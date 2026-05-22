# Design — codemap-vnext-quality-intelligence

## Scope and goals
Implement quality-intelligence for `impact`, `symbol/history` miss diagnostics, and new report-only `deadcode` under `packages/coding-agent/codemap` while preserving envelope compatibility (`schema_version: 1.0`).

## Architecture and file plan

### 1) Shared contracts (`cli/envelope.go`)
- Add typed payloads:
  - `ExplainNotFound` with `cause` + `recommended_actions`.
  - `ImpactFinding` + upgraded `ImpactData` (`findings[]` with `risk_tier|confidence|evidence`).
  - `DeadcodeFinding` + `DeadcodeData` (`classification|suggestion|confidence|evidence`).
- Keep existing top-level `Envelope` unchanged.
- Add enum constants and validation helpers to prevent drift.

### 2) Explain-not-found (`cli/symbol.go`, `cli/history.go`)
- On unresolved symbol/history, return `ok=false` envelope with structured `data.explain_not_found` instead of plain string-only diagnostics.
- Cause derivation (deterministic, first matching strongest cause):
  1. `stale_index` when snapshot stale.
  2. `parse_error` when parse errors exist for candidate file scope.
  3. `missing_history_links` (history only) when symbol exists but history edges absent.
  4. fallback `name_mismatch`.
- Recommended action mapping is static per cause and command to remain deterministic.

### 3) Impact enrichment (`cli/impact.go`)
- Replace flat `affected_symbols[]` response with `findings[]` entries.
- Populate per finding:
  - identity/location from symbol lookup
  - `risk_tier` from edge type heuristics
  - `confidence` from evidence density and edge/source quality
  - `evidence[]` with stable source references
- Add default cap constant (e.g. `defaultImpactLimit = 50`) applied when no explicit limit flag.
- Deterministic ordering: `risk_tier` priority (`high>medium>low`), then confidence rank, then symbol name, then file path.

### 4) Deadcode command (`cmd/codemap/main.go`, new `cli/deadcode.go`)
- Add `deadcode` command route and help text.
- Detection pipeline:
  1. candidate symbols with zero inbound references.
  2. exclusions (generated/test fixtures/configured allowlist if present).
  3. classification + suggestion mapping.
- Output is strictly report-only (no writes/mutations).
- Deterministic ordering by classification rank, then confidence rank, then symbol/file.

## Data flow
1. CLI command parses args and resolves DB.
2. Store queries fetch snapshot/meta/symbol/edges/history/parse-error context.
3. Command-level classifiers map raw graph state → constrained enums.
4. Response assembler emits stable JSON envelope.

## Risk controls
- **False-positive deadcode**: conservative classification; ambiguous cases downgraded to `uncertain` + `justify`.
- **Output explosion in impact**: default cap + explicit sorting + evidence truncation guard.
- **Contract drift**: central enum constants and table-driven contract tests.
- **Non-determinism**: all slices sorted with fully-specified tiebreakers.
- **Unsafe behavior**: deadcode implementation contains no write paths and tests assert source immutability.

## Strict TDD plan
Test runner: `go test ./packages/coding-agent/codemap/...`

### RED-1 (contracts)
- Add envelope serialization tests for new payload schemas/enums.
- Add negative tests that reject enum values outside spec.

### RED-2 (symbol/history explain)
- Add failing command tests for not-found responses containing allowed causes + actions.
- Add stale/parse/missing-history scenario fixtures.

### RED-3 (impact)
- Add failing tests for required finding fields, cap defaulting, and repeat-run deterministic order.

### RED-4 (deadcode)
- Add failing tests for command wiring, required finding fields, enum constraints, deterministic order, and no-mutation behavior.

### GREEN
- Implement minimal code per red block, one block at a time.

### REFACTOR
- Extract reusable rank/sort helpers and cause/action mappers.
- Keep behavior unchanged; re-run full suite after each refactor.

## Rollout plan
1. Ship contracts + tests first.
2. Ship explain-not-found slice.
3. Ship impact enrichment + cap.
4. Ship deadcode command.
5. Update docs/skill and run smoke (`index`, `symbol`, `history`, `impact`, `deadcode`).

## Verification gates
- Unit + command tests green.
- Determinism tests pass across repeated runs.
- Manual smoke confirms deadcode is report-only and explain diagnostics actionable.
