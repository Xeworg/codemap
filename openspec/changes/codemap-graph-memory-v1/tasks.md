# Tasks — codemap-graph-memory-v1

## Slice A (implemented)
- [x] Add migration `0004_graph_cache.sql`.
- [x] Wire migration embed + runner registration.
- [x] Add `store/graph.go`:
  - [x] recursive blast radius query
  - [x] cache read/write helpers
  - [x] cache invalidation helper
  - [x] top-N connectivity helper
- [x] Extend impact envelope (`depth`, `edge_path`).
- [x] Extend `impact` command with `--depth` and `--no-cache`.
- [x] Add graph store tests.
- [x] Validate `go test ./...`.

## Slice B
- [ ] Cache warm at index completion.
- [ ] Incremental invalidation on changed symbols.
- [ ] Optional file-scope output mode.

## Slice C (implemented)
- [x] AI settings SQLite persistence (`store/ai_settings.go`)
- [x] CLI AI config model and router (`cli/ai_router.go`, `cli/ai_settings.go`)
- [x] TUI Bubble Tea settings screen (`cli/ai_settings_tui.go`)
- [x] `ai-test` connectivity command
- [x] Commands wired in `cmd/codemap/main.go` with help
- [x] Tests: settings roundtrip, provider validation, ActiveConfig
- [x] `go test ./...` PASS

## Slice D
- [ ] `graph-query` offline command.
- [ ] Extra smoke/benchmark script for graph queries.
