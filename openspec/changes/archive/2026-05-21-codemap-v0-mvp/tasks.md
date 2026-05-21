# Tasks — codemap-v0-mvp

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1,500–1,900 (full MVP slice with tests) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR1 (store+migrations) → PR2 (incremental index core) → PR3 (Go parse fail-soft) → PR4 (history/link strength) → PR5 (CLI JSON + integration) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

## Execution Rules
- Strict TDD required: RED → GREEN → TRIANGULATE → REFACTOR per PR slice.
- Keep each PR under ~400 changed lines (additions + deletions).
- Include verification evidence in each PR: failing tests first, then passing tests, then refactor-safe rerun.

## PR1 — Store, migrations, and snapshot metadata foundation
Dependencies: none

- [x] 1. **RED: migration/idempotency tests**
  - Add tests for migration bootstrap and repeated migrate calls in:
    - `packages/coding-agent/codemap/store/migrations_test.go`
  - Cover expected tables: `schema_migrations`, `snapshots`, `files`, `symbols`, `edges`, `commits`, `symbol_commits`, `intent_notes`, `parse_errors`.
  - Evidence: failing `go test ./...` showing missing migration implementation.

- [x] 2. **GREEN: implement migration runner + initial SQL**
  - Create/implement:
    - `packages/coding-agent/codemap/migrations/0001_init.sql`
    - `packages/coding-agent/codemap/store/migrate.go`
  - Ensure idempotent apply and persisted migration version rows.
  - Evidence: tests from task 1 pass.

- [x] 3. **TRIANGULATE + REFACTOR: meta retrieval coverage and cleanup**
  - Add tests for meta fields retrieval (`snapshot_id`, `head_ref`, `indexed_at`) in:
    - `packages/coding-agent/codemap/store/meta_test.go`
  - Implement/refactor in:
    - `packages/coding-agent/codemap/store/meta.go`
  - Evidence: `go test ./...` green; migration rerun remains idempotent.

## PR2 — Incremental indexing core (walk, ignore, hash diff)
Dependencies: PR1

- [x] 4. **RED: ignore + hash diff tests**
  - Add unit tests for default exclusions and `.codemapignore` matching:
    - `packages/coding-agent/codemap/indexer/ignore_test.go`
  - Add unit tests for new/changed/unchanged/deleted classification:
    - `packages/coding-agent/codemap/indexer/diff_test.go`
  - Evidence: failing tests before implementation.

- [x] 5. **GREEN: implement file discovery + hash diff**
  - Implement:
    - `packages/coding-agent/codemap/indexer/walk.go`
    - `packages/coding-agent/codemap/indexer/hash.go`
    - `packages/coding-agent/codemap/indexer/diff.go`
  - Restrict parse candidates to `.go` files only.
  - Evidence: tests from task 4 pass.

- [x] 6. **TRIANGULATE + REFACTOR: incremental integration check**
  - Add fixture/integration test for reindex only changed files:
    - `packages/coding-agent/codemap/indexer/incremental_integration_test.go`
    - `packages/coding-agent/codemap/testdata/repos/incremental-go/` (fixture)
  - Refactor classifier/helpers without behavior drift.
  - Evidence: integration + unit suite green.

## PR3 — Go parser extraction + parse-error fail-soft
Dependencies: PR2

- [x] 7. **RED: parser success/failure behavior tests**
  - Add tests for symbol/range extraction and parse-failure continuation:
    - `packages/coding-agent/codemap/indexer/go_parser_test.go`
    - `packages/coding-agent/codemap/indexer/parse_failsoft_test.go`
  - Validate parse errors recorded in `parse_errors`.
  - Evidence: failing tests showing missing fail-soft path.

- [x] 8. **GREEN: implement AST extraction + fail-soft persistence**
  - Implement:
    - `packages/coding-agent/codemap/indexer/go_parser.go`
    - `packages/coding-agent/codemap/indexer/index.go`
    - store hooks in `packages/coding-agent/codemap/store/*.go` for `RecordParseError`, `ReplaceFileSymbols`, edge persistence.
  - Ensure index run continues after per-file parse errors.
  - Evidence: tests from task 7 pass.

