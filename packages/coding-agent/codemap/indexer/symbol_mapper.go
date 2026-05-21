package indexer

import (
	"go/ast"
	"go/token"
)

// MapASTSymbols is a synonym for ExtractGoSymbols, exposed as a refactoring
// target for extracting shared AST-to-Symbol mapping logic.
// This allows the symbol mapper to be reused by both the parser and any future
// AST-aware tooling (e.g., edge extraction via type checking).
func MapASTSymbols(f *ast.File, fset *token.FileSet) []Symbol {
	return ExtractGoSymbols(f, fset)
}
