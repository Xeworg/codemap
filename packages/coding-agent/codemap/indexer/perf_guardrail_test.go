package indexer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// perfFixture is the same helper used by perf_benchmark_test.go.
// Defined here so this test file is self-contained.
func perfFixturePerf(repo string) string {
	return filepath.Join("..", "testdata", "repos", repo)
}

func copyDirPerfGuard(src, dst string) error {
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

// TestPerfIncrementalScalingRegression asserts that unchanged reindex is
// cheaper than changed reindex, which is cheaper than full scan.
// This is a deterministic regression guard: it compares the number of
// files processed, not wall-clock time, so it is not flaky.
func TestPerfIncrementalScalingRegression(t *testing.T) {
	fixture := perfFixturePerf("incremental-go")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found:", fixture)
	}

	tmpDir := t.TempDir()
	if err := copyDirPerfGuard(fixture, tmpDir); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	indexer := NewIndexer(tmpDir)

	// --- Full scan ---
	discovered, err := indexer.DiscoverFiles()
	if err != nil {
		t.Fatal(err)
	}
	fullCount := len(discovered)

	// Build prev snapshot.
	prevFiles := make(map[string]string, len(discovered))
	for _, e := range discovered {
		prevFiles[e.Path] = e.Hash
	}

	// --- Unchanged reindex ---
	reqUnchanged := IndexRequest{RepoRoot: tmpDir, PrevFiles: prevFiles, SnapshotID: 2}
	resultUnchanged, err := RunIndex(ctx, reqUnchanged)
	if err != nil {
		t.Fatal(err)
	}
	unchangedProcessed := resultUnchanged.FilesParsed

	// Mutate one file for the changed run.
	mathPath := filepath.Join(tmpDir, "pkg", "math.go")
	orig, err := os.ReadFile(mathPath)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(mathPath, append(orig, []byte("// guardrail touch\n")...), 0644)

	// Rebuild prevFiles (hash for math.go is now stale).
	prevFiles2 := make(map[string]string, len(discovered))
	for _, e := range discovered {
		prevFiles2[e.Path] = e.Hash
	}

	// --- One-file changed reindex ---
	reqChanged := IndexRequest{RepoRoot: tmpDir, PrevFiles: prevFiles2, SnapshotID: 3}
	resultChanged, err := RunIndex(ctx, reqChanged)
	if err != nil {
		t.Fatal(err)
	}
	changedProcessed := resultChanged.FilesParsed

	// --- Regression guards ---
	// Guard: unchanged reindex must process FEWER files than full scan.
	// An unchanged reindex should skip all files that haven't changed.
	if unchangedProcessed >= fullCount {
		t.Errorf("regression: unchanged reindex processed %d files, expected < full scan %d",
			unchangedProcessed, fullCount)
	}

	// Guard: changed reindex must process FEWER files than full scan.
	if changedProcessed >= fullCount {
		t.Errorf("regression: changed reindex processed %d files, expected < full scan %d",
			changedProcessed, fullCount)
	}

	// Guard: changed reindex should process AT LEAST as many files as unchanged.
	// (It must parse the changed file(s) at minimum; unchanged skips everything.)
	if changedProcessed < unchangedProcessed {
		t.Errorf("regression: changed reindex processed %d, expected >= unchanged %d",
			changedProcessed, unchangedProcessed)
	}
}

// TestPerfFileDiscoveryExcludesVendor asserts that the incremental indexer
// never walks into vendor/ directories, keeping scan cost bounded.
func TestPerfFileDiscoveryExcludesVendor(t *testing.T) {
	fixture := perfFixturePerf("incremental-go")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found:", fixture)
	}

	tmpDir := t.TempDir()
	if err := copyDirPerfGuard(fixture, tmpDir); err != nil {
		t.Fatal(err)
	}

	indexer := NewIndexer(tmpDir)
	candidates, err := indexer.DiscoverFiles()
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range candidates {
		if strings.Contains(e.Path, "vendor/") {
			t.Errorf("vendor path %q found in candidates; vendor exclusion is broken", e.Path)
		}
	}
}

// TestPerfParseErrorDoesNotAbort asserts that a single parse error
// does not prevent processing of remaining files (fail-soft contract).
func TestPerfParseErrorDoesNotAbort(t *testing.T) {
	fixture := perfFixturePerf("parse-mixed")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found:", fixture)
	}

	tmpDir := t.TempDir()
	if err := copyDirPerfGuard(fixture, tmpDir); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	req := IndexRequest{RepoRoot: tmpDir, PrevFiles: nil, SnapshotID: 1}
	result, err := RunIndex(ctx, req)
	if err != nil {
		t.Fatalf("RunIndex should not return error on parse error: %v", err)
	}

	// The fixture has 2 .go files: one valid, one broken.
	// We should have scanned both, parsed at least the valid one.
	if result.FilesScanned == 0 {
		t.Error("expected FilesScanned > 0")
	}
	if result.FilesParsed == 0 {
		t.Error("regression: expected at least one file parsed despite another being broken")
	}
	if result.ParseErrors == 0 {
		t.Error("expected at least one parse error for broken.go")
	}
}
