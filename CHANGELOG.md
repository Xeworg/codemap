# Changelog

All notable changes to codemap are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

_(no changes yet)_

## [1.1.0] — 2026-05-22

### Added

- **smoke script**: `scripts/smoke/smoke.sh` for end-to-end validation of
  `index`, `symbol`, `history`, `impact`, `deadcode`, `query`, `migrate`,
  `doctor`, and `install` commands.
- **KPI sampling documentation**: `docs/kpi-sampling-m5.md` captures impact
  relevance, explain accuracy, and deadcode false-positive sampling for M5
  release readiness.
- **Pi extension command-surface sync**: extension now registers all CLI
  commands (`impact`, `deadcode`, `query`, `install`, `doctor`).

### Changed

- **impact tier diversity**: `codemap impact` now surfaces `medium` and `low`
  risk findings via indexed `type_use`, `imports`, `references`, and `casts`
  edges, in addition to call-based `high` risk findings. Edge extraction
  expanded at index time using file-local AST resolution.
- **deadcode precision v1**: `codemap deadcode` now uses explicit symbol
  edges (calls resolved at index time) plus heuristic entrypoint detection.
  Exported functions and runtime entrypoints (`main`, `init`) are classified
  `uncertain` when no explicit inbound edges are found, reducing false-positive
  rate for commonly-used patterns. Symbol coverage expanded to include methods
  and `init` functions.

### Known Limitations

- **Go-only.** No support for TypeScript, Python, Java, or other languages.
- **File-local AST resolution.** Cross-module impact analysis may be incomplete.
- **Deadcode heuristics.** Exported symbols consumed by external repos not in
  the index may be over-flagged as `uncertain`.
- **No auto-fix.** All commands are advisory only and never modify source files.
- **Pi extension gap.** Before this release the Pi extension only registered
  `codemap_index`, `codemap_symbol`, and `codemap_history`. Commands
  `impact`, `deadcode`, `query`, `install`, and `doctor` are now added.
- **Query.** Prefix/exact match only; no fuzzy or semantic search.
- **History link strength.** Heuristic based on commit message mention and
  file co-change, not semantic diff analysis.

### Next Steps

- Interface/reflection edge support for deadcode v1.1.
- Cross-file SSA for impact precision improvement.
- TUI/explorer mode enhancements.

## [1.0.0] — 2025-05-21

### Added

- `codemap index`: scan and index a Go repository into a local SQLite database.
- `codemap symbol`: query a symbol by name.
- `codemap history`: query commit history for a symbol.
- `codemap migrate`: run pending schema migrations.
- `codemap impact`: show symbols impacted by a given symbol.
- `codemap query`: look up symbols by name or prefix.
- `codemap install`: install codemap skill and tool into Pi runtime.
- `codemap doctor`: diagnose codemap environment and integration.