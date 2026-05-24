package cli

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codrut/packages/coding-agent/codemap/git"
	"codrut/packages/coding-agent/codemap/indexer"
	"codrut/packages/coding-agent/codemap/store"
)

// RunIndex runs the "index" command and returns an exit code.
func RunIndex(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	// Parse flags.
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	dbPathFlag := fs.String("db", "", "Path to SQLite database (optional)")
	if err := fs.Parse(args); err != nil {
		WriteErrorEnvelope(w, "index", err.Error(), EmptyMeta())
		return 2 // validation
	}

	// Validate repoRoot.
	if repoRoot == "" {
		WriteErrorEnvelope(w, "index", "repo path required", EmptyMeta())
		return 2
	}
	if _, err := os.Stat(repoRoot); err != nil {
		WriteErrorEnvelope(w, "index", "repo path: "+err.Error(), EmptyMeta())
		return 1 // runtime
	}

	dbPath, err := ResolveDBPath(*dbPathFlag, repoRoot)
	if err != nil {
		WriteErrorEnvelope(w, "index", "resolve db path: "+err.Error(), EmptyMeta())
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		WriteErrorEnvelope(w, "index", "open db: "+err.Error(), EmptyMeta())
		return 1
	}
	defer db.Close()

	// Run migrations.
	if err := store.Migrate(ctx, db.DB); err != nil {
		WriteErrorEnvelope(w, "index", "migrate: "+err.Error(), EmptyMeta())
		return 1
	}

	// Compute latest snapshot head for staleness.
	meta, _ := store.GetLatestSnapshotMeta(ctx, db.DB)
	prevFiles := map[string]string{}
	if meta.SnapshotID > 0 {
		prevFiles, _ = getFileHashesForSnapshot(ctx, db.DB, meta.SnapshotID)
	}

	// Run indexer.
	req := indexer.IndexRequest{
		RepoRoot:   repoRoot,
		SnapshotID: meta.SnapshotID + 1,
		PrevFiles:  prevFiles,
	}
	result, err := indexer.RunIndex(ctx, req)
	if err != nil {
		WriteErrorEnvelope(w, "index", "index: "+err.Error(), EmptyMeta())
		return 1
	}

	// No-op: no files changed, new, or deleted — skip snapshot creation.
	if len(result.Entries) == 0 && result.FilesDeleted == 0 {
		envelope := NewEnvelope("index", true, IndexData{
			SnapshotID:   meta.SnapshotID,
			FilesScanned: result.FilesScanned,
			FilesParsed:  result.FilesParsed,
			SymbolsFound: result.SymbolsFound,
			ParseErrors:  result.ParseErrors,
			Evidence:     nil,
		}, nil, Meta{
			SnapshotID: meta.SnapshotID,
			HeadRef:    meta.HeadRef,
			IndexedAt:  meta.IndexedAt,
			IsStale:    false,
		})
		out, _ := envelope.Encode()
		_, _ = w.Write(out)
		return 0
	}

	// Collect paths from changed/new/deleted entries for cache invalidation.
	var affectedPaths []string
	for _, e := range result.Entries {
		affectedPaths = append(affectedPaths, e.Path)
	}
	for _, e := range result.Deleted {
		affectedPaths = append(affectedPaths, e.Path)
	}
	// Invalidate stale cache entries for affected files before writing new data.
	go func() {
		InvalidateCacheForFilePaths(context.Background(), db.DB, affectedPaths)
	}()

	// Persist snapshot + files + symbols under transaction.
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		WriteErrorEnvelope(w, "index", "begin tx: "+err.Error(), EmptyMeta())
		return 1
	}
	defer tx.Rollback()

	headRef := getHeadRef(repoRoot)
	snapshotID, err := store.BeginSnapshot(ctx, tx, repoRoot, headRef)
	if err != nil {
		WriteErrorEnvelope(w, "index", "begin snapshot: "+err.Error(), EmptyMeta())
		return 1
	}

	indexedAt := time.Now().UTC().Format(time.RFC3339)

	// Build git client once; nil if repo is not git-tracked.
	gitClient := git.NewClient(repoRoot)
	isGitRepo := gitClient.IsRepo()

	// Persist files, symbols, and history links.
	for _, e := range result.Entries {
		fileID, err := store.UpsertFile(ctx, tx, repoRoot, e.Path, "go", e.Hash, snapshotID)
		if err != nil {
			WriteErrorEnvelope(w, "index", "upsert file: "+err.Error(), EmptyMeta())
			return 1
		}
		if e.Symbols != nil && len(e.Symbols) > 0 {
			symbolIDs, err := store.ReplaceFileSymbols(ctx, tx, fileID, e.Symbols)
			if err != nil {
				WriteErrorEnvelope(w, "index", "replace symbols: "+err.Error(), EmptyMeta())
				return 1
			}
			// Build name→ID map for edge resolution.
			nameToID := buildSymbolNameMap(ctx, tx, fileID, e.Symbols)
			if len(e.Edges) > 0 {
				resolved := resolveEdges(e.Edges, nameToID)
				if err := store.UpsertEdges(ctx, tx, resolved); err != nil {
					slog.Warn("upsert edges", "file", e.Path, "err", err)
				}
			}
			if err := indexGitHistoryForFile(ctx, tx, gitClient, isGitRepo, e, symbolIDs, snapshotID, prevFiles, indexedAt); err != nil {
				WriteErrorEnvelope(w, "index", "index git history: "+err.Error(), EmptyMeta())
				return 1
			}
		}
	}

	if err := store.FinalizeSnapshot(ctx, tx, snapshotID, result.FilesScanned, result.FilesParsed, result.SymbolsFound, result.ParseErrors); err != nil {
		WriteErrorEnvelope(w, "index", "finalize: "+err.Error(), EmptyMeta())
		return 1
	}

	if err := tx.Commit(); err != nil {
		WriteErrorEnvelope(w, "index", "commit: "+err.Error(), EmptyMeta())
		return 1
	}

	// Cache warm: background cache pre-computation for top hot symbols.
	// Fail-soft — errors are logged but do not affect the index response.
	go func() {
		store.WarmCacheForSnapshot(context.Background(), db.DB, 100, 3, 3)
	}()

	// Emit JSON envelope.
	metaOut, _ := store.GetLatestSnapshotMeta(ctx, db.DB)
	envelope := NewEnvelope("index", true, IndexData{
		SnapshotID:   metaOut.SnapshotID,
		FilesScanned: result.FilesScanned,
		FilesParsed:  result.FilesParsed,
		SymbolsFound: result.SymbolsFound,
		ParseErrors:  result.ParseErrors,
		Evidence:     nil, // no per-file evidence in summary response
	}, nil, Meta{
		SnapshotID: metaOut.SnapshotID,
		HeadRef:    metaOut.HeadRef,
		IndexedAt:  metaOut.IndexedAt,
		IsStale:    false,
	})
	out, _ := envelope.Encode()
	_, _ = w.Write(out)
	return 0
}

