package indexer

import (
	"context"
	"os"
	"path/filepath"

	"go/token"
)

// IndexResult summarizes an index run.
type IndexResult struct {
	FilesScanned int
	FilesChanged int
	FilesNew     int
	FilesDeleted int
	FilesParsed  int
	ParseErrors  int
	SymbolsFound int
	EdgesFound   int
	Entries      []FileEntry // files processed (new/changed) with their symbols
	Errored      bool
}

// IndexRequest describes what to index.
type IndexRequest struct {
	RepoRoot   string
	PrevFiles  map[string]string // path → hash of previous snapshot
	SnapshotID int64
}

// RunIndex orchestrates the full indexing pipeline: file discovery, diff
// classification, and Go AST extraction for changed/new files.
// Parse errors are recorded but do not abort the run (fail-soft contract).
func RunIndex(ctx context.Context, req IndexRequest) (IndexResult, error) {
	indexer := NewIndexer(req.RepoRoot)
	discovered, err := indexer.DiscoverFiles()
	if err != nil {
		return IndexResult{}, err
	}

	result := IndexResult{FilesScanned: len(discovered)}

	// Build current hash map.
	currFiles := make(map[string]string, len(discovered))
	for _, entry := range discovered {
		currFiles[entry.Path] = entry.Hash
	}

	// Classify diffs.
	diff := ClassifyFiles(currFiles, req.PrevFiles)

	result.FilesNew = len(diff.New)
	result.FilesChanged = len(diff.Changed)
	result.FilesDeleted = len(diff.Deleted)

	// Process changed/new files with fail-soft parsing.
	var entries []FileEntry
	for i := range diff.New {
		processParse(req.RepoRoot, &diff.New[i], &result)
		entries = append(entries, diff.New[i])
	}
	for i := range diff.Changed {
		processParse(req.RepoRoot, &diff.Changed[i], &result)
		entries = append(entries, diff.Changed[i])
	}
	result.Entries = entries

	return result, nil
}

// processParse reads a file, parses it, and accumulates symbol/error counts.
// Parse errors are counted but do not abort processing of remaining files.
// repoRoot is used to resolve entry.Path (stored relative to repoRoot) to an
// absolute path for os.ReadFile.
func processParse(repoRoot string, entry *FileEntry, result *IndexResult) {
	path := filepath.Join(repoRoot, entry.Path)
	src, err := os.ReadFile(path)
	if err != nil {
		return
	}
	fset := token.NewFileSet()
	pr, err := ParseGoFile(fset, entry.Path, src)
	if err != nil {
		result.ParseErrors++
		return
	}
	result.SymbolsFound += len(pr.Symbols)
	result.EdgesFound += len(pr.Edges)
	result.FilesParsed++
	entry.Symbols = pr.Symbols
	entry.Edges = pr.Edges
}

// FixturePath returns the path to a fixture directory under testdata.
// Resolves relative to the codemap package root (parent of indexer/).
func FixturePath(repo string) string {
	return filepath.Join("..", "testdata", "repos", repo)
}
