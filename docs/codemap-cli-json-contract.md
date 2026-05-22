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
| `command` | string | Yes | One of: `index`, `symbol`, `history`, `migrate`, `impact`, `query`, `deadcode`, `install`, `doctor`. |
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

**Success (found)**

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

**Not found (explain_not_found payload)**

When the symbol does not exist in the index, the CLI returns `ok: false` with an `explain_not_found` block in `data`:

```json
{
  "schema_version": "1.0",
  "command": "symbol",
  "ok": false,
  "data": {
    "explain_not_found": {
      "cause": "stale_index",
      "recommended_actions": [
        "Run 'codemap index' to update the snapshot",
        "Verify the repository path is correct"
      ]
    }
  },
  "errors": ["symbol \"MyFunction\" not found"],
  "meta": { ... }
}
```

- `confidence`: one of `high`, `medium`, `low`. Default: `high` for func/type/interface, `medium` for var/const, `low` otherwise.
- `evidence`: always non-empty on success. Minimum one item of type `direct`.
- **`explain_not_found.cause`** enum: `stale_index`, `parse_error`, `name_mismatch`, `missing_history_links`
- **`explain_not_found.recommended_actions`**: array of strings with fix guidance, determined by cause

### `codemap history <name>`

**Success (history found)**

```json
{
  "schema_version": "1.0",
  "command": "history",
  "ok": true,
  "data": {
    "symbol_name": "MyFunction",
    "confidence": "medium",
    "evidence": [
      {
        "type": "commit_link",
        "description": "modify on 2026-05-10 (a1b2c3d4)",
        "source": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
      }
    ]
  },
  "errors": null,
  "meta": { ... }
}
```

**No history found (symbol exists but no git history links)**

When the symbol is in the index but has no associated commit history, the CLI returns `ok: true` with `confidence: low` and an `explain_not_found` block that includes `cause` and `recommended_actions`:

```json
{
  "schema_version": "1.0",
  "command": "history",
  "ok": true,
  "data": {
    "symbol_name": "MyFunction",
    "confidence": "low",
    "evidence": [
      { "type": "no_history", "description": "no commit history found for this symbol" }
    ],
    "explain_not_found": {
      "cause": "missing_history_links",
      "recommended_actions": [
        "Run 'codemap index' to rebuild history links",
        "Ensure the repository has commit history",
        "Symbol exists but has no git history associations"
      ]
    }
  },
  "errors": null,
  "meta": { ... }
}
```

**Symbol not found (explain_not_found payload)**

When the symbol does not exist in the index, the CLI returns `ok: false` with an `explain_not_found` block:

```json
{
  "schema_version": "1.0",
  "command": "history",
  "ok": false,
  "data": {
    "explain_not_found": {
      "cause": "name_mismatch",
      "recommended_actions": [
        "Verify the symbol name is spelled correctly",
        "Check for renamed or moved symbols",
        "Run 'codemap index' if the code was recently changed"
      ]
    }
  },
  "errors": ["symbol \"MyFunction\" not found"],
  "meta": { ... }
}
```

- `confidence`: `medium` if history entries exist; `low` if none.
- `evidence`: commit links (type: `commit_link`) when history exists; `no_history` item otherwise.
- **`explain_not_found.cause`** enum for history: `stale_index`, `parse_error`, `name_mismatch`, `missing_history_links`
- **`explain_not_found.recommended_actions`**: array of strings with fix guidance, determined by cause

---

## Confidence Enum

Only these values are valid:

| Value | Meaning |
|---|---|
| `high` | Strong evidence: direct symbol extraction, calls/type_use edge, or confirmed zero edges for func/type |
| `medium` | Moderate evidence: history entries present, `imports`/`casts` edge, or var/const classification |
| `low` | Weak or no evidence: hypothesis, weaker edge types, or uncertain classification |

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

## `codemap impact <symbol>`

Returns all symbols that depend on or reference the target symbol, sorted by risk tier then confidence.

```json
{
  "schema_version": "1.0",
  "command": "impact",
  "ok": true,
  "data": {
    "target_symbol": "MyFunction",
    "findings": [
      {
        "symbol_name": "CallerA",
        "file": "pkg/caller.go",
        "kind": "func",
        "start_line": 20,
        "end_line": 25,
        "risk_tier": "high",
        "confidence": "high",
        "evidence": [
          {
            "type": "symbol_link",
            "description": "linked via calls",
            "source": "pkg/caller.go"
          }
        ]
      },
      {
        "symbol_name": "TypeUser",
        "file": "pkg/user.go",
        "kind": "type",
        "start_line": 5,
        "end_line": 10,
        "risk_tier": "medium",
        "confidence": "high",
        "evidence": [
          {
            "type": "symbol_link",
            "description": "linked via type_use",
            "source": "pkg/user.go"
          }
        ]
      }
    ],
    "evidence": []
  },
  "errors": null,
  "meta": { ... }
}
```

