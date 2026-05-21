package cli

import (
	"context"
	"database/sql"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"

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

	// Persist files and their symbols.
	for _, e := range result.Entries {
		fileID, err := store.UpsertFile(ctx, tx, repoRoot, e.Path, "go", e.Hash, snapshotID)
		if err != nil {
			WriteErrorEnvelope(w, "index", "upsert file: "+err.Error(), EmptyMeta())
			return 1
		}
		if e.Symbols != nil && len(e.Symbols) > 0 {
			_, err = store.ReplaceFileSymbols(ctx, tx, fileID, e.Symbols)
			if err != nil {
				WriteErrorEnvelope(w, "index", "replace symbols: "+err.Error(), EmptyMeta())
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
