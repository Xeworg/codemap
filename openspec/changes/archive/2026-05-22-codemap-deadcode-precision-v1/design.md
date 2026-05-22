# Design: codemap-deadcode-precision-v1

## Executive summary
Implement **Option 1**: produce real symbol edges at index time, expand symbol coverage to methods + `init`, and update deadcode classification to use explicit edge evidence plus implicit runtime/public-entry heuristics.

Primary code scope remains:
- `packages/coding-agent/codemap/indexer`
- `packages/coding-agent/codemap/store`
- `packages/coding-agent/codemap/cli/deadcode.go`

## Current-state findings
- `indexer.ExtractGoSymbols` currently skips methods (`d.Recv != nil`) and only emits top-level funcs/vars/consts/types.
- `cli.RunIndex` persists files/symbols/history, but no production path currently extracts and writes edges.
- `deadcode` currently starts from `GetSymbolsWithZeroInboundEdges`, then always emits `no_inbound_edges` evidence and mostly `unused` classifications.
- Edge store/query support exists (`edges` table, `UpsertEdge`, `GetInboundEdges`), so missing piece is index-time extraction + classifier usage.

## Architecture (Option 1)

### 1) Index-time edge extraction pipeline
Add a new indexer pass that, for each parsed Go file:
1. Builds symbol declarations (including methods) with stable symbol keys.
2. Walks AST call/reference sites and resolves target symbol keys using file/package-local symbol map first (MVP scope).
3. Emits directed edges `(fromSymbolKey -> toSymbolKey, edgeType)` for `calls` and basic `ref`/`type_use` where resolvable.
4. Persists resolved edges during `cli index` transaction after symbols are inserted.

Notes:
- Keep fail-soft behavior: unresolved refs are skipped, indexing still succeeds.
- Keep complexity bounded: no SSA/whole-program call graph.

### 2) Method + `init` symbol support
Update symbol extraction model:
- Include method declarations as `kind=method` with receiver-qualified name (e.g., `Type.Method` or `(*Type).Method`).
- Include `init` declarations as callable symbols (`kind=func`, name `init`), preserving file/range.

This enables:
- Inbound edge tracking to methods.
- Explicit heuristic handling for runtime-invoked `init`.

### 3) Store/query updates for deadcode evidence
- Add/adjust store query helper to fetch **all symbols with inbound counts** (not only zero-inbound set), scoped to latest snapshot.
- Keep deterministic ordering in SQL.
- Ensure edge cleanup remains correct on reindex/replace (existing delete-from-symbol rows + edge delete path may need inbound-side cleanup verification).

### 4) Deadcode classifier updates
Refactor deadcode classification around evidence tiers:
- **Explicit evidence present (inboundCount > 0):** not `unused`; classify `uncertain`/non-removal with evidence `inbound_edges`.
- **No explicit edges + implicit runtime/public-entry heuristic matches** (`main`, `init`, exported API candidates, `cmd/` entrypoint patterns): classify `uncertain` with low confidence and heuristic evidence.
- **No explicit edges + no implicit heuristic:** classify `unused` with higher confidence by kind.

Evidence entries become composable (`no_inbound_edges`, `inbound_edges`, `implicit_runtime_entry`, `public_api_surface`).

## Data flow
1. `codemap index` parses changed files.
2. Indexer emits symbols + edge intents.
3. CLI persistence writes symbols, maps symbol keys→row IDs, upserts edges.
4. `codemap deadcode` queries latest snapshot symbols with inbound counts and metadata.
5. Classifier computes class/suggestion/confidence/evidence.
6. Output sorted deterministically.

## Planned file changes
- `packages/coding-agent/codemap/indexer/go_parser.go` (method + init extraction)
- `packages/coding-agent/codemap/indexer/*` (new edge extraction/types)
- `packages/coding-agent/codemap/cli/index.go` (persist extracted edges)
- `packages/coding-agent/codemap/store/symbols.go` (edge persistence helper usage/cleanup hardening)
- `packages/coding-agent/codemap/store/edges.go` (inbound-count query helpers)
- `packages/coding-agent/codemap/cli/deadcode.go` (new classifier and evidence logic)
- `packages/coding-agent/codemap/**/*_test.go` (new/updated tests)

## Strict TDD strategy (mandatory gate: `go test ./...`)

### RED
1. Indexer tests fail for method extraction and `init` symbol inclusion.
2. Indexer/store integration tests fail expecting persisted edges from fixture calls.
3. Deadcode tests fail expecting:
   - method usage not marked `unused`,
   - `init`/`main`/exported entrypoints classified `uncertain` when no explicit inbound edges,
   - explicit inbound evidence reflected in confidence/evidence.
4. Determinism tests fail if new evidence ordering/classification ordering regresses.

### GREEN
- Implement minimum code to satisfy each failing test slice in order:
  1) symbols (methods/init),
  2) edge extraction + persistence,
  3) deadcode query/classifier.

### REFACTOR
- Extract classifier helpers (heuristic predicates, evidence builders).
- Keep sort/order and envelope contract locked by tests.

### CI gate
- Required: `go test ./...` passes at each slice and final merge.

## Rollout slices (review-budget aware)
Target <=400 changed lines per PR.

1. **Slice A: Symbol coverage foundation**
   - Methods + `init` extraction
   - Parser/indexer unit tests
2. **Slice B: Edge extraction + persistence**
   - Edge intent model, index transaction wiring, store/integration tests
3. **Slice C: Deadcode precision classifier**
   - Inbound-aware queries, heuristics, evidence/confidence tests
4. **Slice D: Precision regression corpus/docs**
   - Curated fixtures, determinism/precision checks, operational docs update

## Risk controls
- **False confidence from partial edge resolution**: classify heuristic-only cases as `uncertain`, never auto-remove.
- **Performance regression**: bound extraction to AST-local resolution; add/extend perf guardrail tests.
- **Edge integrity on reindex**: validate cleanup/rewrite behavior with replace-file integration tests.
- **Determinism drift**: enforce stable SQL ordering + `sortDeadcodeFindings` tests.
- **Scope creep to whole-program analysis**: keep non-goals explicit (no SSA/full call graph in v1).

## Test matrix additions
- Indexer unit: methods, pointer/value receivers, multiple `init` funcs.
- Indexer/store integration: cross-symbol call in fixture creates inbound edge.
- Deadcode command: explicit inbound => non-unused; heuristic-only entrypoints => uncertain; private no-edge => unused.
- Regression: deterministic JSON output unchanged across repeated runs.

## Contracts and compatibility
- No breaking CLI envelope changes.
- Deadcode classifications remain within existing enum set.
- Behavior change is precision-oriented: fewer high-confidence false positives, more explicit uncertainty labeling where evidence is implicit.

## Open questions (implementation-time)
- Exact exported-API heuristic boundaries (all exported symbols vs package-mode sensitive rule).
- Minimum viable edge kinds included in v1 (`calls` only vs `calls + type_use` for better precision).

## Skill resolution
- `skill_resolution: none`
