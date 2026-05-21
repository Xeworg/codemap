# Design: codemap-v0-mvp

Go-only MVP will ship as a modular CLI-first vertical slice with explicit `indexer`, `store`, `cli`, and minimal `intent` layers. It prioritizes deterministic JSON, incremental indexing, and fail-soft indexing behavior.

## Quick path

1. Implement schema + migration runner (`store`) and snapshot metadata.
2. Implement Go index pipeline (`indexer`) with hash-based incremental updates and parse-error recording.
3. Implement `index`, `symbol --json`, `history --json` (`cli`) with stable envelope/exit codes.
4. Add strict TDD coverage (unit + integration + golden JSON) before refactors.

## Scope and non-goals

- In: Go parsing only, SQLite local index, deterministic JSON v1.0, evidence/confidence, stale signaling, link strength.
- Out: non-Go parsers, TUI/Explorer UI, cloud sync, advanced intent generation.

## Architecture layers and interfaces

| Layer | Responsibility | Public interfaces (Go) |
|---|---|---|
| `indexer` | file discovery, ignore filtering, hash diffing, Go symbol/relation extraction, commit-link derivation, parse fail-soft | `type Indexer interface { Index(ctx context.Context, req IndexRequest) (IndexResult, error) }` |
| `store` | migrations, transactional persistence, query APIs for symbol/history/meta, parse error registry | `type Store interface { Migrate(ctx context.Context) error; BeginSnapshot(...); UpsertFiles(...); ReplaceFileSymbols(...); UpsertCommitLinks(...); RecordParseError(...); GetSymbol(...); GetHistory(...); GetMeta(...) }` |
| `intent` (minimal) | normalize confidence + evidence container for responses | `type IntentService interface { BuildSymbolIntent(...); BuildHistoryIntent(...) }` |
| `cli` | command routing, argument validation, envelope formatting, exit-code policy | `RunIndex`, `RunSymbol`, `RunHistory` |

## Data flow

1. `cli index` validates repo path and opens DB.
2. `store.Migrate` ensures target schema version.
3. `indexer` loads ignore rules (defaults + `.codemapignore`) and computes candidate Go files.
4. `indexer` compares content hashes with previous snapshot and emits changed/deleted/unchanged sets.
5. Changed files are parsed via Go AST extractor:
   - success: symbols + edges + ranges persisted transactionally per file.
   - parse failure: `store.RecordParseError` and optional shallow file row update only.
6. `indexer` links commits to symbols and computes `link_strength`.
7. Snapshot finalization writes `head_ref`, `indexed_at`, parse summary.
8. `cli symbol/history --json` query store + intent normalizer and emit stable envelope with `meta.is_stale`.

## Storage and migration strategy

- Keep migration files versioned and monotonic (e.g., `0001_init.sql`, `0002_link_strength.sql`).
- `schema_migrations(version, applied_at)` is authoritative.
- MVP tables: `snapshots`, `files`, `symbols`, `edges`, `commits`, `symbol_commits`, `intent_notes`, `parse_errors`.
- Use additive migrations in MVP; avoid destructive alters.
- `codemap index` auto-runs safe migrations; future `codemap migrate` can expose explicit mode.
- Rollback: reversible `down` for safe changes; otherwise fail-fast with actionable error.

## Incremental indexing algorithm (MVP)

1. Resolve active snapshot `S_prev` and HEAD ref.
2. Walk repo with exclusions: `.git/`, `node_modules/`, `vendor/`, `dist/`, `build/` + `.codemapignore` patterns.
3. Keep `.go` files only.
4. Compute `sha256(content)` per candidate file.
5. Diff against `files(path, hash, snapshot_id=S_prev)`:
   - unchanged: skip parse/persist.
   - changed/new: parse + replace symbols/edges for file.
   - deleted: mark file removed and delete symbol/edge rows for path in new snapshot.
6. Persist under `S_new` (copy-on-write semantics at record level).
7. Finalize snapshot and stale basis (`head_ref` + indexed file hashes).

Complexity target: O(files scanned + changed files parsed).