### Finding fields

| Field | Type | Description |
|---|---|---|
| `symbol_name` | string | Name of the affected symbol |
| `file` | string | File path of the affected symbol |
| `kind` | string | Symbol kind (func, type, var, etc.) |
| `start_line` | int | Start line of the symbol definition |
| `end_line` | int | End line of the symbol definition |
| `risk_tier` | string | `high`, `medium`, or `low` |
| `confidence` | string | `high`, `medium`, or `low` |
| `evidence` | array | Always non-empty; at least one `symbol_link` item |

### `risk_tier` derivation

| Edge type | Risk tier |
|---|---|---|
| `calls` | `high` |
| `type_use`, `references` | `medium` |
| `imports`, `casts`, `subtype`, `exports` | `medium` |
| (all others) | `low` |

### `confidence` derivation

| Condition | Confidence |
|---|---|---|
| `calls`/`type_use` edge + `func`/`type`/`interface` kind | `high` |
| `calls`/`type_use` edge + `var`/`const` kind | `medium` |
| `imports`/`casts` edge | `medium` |
| (all others) | `low` |

### Default cap behavior

`findings` is capped at **50 findings** by default. Results are sorted by `risk_tier` (high first), then `confidence` (high first), then `symbol_name` (alphabetical), then `file` before the cap is applied. The cap truncates lower-priority findings; there is no signal in the response when truncation occurs.


Exit codes: `0` success, `1` runtime error, `2` validation error, `3` no index / symbol not found.

---

## `codemap deadcode`

Reports symbols with zero inbound references, classified as unused/likely-unused.

```json
{
  "schema_version": "1.0",
  "command": "deadcode",
  "ok": true,
  "data": {
    "findings": [
      {
        "symbol_name": "UnusedFunction",
        "file": "pkg/foo.go",
        "kind": "func",
        "start_line": 30,
        "end_line": 35,
        "classification": "unused",
        "suggestion": "remove",
        "confidence": "high",
        "evidence": [
          {
            "type": "no_inbound_edges",
            "description": "symbol has no inbound references in the code graph"
          }
        ]
      },
      {
        "symbol_name": "MaybeUnused",
        "file": "pkg/bar.go",
        "kind": "type",
        "start_line": 5,
        "end_line": 9,
        "classification": "likely-unused",
        "suggestion": "remove",
        "confidence": "medium",
        "evidence": [
          {
            "type": "no_inbound_edges",
            "description": "symbol has no inbound references in the code graph"
          }
        ]
      }
    ]
  },
  "errors": null,
  "meta": { ... }
}
```

### Finding fields

| Field | Type | Description |
|---|---|---|
| `symbol_name` | string | Name of the dead symbol |
| `file` | string | File path |
| `kind` | string | Symbol kind |
| `start_line` | int | Start line |
| `end_line` | int | End line |
| `classification` | string | `unused`, `likely-unused`, or `uncertain` |
| `suggestion` | string | `remove`, `deprecate`, or `justify` |
| `confidence` | string | `high`, `medium`, or `low` |
| `evidence` | array | Always non-empty; `no_inbound_edges` item |

### `classification` values

| Value | Meaning |
|---|---|---|
| `unused` | Zero inbound edges confirmed; strong signal |
| `likely-unused` | Indirect/weak signal; symbol kind suggests caution |
| `uncertain` | Some edges exist but may be dead-code-adjacent |

### `suggestion` values

| Value | Meaning |
|---|---|---|
| `remove` | Safe to delete; strong unused signal |
| `deprecate` | Mark for future removal or migration path |
| `justify` | Retain only with documented reason; uncertain signal |

### Classification and suggestion mapping

| Symbol kind | Inbound edges | Classification | Suggestion | Confidence |
|---|---|---|---|---|
| `func`, `type` | 0 | `unused` | `remove` | `high` |
| `var`, `const` | 0 | `unused` | `remove` | `medium` |
| (any other kind) | 0 | `unused` | `remove` | `low` |
| (any kind) | > 0 | `uncertain` | `justify` | `low` |

### Default cap behavior

`findings` is capped at **100 findings** by default. Results are sorted by `classification` rank (`unused` first), then `confidence` (high first), then `symbol_name` (alphabetical), then `file` before the cap is applied. The cap truncates lower-priority findings; there is no signal in the response when truncation occurs.

### Exclusions

Files matching any of the following patterns are excluded from deadcode analysis:
`_generated`, `.gen.go`, `_test.go`, `_mock.go`, `_fake.go`, `testdata/`, `vendor/`, `third_party/`, `_pb.go`, `.pb.go` (protobuf), `_grpc.go` (grpc).

### Command flags

| Flag | Default | Description |
|---|---|---|
| `--db` | auto | Path to SQLite database |
| `--limit` | 100 | Maximum findings to return |

