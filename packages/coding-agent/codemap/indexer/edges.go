package indexer

import (
	"go/ast"
	"go/token"
)

type SymbolKey struct {
	File string // source file (for disambiguation)
	Name string
	Recv string // empty for top-level, receiver-qualified for methods (e.g. "T" or "*T")
}

// EdgeIntent represents a single resolved call/reference edge between two symbols.
type EdgeIntent struct {
	From SymbolKey
	To   SymbolKey
	Kind string // "call", "ref", "type_use", "imports", "references", "casts"
}

// EdgeExtractor processes a single parsed file and emits EdgeIntents.
type EdgeExtractor struct {
	file           *ast.File
	fset           *token.FileSet
	syms           map[string]SymbolKey // name -> key (file-local resolver)
	fileLevelNames map[string]bool      // names declared at file scope (excluded from write-set)
}

// NewEdgeExtractor builds an EdgeExtractor for the given AST file.
func NewEdgeExtractor(f *ast.File, fset *token.FileSet) *EdgeExtractor {
	ee := &EdgeExtractor{
		file:           f,
		fset:           fset,
		syms:           make(map[string]SymbolKey),
		fileLevelNames: make(map[string]bool),
	}
	ee.buildSymbolMap()
	return ee
}

// buildSymbolMap walks the file's top-level declarations and populates the syms map.
// For top-level symbols, key is just the name.
// For methods, key is "Recv.Name" to disambiguate from top-level funcs.
// On naming collisions, the first declaration wins (deterministic by AST order).
// It also indexes type names and var/const names for type_use resolution.
func (ee *EdgeExtractor) buildSymbolMap() {
	if ee.file == nil {
		return
	}
	for _, decl := range ee.file.Decls {
		// Index functions and methods.
		if d, ok := decl.(*ast.FuncDecl); ok {
			sym := SymbolKey{
				File: ee.file.Name.Name,
				Name: d.Name.Name,
				Recv: "",
			}
			if d.Recv != nil && len(d.Recv.List) > 0 {
				sym.Recv = receiverName(d.Recv)
				key := sym.Recv + "." + sym.Name
				if _, exists := ee.syms[key]; !exists {
					ee.syms[key] = sym
				}
				if _, exists := ee.syms[sym.Name]; !exists {
					ee.syms[sym.Name] = sym
				}
			} else {
				if _, exists := ee.syms[sym.Name]; !exists {
					ee.syms[sym.Name] = sym
				}
			}
			continue
		}
		// Index type declarations for type_use resolution.
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
			for _, spec := range gd.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if _, exists := ee.syms[ts.Name.Name]; !exists {
						ee.syms[ts.Name.Name] = SymbolKey{
							File: ee.file.Name.Name,
							Name: ts.Name.Name,
							Recv: "",
						}
					}
				}
			}
			continue
		}
		// Index var/const declarations for type_use resolution.
		if gd, ok := decl.(*ast.GenDecl); ok && (gd.Tok == token.VAR || gd.Tok == token.CONST) {
			for _, spec := range gd.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					for _, n := range vs.Names {
						ee.fileLevelNames[n.Name] = true
						if _, exists := ee.syms[n.Name]; !exists {
							ee.syms[n.Name] = SymbolKey{
								File: ee.file.Name.Name,
								Name: n.Name,
								Recv: "",
							}
						}
					}
				}
			}
		}
	}
}

// ExtractEdges walks the file's AST and emits resolved edges.
// Order: calls → type_use → imports → references → casts.
func (ee *EdgeExtractor) ExtractEdges() []EdgeIntent {
	if ee.file == nil {
		return nil
	}
	var edges []EdgeIntent

	// 1. calls.
	documentCallEdges(ee, &edges)

	// 2. type_use.
	documentTypeUseEdges(ee, &edges)

	// 3. imports.
	documentImportEdges(ee, &edges)

	// 4. references.
	documentReferenceEdges(ee, &edges)

	// 5. casts.
	documentCastEdges(ee, &edges)

	return edges
}

// documentCallEdges emits "call" edges from *ast.CallExpr nodes.
func documentCallEdges(ee *EdgeExtractor, edges *[]EdgeIntent) {
	ast.Inspect(ee.file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		target := ee.resolveCallee(call.Fun)
		if target.Name == "" {
			return true // unresolved
		}
		caller := ee.callSite(call)
		*edges = append(*edges, EdgeIntent{
			From: caller,
			To:   target,
			Kind: "call",
		})
		return true
	})
}

