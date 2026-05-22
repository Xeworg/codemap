# Tasks — codemap-vnext-quality-intelligence

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 480–620 |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1: Contracts + symbol/history explain-not-found · PR 2: Impact enrichment + deadcode |
| Delivery strategy | feature-branch-chain |

```
Decision resolved: feature-branch-chain
```

---

## PR 1: Contracts + Symbol/History Explain-Not-Found

### Phase 1: RED — Contract Schema Tests

**Task 1.1** `packages/coding-agent/codemap/cli/envelope_test.go`
- Add tests for `ExplainNotFound` struct serialization: `cause` must be one of `stale_index|name_mismatch|parse_error|missing_history_links`, `recommended_actions` must be non-empty slice.
- Add negative tests: reject unknown cause values, reject empty actions slice.
- Add tests for `ImpactFinding` with required fields: `symbol_name`, `file`, `risk_tier` (`high|medium|low`), `confidence` (`high|medium|low`), `evidence` (non-empty).
- Add tests for `DeadcodeFinding` with required fields: `symbol_name`, `file`, `classification` (`unused|likely-unused|uncertain`), `suggestion` (`remove|deprecate|justify`), `confidence`, `evidence`.
- Add table-driven test for enum validation helpers: `IsValidRiskTier`, `IsValidDeadcodeClassification`, `IsValidExplainCause`.

**Task 1.2** `packages/coding-agent/codemap/cli/envelope_test.go`
- Add deterministic order tests: encode/decode roundtrip produces identical JSON for `ExplainNotFound`, `ImpactFinding`, `DeadcodeFinding`.
- Assert sorting stability: same input yields same output across 5 iterations.

### Phase 2: GREEN — Contract Implementation

**Task 2.1** `packages/coding-agent/codemap/cli/envelope.go`
- Add `ExplainNotFound` struct with `Cause` (string) and `RecommendedActions` ([]string) fields.
- Add `ImpactFinding` struct with `SymbolName`, `File`, `Kind`, `StartLine`, `EndLine`, `RiskTier`, `Confidence`, `Evidence` fields.
- Add `ImpactData` replacement struct with `TargetSymbol` and `Findings []ImpactFinding`.
- Add `DeadcodeFinding` struct with `SymbolName`, `File`, `Kind`, `StartLine`, `EndLine`, `Classification`, `Suggestion`, `Confidence`, `Evidence` fields.
- Add `DeadcodeData` struct with `Findings []DeadcodeFinding`.
- Add enum validation helpers: `IsValidRiskTier`, `IsValidDeadcodeClassification`, `IsValidExplainCause`, `IsValidDeadcodeSuggestion`.

### Phase 3: RED — Symbol/History Explain Tests

**Task 3.1** `packages/coding-agent/codemap/cli/symbol_test.go`
- Add test `TestSymbol_NotFound_ReturnsExplainWithCause`: query non-existent symbol; assert `ok=false`, `data.explain_not_found.cause` is one of valid causes, `recommended_actions` non-empty.
- Add test `TestSymbol_NotFound_StaleIndex`: mock stale snapshot; assert cause = `stale_index`.
- Add test `TestSymbol_NotFound_NameMismatch`: mock fresh snapshot with no parse errors; assert cause = `name_mismatch`.

**Task 3.2** `packages/coding-agent/codemap/cli/history_test.go`
- Add test `TestHistory_NotFound_ReturnsExplainWithCause`: query non-existent symbol; assert structured `explain_not_found`.
- Add test `TestHistory_NotFound_MissingHistoryLinks`: mock symbol exists but `GetSymbolHistory` returns empty; assert cause = `missing_history_links`.

### Phase 4: GREEN — Symbol/History Explain Implementation

**Task 4.1** `packages/coding-agent/codemap/cli/symbol.go`
- Add import for `store` package.
- In `RunSymbol` not-found branch: call `store.GetParseErrorsForSnapshot` to check for parse errors in candidate file scope.
- Call `store.GetLatestSnapshotMeta` to check `IsStale`.
- Derive cause using deterministic first-match: `stale_index` → `parse_error` → `name_mismatch`.
- Build `ExplainNotFound` with cause and static `RecommendedActions` map.
- Return `NewEnvelope("symbol", false, map[string]interface{}{"explain_not_found": enf}, nil, meta)`.

**Task 4.2** `packages/coding-agent/codemap/cli/history.go`
- In `RunHistory` not-found branch: derive cause with `missing_history_links` check (symbol exists but history edges absent).
- Return structured `explain_not_found` envelope.

**Task 4.3** `packages/coding-agent/codemap/cli/explain.go` (new file)
- Extract `ExplainCause` type and `CauseRecommendedActions` map as reusable constants.
- Add `DeriveSymbolNotFoundCause(ctx, db, symbolArg, symExists bool) (string, []string)` helper.
- Add `DeriveHistoryNotFoundCause(ctx, db, symbolArg string, symExists, hasHistory bool) (string, []string)` helper.

### Phase 5: REFACTOR

**Task 5.1** `packages/coding-agent/codemap/cli/symbol.go`, `history.go`
- Replace inline cause derivation with calls to `explain.go` helpers.

**Task 5.2** Verify `go test ./packages/coding-agent/codemap/cli/...` passes.

---

## PR 2: Impact Enrichment + Deadcode Command

### Phase 1: RED — Impact Enrichment Tests

