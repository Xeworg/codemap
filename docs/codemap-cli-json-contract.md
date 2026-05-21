# CodeMap CLI JSON Contract v1.0

## Overview

All CLI commands output deterministic JSON when called in normal mode. This document defines the envelope structure, field semantics, exit codes, and examples.

---

## Envelope Structure

Every response is a JSON object with these top-level fields:

```json
{
  "schema_version": "1.0",
  "command": "<command-name>",
  "ok": true,
  "data": { ... },
  "errors": null,
  "meta": { ... }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `schema_version` | string | Yes | Always `"1.0"` for MVP. |
| `command` | string | Yes | One of: `index`, `symbol`, `history`. |
| `ok` | bool | Yes | `true` if the command succeeded; `false` if errors occurred. |
| `data` | object | Yes | Command-specific payload (see below). |
| `errors` | `null` or `string[]` | Yes | `null` on success; array of error strings on failure. |
| `meta` | object | Yes | Snapshot and freshness metadata (see Meta section). |

---

## Meta Fields

Every response includes:

```json
"meta": {
  "snapshot_id": 1,
  "head_ref": "refs/heads/main",
  "indexed_at": "2026-05-21T10:00:00Z",
  "is_stale": false
}
```

| Field | Type | Description |
|---|---|---|
| `snapshot_id` | int | ID of the snapshot used for this response. |
| `head_ref` | string | Git ref at time of last index. Empty string if not a git repo. |
| `indexed_at` | string | RFC3339 timestamp of last index run. |
| `is_stale` | bool | `true` if last index is older than 24 hours or HEAD has moved. |

---

## Command Payloads

### `codemap index`

```json
{
  "schema_version": "1.0",
  "command": "index",
  "ok": true,
  "data": {
    "files_scanned": 10,
    "files_parsed": 8,
    "symbols_found": 42,
    "parse_errors": 1,
    "evidence": null
  },
  "errors": null,
  "meta": { ... }
}
```

- `files_scanned`: total `.go` files discovered (excluding ignored paths).
- `files_parsed`: files successfully parsed (fail-soft: parse errors don't abort).
- `symbols_found`: total symbols extracted across all parsed files.
- `parse_errors`: count of files that failed to parse; these are recorded in `parse_errors` table.
- `evidence`: `null` for index summary (not applicable). Reserved for future per-file evidence.

### `codemap symbol <name>`

```json
{
  "schema_version": "1.0",
  "command": "symbol",
  "ok": true,
  "data": {
    "name": "MyFunction",
    "kind": "func",
    "signature": "func(a, b)",
    "start_line": 10,
    "end_line": 15,
    "file": "pkg/foo.go",
    "confidence": "high",
    "evidence": [
      { "type": "direct", "description": "symbol extracted from source" },
      { "type": "file_location", "description": "found in pkg/foo.go" }
    ]
  },
  "errors": null,
  "meta": { ... }
}
```

- `confidence`: one of `high`, `medium`, `low`. Default: `high` for func/type/interface, `medium` for var/const, `low` otherwise.
- `evidence`: always non-empty. Minimum one item of type `direct`.

### `codemap history <name>`

```json
{
  "schema_version": "1.0",
  "command": "history",
  "ok": true,
  "data": {
    "symbol_name": "MyFunction",
    "confidence": "low",
    "evidence": [
      {
        "type": "no_history",
        "description": "no commit history found for this symbol"
      }
    ]
  },
  "errors": null,
  "meta": { ... }
}
```

- `confidence`: `medium` if history entries exist; `low` if none.
- `evidence`: commit links (type: `commit_link`) when history exists; `no_history` item otherwise.

---

## Confidence Enum

Only these values are valid:

| Value | Meaning |
|---|---|
| `high` | Strong evidence: direct symbol extraction or strong link strength |
| `medium` | Moderate evidence: history entries present |
| `low` | Weak or no evidence: hypothesis or no data |

---

## Exit Codes

| Code | Meaning | When |
|---|---|---|
| `0` | Success | Command completed normally |
| `1` | Runtime error | DB open failure, file read error, unexpected panic |
| `2` | Validation error | Missing `--db`, empty symbol argument, invalid flags |
| `3` | Data/index state error | No index found, symbol not found, stale/corrupt DB |

---

## Stale Detection

A snapshot is considered **stale** when:

1. `indexed_at` is more than 24 hours ago.
2. `indexed_at` cannot be parsed as a valid RFC3339 timestamp.

Staleness does not block queries; it is informational only via `meta.is_stale`.

---

## Parse Error Fail-Soft

If one or more `.go` files fail to parse during `index`:

- The run completes with `ok: true`.
- `parse_errors` count is non-zero.
- Successful files are still indexed and available.
- Parse errors are recorded in the `parse_errors` table with `parser='go/ast'`.

---

## Incremental Indexing

On repeated `index` calls:

- Unchanged files (hash unchanged) are skipped (not reparsed).
- Changed/new files are reparsed.
- Deleted files (present in previous snapshot, absent now) are removed from the index.

---

## Non-Goals

- This contract covers Go-only indexing.
- TUI/Explorer UI is out of scope.
- Multi-language parsing is out of scope for MVP.

---

## Running

Build the binary and run commands:

```bash
go build -o codemap ./cmd/codemap

# Index a repository
codemap index --db myrepo.db

# Query a symbol
codemap symbol --db myrepo.db MyFunction

# Query symbol history
codemap history --db myrepo.db MyFunction
```

Global flags:
- `-repo path` — repository root (default: current directory)

All commands return JSON; error paths also return JSON envelopes with `ok:false` and `errors[]`.
