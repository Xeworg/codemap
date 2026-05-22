# Apply Progress: codemap-impact-edges-v1 — Phase 2

## Phase 2 — `references` + `casts` (PR B)

**Status:** COMPLETE

### RED → GREEN Evidence

| Attempt | Problem | Fix |
|---------|---------|-----|
| RED 1 | `References_Declaration_Skips` failed: A and B emitted as references edges (top-level `var` in writeSet) | Built `fileLevelNames` set in `EdgeExtractor`; `buildWriteSet` now skips file-level declarations |
| RED 2 | `References_ResolvableIdent` failed: `Config` in `Config = "value"` emitted as write not read, plus `fetch()` call not in references | Changed fixture to read-only pattern (`return Config`); added `buildCallTargetSet` to exclude call callees from references |
| RED 3 | Build error: `n.Lhs` undefined on `*ast.RangeStmt` | Replaced with `ast.Inspect` walk on `RangeStmt` body to capture declared names |
| RED 4 | Build error: `fun.Sel` type assertion wrong | `SelectorExpr.Sel` is already `*ast.Ident`, not interface; used `fun.Sel.Name` directly |
| RED 5 | Existing call tests (`MethodCall`, `TopLevelCall`, etc.) doubled edges (call + references) | Added `buildCallTargetSet` to exclude call-callee identifiers from `references` emission |
| RED 6 | `T` type name in method tests emitted as `references` | Added `typeNames` pre-built set to skip type-name identifiers in `references` emission |

**GREEN at:** `go test ./...` — all packages pass.

### Files Changed (Phase 2)

| File | Change |
|------|--------|
| `packages/coding-agent/codemap/indexer/edges.go` | Added `documentReferenceEdges`, `documentCastEdges`, `buildWriteSet`, `buildCallTargetSet`, `markIdents`, `isPackageQualifier`; added `fileLevelNames` field to `EdgeExtractor`; `ExtractEdges` updated to 5-phase emission order |
| `packages/coding-agent/codemap/indexer/edges_test.go` | Added 5 new tests: `References_ResolvableIdent`, `References_Declaration_Skips`, `References_Unresolved_Skips`, `References_PackageAlias_Skips`, `Casts_ResolvableAssertion` |
| `packages/coding-agent/codemap/cli/impact_cmd_test.go` | Added 2 integration tests: `TestImpact_TierDiversity_WithAllEdgeKinds`, `TestImpact_Determinism_WithExpandedEdgeTypes` |
| `CHANGELOG.md` | Added `impact tier diversity` entry under Unreleased |

### Tasks Completed (P7–P11)

- [x] **P7** — `references` extractor (`documentReferenceEdges`, `buildWriteSet`, write-context guard)
- [x] **P8** — `casts` extractor (`documentCastEdges`, `*ast.TypeAssertExpr` resolution)
- [x] **P9** — Unit tests for `references` (4 tests) + `casts` (2 tests)
- [x] **P10** — Extended integration tests: tier diversity + determinism with expanded edge types
- [x] **P11** — Regression gate: `go test ./...` PASS

### Test Commands Run

```bash
go test -run 'References|Casts' ./packages/coding-agent/codemap/indexer/...
# → 6/6 PASS

go test ./...
# → all packages pass
```

### Deviations from Design

1. **references excludes type names**: Type names declared at file scope (`type T struct{}`) are not emitted as `references` edges to prevent noise from type-identifier occurrences in method receivers and field access. This is more conservative than the original design intent but improves signal quality.

2. **references excludes call targets**: Call-expression callees are excluded from `references` emission because they are already covered by `call` edges. This ensures no double-counting and preserves existing test expectations.

### Descriptors Added

- `builtins` — Go built-in identifier set for `references` guard
- `fileLevelNames` — set of names declared at file scope (excluded from writeSet)
- `buildCallTargetSet` — per-function map of call-callee identifiers for `references` guard
- `typeNames` — pre-built set of file-scope type declarations for `references` guard

### Risks Observed

| Risk | Status |
|------|--------|
| references false positives from type-name occurrences | ✅ Mitigated: `typeNames` guard excludes type-name idents |
| references double-counting call targets | ✅ Mitigated: `buildCallTargetSet` excludes call callees |
| RangeStmt LHS not in `*ast.RangeStmt` fields | ✅ Mitigated: `ast.Inspect` walk captures all declared identifiers |
| Write-set includes file-level var declarations | ✅ Mitigated: `fileLevelNames` pre-built set |

### Next Recommended

1. **SDD verify** for `codemap-impact-edges-v1` (full change: Phase 1 + Phase 2)
2. **SDD archive** with canonical spec sync
3. Proceed to next change in backlog or evaluate deadcode v1.1 (interface/reflection edges)