- [x] 9. **TRIANGULATE + REFACTOR: mixed-validity fixture coverage**
  - Add fixture with valid + broken Go files:
    - `packages/coding-agent/codemap/testdata/repos/parse-mixed/`
  - Refactor AST mapping utilities:
    - `packages/coding-agent/codemap/indexer/symbol_mapper.go`
  - Evidence: parse summary assertions and full suite green.

## PR4 — History pipeline + link_strength semantics
Dependencies: PR3

- [x] 10. **RED: link_strength classifier and ordering tests**
  - Add unit tests for `strong|medium|weak` boundaries and default ordering:
    - `packages/coding-agent/codemap/indexer/link_strength_test.go`
    - `packages/coding-agent/codemap/store/history_query_test.go`
  - Evidence: failing tests for unimplemented classifier/query behavior.

- [x] 11. **GREEN: implement commit-symbol linking + history query**
  - Implement:
    - `packages/coding-agent/codemap/indexer/history_linker.go`
    - `packages/coding-agent/codemap/store/history.go`
    - migration if needed for link field (e.g., `packages/coding-agent/codemap/migrations/0002_link_strength.sql`)
  - Ensure output stores/returns allowed enum values only.
  - Evidence: tests from task 10 pass.

- [x] 12. **TRIANGULATE + REFACTOR: edge-case history fixtures**
  - Add tests for commit touching file but outside symbol range and rename-like edits in same file:
    - `packages/coding-agent/codemap/indexer/history_linker_edge_test.go`
  - Refactor proximity/token matcher into reusable helper.
  - Evidence: test suite green with expanded cases.

## PR5 — CLI commands, deterministic JSON envelope, exit codes
Dependencies: PR4

- [x] 13. **RED: CLI golden + exit-code tests**
  - Add command tests for:
    - `codemap index`
    - `codemap symbol <id|name> --json`
    - `codemap history <symbol> --json`
  - Files:
    - `packages/coding-agent/codemap/cli/index_cmd_test.go`
    - `packages/coding-agent/codemap/cli/symbol_cmd_test.go`
    - `packages/coding-agent/codemap/cli/history_cmd_test.go`
    - golden fixtures in `packages/coding-agent/codemap/testdata/golden/*.json`
  - Assert deterministic envelope fields, `schema_version="1.0"`, `meta.is_stale`, `data.evidence[]`, confidence enum, exit codes 0/1/2/3.
  - Evidence: failing golden/behavior tests first.

- [x] 14. **GREEN: implement CLI handlers + envelope encoder**
  - Implement:
    - `packages/coding-agent/codemap/cli/root.go`
    - `packages/coding-agent/codemap/cli/index.go`
    - `packages/coding-agent/codemap/cli/symbol.go`
    - `packages/coding-agent/codemap/cli/history.go`
    - `packages/coding-agent/codemap/cli/envelope.go`
    - minimal intent normalizer in `packages/coding-agent/codemap/intent/service.go`
  - Evidence: golden tests pass unchanged after regeneration lock.

- [x] 15. **TRIANGULATE + REFACTOR: integration, docs, and contract lock**
  - Add end-to-end fixture test:
    - `packages/coding-agent/codemap/cli/integration_test.go`
  - Update operator contract docs:
    - `docs/codemap-cli-json-contract.md`
  - Refactor serializer/validation paths while preserving golden outputs.
  - Evidence: `go test ./...` green + documented sample outputs match tests.

## Final verification gate (post-PR5)
- [x] 16. Run full verification and capture evidence for apply phase:
  - `go test ./...`
  - migration idempotency rerun
  - incremental reindex fixture rerun
  - CLI golden snapshot consistency check
  - Evidence artifact target: `openspec/changes/codemap-v0-mvp/verify-report.md` (next phase).