// documentTypeUseEdges emits "type_use" edges for type references in:
// - struct field types
// - function param/return types
// - var/const declarations with explicit type
func documentTypeUseEdges(ee *EdgeExtractor, edges *[]EdgeIntent) {
	// Emit type_use for var/const declarations with explicit type.
	for _, decl := range ee.file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || (gd.Tok != token.VAR && gd.Tok != token.CONST) {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || vs.Type == nil {
				continue
			}
			for _, n := range vs.Names {
				if to := ee.resolveTypeExpr(vs.Type); to.Name != "" {
					*edges = append(*edges, EdgeIntent{
						From: SymbolKey{File: ee.file.Name.Name, Name: n.Name, Recv: ""},
						To:   to,
						Kind: "type_use",
					})
				}
			}
		}
	}

	// Emit type_use for FuncDecl param/result types.
	for _, decl := range ee.file.Decls {
		d, ok := decl.(*ast.FuncDecl)
		if !ok || d.Type == nil {
			continue
		}
		from := SymbolKey{File: ee.file.Name.Name, Name: d.Name.Name, Recv: funcRecv(d)}
		if d.Type.Params != nil {
			for _, field := range d.Type.Params.List {
				if to := ee.resolveTypeExpr(field.Type); to.Name != "" {
					*edges = append(*edges, EdgeIntent{From: from, To: to, Kind: "type_use"})
				}
			}
		}
		if d.Type.Results != nil {
			for _, field := range d.Type.Results.List {
				if to := ee.resolveTypeExpr(field.Type); to.Name != "" {
					*edges = append(*edges, EdgeIntent{From: from, To: to, Kind: "type_use"})
				}
			}
		}
	}

	// Emit type_use for struct field types.
	ast.Inspect(ee.file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		ee.emitTypeUsesFromTypeSpec(ts, edges)
		return true
	})
}

// emitTypeUsesFromTypeSpec emits type_use edges for all types used in a TypeSpec's
// underlying type and (for structs) its fields.
func (ee *EdgeExtractor) emitTypeUsesFromTypeSpec(ts *ast.TypeSpec, edges *[]EdgeIntent) {
	from := SymbolKey{File: ee.file.Name.Name, Name: ts.Name.Name, Recv: ""}
	if st, ok := ts.Type.(*ast.StructType); ok {
		for _, field := range st.Fields.List {
			if field.Type != nil {
				if to := ee.resolveTypeExpr(field.Type); to.Name != "" {
					*edges = append(*edges, EdgeIntent{From: from, To: to, Kind: "type_use"})
				}
			}
		}
	}
	if ft, ok := ts.Type.(*ast.FuncType); ok {
		ee.emitTypeUsesFromFieldList(ft.Params, from, edges)
		ee.emitTypeUsesFromFieldList(ft.Results, from, edges)
	}
}

// emitTypeUsesFromFieldList emits type_use edges for all named parameter or result
// types in a field list, from the enclosing function/type to each type.
func (ee *EdgeExtractor) emitTypeUsesFromFieldList(fl *ast.FieldList, from SymbolKey, edges *[]EdgeIntent) {
	if fl == nil {
		return
	}
	for _, field := range fl.List {
		if field.Type != nil {
			if to := ee.resolveTypeExpr(field.Type); to.Name != "" {
				*edges = append(*edges, EdgeIntent{From: from, To: to, Kind: "type_use"})
			}
		}
	}
}

// resolveTypeExpr resolves a type expression to a SymbolKey.
// Strips pointer stars and package qualifiers at AST level.
func (ee *EdgeExtractor) resolveTypeExpr(expr ast.Expr) SymbolKey {
	for expr != nil {
		switch e := expr.(type) {
		case *ast.Ident:
			if sym, ok := ee.syms[e.Name]; ok {
				return sym
			}
			return SymbolKey{}
		case *ast.StarExpr:
			expr = e.X // unwrap pointer
		case *ast.SelectorExpr:
			// Package-qualified: resolve via selector name if in syms.
			if sym, ok := ee.syms[e.Sel.Name]; ok {
				return sym
			}
			return SymbolKey{}
		default:
			return SymbolKey{}
		}
	}
	return SymbolKey{}
}

// importAlias maps a local import alias to its package path.
type importAlias map[string]string

// buildImportAlias scans the file's import specs and populates the alias map.
func (ee *EdgeExtractor) buildImportAlias() importAlias {
	m := make(importAlias)
	for _, decl := range ee.file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		for _, spec := range gd.Specs {
			is := spec.(*ast.ImportSpec)
			alias := is.Name
			path := ""
			if is.Path != nil {
				path = is.Path.Value // "fmt" → "fmt"
			}
			if alias != nil {
				m[alias.Name] = path
			} else {
				// No alias: store the last component of the path as the default alias.
				// e.g. import "fmt" → key "fmt".
				if len(path) >= 2 {
					m[path[1:len(path)-1]] = path // strip quotes
				}
			}
		}
	}
	return m
}

