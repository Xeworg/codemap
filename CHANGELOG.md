# Changelog

All notable changes to codemap are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **impact tier diversity**: `codemap impact` now surfaces `medium` and `low` risk
  findings via indexed `type_use`, `imports`, and `references` edges, in addition
  to call-based `high` risk findings. Edge extraction expanded at index time
  using file-local AST resolution.

- **deadcode precision v1**: `codemap deadcode` now uses explicit symbol
  edges (calls resolved at index time) plus heuristic entrypoint detection.
  Exported functions and runtime entrypoints (`main`, `init`) are classified
  `uncertain` when no explicit inbound edges are found, reducing false-positive
  rate for commonly-used patterns. Symbol coverage expanded to include methods
  and `init` functions.

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