Exit codes: `0` success, `1` runtime error, `2` validation error, `3` no index found.

---

## `explain_not_found` cause enum

When `symbol` or `history` cannot resolve a symbol, the response `data` contains an `explain_not_found` block:


```json
"data": {
  "explain_not_found": {
    "cause": "<cause-value>",
    "recommended_actions": ["...", "..."]
  }
}
```

| Cause | Trigger | Recommended actions |
|---|---|---|
| `stale_index` | Snapshot older than 24 h or unreadable timestamp | Run `codemap index`; verify repo path |
| `parse_error` | One or more files failed to parse in the last index | Re-run `codemap index`; check syntax; review parse_errors table |
| `name_mismatch` | Fresh snapshot, no parse errors, symbol absent | Verify spelling; check for renames/moves; re-index if recently changed |
| `missing_history_links` | Symbol exists but has no git history associations | Rebuild index; ensure repo has commits |

---

## Non-Goals

### `codemap install`

```json
{
  "schema_version": "1.0",
  "command": "install",
  "ok": true,
  "data": {
    "status": "applied",
    "checks": [
      {
        "name": "repo_root",
        "passed": true,
        "info": "/path/to/repo"
      },
      {
        "name": "template_skill",
        "passed": true,
        "info": "/path/to/repo/integrations/pi/skills/codemap-usage/SKILL.md"
      },
      {
        "name": "template_extension",
        "passed": true,
        "info": "/path/to/repo/integrations/pi/extensions/codemap-extension.ts"
      },
      {
        "name": "pi_runtime",
        "passed": true,
        "exists": true,
        "info": "~/.pi/agent"
      }
    ],
    "actions": [
      {
        "kind": "copy",
        "source": "/path/to/repo/integrations/pi/skills/codemap-usage/SKILL.md",
        "target": "~/.pi/agent/skills/codemap-usage/SKILL.md",
        "changed": false
      }
    ],
    "timestamp": "2026-05-21T10:00:00Z"
  },
  "errors": null,
  "meta": {}
}
```

- `status`: one of `applied` (changes applied), `up-to-date` (nothing changed), `dry-run` (planned, not applied), `error` (pre-flight or apply failed).
- `checks`: pre-flight validation results. `exists` is present for runtime checks; `skipped` surfaces reasons when a check was not run.
- `actions`: copy operations planned or executed. `changed: true` means the file was (or would be) written.

Exit codes for `install`: `0` for applied/up-to-date/dry-run; `1` for error; `2` for flag/validation error.

### `codemap doctor`

```json
{
  "schema_version": "1.0",
  "command": "doctor",
  "ok": true,
  "data": {
    "status": "pass",
    "checks": [
      {
        "check": "repo_root",
        "level": "pass",
        "message": "repo root: /path/to/repo"
      },
      {
        "check": "integrations_dir",
        "level": "pass",
        "message": "integrations dir: /path/to/repo/integrations/pi"
      },
      {
        "check": "skill",
        "level": "warn",
        "message": "skill template: /path/to/repo/integrations/pi/skills/codemap-usage/SKILL.md — NOT FOUND"
      },
      {
        "check": "pi_runtime",
        "level": "warn",
        "message": "Pi runtime not found at ~/.pi/agent (will be created on install)"
      },
      {
        "check": "installed_skill",
        "level": "warn",
        "message": "skill not installed at ~/.pi/agent/skills/codemap-usage/SKILL.md (run 'codemap install' to install)"
      },
      {
        "check": "installed_extension",
        "level": "warn",
        "message": "extension not installed at ~/.pi/agent/extensions/codemap-extension.ts (run 'codemap install' to install)"
      },
      {
        "check": "default_db",
        "level": "warn",
        "message": "DB path: ~/.cache/codemap/<hash>.db (not created yet; run 'codemap index' to create)"
      }
    ],
    "db_path": "~/.cache/codemap/<hash>.db",
    "db_exists": false
  },
  "errors": null,
  "meta": {}
}
```

- `status`: `pass` (all checks pass), `warn` (one or more warnings, no failures), `fail` (one or more failures).
- `checks[].level`: `pass`, `warn`, or `fail`.
- `db_path`: the effective default DB path (`CODEMAP_DB_PATH` env → `~/.cache/codemap/<hash>.db`).
- `db_exists`: `true` if the DB file already exists; `false` otherwise.

Exit codes for `doctor`: `0` for pass/warn; `1` for fail or runtime error.

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

# Install codemap skill and tool into Pi runtime
codemap install
codemap install --dry-run

# Diagnose codemap environment
codemap doctor
codemap doctor --json
```

Global flags:
- `-repo path` — repository root (default: current directory)

All commands return JSON; error paths also return JSON envelopes with `ok:false` and `errors[]`.
