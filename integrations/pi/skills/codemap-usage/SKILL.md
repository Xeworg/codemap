---
title: codemap-usage
trigger: symbol lookup, code navigation, function search, refactor, bug context, commit history, understanding code structure
scope: project
version: 1.0
description: Use CodeMap to query Go symbols, definitions, and commit history before editing or reasoning about code.
---

# CodeMap Usage Skill

## When to use

Call CodeMap whenever you need to:

1. **Understand a symbol** (function, type, interface, var, const)
2. **Navigate to a definition** before editing
3. **Check commit history** of a symbol to understand changes
4. **Debug or refactor** with evidence of how/when code evolved
5. **Verify a symbol exists** before creating new one with same name

## Command sequence for coding tasks

### Before writing or editing code

```bash
codemap index           # ensure index is fresh
codemap symbol <name>   # verify definition exists, get location/signature
```

### When investigating a bug or regression

```bash
codemap history <name>  # get commit history with link strength
```

### Sequence for refactor

```bash
codemap index
codemap symbol <target>
codemap history <target>
```

## DB behavior (default)

- Default DB path: `~/.cache/codemap/<repo-hash>.db`
- Override via `--db <path>` flag or `CODEMAP_DB_PATH` env var
- If DB does not exist, `codemap index` auto-creates it
- If DB is stale (>24h), `meta.is_stale: true` in response

## Auto-index fallback

If `symbol` or `history` returns "no index found":
1. Run `codemap index` first
2. Re-run the original query

## Evidence and citation expectations

When using CodeMap evidence in responses:

- **Include symbol name, kind, file path, and line range**
- **Quote the signature** if relevant
- **Cite history entries** as `commit_hash (link_strength): description`
- **Do not fabricate** symbol data; if DB is empty or symbol absent, say so

Example citation:
```
`RunIndex` (func, cli/index.go:17-154, signature: func(ctx, w, args, repoRoot))
```

## JSON output contract

All commands return deterministic JSON with this envelope:

```json
{
  "schema_version": "1.0",
  "command": "index|symbol|history",
  "ok": true,
  "data": { ... },
  "errors": [],
  "meta": {
    "snapshot_id": 1,
    "head_ref": "refs/heads/main",
    "indexed_at": "2026-05-21T10:00:00Z",
    "is_stale": false
  }
}
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Runtime error (DB/file) |
| 2 | Validation error (missing args) |
| 3 | Data/index state error (no index, not found) |

## Non-goals

- Non-Go language support (Go-only MVP)
- Multi-symbol batch queries
- TUI/Explorer mode