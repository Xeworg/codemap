package indexer

import (
	"go/parser"
	"go/token"
	"testing"
)

// TestExtractEdges_MethodCall verifies that a method call in a function
// produces an edge from the caller to the method.
// v1 limitation: resolution requires variable name to match receiver type name
// (var t T; t.Method() → key "T.Method").
func TestExtractEdges_MethodCall(t *testing.T) {
	src := `package test

type T struct{}

func (t T) Method() {}

func Caller() {
	var t T
	t.Method()
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	// We expect exactly one edge: Caller → T.Method (call).
	if len(edges) != 1 {
		t.Errorf("expected 1 edge, got %d: %+v", len(edges), edges)
		return
	}
	e := edges[0]
	if e.Kind != "call" {
		t.Errorf("expected kind 'call', got %q", e.Kind)
	}
	// Caller is a top-level func: Recv should be "".
	if e.From.Name != "Caller" {
		t.Errorf("expected from 'Caller', got %q", e.From.Name)
	}
	if e.From.Recv != "" {
		t.Errorf("expected from.Recv '', got %q", e.From.Recv)
	}
	// T.Method is a method with Recv "T".
	if e.To.Name != "Method" {
		t.Errorf("expected to 'Method', got %q", e.To.Name)
	}
	if e.To.Recv != "T" {
		t.Errorf("expected to.Recv 'T', got %q", e.To.Recv)
	}
}

// TestExtractEdges_TopLevelCall verifies that a call to a top-level function
// produces an edge between two top-level functions.
func TestExtractEdges_TopLevelCall(t *testing.T) {
	src := `package test

func Helper() {}

func Caller() {
	Helper()
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	if len(edges) != 1 {
		t.Errorf("expected 1 edge, got %d: %+v", len(edges), edges)
		return
	}
	e := edges[0]
	if e.Kind != "call" {
		t.Errorf("expected kind 'call', got %q", e.Kind)
	}
	if e.From.Name != "Caller" {
		t.Errorf("expected from 'Caller', got %q", e.From.Name)
	}
	if e.To.Name != "Helper" {
		t.Errorf("expected to 'Helper', got %q", e.To.Name)
	}
}

// TestExtractEdges_NoEdgesWhenNoCalls verifies that a file with no function
// calls produces zero edges.
func TestExtractEdges_NoEdgesWhenNoCalls(t *testing.T) {
	src := `package test

func A() {}
func B() {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	if len(edges) != 0 {
		t.Errorf("expected 0 edges, got %d: %+v", len(edges), edges)
	}
}

// TestExtractEdges_UnresolvedCalleeIsSkipped verifies that calls to
// undeclared functions are silently skipped (no edge emitted).
func TestExtractEdges_UnresolvedCalleeIsSkipped(t *testing.T) {
	src := `package test

func Caller() {
	UnknownFunc()
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	if len(edges) != 0 {
		t.Errorf("expected 0 edges for unresolved callee, got %d: %+v", len(edges), edges)
	}
}

// TestExtractEdges_MultipleCalls verifies that multiple calls in the same
// function produce multiple edges.
func TestExtractEdges_MultipleCalls(t *testing.T) {
	src := `package test

func One() {}
func Two() {}

func Many() {
	One()
	Two()
	One()
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	if len(edges) != 3 {
		t.Errorf("expected 3 edges, got %d: %+v", len(edges), edges)
	}
}

// TestExtractEdges_BuildSymbolMapSkipsNonFuncDecl verifies that non-func
// declarations don't cause issues in the symbol map.
func TestExtractEdges_BuildSymbolMapSkipsNonFuncDecl(t *testing.T) {
	src := `package test

type T struct{}
const C = 1
var V int

func F() {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	// Build shouldn't panic. F must be in syms.
	if _, ok := ee.syms["F"]; !ok {
		t.Error("expected 'F' in symbol map")
	}
	// T and V are indexed for type_use and reference resolution.
	// The old "func-only" assertion is replaced by type_use feature.
	if _, ok := ee.syms["T"]; !ok {
		t.Error("expected 'T' in symbol map (needed for type_use resolution)")
	}
}

// TestExtractEdges_NilFileReturnsNil verifies that nil input returns nil.
func TestExtractEdges_NilFileReturnsNil(t *testing.T) {
	ee := NewEdgeExtractor(nil, nil)
	if ee.ExtractEdges() != nil {
		t.Error("expected nil edges for nil file")
	}
}

// TestExtractEdges_MethodWithPointerReceiver verifies that calling a method
// with a pointer receiver works when the variable name matches the type.
func TestExtractEdges_MethodWithPointerReceiver(t *testing.T) {
	src := `package test

type T struct{}

func (t *T) PointerMethod() {}

func Caller() {
	var t T
	t.PointerMethod()
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	// Method receiver is *T but call uses value variable t (T).
	// For v1 resolution we match on the receiver type name as written (*T → "*T").
	// The call's variable name "t" doesn't match "*T", so this edge may not resolve.
	// We document this as a known limitation for pointer receiver methods.
	if len(edges) != 1 {
		t.Errorf("expected 1 edge (pointer receiver), got %d: %+v", len(edges), edges)
		return
	}
	e := edges[0]
	if e.Kind != "call" {
		t.Errorf("expected kind 'call', got %q", e.Kind)
	}
	if e.To.Name != "PointerMethod" {
		t.Errorf("expected to 'PointerMethod', got %q", e.To.Name)
	}
}

// TestExtractEdges_NestedCalls verifies that nested call chains are captured.
func TestExtractEdges_NestedCalls(t *testing.T) {
	src := `package test

func A() {}
func B() { A() }
func C() { B() }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	if len(edges) != 2 {
		t.Errorf("expected 2 edges, got %d: %+v", len(edges), edges)
	}
}

// TestExtractEdges_MethodCallDifferentName verifies resolution when the variable
// name differs from the receiver type name. With the bare-method-name fallback,
// the call resolves via the method name alone (v1 pragmatic heuristic).
// This test documents the current behavior.
func TestExtractEdges_MethodCallDifferentName(t *testing.T) {
	src := `package test

type MyType struct{}

func (m MyType) DoSomething() {}

func Caller() {
	var obj MyType
	obj.DoSomething()
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	// With bare-name fallback, obj.DoSomething() resolves via key "DoSomething".
	// This succeeds for methods where no top-level function shares the name.
	if len(edges) != 1 {
		t.Errorf("expected 1 edge (bare-name fallback), got %d: %+v", len(edges), edges)
	}
}

// TestExtractEdges_MethodCallSameName verifies that method resolution works
// when the variable name case-matches the receiver type name.
// With bare-method-name fallback, resolution works regardless of
// whether the variable name matches the receiver type name.
func TestExtractEdges_MethodCallSameName(t *testing.T) {
	src := `package test

type MyType struct{}

func (m MyType) DoSomething() {}

func Caller() {
	var mytype MyType
	mytype.DoSomething()
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	// Key "mytype.DoSomething" not found (case-sensitive). Bare fallback: "DoSomething" → found.
	if len(edges) != 1 {
		t.Errorf("expected 1 edge via bare-name fallback, got %d: %+v", len(edges), edges)
	}
}

// -- Phase 1: type_use edge tests (P3) --

// TestEdgeExtractor_TypeUse_FromStructFields verifies that struct field type declarations
// emit type_use edges from the struct to the field type.
func TestEdgeExtractor_TypeUse_FromStructFields(t *testing.T) {
	src := `package test

type Inner struct{}

type Outer struct {
	Field Inner
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	// We expect 1 type_use edge: Outer → Inner (via field type Inner).
	typeUse := filterByKind(edges, "type_use")
	if len(typeUse) != 1 {
		t.Errorf("expected 1 type_use edge from struct field, got %d: %+v", len(typeUse), typeUse)
		return
	}
	e := typeUse[0]
	if e.Kind != "type_use" {
		t.Errorf("expected kind 'type_use', got %q", e.Kind)
	}
	if e.To.Name != "Inner" {
		t.Errorf("expected to 'Inner', got %q", e.To.Name)
	}
	if e.From.Name != "Outer" {
		t.Errorf("expected from 'Outer', got %q", e.From.Name)
	}
}

// TestEdgeExtractor_TypeUse_FromFuncParams verifies that function parameter types
// emit type_use edges.
func TestEdgeExtractor_TypeUse_FromFuncParams(t *testing.T) {
	src := `package test

type Config struct{}

func TakesConfig(c Config) {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	typeUse := filterByKind(edges, "type_use")
	if len(typeUse) != 1 {
		t.Errorf("expected 1 type_use edge from func param, got %d: %+v", len(typeUse), typeUse)
		return
	}
	if typeUse[0].To.Name != "Config" {
		t.Errorf("expected to 'Config', got %q", typeUse[0].To.Name)
	}
	if typeUse[0].From.Name != "TakesConfig" {
		t.Errorf("expected from 'TakesConfig', got %q", typeUse[0].From.Name)
	}
}

// TestEdgeExtractor_TypeUse_FromFuncResults verifies that function return types
// emit type_use edges.
func TestEdgeExtractor_TypeUse_FromFuncResults(t *testing.T) {
	src := `package test

type Result struct{}

func ReturnsResult() Result {
	return Result{}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	typeUse := filterByKind(edges, "type_use")
	if len(typeUse) != 1 {
		t.Errorf("expected 1 type_use edge from func result, got %d: %+v", len(typeUse), typeUse)
		return
	}
	if typeUse[0].To.Name != "Result" {
		t.Errorf("expected to 'Result', got %q", typeUse[0].To.Name)
	}
}

// TestEdgeExtractor_TypeUse_FromVarSpec verifies that var declarations with explicit
// types emit type_use edges.
func TestEdgeExtractor_TypeUse_FromVarSpec(t *testing.T) {
	src := `package test

type T int

var V T
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	typeUse := filterByKind(edges, "type_use")
	if len(typeUse) != 1 {
		t.Errorf("expected 1 type_use edge from var type, got %d: %+v", len(typeUse), typeUse)
		return
	}
	if typeUse[0].To.Name != "T" {
		t.Errorf("expected to 'T', got %q", typeUse[0].To.Name)
	}
}

// TestEdgeExtractor_TypeUse_Unresolved_Skips verifies that unresolved type references
// are silently skipped (no edge emitted, no error).
func TestEdgeExtractor_TypeUse_Unresolved_Skips(t *testing.T) {
	src := `package test

type Outer struct {
	Field UnresolvedType
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	typeUse := filterByKind(edges, "type_use")
	if len(typeUse) != 0 {
		t.Errorf("expected 0 type_use edges for unresolved type, got %d", len(typeUse))
	}
}

// TestEdgeExtractor_TypeUse_PointerType_Resolves verifies that pointer types
// used as field types resolve to the underlying symbol.
func TestEdgeExtractor_TypeUse_PointerType_Resolves(t *testing.T) {
	src := `package test

type T struct{}

type Outer struct {
	Field *T
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	typeUse := filterByKind(edges, "type_use")
	foundT := false
	for _, e := range typeUse {
		if e.To.Name == "T" {
			foundT = true
		}
	}
	if !foundT {
		t.Errorf("expected type_use edge to 'T' from pointer field type, got: %+v", typeUse)
	}
}

// -- Phase 1: imports edge tests (P4) --

// TestEdgeExtractor_Imports_AliasUsed verifies that an aliased import whose selector
// resolves to a locally-defined symbol emits an imports edge.
func TestEdgeExtractor_Imports_AliasUsed(t *testing.T) {
	src := `package test

import alias1 "fmt"

func Alias1() {}

func Use() {
	alias1.Alias1()
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	imports := filterByKind(edges, "imports")
	if len(imports) != 1 {
		t.Errorf("expected 1 imports edge from alias selector, got %d: %+v", len(imports), imports)
		return
	}
	e := imports[0]
	if e.Kind != "imports" {
		t.Errorf("expected kind 'imports', got %q", e.Kind)
	}
	if e.From.Name != "Use" {
		t.Errorf("expected from 'Use', got %q", e.From.Name)
	}
	// To should be the locally-defined Alias1 symbol.
	if e.To.Name != "Alias1" {
		t.Errorf("expected to 'Alias1', got %q", e.To.Name)
	}
}

// TestEdgeExtractor_Imports_UnresolvedAlias_Skips verifies that unresolved alias
// references are silently skipped.
func TestEdgeExtractor_Imports_UnresolvedAlias_Skips(t *testing.T) {
	src := `package test

import unknown "fmt"

func Use() {
	unknown.UnresolvedSymbol()
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	imports := filterByKind(edges, "imports")
	if len(imports) != 0 {
		t.Errorf("expected 0 imports edges for unresolved selector, got %d: %+v", len(imports), imports)
	}
}

// TestEdgeExtractor_Imports_PackagePathNotResolved_Skips verifies that a bare
// package path selector (no alias) does not emit an edge.
func TestEdgeExtractor_Imports_PackagePathNotResolved_Skips(t *testing.T) {
	src := `package test

import "fmt"

func Use() {
	fmt.Println("hello")
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	// "fmt" is not in the symbol map; should not crash, should emit 0 imports.
	imports := filterByKind(edges, "imports")
	if len(imports) != 0 {
		t.Errorf("expected 0 imports edges for package path selector, got %d: %+v", len(imports), imports)
	}
}

func TestEdgeExtractor_Casts_UnresolvedType_Skips(t *testing.T) {
	src := `package test

type Outer struct {
	Field interface{}
}

func GetField(o Outer) {
	if v, ok := o.Field.(UnresolvedType); ok {
		_ = v
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	casts := filterByKind(edges, "casts")
	if len(casts) != 0 {
		t.Errorf("expected 0 casts edges for unresolved type, got %d: %+v", len(casts), casts)
	}
}

// -- Phase 2 P9: references edge tests --

// TestEdgeExtractor_References_ResolvableIdent verifies that a value-position identifier
// use emits a references edge.
func TestEdgeExtractor_References_ResolvableIdent(t *testing.T) {
	src := `package test

var Config string

func read() string {
	return Config
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	refs := filterByKind(edges, "references")
	if len(refs) == 0 {
		t.Fatalf("expected at least 1 references edge, got 0: %+v", edges)
	}
	found := false
	for _, e := range refs {
		if e.From.Name == "read" && e.To.Name == "Config" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected references edge read→Config, got: %+v", refs)
	}
}

// TestEdgeExtractor_References_Declaration_Skips verifies that identifiers in
// assignment LHS positions are not emitted as reference edges.
func TestEdgeExtractor_References_Declaration_Skips(t *testing.T) {
	src := `package test

var A string
var B string

func Assign() {
	A, B = "x", "y"
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	// A and B appear on the LHS of assignment — no reference edges for them.
	refs := filterByKind(edges, "references")
	for _, e := range refs {
		if e.To.Name == "A" || e.To.Name == "B" {
			t.Errorf("declaration identifier %q should not emit references edge", e.To.Name)
		}
	}
}

// TestEdgeExtractor_References_Unresolved_Skips verifies that identifiers not in
// the symbol map are silently skipped (no edge, no error).
func TestEdgeExtractor_References_Unresolved_Skips(t *testing.T) {
	src := `package test

var Local string

func Use() {
	_ = UnknownVar
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	// UnknownVar is not in symbol map; should not crash.
	refs := filterByKind(edges, "references")
	for _, e := range refs {
		if e.To.Name == "UnknownVar" {
			t.Errorf("unresolved identifier should not emit references edge")
		}
	}
}

// TestEdgeExtractor_References_PackageAlias_Skips verifies that package-qualified
// identifiers (import aliases used as package qualifiers) do not emit reference edges.
func TestEdgeExtractor_References_PackageAlias_Skips(t *testing.T) {
	src := `package test

import fmt "fmt"

func Use() {
	fmt.Println("hi")
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	// "fmt" is the package qualifier; should not emit references edge.
	refs := filterByKind(edges, "references")
	for _, e := range refs {
		if e.To.Name == "fmt" {
			t.Errorf("package alias should not emit references edge")
		}
	}
}

// TestEdgeExtractor_Casts_ResolvableAssertion verifies that a type assertion where
// the asserted type resolves to an indexed symbol emits a casts edge.
func TestEdgeExtractor_Casts_ResolvableAssertion(t *testing.T) {
	src := `package test

type MyType struct{}

func Get(o interface{}) {
	if v, ok := o.(MyType); ok {
		_ = v
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ee := NewEdgeExtractor(f, fset)
	edges := ee.ExtractEdges()

	casts := filterByKind(edges, "casts")
	if len(casts) == 0 {
		t.Fatalf("expected at least 1 casts edge, got 0: %+v", edges)
	}
	found := false
	for _, e := range casts {
		if e.To.Name == "MyType" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected casts edge to MyType, got: %+v", casts)
	}
}

// filterByKind is a test helper that returns edges with the given Kind.
func filterByKind(edges []EdgeIntent, kind string) []EdgeIntent {
	var result []EdgeIntent
	for _, e := range edges {
		if e.Kind == kind {
			result = append(result, e)
		}
	}
	return result
}