// indexGitHistoryForFile wires real git commit history to symbols for one file.
// When the repo is tracked by git, it queries the file's commit log, fetches
// diff hunks per commit, classifies each symbol's link strength, and writes the
// commit + symbol_commit rows. When the repo is not a git repo (or the file has
// no history) it falls back to a single synthetic commit so that history queries
// still return meaningful results.
func indexGitHistoryForFile(
	ctx context.Context,
	tx *sql.Tx,
	gitClient *git.Client,
	isGitRepo bool,
	e indexer.FileEntry,
	symbolIDs []int64,
	snapshotID int64,
	prevFiles map[string]string,
	indexedAt string,
) error {
	changeType := classifyChangeType(e.Path, prevFiles)
	if isGitRepo {
		commits, err := gitClient.FileHistory(e.Path, 50)
		if err != nil {
			slog.Warn("git history for file", "path", e.Path, "err", err)
			// fall through to synthetic fallback
		} else if len(commits) > 0 {
			for _, c := range commits {
				// Upsert commit record (INSERT OR IGNORE handles duplicates).
				if err := store.EnsureCommit(ctx, tx, c.Hash, c.Author, c.Date, c.Message); err != nil {
					slog.Warn("ensure commit", "hash", c.Hash, "err", err)
					continue
				}
				// Fetch diff hunks for this commit.
				hunks, err := gitClient.FileDiffHunks(c.Hash, e.Path)
				if err != nil {
					slog.Warn("git diff hunks", "hash", c.Hash, "path", e.Path, "err", err)
					continue
				}
				for i, sym := range e.Symbols {
					symbolID := symbolIDs[i]
					strength := indexer.ClassifyLink(indexer.SymbolRange{
						Name:      sym.Name,
						StartLine: sym.StartLine,
						EndLine:   sym.EndLine,
					}, hunks)
					if err := store.UpsertSymbolCommit(ctx, tx, symbolID, c.Hash, string(strength), changeType); err != nil {
						slog.Warn("upsert symbol commit", "symbolID", symbolID, "err", err)
					}
				}
			}
			return nil
		}
	}
	// Synthetic fallback: single synthetic commit for non-git repos or files
	// with no history. This preserves the existing contract that every indexed
	// symbol has at least one history entry.
	return writeSyntheticCommit(ctx, tx, e, symbolIDs, snapshotID, changeType, indexedAt)
}

