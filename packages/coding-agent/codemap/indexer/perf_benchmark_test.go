package indexer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func perfFixture(repo string) string {
	return filepath.Join("..", "testdata", "repos", repo)
}

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

// BenchmarkRunIndexFullScan measures a cold full scan (no PrevFiles).
func BenchmarkRunIndexFullScan(b *testing.B) {
	repo := setupPerfBench(b, "incremental-go")
	req := IndexRequest{
		RepoRoot:   repo,
		PrevFiles:  nil,
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
// Each iteration resets the file to original content, mutates it, runs index, then restores.
// This ensures the benchmarked unit of work is stable across all N iterations.
func BenchmarkRunIndexOneFileChanged(b *testing.B) {
	repo := setupPerfBench(b, "incremental-go")
	ctx := context.Background()

	mathPath := filepath.Join(repo, "pkg", "math.go")
	orig, err := os.ReadFile(mathPath)
	if err != nil {
		b.Fatal("read original math.go:", err)
	}
	// Write original once before the timed loop to prime the baseline.
	if err := os.WriteFile(mathPath, orig, 0644); err != nil {
		b.Fatal("write original math.go:", err)
	}

	// First scan establishes stable prevFiles.
	req := IndexRequest{RepoRoot: repo, PrevFiles: nil, SnapshotID: 1}
	first, err := RunIndex(ctx, req)
	if err != nil {
		b.Fatal("first scan:", err)
	}
	prevFiles := make(map[string]string, len(first.Entries))
	for _, e := range first.Entries {
		prevFiles[e.Path] = e.Hash
	}

	modified := append(orig, []byte("// perf bench touch\n")...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Apply one small change.
		if err := os.WriteFile(mathPath, modified, 0644); err != nil {
			b.Fatal("write modified:", err)
		}

		// Run index with previous hashes.
		req := IndexRequest{RepoRoot: repo, PrevFiles: prevFiles, SnapshotID: 2}
		if _, err := RunIndex(ctx, req); err != nil {
			b.Fatal("index run:", err)
		}

		// Restore original content so the next iteration starts from the same state.
		if err := os.WriteFile(mathPath, orig, 0644); err != nil {
			b.Fatal("restore original:", err)
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
