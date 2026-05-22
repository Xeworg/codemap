package indexer

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
)

// ExtractGoSymbols walks the AST and returns top-level declarations as Symbols.
// It does not recurse into nested scopes.
func ExtractGoSymbols(f *ast.File, fset *token.FileSet) []Symbol {
	if f == nil || fset == nil {
		return nil
	}
	var out []Symbol
	for _, decl := range f.Decls {
		decl := decl
		switch d := decl.(type) {
		case *ast.FuncDecl:
			out = append(out, symbolFromFuncDecl(d, fset))
		case *ast.GenDecl:
			kind := "var"
			if d.Tok == token.CONST {
				kind = "const"
			}
			for _, spec := range d.Specs {
				spec := spec
				out = append(out, symbolFromSpec(spec, fset, kind)...)
			}
		}
	}
	return out
}

// ParseGoFile parses a Go source file and returns the extraction result.
// It returns an error for unparseable source (syntax-level).
// Both symbols and call edges are extracted.
func ParseGoFile(fset *token.FileSet, filename string, src []byte) (ParseResult, error) {
	if fset == nil {
		return ParseResult{}, errors.New("nil FileSet")
	}
	file, err := parser.ParseFile(fset, filename, src, parser.AllErrors)
	if err != nil {
		return ParseResult{}, err
	}
	syms := ExtractGoSymbols(file, fset)
	ee := NewEdgeExtractor(file, fset)
	edges := ee.ExtractEdges()
	return ParseResult{Symbols: syms, Edges: edges}, nil
}

// symbolFromFuncDecl converts a FuncDecl to a Symbol.
func symbolFromFuncDecl(d *ast.FuncDecl, fset *token.FileSet) Symbol {
	sig := typeSignature(d)
	kind := "func"
	var recv string
	if d.Recv != nil {
		kind = "method"
		recv = receiverName(d.Recv)
	}
	return Symbol{
		Name:      d.Name.Name,
		Kind:      kind,
		Signature: sig,
		Recv:      recv,
		StartLine: fset.Position(d.Pos()).Line,
		EndLine:   fset.Position(d.End()).Line,
	}
}

// receiverName extracts the receiver type name from an AST receiver list.
// For a value receiver (T), it returns "T".
// For a pointer receiver (*T), it returns "*T".
// For a package-qualified receiver (pkg.T), it returns the last identifier.
func receiverName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	r := recv.List[0]
	switch x := r.Type.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		if ident, ok := x.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
		return "*"
	}
	return ""
}

// symbolFromSpec converts a GenDecl spec to one or more Symbols.
// kind is passed from the parent GenDecl (const/var/type).
func symbolFromSpec(spec ast.Spec, fset *token.FileSet, kind string) []Symbol {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		symKind := "type"
		if _, ok := s.Type.(*ast.InterfaceType); ok {
			symKind = "interface"
		}
		return []Symbol{{
			Name:      s.Name.Name,
			Kind:      symKind,
			Signature: "",
			StartLine: fset.Position(s.Pos()).Line,
			EndLine:   fset.Position(s.End()).Line,
		}}
	case *ast.ValueSpec:
		var out []Symbol
		for _, n := range s.Names {
			out = append(out, Symbol{
				Name:      n.Name,
				Kind:      kind,
				Signature: "",
				StartLine: fset.Position(s.Pos()).Line,
				EndLine:   fset.Position(s.End()).Line,
			})
		}
		return out
	}
	return nil
}

// typeSignature produces a minimal string representation of the function signature.
func typeSignature(d *ast.FuncDecl) string {
	if d.Type.Params != nil {
		return "func(" + paramNames(d.Type.Params.List) + ")"
	}
	return "func()"
}

// paramNames returns a comma-separated list of parameter names for a field list.
func paramNames(fields []*ast.Field) string {
	var names []string
	for _, f := range fields {
		for _, n := range f.Names {
			names = append(names, n.Name)
		}
	}
	return joinNames(names)
}

func joinNames(names []string) string {
	if len(names) == 0 {
		return ""
	}
	result := names[0]
	for _, n := range names[1:] {
		result += ", " + n
	}
	return result
}