// writeSyntheticCommit creates a single synthetic commit anchored to the file's
// snapshot and assigns "strong" link strength to all symbols.
// Used as fallback when real git history is unavailable.
func writeSyntheticCommit(
	ctx context.Context,
	tx *sql.Tx,
	e indexer.FileEntry,
	symbolIDs []int64,
	snapshotID int64,
	changeType string,
	indexedAt string,
) error {
	// Synthetic hash is deterministic so re-indexing the same file always
	// hits INSERT OR IGNORE and does not duplicate commit rows.
	commitHash := syntheticCommitHash(snapshotID, e.Path)
	if err := store.EnsureCommit(ctx, tx, commitHash, "codemap-indexer", indexedAt, "index symbols for "+e.Path); err != nil {
		return err
	}
	for _, symbolID := range symbolIDs {
		if err := store.UpsertSymbolCommit(ctx, tx, symbolID, commitHash, "strong", changeType); err != nil {
			return err
		}
	}
	return nil
}

// classifyChangeType derives the change type from the previous snapshot.
// add = file is new in this snapshot; modify = file existed previously.
func classifyChangeType(path string, prevFiles map[string]string) string {
	if _, ok := prevFiles[path]; ok {
		return "modify"
	}
	return "add"
}

// getHeadRef reads the current git HEAD ref for the repo root.
func getHeadRef(repoRoot string) string {
	headFile := filepath.Join(repoRoot, ".git", "HEAD")
	data, err := os.ReadFile(headFile)
	if err != nil {
		return ""
	}
	ref := strings.TrimSpace(string(data))
	if strings.HasPrefix(ref, "ref: ") {
		return strings.TrimPrefix(ref, "ref: ")
	}
	return ref
}

// syntheticCommitHash produces a deterministic hash from path and snapshot.
// Kept only for the synthetic fallback path; not used when real git is available.
func syntheticCommitHash(snapshotID int64, path string) string {
	// Simple fold of path+snapshot into a hex string, matching the prior behavior.
	h := uint64(2166136261)
	for _, c := range path {
		h = h*16777619 ^ uint64(c)
	}
	h = h*16777619 ^ uint64(snapshotID)
	return fmt.Sprintf("%016x", h)
}

