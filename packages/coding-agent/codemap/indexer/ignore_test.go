package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultExclusions(t *testing.T) {
	indexer := &Indexer{}

	// These patterns should be excluded by default
	excluded := []string{
		".git/config",
		"node_modules/package/index.js",
		"vendor/foo/bar.go",
		"dist/build.js",
		"build/output.o",
		".codemap/db",
	}

	for _, path := range excluded {
		if !indexer.shouldExclude(path) {
			t.Errorf("expected path %q to be excluded by default, but it was not", path)
		}
	}

	// These should NOT be excluded
	included := []string{
		"cmd/main.go",
		"pkg/foo/bar.go",
		"internal/indexer/walk.go",
		"src/index.js",
	}

	for _, path := range included {
		if indexer.shouldExclude(path) {
			t.Errorf("expected path %q to NOT be excluded, but it was", path)
		}
	}
}

func TestCodemapignoreMatching(t *testing.T) {
	indexer := &Indexer{
		customRules: []string{"*.tmp", "testdata/**", "*.log"},
	}

	// Custom rule: *.tmp should be excluded
	if !indexer.shouldExclude("backup.tmp") {
		t.Error("expected *.tmp pattern to exclude backup.tmp")
	}

	// Custom rule: testdata/** should exclude nested paths
	if !indexer.shouldExclude("testdata/fixtures/repo/main.go") {
		t.Error("expected testdata/** to exclude testdata/fixtures/repo/main.go")
	}

	// Custom rule: *.log should exclude logs
	if !indexer.shouldExclude("app.log") {
		t.Error("expected *.log pattern to exclude app.log")
	}

	// Go files should NOT be excluded by custom rules
	if indexer.shouldExclude("cmd/main.go") {
		t.Error("expected Go files to NOT be excluded by custom rules")
	}
}

func TestCodemapignoreFileLoading(t *testing.T) {
	// Create temp dir with .codemapignore file
	tmpDir := t.TempDir()
	ignorePath := filepath.Join(tmpDir, ".codemapignore")

	ignoreContent := "*.bak\nsecret/**\n!secret/.keep\n"
	if err := os.WriteFile(ignorePath, []byte(ignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	indexer := NewIndexer(tmpDir)

	// Verify custom exclusions loaded
	if !indexer.shouldExclude("old.bak") {
		t.Error("expected *.bak from .codemapignore to exclude old.bak")
	}

	// But .keep should NOT be excluded (negation rule)
	if indexer.shouldExclude("secret/.keep") {
		t.Error("expected !secret/.keep negation to NOT exclude secret/.keep")
	}
}

func TestGoFilesOnly(t *testing.T) {
	indexer := &Indexer{}

	files := []string{
		"cmd/main.go",
		"pkg/util/parser.go",
		"internal/types/types.go",
	}

	for _, f := range files {
		if !indexer.isParseCandidate(f) {
			t.Errorf("expected %q to be a Go parse candidate", f)
		}
	}

	nonGoFiles := []string{
		"pkg/util/parser.js",
		"pkg/util/parser.ts",
		"README.md",
		"cmd/main.py",
		"script.sh",
	}

	for _, f := range nonGoFiles {
		if indexer.isParseCandidate(f) {
			t.Errorf("expected %q to NOT be a parse candidate", f)
		}
	}
}
