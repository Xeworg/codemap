---
title: codemap-usage
trigger: symbol lookup, code navigation, function search, impact analysis, text query, migration, refactor, bug context, commit history, understanding code structure
scope: project
version: 1.0
description: Use CodeMap to query Go symbols, definitions, impact, textual matches, and commit history before editing or reasoning about code.
---

# CodeMap Usage Skill

## When to use

Call CodeMap whenever you need to:

1. **Understand a symbol** (function, type, interface, var, const)
2. **Navigate to a definition** before editing
3. **Check commit history** of a symbol to understand changes
4. **Debug or refactor** with evidence of how/when code evolved
5. **Verify a symbol exists** before creating new one with same name
6. **Estimate change impact** (related callers/callees/edges) before edits
7. **Run global textual query** when the question is broad
8. **Run explicit migration** when schema state is uncertain

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
codemap impact <target>
```

### When estimating blast radius

```bash
codemap impact <name>   # related graph/evidence for likely affected areas
```

### When question is broad/open-ended

```bash
codemap query "<text>"  # deterministic JSON search over indexed entities
```

### When schema state is uncertain

```bash
codemap migrate         # explicit schema migration command
```

### When running a dead code audit

```bash
codemap index           # ensure index is fresh
codemap deadcode        # report unused symbols (up to 100 findings)
codemap deadcode --limit 50  # reduce output to top 50
```

### Installing codemap into Pi runtime

```bash
codemap install          # apply install
codemap install --dry-run  # preview planned actions without applying
```

### Diagnosing codemap environment

```bash
codemap doctor           # human-readable health report
codemap doctor --json    # machine-readable diagnostic output
```

## Impact analysis — risk tiers

`codemap impact` returns findings sorted by `risk_tier` (high → medium → low), then `confidence`, then name. Use it to scope change impact before refactoring a symbol.


```bash
# Basic impact query
codemap impact MyFunction
```

**Key fields in impact findings:**

| Field | What it means |
|---|---|
| `risk_tier` | `high` = callers; `medium` = type uses/imports; `low` = weaker links |
| `confidence` | Strength of evidence for this finding |
| `evidence[].description` | Shows the edge type (e.g., "linked via calls") |

**Example JSON excerpt:**

```json
{
  "data": {
    "target_symbol": "MyFunction",
    "findings": [
      {
        "symbol_name": "CallerA",
        "risk_tier": "high",
        "confidence": "high",
        "evidence": [{ "type": "symbol_link", "description": "linked via calls" }]
      }
    ]
  }
}
```

**Workflow: refactor scoping**
1. Run `codemap impact <target>`
2. Filter by `risk_tier: "high"` to identify direct callers
3. For each high-risk caller, run `codemap symbol <caller>` to verify location
4. Proceed with changes with full blast-radius context

**Default cap:** 50 findings. High-risk findings are returned first.

## Symbol/history miss troubleshooting

When `codemap symbol` or `codemap history` returns exit 3 or `ok: false`, check the `explain_not_found.cause` field for structured guidance:

```bash
codemap symbol NonExistent
# Returns: explain_not_found.cause = "name_mismatch" or "stale_index" etc.
```

| Cause | Meaning | Action |
|---|---|---|
| `stale_index` | Snapshot older than 24 h | Run `codemap index` |
| `parse_error` | Files failed to parse; symbol may be in a broken file | Check `codemap index` output for parse errors; fix syntax |
| `name_mismatch` | Symbol not in fresh index; may be renamed/moved | Verify spelling; check for renames |
| `missing_history_links` | Symbol found but no git history | Rebuild index; ensure repo has commits |

**Workflow: symbol miss**
1. Note `explain_not_found.cause` from the response
2. Follow the `recommended_actions` array
3. Re-run the original command after fixing

**Example: stale index**
```json
{
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
  "errors": ["symbol \"MyFunction\" not found"]
}
```

## Deadcode report usage

`codemap deadcode` reports symbols with zero inbound references. Use it for code hygiene, pre-refactor audits, or cleanup sprints.

```bash
# Full report (up to 100 findings)
codemap deadcode

# Limit output
codemap deadcode --limit 50

# Custom DB path
codemap deadcode --db myrepo.db
```

**Key fields in each finding:**

| Field | Description |
|---|---|
| `classification` | `unused` (strong signal), `likely-unused`, `uncertain` |
| `suggestion` | `remove` (safe), `deprecate`, `justify` (keep with reason) |
| `confidence` | `high` for func/type with zero edges; `medium` for var/const |
| `evidence` | Always contains `no_inbound_edges` |

**Classification logic:**
- `func`/`type` with 0 inbound edges → `unused`, `remove`, `high` confidence
- `var`/`const` with 0 inbound edges → `unused`, `remove`, `medium` confidence
- Any symbol with > 0 inbound edges → `uncertain`, `justify`, `low` confidence

**Workflow: deadcode audit**
1. Run `codemap index` first (deadcode analysis depends on current graph)
2. Run `codemap deadcode` and inspect `classification: "unused"` findings
3. For each `suggestion: "remove"` with `confidence: "high"`, verify manually before deletion
4. For `classification: "uncertain"`, add a comment or skip

**Excluded files:** `_test.go`, `vendor/`, `testdata/`, `_pb.go`, `_grpc.go`, `_mock.go`, `_fake.go`, `_generated`, `.gen.go`, `third_party/` — these are never flagged.

**Default cap:** 100 findings, sorted by severity (unused first).

## DB behavior (default)

- Default DB path: `~/.cache/codemap/<repo-hash>.db`
- Override via `--db <path>` flag or `CODEMAP_DB_PATH` env var
- If DB does not exist, `codemap index` auto-creates it
- If DB is stale (>24h), `meta.is_stale: true` in response

## Auto-index fallback

If `symbol`, `history`, `impact`, or `query` returns "no index found":
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
  "command": "index|symbol|history|impact|query|migrate|deadcode|install|doctor",
  "ok": true,
  "data": { ... },
  "errors": [],
  "meta": { ... }
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