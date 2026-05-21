# codemap-mvp Specification

## Purpose

Define the MVP Go-only vertical slice behavior for local indexing and deterministic CLI JSON responses for AI/automation consumers.

## Requirements

### Requirement: Go-only indexing scope

The system MUST implement the first delivery slice for Go repositories only, and SHALL limit MVP command support to `codemap index`, `codemap symbol <id|name> --json`, and `codemap history <symbol> --json`.

#### Scenario: Non-Go files are outside MVP parsing scope

- GIVEN a repository containing Go and non-Go source files
- WHEN `codemap index` runs in MVP mode
- THEN Go files are parsed for symbol and relation extraction
- AND non-Go language parsing is not required for MVP correctness.

### Requirement: Incremental local indexing with persistence

The system MUST build and maintain a local SQLite index with versioned migrations for files, symbols, edges, commits, symbol_commits, snapshots, intent_notes, and parse_errors.

The system MUST support incremental reindex by file hash, updating only changed files and their affected records.

#### Scenario: Small diff reindex updates only changed files

- GIVEN an existing snapshot and stored file hashes
- WHEN only a subset of Go files changes and `codemap index` runs again
- THEN unchanged files are not reparsed
- AND changed files are reparsed and persisted in a new or updated snapshot context.

### Requirement: Deterministic JSON envelope v1.0

All MVP commands executed with `--json` MUST return a deterministic envelope with `schema_version` equal to `"1.0"`, stable top-level field structure, and machine-parseable `ok`, `data`, `errors`, and `meta` fields.

The `meta` object MUST include `snapshot_id`, `head_ref`, `indexed_at`, and `is_stale`.

#### Scenario: Symbol command JSON shape is stable

- GIVEN a successful `codemap symbol <id|name> --json` request
- WHEN the response is emitted
- THEN `schema_version` is `"1.0"`
- AND the top-level envelope shape is deterministic across equivalent runs
- AND `meta.snapshot_id`, `meta.head_ref`, `meta.indexed_at`, and `meta.is_stale` are present.

### Requirement: Evidence-first explanatory responses

For explanatory MVP JSON responses, `data.evidence[]` MUST be present and non-null.

The system MUST include confidence labels using `high`, `medium`, or `low`.

If direct evidence is unavailable, the response MUST still include `evidence[]` and SHALL mark the claim as low-confidence or hypothesis-compatible.

#### Scenario: History response includes evidence and confidence

- GIVEN `codemap history <symbol> --json`
- WHEN commit-symbol links are returned
- THEN the response includes `data.evidence[]`
- AND confidence is expressed using only `high`, `medium`, or `low`.

### Requirement: Stale index signaling

The system MUST compute and expose stale-index state through `meta.is_stale` for MVP JSON commands.

When the index is stale relative to repository head or indexed content state, the command MUST signal staleness in JSON metadata without changing the envelope contract.

#### Scenario: Stale snapshot is reported

- GIVEN the repository HEAD has advanced after the last completed index
- WHEN `codemap symbol --json` is executed
- THEN the command returns a valid v1.0 envelope
- AND `meta.is_stale` is `true`.

### Requirement: Parse-error fail-soft behavior

Indexing MUST fail soft on per-file parse failures: it SHALL record failures in `parse_errors` and continue indexing other files.

At the end of `codemap index`, the system MUST report parse-failure summary information while preserving overall index usability for successfully parsed files.

#### Scenario: One file fails parse during index

- GIVEN one Go file cannot be parsed and others are valid
- WHEN `codemap index` runs
- THEN indexing continues for parseable files
- AND the failed file incident is recorded in `parse_errors`
- AND end-of-run output includes parse-failure summary information.

### Requirement: Commit-to-symbol link strength semantics

History outputs MUST classify commit-symbol associations with `link_strength` values limited to `strong`, `medium`, or `weak`.

MVP outputs SHOULD prioritize `strong` and `medium` links in default result ordering.

#### Scenario: History includes link strength labels

- GIVEN a symbol with multiple associated commits
- WHEN `codemap history <symbol> --json` is returned
- THEN each returned association includes `link_strength`
- AND `link_strength` values are only `strong`, `medium`, or `weak`.

### Requirement: Safe-path exclusions and ignore support

Indexing MUST apply default safe-path exclusions for common non-source or generated directories and MUST support `.codemapignore` custom exclusions.

#### Scenario: Ignored paths are excluded from indexing

- GIVEN `.codemapignore` excludes a path
- WHEN `codemap index` runs
- THEN files under excluded paths are not indexed.

### Requirement: Stable CLI exit code policy

MVP commands MUST use stable exit codes:
- `0` for success
- `1` for execution/runtime failure
- `2` for input/validation failure
- `3` for index/data-state failure.

#### Scenario: Validation error returns exit code 2

- GIVEN an invalid symbol argument for `codemap symbol`
- WHEN command validation fails
- THEN the process exits with code `2`.
