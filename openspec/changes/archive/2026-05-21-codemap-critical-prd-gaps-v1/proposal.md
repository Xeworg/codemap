# Proposal: codemap-critical-prd-gaps-v1

## Intent
Close critical PRD contract gaps by adding machine-consumable JSON output for `codemap impact` and `codemap query`, plus explicit schema migration control through `codemap migrate`, while preserving current JSON envelope v1 and stable CLI exit-code behavior.

## Scope
### In scope
1. Implement `codemap impact --json` using the existing JSON envelope v1 (`schema_version`, `command`, `ok`, `data`, `errors`, `meta`).
2. Implement `codemap query --json` with deterministic envelope/output shape aligned to v1 conventions.
3. Implement explicit `codemap migrate` command for applying SQLite schema migrations intentionally (rather than implicit/hidden behavior).
4. Keep stable exit-code mapping unchanged (`0` success, `1` runtime, `2` input/validation, `3` data/index).
5. Add focused tests for envelope compatibility and exit-code stability on these commands.
6. Update CLI docs/examples for the three commands.

### Explicitly out of scope
- Intent layer expansion or quality improvements.
- Multi-language parser expansion.
- New JSON schema versions or envelope redesign.
- Non-critical UX/UI enhancements.

## Affected areas
- `cli/` command registration/handlers for `impact`, `query`, `migrate`.
- `store/` migration execution entrypoint and migration-status handling.
- `indexer/query` read-path integration used by `impact` and `query` outputs.
- `docs/` command usage and JSON examples.
- Tests: integration/golden coverage for JSON and exit codes.

## Risks and mitigations
1. **Envelope drift from v1 contract**  
   Mitigation: golden JSON tests and shared response builder reuse.
2. **Exit-code regressions**  
   Mitigation: explicit integration tests for representative error paths per command.
3. **Migration command side effects**  
   Mitigation: idempotent migrate behavior, clear no-op messaging, and safe failure classification.
4. **PR size budget overrun**  
   Mitigation: chained PR slicing with strict <400 changed lines per PR target.

## Rollback plan
- Revert individual PR slices cleanly (command-by-command) without schema-envelope break.
- If `migrate` path is unstable, disable command wiring and retain prior startup behavior while keeping schema unchanged.
- Preserve JSON v1 envelope and exit-code mapping throughout rollback.

## Success criteria
1. `codemap impact --json` returns valid envelope v1 with deterministic `data`/`errors`/`meta` structure.
2. `codemap query --json` returns valid envelope v1 and parseable machine output for query results.
3. `codemap migrate` runs migrations explicitly, is idempotent on already-migrated DBs, and reports failures with stable exit codes.
4. Existing consumers observe no JSON envelope-version change and no exit-code contract change.
5. Tests cover success + failure paths for all three commands and pass in CI.

## Chained PR forecast (under 400 changed lines each)
- **PR1 (~250–350 lines):** `codemap migrate` command wiring + migration runner tests + docs.
- **PR2 (~300–380 lines):** `codemap impact --json` implementation + envelope/exit-code integration tests.
- **PR3 (~300–380 lines):** `codemap query --json` implementation + deterministic output tests + docs/examples.

## Skill resolution
- `skill_resolution: none` (no parent-injected skill paths were provided in this session).