// documentImportEdges emits "imports" edges for selector expressions that reference
// an aliased import whose symbol resolves to a known local symbol.
// Unresolved external selectors are skipped (fail-soft).
func documentImportEdges(ee *EdgeExtractor, edges *[]EdgeIntent) {
	aliasMap := ee.buildImportAlias()

	ast.Inspect(ee.file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if _, isAlias := aliasMap[ident.Name]; !isAlias {
			return true // not an aliased import reference
		}
		// Only emit if the selector resolves to a known local symbol.
		if sym, ok := ee.syms[sel.Sel.Name]; ok {
			caller := ee.callSiteForSelector(sel)
			*edges = append(*edges, EdgeIntent{
				From: caller,
				To:   sym,
				Kind: "imports",
			})
		}
		return true
	})
}

// callSiteForSelector returns the enclosing function's SymbolKey for a selector expr.
func (ee *EdgeExtractor) callSiteForSelector(sel *ast.SelectorExpr) SymbolKey {
	for _, decl := range ee.file.Decls {
		d, ok := decl.(*ast.FuncDecl)
		if !ok || d.Body == nil {
			continue
		}
		if containsNode(d.Body, sel) {
			return SymbolKey{
				File: ee.file.Name.Name,
				Name: d.Name.Name,
				Recv: funcRecv(d),
			}
		}
	}
	return SymbolKey{}
}

// callSite returns the SymbolKey for the enclosing FuncDecl, or an empty key.
func (ee *EdgeExtractor) callSite(call *ast.CallExpr) SymbolKey {
	for _, decl := range ee.file.Decls {
		d, ok := decl.(*ast.FuncDecl)
		if !ok || d.Body == nil {
			continue
		}
		if containsNode(d.Body, call) {
			return SymbolKey{
				File: ee.file.Name.Name,
				Name: d.Name.Name,
				Recv: funcRecv(d),
			}
		}
	}
	return SymbolKey{}
}

// funcRecv returns the receiver name for a FuncDecl, or "" if none.
func funcRecv(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return ""
	}
	return receiverName(d.Recv)
}

// resolveCallee resolves a CallExpr's Fun expression to a SymbolKey.
// Handles bare identifiers (e.g. foo → top-level foo)
// and selector expressions (e.g. x.Foo → method on x's type).
// v1 limitation: method resolution only succeeds when the variable name
// matches the receiver type name (e.g. var t T; t.Method() → T.Method).
func (ee *EdgeExtractor) resolveCallee(fun ast.Expr) SymbolKey {
	switch v := fun.(type) {
	case *ast.Ident:
		if sym, ok := ee.syms[v.Name]; ok {
			return sym
		}
	case *ast.SelectorExpr:
		if ident, ok := v.X.(*ast.Ident); ok {
			key := ident.Name + "." + v.Sel.Name
			if sym, ok := ee.syms[key]; ok {
				return sym
			}
			if sym, ok := ee.syms[v.Sel.Name]; ok {
				return sym
			}
		}
	}
	return SymbolKey{}
}

// containsNode returns true if the inner node is inside the outer AST subtree.
func containsNode(outer ast.Node, inner ast.Node) bool {
	found := false
	ast.Inspect(outer, func(n ast.Node) bool {
		if n == inner {
			found = true
			return false
		}
		return !found
	})
	return found
}

// builtins marks Go built-in identifiers that should never be emitted as reference edges.
var builtins = map[string]bool{
	"len": true, "cap": true, "append": true, "make": true, "new": true,
	"delete": true, "copy": true, "complex": true, "real": true, "imag": true,
	"panic": true, "recover": true, "print": true, "println": true,
	"close": true, "clear": true, "min": true, "max": true,
}

// documentReferenceEdges emits "references" edges for value-position identifier
// uses that resolve to a known local symbol.
// Excludes: declarations (LHS of assignment), keywords, builtins, package-qualified,
// and call-expression callees (already covered by "call" edges).
func documentReferenceEdges(ee *EdgeExtractor, edges *[]EdgeIntent) {
	// Pre-build set of written identifiers per function body to exclude write contexts.
	writeSet := ee.buildWriteSet()

	// Pre-build set of call-callee identifiers per function body to exclude call targets.
	callTargets := ee.buildCallTargetSet()
	for _, decl := range ee.file.Decls {
		d, ok := decl.(*ast.FuncDecl)
		if !ok || d.Body == nil {
			continue
		}
		from := SymbolKey{File: ee.file.Name.Name, Name: d.Name.Name, Recv: funcRecv(d)}
		writes := writeSet[d]
		targets := callTargets[d]
		// Pre-build set of type names declared at file scope.
		typeNames := make(map[string]bool)
		for _, decl := range ee.file.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
				for _, spec := range gd.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						typeNames[ts.Name.Name] = true
					}
				}
			}
		}

		ast.Inspect(d.Body, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			// Skip written (declaration/LHS) identifiers.
			if writes[ident.Name] {
				return true
			}
			// Skip call targets (already covered by "call" edges).
			if targets[ident.Name] {
				return true
			}
			// Skip language builtins.
			if builtins[ident.Name] {
				return true
			}
			// Skip package-qualified idents.
			if isPackageQualifier(ee.file, ident) {
				return true
			}
			// Skip type-name references (they belong in type_use, not references).
			if typeNames[ident.Name] {
				return true
			}
			// Resolve.
			if sym, ok := ee.syms[ident.Name]; ok {
				*edges = append(*edges, EdgeIntent{
					From: from,
					To:   sym,
					Kind: "references",
				})
			}
			return true
		})
	}
}

