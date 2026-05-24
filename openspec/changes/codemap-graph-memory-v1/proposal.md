# Proposal — codemap-graph-memory-v1

## Summary
Add SQLite-backed project graph memory so codemap can answer multi-hop impact questions deterministically ("if I change function X, what else changes?").

## Why
Current impact is mostly direct-link oriented. AI-assisted edits need deeper, reproducible blast-radius context with low latency.

## Goals
1. Recursive graph traversal over existing symbol/edge data.
2. Impact cache tables for fast repeat queries.
3. `impact` command support for depth-aware results.
4. Keep JSON envelope stable and deterministic.

## Scope
### In scope (v1)
- Migration `0004_graph_cache`.
- `store/graph.go` traversal + cache helpers.
- `impact --depth` and `impact --no-cache`.
- Tests + smoke validation.

### Out of scope (later slices)
- Cache warm + incremental invalidation wiring.
- `graph-query` command.
- AI provider TUI (Ollama/Minimax).

## Risks
- Recursive query performance on large repos.
- Cache staleness.

## Mitigations
- Depth cap.
- TTL + no-cache bypass.
- Deterministic ordering and tests.
