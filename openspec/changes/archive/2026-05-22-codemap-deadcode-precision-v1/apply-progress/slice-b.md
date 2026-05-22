# Slice B: Edge extraction + persistence — COMPLETED

## Status: PASS ✅

## Changes

### New files
- `packages/coding-agent/codemap/indexer/edges.go`: `SymbolKey`, `EdgeIntent`, `EdgeExtractor` with call resolution.
- `packages/coding-agent/codemap/indexer/edges_test.go`: method-call integration test.
- `packages/coding-agent/codemap/store/edges_test.go`: reindex edge replacement test.

### Modified files
- `packages/coding-agent/codemap/indexer/parse_result.go`: `ParseResult` gains `Edges []EdgeIntent`.
- `packages/coding-agent/codemap/indexer/go_parser.go`: removed `d.Recv == nil` guard for methods.
- `packages/coding-agent/codemap/indexer/index.go`: `FileEntry` gains `Edges []EdgeIntent`; `RunIndex` gains `EdgesFound int`.
- `packages/coding-agent/codemap/cli/index.go`: edge resolution wiring via `UpsertEdges`.
- `packages/coding-agent/codemap/store/edges.go`: `ResolvedEdge`, `UpsertEdges`, `GetInboundEdges` helpers.
- `packages/coding-agent/codemap/indexer/diff.go`: edge propagation through `DiffFileEntry`.

## Evidence

```
ok  	codrut/packages/coding-agent/codemap/indexer
ok  	codrut/packages/coding-agent/codemap/store
```

## TDD cycle

- B1-B4 (edge types + extractor + call resolution): RED first (compile), GREEN.
- B5-B6 (store upsert + index wiring): GREEN.
- B7-B8 (integration tests): RED → GREEN with concrete call-graph fixture assertions.

## Notes

- Scope held: only `call` edges; `ref` and `type_use` deferred to v1.1.
- Edge cleanup on reindex confirmed via `UpsertEdges` with `INSERT OR IGNORE` and prior `DELETE edges WHERE from_symbol_id IN (SELECT id FROM symbols WHERE file_id = ?)`.