**Task 1.1** `packages/coding-agent/codemap/cli/impact_test.go`
- Add test `TestImpact_ResponseHasFindingsArray`: assert `data.findings` is non-nil slice (not `affected_symbols`).
- Add test `TestImpact_FindingHasRequiredFields`: each finding must have `symbol_name`, `risk_tier`, `confidence`, `evidence`.
- Add test `TestImpact_DefaultCap`: query symbol with >50 findings; assert response length ≤ 50.
- Add test `TestImpact_DeterministicOrder`: run twice with same args; assert JSON diff is empty.
- Add test `TestImpact_RiskTierOrdering`: assert `high` findings appear before `medium` and `low`.

### Phase 2: GREEN — Impact Enrichment Implementation

**Task 2.1** `packages/coding-agent/codemap/cli/impact.go`
- Add `defaultImpactLimit = 50` constant.
- Add `riskTierPriority` map: `high=0, medium=1, low=2`.
- Refactor `RunImpact`: replace `AffectedSymbols []string` with `Findings []ImpactFinding`.
- For each affected symbol, build `ImpactFinding`: resolve symbol details, derive `RiskTier` from `EdgeType` heuristics, compute `Confidence` from evidence density, populate `Evidence` entries.
- Apply `defaultImpactLimit` cap after sorting.
- Sort with tiebreaker: `risk_tier` priority → confidence rank → symbol name → file path.
- Update envelope response to use new `ImpactData` struct.

### Phase 3: RED — Deadcode Command Tests

**Task 3.1** `packages/coding-agent/codemap/cli/deadcode_test.go` (new file)
- Add test `TestDeadcode_Wiring`: run `codemap deadcode`; assert exit 0 and valid JSON envelope.
- Add test `TestDeadcode_FindingsHaveRequiredFields`: each finding must have `symbol_name`, `classification`, `suggestion`, `confidence`, `evidence`.
- Add test `TestDeadcode_ClassificationsValid`: assert all `classification` values are `unused|likely-unused|uncertain`.
- Add test `TestDeadcode_SuggestionsValid`: assert all `suggestion` values are `remove|deprecate|justify`.
- Add test `TestDeadcode_DeterministicOrder`: run twice; assert JSON diff is empty.
- Add test `TestDeadcode_NoMutation`: assert database file unchanged after command (stat before/after mtime and size match).

**Task 3.2** `packages/coding-agent/codemap/store/edges_test.go`
- Add `TestGetInboundEdges` helper test: `GetSymbolEdges` filter for `to_symbol_id = ?` only.
- Add `TestGetSymbolWithZeroInboundEdges`: integration test returning symbols with no incoming edges.

### Phase 4: GREEN — Deadcode Implementation

**Task 4.1** `packages/coding-agent/codemap/store/edges.go`
- Add `GetInboundEdges(ctx, db, symbolID) ([]SymbolEdge, error)`: `SELECT * FROM edges WHERE to_symbol_id = ?`.
- Add `GetSymbolsWithZeroInboundEdges(ctx, db, snapshotID) ([]SymbolRow, error)`: subquery for symbols with no inbound edges.

**Task 4.2** `packages/coding-agent/codemap/cli/deadcode.go` (new file)
- Implement `RunDeadcode(ctx, w, args, repoRoot) int`.
- Parse `-limit` flag (default 100) and optional symbol-name filter.
- Query `store.GetSymbolsWithZeroInboundEdges`.
- Apply exclusions: skip generated files, test fixtures, configured allowlist (if present).
- Classify each finding: `unused` (zero edges + excluded), `likely-unused` (single edge to other unused), `uncertain` (default).
- Map classification to suggestion: `unused` → `remove`, `likely-unused` → `deprecate`, `uncertain` → `justify`.
- Compute confidence from edge count and file kind heuristics.
- Sort by classification rank → confidence → symbol name → file.
- Apply limit cap.
- Return `DeadcodeData` envelope.

**Task 4.3** `cmd/codemap/main.go`
- Add `case "deadcode"` in switch: route to `cli.RunDeadcode`.
- Add `deadcode` case in `helpRoot` and `helpFor`.

### Phase 5: REFACTOR

**Task 5.1** `packages/coding-agent/codemap/cli/deadcode.go`
- Extract `classifyDeadcode(symbol, edgeCount int) (classification, suggestion, confidence)` helper.
- Extract `sortDeadcodeFindings(findings []DeadcodeFinding)` helper.

**Task 5.2** Verify `go test ./packages/coding-agent/codemap/...` passes.

---

## Documentation + Smoke (Post-PR2)

**Task D.1** `docs/codemap-cli-json-contract.md`
- Document `explain_not_found` payload for `symbol` and `history` commands.
- Document `findings[]` format for `impact` command with `risk_tier|confidence|evidence` fields.
- Document `deadcode` command response format with `classification|suggestion|confidence|evidence`.
- Add JSON examples for each new response type.

**Task D.2** `integrations/pi/skills/codemap-usage/SKILL.md`
- Update `impact` command description to mention risk tiers and cap.
- Add `deadcode` command entry with usage and example.
- Update `symbol`/`history` miss handling guidance to reference `explain_not_found`.

**Task D.3** Smoke validation
- Run `codemap index` on test fixture.
- Run `codemap symbol NonExistent`; assert `explain_not_found` present.
- Run `codemap history NonExistent`; assert `explain_not_found` present.
- Run `codemap impact SomeSymbol`; assert `findings[]` with risk tiers.
- Run `codemap deadcode`; assert report-only, no file mutations.

---

## Verification Gates

1. `go test ./packages/coding-agent/codemap/...` passes for each PR.
2. Determinism tests pass across 5 repeated runs (JSON diff = empty).
3. Deadcode assertion: source DB mtime/size unchanged after command.
4. Smoke manual validation for all commands.

## Rollback Triggers

- PR1 regression: revert to prior envelope compatibility.
- PR2 regression: disable `deadcode` routing in `main.go` until fixed.
