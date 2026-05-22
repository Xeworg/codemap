# Slice C: Deadcode precision classifier — COMPLETED

## Status: PASS ✅

## Changes

### Modified files
- `packages/coding-agent/codemap/cli/deadcode.go`: full rewrite — inbound-aware `classifyDeadcode` (4-arg: `inbound int, kind, name, file string`), `deadcodeEvidence` composable helper, heuristic predicates (`isRuntimeEntrypoint`, `isPublicAPI`, `isEntrypointFile`), switched main loop to `store.GetAllSymbolsWithInboundCounts` single query, removed per-symbol `GetSymbolEdges` loop.
- `packages/coding-agent/codemap/cli/envelope.go`: added evidence type constants (`EvidenceNoInboundEdges`, `EvidenceInboundEdges`, `EvidenceImplicitRuntime`, `EvidencePublicAPISurface`); added `"review"` to `ValidDeadcodeSuggestionValues`.
- `packages/coding-agent/codemap/store/edges.go`: added `SymbolWithInbound` struct + `GetAllSymbolsWithInboundCounts` single-query function.

### New files
- `packages/coding-agent/codemap/cli/deadcode_test.go`: `TestClassifyDeadcode_Deterministic` — 10-run same-input determinism assertion.

### Updated test file
- `packages/coding-agent/codemap/cli/deadcode_cmd_test.go`: added unit tests for new classifier:
  - `TestClassify_WithInboundEdges_ClassifiesUncertain`
  - `TestClassify_MainFunc_NoEdges_ClassifiesUncertain`
  - `TestClassify_InitFunc_NoEdges_ClassifiesUncertain`
  - `TestClassify_ExportedNoEdges_Uncertain`
  - `TestClassify_PrivateFuncNoEdges_Unused`
  - `TestClassify_MethodNoEdges_NotHighConfidenceUnused`
  - `TestEvidence_Composable`
  - `TestEvidence_PublicAPIComposes`
  - `TestEvidence_InboundComposes`

## Evidence

```
go test ./packages/coding-agent/codemap/cli/... ./packages/coding-agent/codemap/store/...
ok  	codrut/packages/coding-agent/codemap/cli
ok  	codrut/packages/coding-agent/codemap/store

go test ./...
ok  	codrut/packages/coding-agent/codemap/cli
ok  	codrut/packages/coding-agent/codemap/cli/installer
ok  	codrut/packages/coding-agent/codemap/git
ok  	codrut/packages/coding-agent/codemap/indexer
ok  	codrut/packages/coding-agent/codemap/store
```

## TDD cycle

- RED: compile errors on old `classifyDeadcode(inboundCount, kind)` signature (3 tests expecting 4-arg variant).
- GREEN: updated signature + added evidence constants + new store query + heuristic helpers → all compile and pass.

## Evidence tiers implemented

| Evidence type | When |
|---|---|
| `no_inbound_edges` | `inbound == 0` always present |
| `inbound_edges` | `inbound > 0` |
| `implicit_runtime_entry` | `name == "main" \|\| name == "init"` |
| `public_api_surface` | `name[0] >= 'A' && name[0] <= 'Z'` |

## Classification rules

| Condition | Classification | Suggestion | Confidence |
|---|---|---|---|
| `inbound > 0` | uncertain | review | low |
| `name == "main" \|\| "init"` | uncertain | review | low |
| Exported (uppercase start) | uncertain | justify | low |
| Private func/type | unused | remove | high |
| Private var/const | unused | remove | medium |
| method | likely-unused | remove | medium |
| Other | unused | remove | low |

## Open risks

- `review` suggestion added to enum; it was not present before. This is a correct extension for v1 precision semantics.
- Edge resolution quality depends on Slice B edge extraction coverage; symbols with real callers but unresolvable call targets will still appear as zero-inbound.