// getFileHashesForSnapshot returns a map of path -> hash for the given snapshot.
func getFileHashesForSnapshot(ctx context.Context, db *sql.DB, snapshotID int64) (map[string]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT path, hash FROM files WHERE snapshot_id = ?`, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var path, hash string
		if err := rows.Scan(&path, &hash); err != nil {
			return nil, err
		}
		m[path] = hash
	}
	return m, rows.Err()
}

// InvalidateCacheForFilePaths removes cached impact entries for symbols in the
// given file paths. This is called after a re-index to clear stale cache entries
// for files that were changed or deleted.
func InvalidateCacheForFilePaths(ctx context.Context, db *sql.DB, paths []string) {
	if len(paths) == 0 {
		return
	}
	// Collect symbol IDs for all affected files.
	var symbolIDs []int64
	for _, path := range paths {
		ids, err := getSymbolIDsForFile(ctx, db, path)
		if err != nil {
			slog.Warn("InvalidateCacheForFilePaths: getSymbolIDsForFile", "path", path, "err", err)
			continue
		}
		symbolIDs = append(symbolIDs, ids...)
	}
	for _, symID := range symbolIDs {
		if err := store.InvalidateCacheForSymbol(ctx, db, symID); err != nil {
			slog.Warn("InvalidateCacheForFilePaths: InvalidateCacheForSymbol",
				"symbolID", symID, "err", err)
		}
	}
}

// getSymbolIDsForFile returns the symbol IDs for a file in the most recent snapshot.
func getSymbolIDsForFile(ctx context.Context, db *sql.DB, path string) ([]int64, error) {
	meta, err := store.GetLatestSnapshotMeta(ctx, db)
	if err != nil || meta.SnapshotID == 0 {
		return nil, err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT s.id FROM symbols s
		 JOIN files f ON f.id = s.file_id
		 WHERE f.path = ? AND f.snapshot_id = ?`,
		path, meta.SnapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// buildSymbolNameMap creates a lookup from (Name, Recv) → symbolID for the
// given file. This is used to resolve EdgeIntent symbols to concrete DB IDs.
func buildSymbolNameMap(ctx context.Context, tx *sql.Tx, fileID int64, symbols []indexer.Symbol) map[string]int64 {
	nameToID := make(map[string]int64)
	// We need to query IDs from the just-inserted symbols.
	// Since ReplaceFileSymbols clears and re-inserts, we can query by file_id.
	rows, err := tx.QueryContext(ctx,
		`SELECT id, name, kind FROM symbols WHERE file_id = ?`, fileID)
	if err != nil {
		return nameToID
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name, kind string
		if err := rows.Scan(&id, &name, &kind); err != nil {
			continue
		}
		// Key by simple name for top-level funcs.
		key := name
		// For methods, also key by receiver + name.
		if kind == "method" {
			// We need the receiver from the original symbols list.
			for _, s := range symbols {
				if s.Name == name && s.Kind == "method" {
					methodKey := s.Recv + "." + name
					nameToID[methodKey] = id
					break
				}
			}
		}
		nameToID[key] = id
	}
	return nameToID
}

// resolveEdges converts a slice of EdgeIntent (with Name/Recv keys) to
// ResolvedEdge (with concrete symbol IDs) using the nameToID lookup.
// Symbols whose IDs cannot be resolved are skipped.
func resolveEdges(edges []indexer.EdgeIntent, nameToID map[string]int64) []store.ResolvedEdge {
	var resolved []store.ResolvedEdge
	for _, e := range edges {
		fromID := resolveSymbolID(e.From, nameToID)
		toID := resolveSymbolID(e.To, nameToID)
		if fromID == 0 || toID == 0 {
			continue
		}
		resolved = append(resolved, store.ResolvedEdge{
			FromSymbolID: fromID,
			ToSymbolID:   toID,
			EdgeType:     e.Kind,
		})
	}
	return resolved
}

// resolveSymbolID maps an EdgeIntent's SymbolKey to a concrete symbol ID.
// It tries qualified key (Recv.Name) first, then bare Name.
func resolveSymbolID(key indexer.SymbolKey, nameToID map[string]int64) int64 {
	if key.Name == "" {
		return 0
	}
	// Try qualified key for methods.
	if key.Recv != "" {
		if id, ok := nameToID[key.Recv+"."+key.Name]; ok {
			return id
		}
	}
	// Fall back to bare name.
	if id, ok := nameToID[key.Name]; ok {
		return id
	}
	return 0
}
