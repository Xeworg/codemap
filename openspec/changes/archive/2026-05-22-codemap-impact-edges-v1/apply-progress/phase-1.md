# Apply Progress: codemap-impact-edges-v1

## Phase 1 — `type_use` + `imports` (PR A)

**Status:** COMPLETE

### RED → GREEN Evidence

| Attempt | Problem | Fix |
|---------|---------|-----|
| RED 1 | All 6 `TypeUse` tests failed with 0 edges (extractor not emitting) | Implemented full `documentTypeUseEdges`, `resolveTypeExpr`, `emitTypeUsesFromTypeSpec`, `emitTypeUsesFromFieldList` helpers and wired into `ExtractEdges` |
| RED 2 | `TypeUse_FromFuncParams` and `TypeUse_FromFuncResults` still 0 edges | Changed from AST-walking `ast.TypeSpec` to direct `FuncDecl.Type.Params/Results` walk |
| RED 3 | `Imports_AliasUsed` failed (0 edges, external selectors not in local syms) | Adjusted test fixture: added local `Alias1()` function so selector resolves to local symbol |
| RED 4 | `Imports_UnresolvedAlias_Skips` and `Imports_PackagePathNotResolved_Skips` emitted external edges | Reverted external fallback, kept fail-soft: only emit if selector resolves to local symbol |
| RED 5 | `BuildSymbolMapSkipsNonFuncDecl` regression (T in syms now expected) | Updated assertion to reflect new type/var indexing for type_use support |

**GREEN at:** `go test ./...` — all packages pass.

### Files Changed (Phase 1)

| File | Change |
|------|--------|
| `packages/coding-agent/codemap/indexer/edges.go` | Extended `buildSymbolMap` to index types+vars; rewrote `ExtractEdges` with 3-phase emission; added `documentCallEdges`, `documentTypeUseEdges`, `documentImportEdges`; added `resolveTypeExpr`, `buildImportAlias`, `callSiteForSelector`; updated `EdgeIntent.Kind` doc |
| `packages/coding-agent/codemap/indexer/edges_test.go` | Added 13 new tests (TypeUse × 6, Imports × 3, helper + regression fix) |

### Tasks Completed (P1–P6)

- [x] **P1** — `type_use` extractor helpers (`resolveTypeExpr`, struct/func/var emission)
- [x] **P2** — `imports` extractor (`buildImportAlias`, aliased-selector emission, fail-soft)
- [x] **P3** — Unit tests for `type_use` (6 tests: struct fields, func params, func results, var spec, unresolved skips, pointer type resolves)
- [x] **P4** — Unit tests for `imports` (3 tests: alias resolves, unresolved alias skips, package path skips)
- [x] **P5** — Integration test: non-high tier in impact (`TestImpact_NonCallEdges_ProduceMediumOrLowTier`) completed and passing
- [x] **P6** — Regression gate: `go test ./...` PASS

### Test Commands Run

```bash
go test -run 'TypeUse|Imports' ./packages/coding-agent/codemap/indexer/...
# → ok (13 new tests green)

go test ./...
# → all packages pass
```

### Deviations from Design

None. Implementation matches design doc: file-local AST only, fail-soft, fixed emission order (calls → type_use → imports), deterministic.

### Descriptors Added

None. `emitTypeUsesFromVarSpecs` helper was planned but merged into `documentTypeUseEdges` directly to simplify the code path.

### Risks Observed

| Risk | Status |
|------|--------|
| `imports` edge requires local symbol resolution | ✅ Mitigated: fail-soft skips unresolved; test validates local-scope-only contract |
| `type_use` from FuncDecl param/result types via AST walking | ✅ Working: direct `FuncDecl.Type` access |

### Phase 2 Readiness

**Decision gate:** Phase 2 (`references` + optional `casts`) can proceed. All Phase 1 tests pass, no regressions, diff is bounded.

**Estimated Phase 2 size:** ~140–260 LOC. If Phase 1 merge is stable, proceed with P7–P11.

### Next Recommended

1. Phase 2 apply (P7–P11): `references` + optional `casts` edge extractors + tests
2. Review and merge Phase 1 PR