# Spec — codemap-graph-memory-v1

## Requirement 1: Multi-hop impact traversal
The system MUST support deterministic multi-hop impact traversal from a symbol using SQLite recursive queries.

### Scenario: Depth-bounded impact
- WHEN user runs `codemap impact <symbol> --depth N`
- THEN findings include reachable symbols up to depth `N`
- AND each finding includes `depth` and `edge_path`.

## Requirement 2: Impact cache
The system MUST support symbol impact caching with TTL and bypass control.

### Scenario: Cache-first behavior
- WHEN user runs `codemap impact <symbol>`
- THEN command checks cache first
- AND falls back to live traversal on cache miss/stale.

### Scenario: No-cache behavior
- WHEN user runs `codemap impact <symbol> --no-cache`
- THEN command always executes live traversal.

## Requirement 3: Index lifecycle cache maintenance
The system MUST warm impact cache after index completion and invalidate cache for affected files/symbols on re-index.

### Scenario: Post-index warming
- WHEN index completes
- THEN top hot symbols are warmed asynchronously with bounded concurrency.

### Scenario: Re-index invalidation
- WHEN files are changed/deleted in index run
- THEN related cache entries are invalidated.

## Requirement 4: Offline graph query command
The system MUST provide an offline deterministic natural-language-like command for impact routing.

### Scenario: Prompt-driven routing
- WHEN user runs `codemap graph-query --prompt "..."`
- THEN parser derives target symbol (+ optional depth)
- AND returns standard impact envelope without LLM calls.

## Requirement 5: AI provider config (Ollama + Minimax)
The system MUST support provider configuration persisted in SQLite and testable via CLI.

### Scenario: Configure provider
- WHEN user sets provider/model/url/key/timeout via `ai-settings` or TUI
- THEN values are persisted in `settings` table.

### Scenario: Connectivity test
- WHEN user runs `codemap ai-test`
- THEN command attempts provider connectivity (ollama/minimax only)
- AND returns deterministic success/error output.
