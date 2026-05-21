package indexer

import (
	"context"
	"os"
	"testing"

	"go/token"
)

// TestParseMixedFixture verifies the parse-mixed fixture has both valid and broken files,
// and that the parser processes them with correct fail-soft behavior.
func TestParseMixedFixture(t *testing.T) {
	fixture := FixturePath("parse-mixed")
	files := []string{
		"valid.go",
		"broken.go",
	}

	succeeded := 0
	errored := 0

	for _, file := range files {
		src, err := os.ReadFile(fixture + "/" + file)
		if err != nil {
			t.Fatalf("failed to read fixture %s: %v", file, err)
		}

		fset := token.NewFileSet()
		_, err = ParseGoFile(fset, file, src)
		if err != nil {
			errored++
			t.Logf("%s: parse error (expected for broken): %v", file, err)
		} else {
			succeeded++
			t.Logf("%s: parsed successfully", file)
		}
	}

	// At least one should parse successfully and at least one should fail.
	if succeeded == 0 {
		t.Error("expected at least one valid file in parse-mixed fixture")
	}
	if errored == 0 {
		t.Error("expected at least one broken file in parse-mixed fixture")
	}
	t.Logf("parse-mixed fixture: %d ok, %d errors; fail-soft contract maintained", succeeded, errored)
}

// TestParseMixedFixtureAllFilesProcessed verifies the parser does not abort on broken files.
func TestParseMixedFixtureAllFilesProcessed(t *testing.T) {
	fixture := FixturePath("parse-mixed")
	files := []string{"valid.go", "broken.go"}

	fset := token.NewFileSet()
	total := 0
	for _, file := range files {
		src, err := os.ReadFile(fixture + "/" + file)
		if err != nil {
			continue
		}
		_, _ = ParseGoFile(fset, file, src) // error is acceptable; continue.
		total++
	}

	if total != len(files) {
		t.Errorf("expected all %d files processed, got %d", len(files), total)
	}
}

// TestParseMixedValidGoExtractsSymbols verifies the valid file produces symbols.
func TestParseMixedValidGoExtractsSymbols(t *testing.T) {
	fixture := FixturePath("parse-mixed")
	src, err := os.ReadFile(fixture + "/valid.go")
	if err != nil {
		t.Fatalf("failed to read valid.go: %v", err)
	}

	fset := token.NewFileSet()
	pr, err := ParseGoFile(fset, "valid.go", src)
	if err != nil {
		t.Fatalf("valid.go should parse successfully: %v", err)
	}

	if len(pr.Symbols) == 0 {
		t.Error("expected symbols from valid.go")
	}

	// Verify we have the expected top-level symbols.
	names := make(map[string]bool)
	for _, s := range pr.Symbols {
		names[s.Name] = true
	}

	if !names["Valid"] {
		t.Error("expected 'Valid' function symbol")
	}
	if !names["AnotherValid"] {
		t.Error("expected 'AnotherValid' function symbol")
	}
	if !names["MyStruct"] {
		t.Error("expected 'MyStruct' type symbol")
	}
}

// TestRunIndexProcessParseCallsParseGoFileWithFset verifies the fixed processParse
// path: real token.FileSet allocated per file, and path resolved against
// RepoRoot. Regression test for reviewer block #1 and #2 from PR3 review.
func TestRunIndexProcessParseCallsParseGoFileWithFset(t *testing.T) {
	fixture := FixturePath("parse-mixed")

	req := IndexRequest{
		RepoRoot:   fixture,
		PrevFiles:  map[string]string{}, // all files are "new"
		SnapshotID: 1,
	}

	result, err := RunIndex(context.Background(), req)
	if err != nil {
		t.Fatalf("RunIndex failed: %v", err)
	}

	// We expect FilesScanned to reflect discovered .go files.
	if result.FilesScanned == 0 {
		t.Error("expected at least one file scanned in parse-mixed fixture")
	}

	// valid.go should parse successfully.
	if result.FilesParsed == 0 {
		t.Errorf("expected at least one file parsed (valid.go), got FilesParsed=%d", result.FilesParsed)
	}

	// broken.go should produce a parse error but must NOT abort the run.
	if result.ParseErrors == 0 {
		t.Error("expected at least one parse error (broken.go)")
	}

	// Symbols should be found from valid.go.
	if result.SymbolsFound == 0 {
		t.Error("expected symbols from valid.go")
	}

	// Run must not have errored.
	if result.Errored {
		t.Error("RunIndex should not set Errored=true on fail-soft parse errors")
	}

	t.Logf("RunIndex result: FilesScanned=%d FilesParsed=%d ParseErrors=%d SymbolsFound=%d",
		result.FilesScanned, result.FilesParsed, result.ParseErrors, result.SymbolsFound)
}
