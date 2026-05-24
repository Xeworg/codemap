# Apply Progress — codemap-graph-memory-v1

## Status
Implemented through slices A-D.

## Slice A
- ✅ Migration `0004_graph_cache.sql` added and wired.
- ✅ `store/graph.go` added (recursive CTE + cache helpers).
- ✅ `impact` extended with `--depth` and `--no-cache`.
- ✅ `ImpactFinding` extended with `depth` and `edge_path`.
- ✅ Unit tests for graph cache/traversal added.

## Slice B
- ✅ Cache warm after index completion implemented (bounded concurrency).
- ✅ Cache invalidation on affected paths during re-index wired.
- ✅ Additional warm/invalidation tests added.

## Slice C
- ✅ AI settings persistence added (SQLite `settings` table).
- ✅ AI config/router implemented for **Ollama + Minimax** only.
- ✅ TUI configuration command implemented.
- ✅ `ai-test` connectivity command implemented.

## Slice D
- ✅ `graph-query` offline deterministic command implemented.
- ✅ Parser tests added.
- ✅ Smoke suite extended for graph-query + impact flags.

## Validation
- ✅ `go test ./...`
- ✅ `bash scripts/smoke/smoke.sh`

## Notes
- `graph-query` currently offline deterministic (no LLM parsing by design).
- Provider scope intentionally limited to Ollama/Minimax per user decision.
