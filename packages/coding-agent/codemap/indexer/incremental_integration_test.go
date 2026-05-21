package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIncrementalIntegrationFullScan(t *testing.T) {
	fixture := filepath.Join("..", "testdata", "repos", "incremental-go")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found:", fixture)
	}

	indexer := NewIndexer(fixture)
	candidates, err := indexer.DiscoverFiles()
	if err != nil {
		t.Fatalf("DiscoverFiles failed: %v", err)
	}

	// Should find only .go files outside vendor/
	var paths []string
	for _, e := range candidates {
		paths = append(paths, e.Path)
	}

	// Must include these files
	wantPaths := []string{
		"pkg/math.go",
		"pkg/math_v2.go",
		"cmd/main.go",
	}
	for _, want := range wantPaths {
		found := false
		for _, got := range paths {
			if strings.HasSuffix(got, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find %q in candidates; got %v", want, paths)
		}
	}

	// Must NOT include vendor/ files
	for _, p := range paths {
		if strings.Contains(p, "vendor/") {
			t.Errorf("vendor file %q should be excluded", p)
		}
		if !strings.HasSuffix(p, ".go") {
			t.Errorf("non-Go file %q should not be a parse candidate", p)
		}
	}
}

func TestIncrementalIntegrationReindexUnchanged(t *testing.T) {
	fixture := filepath.Join("..", "testdata", "repos", "incremental-go")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found:", fixture)
	}

	// Copy fixture to temp dir for mutation
	tmpDir := t.TempDir()
	if err := copyDir(fixture, tmpDir); err != nil {
		t.Fatal(err)
	}

	indexer := NewIndexer(tmpDir)

	// First full scan
	firstScan, err := indexer.DiscoverFiles()
	if err != nil {
		t.Fatalf("first DiscoverFiles failed: %v", err)
	}
	if len(firstScan) == 0 {
		t.Fatal("first scan returned no files")
	}

	// Build prev snapshot map
	prevSnapshot := make(map[string]string)
	for _, e := range firstScan {
		prevSnapshot[e.Path] = e.Hash
	}

	// Second scan without changes — all should be unchanged
	secondScan, err := indexer.DiscoverFiles()
	if err != nil {
		t.Fatalf("second DiscoverFiles failed: %v", err)
	}

	// Build current snapshot
	currSnapshot := make(map[string]string)
	for _, e := range secondScan {
		currSnapshot[e.Path] = e.Hash
	}

	// Classify
	ds := ClassifyFiles(currSnapshot, prevSnapshot)

	if len(ds.Changed) > 0 {
		t.Errorf("expected no changed files after re-scan; got %v", ds.Changed)
	}
	if len(ds.New) > 0 {
		t.Errorf("expected no new files; got %v", ds.New)
	}
	if len(ds.Deleted) > 0 {
		t.Errorf("expected no deleted files; got %v", ds.Deleted)
	}
	if len(ds.Unchanged) != len(firstScan) {
		t.Errorf("expected all %d files unchanged; got %d", len(firstScan), len(ds.Unchanged))
	}
}

func TestIncrementalIntegrationReindexWithChange(t *testing.T) {
	fixture := filepath.Join("..", "testdata", "repos", "incremental-go")
	if _, err := os.Stat(fixture); os.IsNotExist(err) {
		t.Skip("fixture not found:", fixture)
	}

	tmpDir := t.TempDir()
	if err := copyDir(fixture, tmpDir); err != nil {
		t.Fatal(err)
	}

	indexer := NewIndexer(tmpDir)

	// First scan
	firstScan, err := indexer.DiscoverFiles()
	if err != nil {
		t.Fatal(err)
	}

	prevSnapshot := make(map[string]string)
	for _, e := range firstScan {
		prevSnapshot[e.Path] = e.Hash
	}

	// Mutate one file: add a comment to math.go
	mathPath := filepath.Join(tmpDir, "pkg", "math.go")
	orig, err := os.ReadFile(mathPath)
	if err != nil {
		t.Fatal(err)
	}
	modified := append(orig, []byte("// modified\n")...)
	if err := os.WriteFile(mathPath, modified, 0644); err != nil {
		t.Fatal(err)
	}

	// Second scan
	secondScan, err := indexer.DiscoverFiles()
	if err != nil {
		t.Fatal(err)
	}

	currSnapshot := make(map[string]string)
	for _, e := range secondScan {
		currSnapshot[e.Path] = e.Hash
	}

	ds := ClassifyFiles(currSnapshot, prevSnapshot)

	// Only the modified math.go should be changed
	if len(ds.Changed) == 0 {
		t.Error("expected exactly one changed file (math.go); got none")
	}
	if len(ds.Changed) > 1 {
		t.Errorf("expected only math.go changed; got %v", ds.Changed)
	}

	// Verify the changed file is pkg/math.go
	if len(ds.Changed) == 1 && !strings.Contains(ds.Changed[0].Path, "math.go") {
		t.Errorf("expected math.go to be changed; got %v", ds.Changed[0].Path)
	}

	// Other files should be unchanged
	unchangedPaths := make(map[string]bool)
	for _, e := range ds.Unchanged {
		unchangedPaths[e.Path] = true
	}
	for _, e := range firstScan {
		if strings.Contains(e.Path, "math.go") {
			continue // this one changed
		}
		if !unchangedPaths[e.Path] {
			t.Errorf("file %q should be unchanged but is missing from unchanged set", e.Path)
		}
	}
}

// copyDir recursively copies src to dst.
func copyDir(src, dst string) error {
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
