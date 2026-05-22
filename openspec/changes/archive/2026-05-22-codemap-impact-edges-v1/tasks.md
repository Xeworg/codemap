# Tasks: codemap-impact-edges-v1

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 300–540 (total across phases) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR A (Phase 1: type_use + imports) → PR B (Phase 2: references + casts) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

```text
Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium
```

**Rationale:** Two phases map cleanly to two stacked PRs. Phase 1 (`type_use` + `imports`) delivers measurable tier diversity quickly within budget. Phase 2 adds `references` + optional `casts` only if Phase 1 signal is stable and total diff remains reviewable. Explicit decision gate before apply because total LOC may approach or exceed 400 lines depending on test fixture volume. Stacked-to-main keeps integration pressure minimal.

---

## Phase 1 — `type_use` + `imports` (PR A)

**Scope:** `packages/coding-agent/codemap/indexer/` + `packages/coding-agent/codemap/cli/`

**Target:** Extend `EdgeExtractor.ExtractEdges()` with `type_use` and `imports` emitters; add unit and integration tests.

### P1 — Add `type_use` extractor helpers

**File:** `packages/coding-agent/codemap/indexer/edges.go`

Add:
```go
// resolveTypeExpr resolves a type expression to a SymbolKey.
// Strips pointer stars and package qualifiers at AST level.
func (ee *EdgeExtractor) resolveTypeExpr(expr ast.Expr) (SymbolKey, bool)

// emitTypeUsesFromTypeSpec emits type_use edges from a *ast.TypeSpec node.
func (ee *EdgeExtractor) emitTypeUsesFromTypeSpec(ts *ast.TypeSpec)

// emitTypeUsesFromFieldList emits type_use edges from struct fields or func params/results.
func (ee *EdgeExtractor) emitTypeUsesFromFieldList(fl *ast.FieldList)

// emitTypeUsesFromVarSpec emits type_use edges from var/const declarations with explicit type.
func (ee *EdgeExtractor) emitTypeUsesFromVarSpec(vs *ast.ValueSpec)
```

Wire into `ExtractEdges()` after the existing call-emission loop. Emit only when both `From` and `To` resolve.

**Verification:** `go test ./packages/coding-agent/codemap/indexer/...` (type_use tests, see P3).

### P2 — Add `imports` extractor

**File:** `packages/coding-agent/codemap/indexer/edges.go`

Add:
```go
// importAlias maps local alias to package path, built once per file from *ast.ImportSpec.
type importAlias map[string]string

// buildImportAlias scans file declarations and populates alias map.
func (ee *EdgeExtractor) buildImportAlias() importAlias

// emitImportEdges walks selectors and emits imports edges for resolvable alias references.
func (ee *EdgeExtractor) emitImportEdges()
```

Emit only when selector identifier resolves to a known symbol key. Skip unresolved alias references.

**Verification:** `go test ./packages/coding-agent/codemap/indexer/...` (imports tests, see P4).

### P3 — Unit tests for `type_use`

**File:** `packages/coding-agent/codemap/indexer/edges_test.go`

Add `TestEdgeExtractor_TypeUse_FromStructFields`, `TestEdgeExtractor_TypeUse_FromFuncParams`, `TestEdgeExtractor_TypeUse_FromFuncResults`, `TestEdgeExtractor_TypeUse_FromVarSpec`, `TestEdgeExtractor_TypeUse_Unresolved_Skips`.

Each test:
- Has concrete Go source with typed field/param/result/var.
- Parses file with `NewEdgeExtractor`, calls `ExtractEdges`.
- Asserts correct `EdgeIntent` count, `Kind = "type_use"`, and resolved `From`/`To` keys.
- Negative test: unresolved type → no edge emitted, no error.

**Verification:** `go test -run TypeUse ./packages/coding-agent/codemap/indexer/...` pass.

### P4 — Unit tests for `imports`

**File:** `packages/coding-agent/codemap/indexer/edges_test.go`

Add `TestEdgeExtractor_Imports_AliasUsed`, `TestEdgeExtractor_Imports_UnresolvedAlias_Skips`, `TestEdgeExtractor_Imports_PackagePathNotResolved_Skips`.

Each test:
- Has concrete Go source with import alias + selector use.
- Asserts `Kind = "imports"` edge with resolved `From`/`To`.
- Negative test: unresolved alias → no edge, no error.

**Verification:** `go test -run Imports ./packages/coding-agent/codemap/indexer/...` pass.

### P5 — Integration test: non-high tier appears in impact output

**File:** `packages/coding-agent/codemap/cli/impact_cmd_test.go`

Add `TestImpact_NonCallEdges_ProduceMediumOrLowTier`.

Fixture package in `packages/coding-agent/codemap/testdata/` must contain:
- A struct type `T` used in a function parameter.
- An imported aliased package whose selector is used.
- At least one regular call edge (to maintain baseline).

Assert:
- Impact output includes at least one finding with `risk_tier` in `["medium", "low"]`.
- Envelope structure unchanged (`schema_version`, `ok`, `data`, `meta`).
- `confidence` and `evidence` present per finding.

