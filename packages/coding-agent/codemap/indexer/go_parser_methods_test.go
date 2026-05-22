package indexer

import (
	"go/parser"
	"go/token"
	"testing"
)

// TestExtractGoSymbols_IncludesMethods verifies that method declarations are
// extracted with Kind="method" and a non-empty Recv field.
func TestExtractGoSymbols_IncludesMethods(t *testing.T) {
	src := `package foo

type T struct{}

func (t T) Method() {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	symbols := ExtractGoSymbols(f, fset)
	if len(symbols) == 0 {
		t.Fatal("expected at least one symbol, got 0")
	}

	var method *Symbol
	for i := range symbols {
		if symbols[i].Name == "Method" {
			method = &symbols[i]
			break
		}
	}
	if method == nil {
		t.Fatalf("expected symbol 'Method', got: %v", symbols)
	}
	if method.Kind != "method" {
		t.Errorf("expected kind 'method', got %q", method.Kind)
	}
	if method.Recv == "" {
		t.Error("expected non-empty Recv for method symbol")
	}
}

// TestExtractGoSymbols_IncludesInit verifies that init functions are included
// with Kind="func" (not skipped).
func TestExtractGoSymbols_IncludesInit(t *testing.T) {
	src := `package foo

func init() {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	symbols := ExtractGoSymbols(f, fset)
	if len(symbols) == 0 {
		t.Fatal("expected at least one symbol for init, got 0")
	}

	var initSym *Symbol
	for i := range symbols {
		if symbols[i].Name == "init" {
			initSym = &symbols[i]
			break
		}
	}
	if initSym == nil {
		t.Fatalf("expected symbol 'init', got: %v", symbols)
	}
	if initSym.Kind != "func" {
		t.Errorf("expected kind 'func' for init, got %q", initSym.Kind)
	}
}

// TestExtractGoSymbols_MethodWithPointerReceiver verifies that methods with
// pointer receivers have Recv prefixed with "*".
func TestExtractGoSymbols_MethodWithPointerReceiver(t *testing.T) {
	src := `package foo

type T struct{}

func (t *T) PointerMethod() {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	symbols := ExtractGoSymbols(f, fset)
	var method *Symbol
	for i := range symbols {
		if symbols[i].Name == "PointerMethod" {
			method = &symbols[i]
			break
		}
	}
	if method == nil {
		t.Fatal("expected 'PointerMethod' symbol")
	}
	if method.Kind != "method" {
		t.Errorf("expected kind 'method', got %q", method.Kind)
	}
	if method.Recv != "*T" {
		t.Errorf("expected Recv '*T', got %q", method.Recv)
	}
}

// TestExtractGoSymbols_MultipleInitFuncs verifies that multiple init functions
// in a file are each extracted separately.
func TestExtractGoSymbols_MultipleInitFuncs(t *testing.T) {
	src := `package foo

func init() { _ = 1 }
func init() { _ = 2 }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	symbols := ExtractGoSymbols(f, fset)
	initCount := 0
	for _, s := range symbols {
		if s.Name == "init" {
			initCount++
		}
	}
	if initCount != 2 {
		t.Errorf("expected 2 init symbols, got %d", initCount)
	}
}

// TestExtractGoSymbols_ReceiverNamingVariants verifies that receiver naming
// handles value, pointer, and simple package-qualified receivers correctly.
func TestExtractGoSymbols_ReceiverNamingVariants(t *testing.T) {
	src := `package foo

type T struct{}

func (t T) ValueReceiver() {}
func (t *T) PointerReceiver() {}
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
		"ValueReceiver":   "T",
		"PointerReceiver": "*T",
	}
	for name, wantRecv := range cases {
		s, ok := byName[name]
		if !ok {
			t.Errorf("symbol %q not found", name)
			continue
		}
		if s.Kind != "method" {
			t.Errorf("%s: expected kind 'method', got %q", name, s.Kind)
		}
		if s.Recv != wantRecv {
			t.Errorf("%s: expected Recv %q, got %q", name, wantRecv, s.Recv)
		}
	}
}
