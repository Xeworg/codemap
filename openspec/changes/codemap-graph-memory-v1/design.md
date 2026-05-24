# Design — codemap-graph-memory-v1

## Architecture

### Data layer
- New table: `symbol_impact_cache`
- New table: `file_impact_cache`
- Indexed by source/depth and updated_at for cache lookup and TTL filtering.

### Query layer
- `BlastRadiusQuery(ctx, db, symbolID, maxDepth, edgeTypes)`
- Implemented with `WITH RECURSIVE` over `edges`.
- Returns symbol hits with depth and edge path.

### Cache layer
- `GetCachedImpact(...)`
- `WriteCachedImpact(...)`
- `InvalidateCacheForSymbol(...)`

### CLI layer
- `impact` now accepts:
  - `--depth` (default 3, bounded)
  - `--no-cache`
- Response findings include:
  - `depth`
  - `edge_path`

## Determinism
- Stable sort: depth, risk tier, confidence, symbol, file.
- Stable envelope format and meta semantics preserved.

## Notes for next slice
- Wire automatic cache warm/invalidation from indexer.
- Add file-scope aggregation route and graph-query command.
