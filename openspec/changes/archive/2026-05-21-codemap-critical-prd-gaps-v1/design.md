# Design — codemap-critical-prd-gaps-v1

## Overview
Implement three CLI additions in `packages/coding-agent/codemap`:
1. `codemap migrate` explicit migration command.
2. `codemap query --json` deterministic envelope-v1 command (symbol/history oriented lookup).
3. `codemap impact --json` deterministic envelope-v1 command (impact graph over symbol history links).

All commands preserve envelope v1 (`schema_version: "1.0"`) and exit-code mapping:
- `0` success
- `1` runtime failure
- `2` input/validation failure
- `3` index/data-state failure

## Architecture

### Command routing
- **File:** `cmd/codemap/main.go`
- Add command cases + help text:
  - `query` → `cli.RunQuery(...)`
  - `impact` → `cli.RunImpact(...)`
  - `migrate` → `cli.RunMigrate(...)`
- Reuse existing `runWithHelp` behavior and root `-repo` semantics.

### Shared envelope + deterministic output
- **File:** `packages/coding-agent/codemap/cli/envelope.go`
- Reuse `NewEnvelope`, `WriteErrorEnvelope`, `Meta`.
- Add new payload structs with stable field order:
  - `QueryData`
  - `ImpactData`
  - `MigrateData`
- Determinism rule: explicitly sort all variable-length slices before encoding (e.g., by symbol name then file path).

### Query execution
- **New file:** `packages/coding-agent/codemap/cli/query.go`
- Responsibilities:
  - Parse flags (`--db`, `--json` accepted; JSON remains default output contract).
  - Validate required query term (exit `2` on missing/invalid input).
  - Open DB, load latest snapshot meta.
  - If no snapshot/index: exit `3` with envelope error.
  - Resolve matches from store layer (exact name first; optional prefix fallback if exact miss).
  - Build deterministic `QueryData` + envelope and emit JSON.

### Impact execution
- **New file:** `packages/coding-agent/codemap/cli/impact.go`
- Responsibilities:
  - Parse flags + validate target symbol.
  - Require indexed state (missing snapshot/index => exit `3`).
  - Fetch symbol + linked history evidence and related symbols from same file/snapshot scope.
  - Build stable impact set (`affected_symbols`, `evidence`) with deterministic sort.
  - Emit envelope v1.

### Migrate execution
- **New file:** `packages/coding-agent/codemap/cli/migrate.go`
- Responsibilities:
  - Parse flags (`--db`).
  - Open DB + run `store.NewMigrationRunner(db.DB).Migrate(ctx)`.
  - Read `CurrentSchemaVersion` before/after to report `applied|up_to_date` status.
  - Idempotent success (`0`) when no pending migrations.
  - Runtime DB/migration errors return `1`; malformed flags/input return `2`.

## Data flow
1. CLI parses command + flags in `main.go`.
2. `Run*` command resolves DB path via `ResolveDBPath`.
3. Store access:
   - `query`/`impact`: read snapshot meta + symbol/history tables.
   - `migrate`: migration runner + version checks.
4. Command builds typed `data` payload.
5. Response emitted through `Envelope.Encode()`.
6. Exit code derived from command error classification.

## File change plan
- `cmd/codemap/main.go` (route/help wiring)
- `packages/coding-agent/codemap/cli/envelope.go` (new payload structs)
- `packages/coding-agent/codemap/cli/query.go` (new)
- `packages/coding-agent/codemap/cli/impact.go` (new)
- `packages/coding-agent/codemap/cli/migrate.go` (new)
- `packages/coding-agent/codemap/cli/query_cmd_test.go` (new)
- `packages/coding-agent/codemap/cli/impact_cmd_test.go` (new)
- `packages/coding-agent/codemap/cli/migrate_cmd_test.go` (new)
- `packages/coding-agent/codemap/cli/integration_test.go` (extend cross-command envelope/exit compatibility)

## Strict TDD verification strategy

### Red phase (tests first)
1. Add envelope-compat tests for each command:
   - Top-level fields present.
   - `schema_version == "1.0"`.
   - `ok`, `errors`, `meta` semantics.
2. Add exit-code contract tests for each command covering `0/1/2/3` relevant paths.
3. Add determinism tests:
   - repeated `query --json` with same DB/index => identical JSON bytes.
   - repeated `impact --json` with same DB/index => identical JSON bytes.
4. Add migrate idempotency tests:
   - first run applies pending migrations.
   - second run is success no-op with unchanged schema version.

### Green phase
- Implement minimum command logic to satisfy failing tests.
- Keep logic thin in CLI; reuse store/envelope utilities.

### Refactor phase
- Extract shared helpers for:
  - index/data-state guard
  - deterministic sorting
  - error classification to stable exit codes
- Re-run full suite:
  - `go test -count=1 ./packages/coding-agent/codemap/cli/...`
  - `go test -count=1 ./packages/coding-agent/codemap/store/...`

## Rollout / PR slicing
1. **PR1:** `migrate` command + tests.
2. **PR2:** `impact --json` + tests.
3. **PR3:** `query --json` + determinism tests + docs/help updates.

Each PR must keep compatibility tests green for envelope v1 and stable exit-code mapping.

## Risks and mitigations
- **Ambiguous query/impact semantics:** lock deterministic data shapes in tests before implementation.
- **Exit-code drift:** centralize mapping in test table-driven cases per command.
- **Non-deterministic ordering from SQL/maps:** always sort before envelope encode.

## Skill resolution
- `skill_resolution: none` (no injected skill paths provided).