// buildWriteSet returns a map from FuncDecl to the set of identifiers that are
// written (assigned or declared) within that function body.
func (ee *EdgeExtractor) buildWriteSet() map[*ast.FuncDecl]map[string]bool {
	result := make(map[*ast.FuncDecl]map[string]bool)
	for _, decl := range ee.file.Decls {
		d, ok := decl.(*ast.FuncDecl)
		if !ok || d.Body == nil {
			continue
		}
		writes := make(map[string]bool)
		ast.Inspect(d.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				for _, lhs := range n.Lhs {
					markIdents(lhs, writes)
				}
			case *ast.ValueSpec:
				// Skip file-level ValueSpec (already in fileLevelNames).
				// Only track local ValueSpec declarations.
				for _, name := range n.Names {
					if !ee.fileLevelNames[name.Name] {
						writes[name.Name] = true
					}
				}
			case *ast.RangeStmt:
				// Range with := — variables declared in source are on the LHS of the implicit AssignStmt.
				// We extract them by walking source text via ast.Inspect on the RangeStmt body.
				if n.Tok == token.DEFINE {
					ast.Inspect(n, func(nn ast.Node) bool {
						if id, ok := nn.(*ast.Ident); ok {
							writes[id.Name] = true
						}
						return true
					})
				}
			}
			return true
		})
		result[d] = writes
	}
	return result
}

// markIdents recursively collects identifier names from an expression into the set.
func markIdents(expr ast.Expr, set map[string]bool) {
	ast.Inspect(expr, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			set[id.Name] = true
		}
		return true
	})
}

// buildCallTargetSet returns a map from FuncDecl to the set of identifier names
// used as the callee of a CallExpr within that function's body.
// Handles both bare identifiers (foo()) and selector expressions (obj.Method()).
func (ee *EdgeExtractor) buildCallTargetSet() map[*ast.FuncDecl]map[string]bool {
	result := make(map[*ast.FuncDecl]map[string]bool)
	for _, decl := range ee.file.Decls {
		d, ok := decl.(*ast.FuncDecl)
		if !ok || d.Body == nil {
			continue
		}
		targets := make(map[string]bool)
		ast.Inspect(d.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				switch fun := call.Fun.(type) {
				case *ast.Ident:
					targets[fun.Name] = true
				case *ast.SelectorExpr:
					targets[fun.Sel.Name] = true
				}
			}
			return true
		})
		result[d] = targets
	}
	return result
}

// isPackageQualifier returns true if ident is the X part of a SelectorExpr
// (e.g. "fmt" in fmt.Println).
func isPackageQualifier(f *ast.File, ident *ast.Ident) bool {
	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			for _, spec := range gd.Specs {
				if is, ok := spec.(*ast.ImportSpec); ok && is.Name != nil && is.Name.Name == ident.Name {
					return true
				}
			}
		}
	}
	return false
}

// documentCastEdges emits "casts" edges for type assertions where the asserted
// type resolves to a known local symbol.
// e.g. v, ok := x.(T) where T is an indexed type.
func documentCastEdges(ee *EdgeExtractor, edges *[]EdgeIntent) {
	for _, decl := range ee.file.Decls {
		d, ok := decl.(*ast.FuncDecl)
		if !ok || d.Body == nil {
			continue
		}
		from := SymbolKey{File: ee.file.Name.Name, Name: d.Name.Name, Recv: funcRecv(d)}
		ast.Inspect(d.Body, func(n ast.Node) bool {
			ta, ok := n.(*ast.TypeAssertExpr)
			if !ok {
				return true
			}
			if ta.Type == nil {
				return true
			}
			if to := ee.resolveTypeExpr(ta.Type); to.Name != "" {
				*edges = append(*edges, EdgeIntent{
					From: from,
					To:   to,
					Kind: "casts",
				})
			}
			return true
		})
	}
}