**Verification:** `go test -run NonCall ./packages/coding-agent/codemap/cli/...` pass.

### P6 — Regression gate

**Command:** `go test ./...`

All packages must pass. Specifically: existing `deadcode`, `symbol`, `history`, `impact determinism` tests must remain green.

**Verification:** full `go test ./...` passes before proceeding to Phase 2.

---

## Phase 2 — `references` + `casts` (PR B, conditional)

**Scope:** `packages/coding-agent/codemap/indexer/` + `packages/coding-agent/codemap/cli/`

**Target:** Extend edge extraction with guarded `references` and optional `casts`; add unit tests; extend integration coverage.

**Gate:** Proceed only if Phase 1 post-merge signal is stable and expected Phase 2 diff stays within review budget.

### P7 — Add `references` extractor

**File:** `packages/coding-agent/codemap/indexer/edges.go`

Add:
```go
// emitReferenceEdges walks value-position *ast.Ident nodes and emits references edges.
// Guards: skip declarations, keywords, package aliases, unresolved identifiers.
func (ee *EdgeExtractor) emitReferenceEdges()
```

Exclude `*ast.AssignStmt` LHS (declarations), language builtins (`len`, `append`, etc.), and identifiers not in `ee.syms`.

**Verification:** `go test -run References ./packages/coding-agent/codemap/indexer/...` (see P9).

### P8 — Optional `casts` extractor

**File:** `packages/coding-agent/codemap/indexer/edges.go`

Add:
```go
// emitCastEdges walks *ast.TypeAssertExpr and emits casts edges for resolvable targets.
func (ee *EdgeExtractor) emitCastEdges()
```

Emit only when asserted type resolves to a known symbol key.

**Verification:** `go test -run Casts ./packages/coding-agent/codemap/indexer/...` (see P9).

### P9 — Unit tests for `references` and `casts`

**File:** `packages/coding-agent/codemap/indexer/edges_test.go`

Add:
- `TestEdgeExtractor_References_ResolvableIdent` — positive: var used after declaration emits `references` edge.
- `TestEdgeExtractor_References_Declaration_Skips` — negative: identifier in declaration position → no edge.
- `TestEdgeExtractor_References_Unresolved_Skips` — negative: identifier not in symbol map → no edge, no error.
- `TestEdgeExtractor_References_PackageAlias_Skips` — negative: package-qualified identifier → no edge.
- `TestEdgeExtractor_Casts_ResolvableAssertion` — positive: type assertion where asserted type resolves.
- `TestEdgeExtractor_Casts_UnresolvedType_Skips` — negative: unresolved asserted type → no edge.

**Verification:** `go test -run 'References|Casts' ./packages/coding-agent/codemap/indexer/...` pass.

### P10 — Extended integration test

**File:** `packages/coding-agent/codemap/cli/impact_cmd_test.go`

Add `TestImpact_TierDiversity_WithAllEdgeKinds` asserting that when the fixture has call + type_use + imports + references edges present, the impact output contains findings spanning `high`, `medium`, and `low` tiers.

Add `TestImpact_Determinism_WithExpandedEdgeTypes` — run impact on same symbol twice; assert identical JSON output.

**Verification:** `go test -run 'TierDiversity|Determinism' ./packages/coding-agent/codemap/cli/...` pass.

### P11 — Regression gate

**Command:** `go test ./...`

Full suite must pass. No envelope, store, or CLI contract changes.

---

## Docs/Changelog (final step, no separate PR)

**File:** `docs/codemap-cli-json-contract.md` (update edge-type examples if needed)

**File:** `CHANGELOG.md` — add entry under Unreleased:

```markdown
### Changed

- **impact tier diversity**: `codemap impact` now surfaces `medium` and `low` risk
  findings via indexed `type_use`, `imports`, and `references` edges, in addition
  to call-based `high` risk findings. Edge extraction expanded at index time
  using file-local AST resolution.
```

---

## Dependency ordering

```
Phase 1 gate: go test ./...
P1 → P2 → P3 → P4 → P5 → P6  (Phase 1: type_use + imports)
                                        ↓
                           Phase 1 review + merge
                                        ↓
Phase 2 gate: go test ./...
P7 → P8 → P9 → P10 → P11  (Phase 2: references + casts, conditional)
```

**Rollback boundary per phase:** If `go test ./...` fails after a phase, revert to the last commit on the previous phase and re-evaluate.

---

## Risks and mitigation

| Risk | Mitigation |
|------|-----------|
| False-positive reference edges from unresolved idents | Strict resolvable-only guard; skip declarations/keywords/unresolved |
| Order nondeterminism from map iteration | Fixed extraction order (calls → type_use → imports → references → casts) + determinism tests |
| Phase 2 pushes diff over 400 lines | Stop immediately after Phase 1 and reassess slice; split into separate PR if needed |
| deadcode classifier destabilized by edge expansion | Phase 1 regression gate includes deadcode tests; flag any interference |
| Performance regression from larger AST walks | perf guardrail test runs after each phase; bail if regression detected |