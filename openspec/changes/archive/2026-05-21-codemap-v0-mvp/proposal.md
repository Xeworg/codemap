# Proposal: codemap-v0-mvp

## Intent
Deliver a **review-safe MVP slice** of CodeMap that proves end-to-end value for AI/CLI consumption: index a local repo, resolve symbols/relations/history, and return deterministic JSON with evidence.

## Scope (MVP-first)
### In scope
1. Core local indexing pipeline (single-machine, no network requirement).
2. SQLite schema + migrations for files, symbols, edges, commits, symbol_commits, snapshots, intent_notes, parse_errors.
3. Incremental reindex by file hash (changed files only).
4. CLI commands (human + `--json`):
   - `codemap index`
   - `codemap symbol <id|name> --json`
   - `codemap history <symbol> --json`
   - `codemap impact <symbol> --json`
   - `codemap query "<text>" --json`
5. Stable JSON envelope (`schema_version: "1.0"`) with `meta` status (`snapshot_id`, `head_ref`, `indexed_at`, `is_stale`).
6. Evidence-first responses (`evidence[]` required) and confidence labels (`high|medium|low`).
7. Parse-failure tolerance (record parse errors, continue indexing).
8. Default safe-path exclusions and `.codemapignore` support.

### Non-goals (explicit)
- Cloud sync or remote services.
- Multi-user collaboration or real-time shared state.
- VSCode extension / graphical UI.
- Full multi-language parity in first delivery.
- Mandatory secondary AI agent workflows.

## First delivery slice (PR1 target)
**Single-language vertical slice (Go only)** that validates architecture and contracts:
- Index Go files into SQLite.
- Extract symbols + basic call/import relations.
- Map symbol-to-commit links with `link_strength` (`strong|medium|weak`).
- Support `index`, `symbol --json`, `history --json`.
- Emit deterministic JSON v1.0 + required `evidence[]`.
- Include stale-index metadata and stable exit codes.

This slice is intentionally narrow to stay under review budget and reduce parser risk before adding other languages and commands.

## Affected areas
- `indexer/`: language parser adapters, incremental hashing, relation extraction, parse-error fallback.
- `store/`: SQLite schema, migrations, query layer, FTS bootstrap.
- `cli/`: command handlers, JSON envelope, error/exit code policy.
- `intent/`: initial intent note ingestion and confidence normalization.
- `explorer/`: deferred for MVP CLI-first; no TUI requirement in first slice.
- `docs/`: JSON contract examples and operator usage updates.

## Risks and mitigations
1. **Parser inconsistency / false relations**  
   Mitigation: start with Go-only parser in PR1, add fixtures and strict tests before additional languages.
2. **Weak commit↔symbol precision**  
   Mitigation: expose `link_strength`; default outputs prioritize `strong|medium`.
3. **JSON contract drift**  
   Mitigation: lock v1.0 envelope + golden JSON integration tests for critical commands.
4. **Index fragility on parse errors**  
   Mitigation: fail-soft behavior with `parse_errors` table + end-of-run summary.
5. **Review overload**  
   Mitigation: chained PR plan, MVP slicing, and per-PR changed-lines discipline.

## Rollback plan
- Keep schema changes behind versioned migrations (`up/down` where safe).
- If a migration or parser path regresses, revert the feature branch PR and preserve previous schema version.
- Keep CLI JSON schema backward-compatible within v1.0; breaking changes deferred to explicit major version planning.

## Success criteria
1. `codemap index` builds a local SQLite index for a real Go repo without complex setup.
2. Incremental index updates only modified files and completes faster than full reindex on small diffs.
3. `symbol/history/impact/query --json` return valid v1.0 envelope with deterministic structure.
4. Explanatory outputs always include `evidence[]`; unsupported evidence is marked clearly as hypothesis/low confidence.
5. Parse errors do not abort full indexing; failures are reported and query surface remains usable.
6. External AI CLI can consume `codemap symbol --json` and improve response grounding.

## Slice forecast (auto-forecast)
- **PR1**: Go vertical slice + schema + `index/symbol/history` JSON.
- **PR2**: `impact/query`, FTS query path, stronger evidence propagation.
- **PR3**: intent bootstrap, ignore/strict mode hardening, performance + stale-state polish.
