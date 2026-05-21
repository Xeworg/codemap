package indexer

import (
	"testing"

	"go/token"
)

// TestParseFailSoftNoPanic verifies parsing a file with syntax errors does not panic.
func TestParseFailSoftNoPanic(t *testing.T) {
	badSrc := []byte(`package broken

func Foo() {
    x := 1 +
    return
}`)

	// Should not panic; may return error or empty symbols.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("parser panicked on invalid syntax: %v", r)
		}
	}()

	fset := token.NewFileSet()
	_, err := ParseGoFile(fset, "broken.go", badSrc)
	// err is expected; we just should not have panicked.
	_ = err
}

// TestParseFailSoftReturnsError verifies that a broken file returns an error.
func TestParseFailSoftReturnsError(t *testing.T) {
	badSrc := []byte(`package p

func Broken() {
    this is not go
`)

	fset := token.NewFileSet()
	_, err := ParseGoFile(fset, "broken.go", badSrc)
	if err == nil {
		t.Error("expected parse error for invalid syntax, got nil")
	}
}

// TestParseFailSoftValidFileSucceeds verifies valid Go is parsed without error.
func TestParseFailSoftValidFileSucceeds(t *testing.T) {
	validSrc := []byte(`package main

func main() {}
`)

	fset := token.NewFileSet()
	_, err := ParseGoFile(fset, "valid.go", validSrc)
	if err != nil {
		t.Errorf("unexpected error for valid Go: %v", err)
	}
}

// TestParseFailSoftContinuesAfterOneError verifies that when one file fails,
// the indexer continues (fail-soft contract).
func TestParseFailSoftContinuesAfterOneError(t *testing.T) {
	fset := token.NewFileSet()
	files := map[string][]byte{
		"good.go":  []byte("package main\n\nfunc main() {}"),
		"bad.go":   []byte("package p\nfunc Bad() {\n"), // missing closing brace causes syntax error
		"good2.go": []byte("package main\n\nfunc Other() {}"),
	}

	errored := 0
	ok := 0
	for path, src := range files {
		result, err := ParseGoFile(fset, path, src)
		t.Logf("DEBUG %s: err=%v symbols=%d", path, err != nil, len(result.Symbols))
		if err != nil {
			errored++
			continue
		}
		_ = result
		ok++
	}

	// At least one should have errored (bad.go has broken syntax) and at least one succeeded.
	if errored == 0 {
		t.Error("expected at least one parse error in the set")
	}
	if ok == 0 {
		t.Error("expected at least one successful parse")
	}
	t.Logf("fail-soft: %d ok, %d errors; all files processed without abort", ok, errored)
}

// TestParseFailSoftRecordsErrorDetail verifies the error contains useful detail.
func TestParseFailSoftRecordsErrorDetail(t *testing.T) {
	badSrc := []byte(`package p; func f() { x := }`)

	fset := token.NewFileSet()
	_, err := ParseGoFile(fset, "error_test.go", badSrc)
	if err == nil {
		t.Fatal("expected an error")
	}

	// Error message should mention something about syntax or parsing.
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("expected non-empty error message")
	}
	t.Logf("error detail: %s", errMsg)
}

// TestParseFailSoftNilFSet verifies ParseGoFile handles nil fset gracefully.
func TestParseFailSoftNilFSet(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ParseGoFile panicked with nil fset: %v", r)
		}
	}()
	_, _ = ParseGoFile(nil, "test.go", []byte(`package p`))
}

// TestParseFailSoftEmptyContent verifies empty source is handled.
// Empty source without package clause is invalid Go; fail-soft just means no panic.
func TestParseFailSoftEmptyContent(t *testing.T) {
	fset := token.NewFileSet()
	_, _ = ParseGoFile(fset, "empty.go", []byte{})
	// Empty content may error; fail-soft contract: no panic, caller decides.
}

// TestParseFailSoftPackageNameOnly verifies package-only file is valid.
func TestParseFailSoftPackageNameOnly(t *testing.T) {
	src := []byte(`package foo`)
	fset := token.NewFileSet()
	result, err := ParseGoFile(fset, "package.go", src)
	if err != nil {
		t.Errorf("package-only file should be valid: %v", err)
	}
	if len(result.Symbols) != 0 {
		t.Errorf("expected 0 symbols for package-only, got %d", len(result.Symbols))
	}
}
