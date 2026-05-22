# `codemap deadcode` — Dead Code Precision Analysis

The `deadcode` command analyzes a Go repository's indexed symbols and reports
those with no inbound call-graph edges, along with a classification, confidence
level, and actionable suggestions.

## Output Structure

Every invocation returns a JSON envelope with this shape:

```json
{
  "schema_version": "1.0",
  "command": "deadcode",
  "ok": true,
  "data": {
    "findings": [
      {
        "symbol_name": "...",
        "file": "...",
        "kind": "func|type|var|const|method",
        "start_line": 10,
        "end_line": 15,
        "classification": "unused|likely-unused|uncertain",
        "suggestion": "remove|deprecate|justify",
        "confidence": "high|medium|low",
        "evidence": [
          { "type": "...", "description": "..." }
        ]
      }
    ]
  },
  "meta": {
    "snapshot_id": 1,
    "head_ref": "abc123",
    "indexed_at": "2026-05-22T14:00:00Z",
    "is_stale": false
  }
}
```

## Classification Values

| Value | Meaning |
|-------|---------|
| `unused` | No inbound call-graph edges found; no heuristic protection applies |
| `likely-unused` | Few inbound edges or partial resolution; manual review recommended |
| `uncertain` | Heuristic protection active; classification is not definitive |

## Confidence Levels

| Level | When it applies |
|-------|----------------|
| `high` | Private `func` or `type` with no inbound edges and no heuristic protection |
| `medium` | Private `var`/`const`, or method with no edges |
| `low` | Any symbol with active heuristic protection |

## Evidence Tiers

Each finding includes a list of `evidence` entries that explain *why* the
classifier arrived at its decision. The tiers are:

### 1. Edge-based evidence

- **`inbound_edges`** — the symbol has explicit inbound call edges in the code graph.
  When this is the only evidence, the classification is `uncertain` because an edge
  means *something* calls the symbol from within the indexed scope.

- **`no_inbound_edges`** — no explicit edges found. This is the starting condition
  for deadcode analysis; it is always present when a symbol appears in findings.

### 2. Heuristic protection tiers

These tiers suppress aggressive dead-code classification:

- **`implicit_runtime_entry`** — the symbol is `main` or `init` (runtime entrypoints).
  These functions are always live, even if no edges are found in the index.

- **`public_api_surface`** — the symbol name starts with an uppercase letter
  (Go-exported identifier). Exported symbols may be used by external callers that
  are outside the indexed repository boundary.

### 3. Confidence modifier

- **`method_owner_context`** — (future, v1.1) method is uncertain pending analysis
  of its receiver type's usage and lifetime.

## Heuristic Boundaries (v1)

The classifier uses a conservative boundary:

- **Runtime entrypoints** (`main`, `init`): always `uncertain`.
- **Public API** (exported identifiers, uppercase-first letter): always `uncertain`.
- **Private functions/types with no edges**: `unused` with `high` confidence → safe to remove.
- **Private vars/consts with no edges**: `unused` with `medium` confidence.
- **Methods with no edges**: `unused` with `medium` confidence (defer to owner analysis).

## Guaranteed Safe Actions

Only one combination is considered **safe for automated action**:

> `classification = "unused"` AND `confidence = "high"` → `suggestion = "remove"`

For all other combinations, the recommendation is `review` or `justify`.

## Limitations

- **Cross-package edges are not tracked.** If a symbol in package A is called
  from package B outside the index boundary, it will appear as dead.
- **Dynamic calls** (`reflect.Call`, plugin systems, `os/exec` with constructed
  strings) are not resolved by the AST-based edge extractor.
- **Public API heuristic is shallow.** Only the uppercase-start heuristic is used
  in v1; no package-level analysis (e.g., `_test.go` convention or `export.go`
  patterns) is applied.
- **Methods are classified conservatively.** Owner analysis (v1.1) is required to
  determine whether a method with no external callers should be removed.