## `link_strength` derivation

For each candidate commit affecting a symbol’s file:

- `strong`: commit diff hunks intersect symbol line range `[start_line,end_line]`.
- `medium`: commit touches same file and hunk start/end within configurable proximity window to symbol range (e.g., ±30 lines) or contains symbol identifier token.
- `weak`: commit associated only by broader file/context linkage (same file commit without proximity/token match).

Default history ordering: `strong` desc, `medium` desc, `weak` desc, then recency desc.

## Parse-error fail-soft path

- Parser error on file must not abort run.
- Write `parse_errors(file, parser='go/ast', error, snapshot_id, created_at)`.
- Continue processing remaining files.
- Preserve index usability for successful files.
- `index` output includes summary: `parse_error_count`, sample paths, and warning status while still `ok=true` if run completed.

## JSON and CLI contracts

- Envelope fixed: `schema_version`, `command`, `ok`, `data`, `errors`, `meta`.
- `meta`: `snapshot_id`, `head_ref`, `indexed_at`, `is_stale` always present.
- `data.evidence[]` always present (possibly empty array only for non-explanatory rows is disallowed by MVP spec; use at least one evidence item or hypothesis marker).
- Confidence enum constrained to `high|medium|low`.
- Exit codes: 0 success, 1 runtime, 2 validation, 3 data/index state.

## Strict TDD plan

### RED
- Unit tests for hash diff classification, ignore matching, stale-state calculation.
- Unit tests for link-strength classifier (`strong/medium/weak` boundaries).
- Unit tests for parse-error recording and continuation behavior.
- CLI JSON golden tests for `symbol --json` and `history --json` envelope determinism.

### GREEN
- Implement minimal production code to pass each failing test in vertical order:
  1) store migration + meta
  2) indexer incremental + parse fail-soft
  3) symbol/history query + envelope

### TRIANGULATE
- Add second/third fixtures for edge cases:
  - symbol rename in same file
  - commit touching file but outside range
  - broken Go file among valid files
- Ensure classifier and stale logic generalize.

### REFACTOR
- Extract reusable mappers/builders:
  - AST symbol mapper
  - evidence builder
  - envelope encoder
- Keep behavior locked by golden tests and query integration tests.

## Verification hooks

- `go test ./...` mandatory gate.
- Golden JSON snapshots for deterministic output.
- Integration fixture script:
  - init repo fixture
  - run `index`
  - mutate one file
  - rerun `index`
  - assert unchanged files not reparsed (via counters/trace logs).
- Migration smoke: create DB at empty state, run migrate twice (idempotency).

## Split-ready delivery plan (<400 review lines each PR)

1. **PR1 (~300-380 lines)**: schema+migration runner, DB bootstrap, meta structs, migration tests.
2. **PR2 (~320-390 lines)**: indexer file walk/hash diff/ignore rules + tests.
3. **PR3 (~320-400 lines)**: Go parser extraction + parse-error fail-soft + tests.
4. **PR4 (~300-380 lines)**: commit-link strength derivation + history queries + tests.
5. **PR5 (~260-340 lines)**: CLI `index/symbol/history --json`, envelope/exit codes, golden/integration tests.

## Planned file changes (implementation phase)

- `packages/coding-agent/codemap/indexer/...`
- `packages/coding-agent/codemap/store/...`
- `packages/coding-agent/codemap/cli/...`
- `packages/coding-agent/codemap/intent/...`
- `packages/coding-agent/codemap/testdata/...`
- `packages/coding-agent/codemap/migrations/...`

(Repository currently lacks these modules; this design defines target structure for apply phase.)

## Risks

- Commit diff-to-range mapping can be noisy for refactors; mitigated by exposed `link_strength` and ordering.
- Snapshot storage growth; mitigated by per-file copy-on-write and later compaction strategy.
- Deterministic JSON drift; mitigated with golden tests and strict encoder ordering.

## Rollout

- Phase 1: internal fixture repos + CI tests only.
- Phase 2: dogfood against this repo and one medium Go repo.
- Phase 3: enable downstream AI CLI consumption in required mode.
