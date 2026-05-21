package indexer

import (
	"go/parser"
	"go/token"
	"testing"
)

// TestGoParserExtractsFuncDecl verifies the parser extracts a simple function declaration.
func TestGoParserExtractsFuncDecl(t *testing.T) {
	src := `package foo

func Hello(name string) int {
	return 42
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error (should not happen with valid source): %v", err)
	}

	symbols := ExtractGoSymbols(f, fset)
	if len(symbols) == 0 {
		t.Fatal("expected at least one symbol, got 0")
	}

	// Find the Hello function.
	var hello *Symbol
	for i := range symbols {
		if symbols[i].Name == "Hello" {
			hello = &symbols[i]
			break
		}
	}
	if hello == nil {
		t.Fatalf("expected symbol 'Hello', got symbols: %v", symbols)
	}
	if hello.Kind != "func" {
		t.Errorf("expected kind 'func', got %q", hello.Kind)
	}
	if hello.StartLine == 0 || hello.EndLine == 0 {
		t.Error("expected non-zero start/end line")
	}
	if hello.StartLine > hello.EndLine {
		t.Errorf("start_line %d should be <= end_line %d", hello.StartLine, hello.EndLine)
	}
}

// TestGoParserExtractsMultipleSymbols verifies parser finds all top-level declarations.
func TestGoParserExtractsMultipleSymbols(t *testing.T) {
	src := `package bar

type MyStruct struct{}

const Pi = 3.14

var X int

func F() {}

type MyInterface interface{}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	symbols := ExtractGoSymbols(f, fset)
	if len(symbols) < 4 {
		t.Errorf("expected at least 4 symbols, got %d: %v", len(symbols), symbols)
	}

	// Collect names to verify all expected are present.
	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}

	for _, want := range []string{"MyStruct", "Pi", "X", "F", "MyInterface"} {
		if !names[want] {
			t.Errorf("expected symbol %q not found in %v", want, symbols)
		}
	}
}

// TestGoParserExtractsLineRanges verifies symbol ranges are consistent.
func TestGoParserExtractsLineRanges(t *testing.T) {
	src := `package p

// A is on line 3.
func A() { // line 4
} // line 5

func B() { // line 7
} // line 8
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	symbols := ExtractGoSymbols(f, fset)
	if len(symbols) < 2 {
		t.Fatalf("expected 2 symbols, got %d", len(symbols))
	}

	// First function should start on line 4.
	if symbols[0].StartLine != 4 {
		t.Errorf("first symbol start_line: expected 4, got %d", symbols[0].StartLine)
	}
	if symbols[0].EndLine < symbols[0].StartLine {
		t.Errorf("symbol EndLine %d < StartLine %d", symbols[0].EndLine, symbols[0].StartLine)
	}
}

// TestGoParserHandlesEmptyPackage verifies parser handles empty source.
func TestGoParserHandlesEmptyPackage(t *testing.T) {
	src := `package empty`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	symbols := ExtractGoSymbols(f, fset)
	if len(symbols) != 0 {
		t.Errorf("expected 0 symbols for empty package, got %d", len(symbols))
	}
}

// TestGoParserHandlesPackageClauseOnly verifies parser handles package clause only.
func TestGoParserHandlesPackageClauseOnly(t *testing.T) {
	src := `package main`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	symbols := ExtractGoSymbols(f, fset)
	if len(symbols) != 0 {
		t.Errorf("expected 0 symbols for package clause only, got %d", len(symbols))
	}
}

// TestGoParserExtractsNestedDecls verifies nested declarations are skipped (top-level only).
func TestGoParserExtractsNestedDecls(t *testing.T) {
	src := `package nested

func Outer() {
	_ = 1 // valid statement inside
}

type T struct {
	Field int
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	symbols := ExtractGoSymbols(f, fset)
	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}

	if names["Outer"] == false {
		t.Error("expected 'Outer' top-level function")
	}
	if names["Inner"] == true {
		t.Error("'Inner' is nested; should not appear in top-level symbols")
	}
	if names["T"] == false {
		t.Error("expected 'T' struct type")
	}
	if names["Field"] == true {
		t.Error("'Field' is inside T; should not appear as top-level symbol")
	}
}

// TestGoParserSetsCorrectKind verifies each symbol kind is correct.
func TestGoParserSetsCorrectKind(t *testing.T) {
	src := `package kinds

type S struct {}
const C = 1
var V int
func F() {}
type I interface{}
type E int
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	symbols := ExtractGoSymbols(f, fset)
	byName := make(map[string]Symbol)
	for _, s := range symbols {
		byName[s.Name] = s
	}

	cases := map[string]string{
		"S": "type", "C": "const", "V": "var", "F": "func",
		"I": "interface", "E": "type",
	}
	for name, wantKind := range cases {
		if s, ok := byName[name]; !ok {
			t.Errorf("symbol %q not found", name)
		} else if s.Kind != wantKind {
			t.Errorf("symbol %q: expected kind %q, got %q", name, wantKind, s.Kind)
		}
	}
}

// TestGoParserNilFile verifies ExtractGoSymbols handles nil file gracefully (returns nil, no panic).
func TestGoParserNilFile(t *testing.T) {
	result := ExtractGoSymbols(nil, nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

// TestGoParserNilFSet verifies ExtractGoSymbols handles nil fset gracefully (returns nil, no panic).
func TestGoParserNilFSet(t *testing.T) {
	src := `package p
func F() {}`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	result := ExtractGoSymbols(f, nil)
	if result != nil {
		t.Errorf("expected nil for nil fset, got %v", result)
	}
}
