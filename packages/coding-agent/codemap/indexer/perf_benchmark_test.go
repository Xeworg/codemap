package indexer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// FixturePath returns the path to a fixture under testdata/repos/.
func perfFixture(repo string) string {
	return filepath.Join("..", "testdata", "repos", repo)
}

// setupPerfBench copies a fixture to a temp dir, returning the path.
// The returned path is stable for the lifetime of the test.
func setupPerfBench(b *testing.B, repo string) string {
	fixture := perfFixture(repo)
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		b.Skip("fixture not found:", fixture)
	}
	tmpDir := b.TempDir()
	if err := copyDir(fixture, tmpDir); err != nil {
		b.Fatal(err)
	}
	return tmpDir
}

// copyDir mirrors the same helper used by incremental_integration_test.go.
func copyDirPerfBench(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}

// BenchmarkRunIndexFullScan measures a cold full scan (no PrevFiles).
func BenchmarkRunIndexFullScan(b *testing.B) {
	repo := setupPerfBench(b, "incremental-go")
	req := IndexRequest{
		RepoRoot:   repo,
		PrevFiles:  nil, // full scan: no previous snapshot
		SnapshotID: 1,
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = RunIndex(ctx, req)
	}
}

// BenchmarkRunIndexUnchangedReindex measures reindexing when no file changed.
func BenchmarkRunIndexUnchangedReindex(b *testing.B) {
	repo := setupPerfBench(b, "incremental-go")
	ctx := context.Background()

	// First scan to populate PrevFiles.
	req := IndexRequest{RepoRoot: repo, PrevFiles: nil, SnapshotID: 1}
	first, err := RunIndex(ctx, req)
	if err != nil {
		b.Fatal(err)
	}
	prevFiles := make(map[string]string, len(first.Entries))
	for _, e := range first.Entries {
		prevFiles[e.Path] = e.Hash
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := IndexRequest{RepoRoot: repo, PrevFiles: prevFiles, SnapshotID: 2}
		_, _ = RunIndex(ctx, req)
	}
}

// BenchmarkRunIndexOneFileChanged measures reindexing when exactly one file is modified.
func BenchmarkRunIndexOneFileChanged(b *testing.B) {
	repo := setupPerfBench(b, "incremental-go")
	ctx := context.Background()

	// First scan.
	req := IndexRequest{RepoRoot: repo, PrevFiles: nil, SnapshotID: 1}
	first, err := RunIndex(ctx, req)
	if err != nil {
		b.Fatal(err)
	}
	prevFiles := make(map[string]string, len(first.Entries))
	for _, e := range first.Entries {
		prevFiles[e.Path] = e.Hash
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Mutate one file.
		mathPath := filepath.Join(repo, "pkg", "math.go")
		orig, _ := os.ReadFile(mathPath)
		modified := append(orig, []byte("// perf bench touch\n")...)
		os.WriteFile(mathPath, modified, 0644)
		req := IndexRequest{RepoRoot: repo, PrevFiles: prevFiles, SnapshotID: 2}
		result, _ := RunIndex(ctx, req)
		// Restore prevFiles hash for next iteration (don't carry forward mutations).
		prevFiles = make(map[string]string, len(first.Entries))
		for _, e := range first.Entries {
			prevFiles[e.Path] = e.Hash
		}
		// Mark math.go as having changed from the perspective of next run.
		for _, e := range result.Entries {
			if e.Hash != first.Entries[0].Hash {
				prevFiles[e.Path] = e.Hash
			}
		}
	}
}

// BenchmarkDiscoverFiles measures the file discovery walk cost.
func BenchmarkDiscoverFiles(b *testing.B) {
	repo := setupPerfBench(b, "incremental-go")
	indexer := NewIndexer(repo)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = indexer.DiscoverFiles()
	}
}
