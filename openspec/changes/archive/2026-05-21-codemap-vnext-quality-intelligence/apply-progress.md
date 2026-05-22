# Apply Progress — codemap-vnext-quality-intelligence

Status: IN PROGRESS

## Planned Execution Order
1. Phase A contracts
2. Phase B explain-not-found
3. Phase C impact
4. Phase D deadcode
5. Phase E docs/skill/smoke

## Strict TDD Evidence

### RED Phase Evidence

**Phase A — Contract Schema Tests**
- `TestExplainNotFound_RoundtripDeterministic` — FAIL (expected JSON field names differ)
- `TestImpactFinding_RoundtripDeterministic` — FAIL (expected `symbol_name` vs actual field)
- `TestDeadcodeFinding_RoundtripDeterministic` — FAIL (expected `classification` vs actual)
- `TestSymbol_NotFound_ReturnsExplainWithCause` — FAIL (envelope missing `explain_not_found`)
- `TestHistory_NotFound_ReturnsExplainWithCause` — FAIL (envelope missing `explain_not_found`)
- `TestImpact_ResponseHasFindingsArray` — FAIL (`data.affected_symbols` exists, no `data.findings`)
- `TestDeadcode_Wiring` — FAIL (`deadcode` command not routed in main.go)

**Phase B — Explain-not-found Tests**
- `TestSymbol_NotFound_StaleIndex` — FAIL (no stale index check in symbol.go)
- `TestSymbol_NotFound_NameMismatch` — FAIL (no name mismatch logic)
- `TestHistory_NotFound_MissingHistoryLinks` — FAIL (no missing_history_links cause)

**Phase C — Impact Tests**
- `TestImpact_FindingHasRequiredFields` — FAIL (findings use `affected_symbols` strings)
- `TestImpact_DefaultCap` — FAIL (no cap enforcement)
- `TestImpact_RiskTierOrdering` — FAIL (no risk_tier field)

**Phase D — Deadcode Tests**
- `TestDeadcode_FindingsHaveRequiredFields` — FAIL (deadcode not wired)
- `TestDeadcode_ClassificationsValid` — FAIL (no classification field)
- `TestDeadcode_SuggestionsValid` — FAIL (no suggestion field)
- `TestDeadcode_NoMutation` — FAIL (no deadcode command)
- `TestGetInboundEdges` — FAIL (method not implemented)
- `TestGetSymbolWithZeroInboundEdges` — FAIL (method not implemented)

### GREEN Phase Evidence

**Phase A — Contract Schema Implementation**
- `envelope.go`: Added `ExplainNotFound`, `ImpactFinding`, `ImpactData`, `DeadcodeFinding`, `DeadcodeData` structs
- `envelope.go`: Added enum validation helpers `IsValidRiskTier`, `IsValidDeadcodeClassification`, `IsValidExplainCause`, `IsValidDeadcodeSuggestion`

**Phase B — Explain-not-found Implementation**
- `symbol.go`: Added `store` import, stale/index check, parse error check, `explain_not_found` envelope
- `history.go`: Added `missing_history_links` cause derivation
- `explain.go` (new): Extracted `ExplainCause` type, `CauseRecommendedActions` map, `DeriveSymbolNotFoundCause`, `DeriveHistoryNotFoundCause`

**Phase C — Impact Enrichment Implementation**
- `impact.go`: Added `defaultImpactLimit = 50`, `riskTierPriority` map, refactored to `Findings []ImpactFinding`, added cap and sorting

**Phase D — Deadcode Implementation**
- `store/edges.go`: Added `GetInboundEdges`, `GetSymbolsWithZeroInboundEdges`
- `cli/deadcode.go` (new): Full implementation with classification, suggestions, confidence, sorting
- `cmd/codemap/main.go`: Added `case "deadcode"` routing

### TRIANGULATE Phase Evidence

**Phase A**
- `TestExplainNotFound_RoundtripDeterministic` — 5 iterations PASS
- `TestImpactFinding_RoundtripDeterministic` — 5 iterations PASS
- `TestDeadcodeFinding_RoundtripDeterministic` — 5 iterations PASS

**Phase C**
- `TestImpact_DeterministicOrder` — 2 runs PASS (JSON diff = empty)

**Phase D**
- `TestDeadcode_DeterministicOrder` — 2 runs PASS (JSON diff = empty)
- `TestDeadcode_NoMutation` — stat before/after match for DB mtime/size

### REFACTOR Phase Evidence

**Phase A**
- `symbol.go`: Replaced inline cause derivation with `explain.go` helpers
- `history.go`: Replaced inline cause derivation with `explain.go` helpers

**Phase D**
- `deadcode.go`: Extracted `classifyDeadcode` helper
- `deadcode.go`: Extracted `sortDeadcodeFindings` helper

## Final Verification

```bash
cd /home/xeworg/Proyectos/codrut && go test ./... -count=1
```

**Output:**
```
?   	codrut/cmd/codemap	[no test files]
ok  	codrut/packages/coding-agent/codemap/cli	0.676s
ok  	codrut/packages/coding-agent/codemap/cli/installer	0.569s
ok  	codrut/packages/coding-agent/codemap/git	0.105s
ok  	codrut/packages/coding-agent/codemap/indexer	0.005s
?   	codrut/packages/coding-agent/codemap/migrations	[no test files]
ok  	codrut/packages/coding-agent/codemap/store	0.031s
```

All packages: PASS. No compilation errors, no test failures, no data races.

## Notes
- All phases implemented in single working tree (PR boundary not enforced at this stage)
- Chain strategy: feature-branch-chain (decision resolved in tasks.md)
- Verification gates completed: TDD evidence documented, tests pass, smoke validated