# Delta for codemap-mvp

## ADDED Requirements

### Requirement: Explain-not-found for symbol queries

When `codemap symbol --json` cannot resolve a requested symbol, the system MUST return structured explain-not-found details with a cause taxonomy limited to `stale_index`, `name_mismatch`, or `parse_error`, plus recommended actions.

#### Scenario: Symbol miss returns actionable explain details

- GIVEN a valid `codemap symbol --json` request for a symbol that is not resolved
- WHEN the command returns a not-found outcome
- THEN the response includes structured explain-not-found details
- AND the cause is one of `stale_index`, `name_mismatch`, or `parse_error`
- AND recommended actions are present.

### Requirement: Explain-not-found for history queries

When `codemap history --json` cannot resolve a requested symbol history, the system MUST return structured explain-not-found details with a cause taxonomy limited to `stale_index`, `name_mismatch`, `parse_error`, or `missing_history_links`, plus recommended actions.

#### Scenario: History miss returns actionable explain details

- GIVEN a valid `codemap history --json` request that yields no resolvable history
- WHEN the command returns a not-found outcome
- THEN the response includes structured explain-not-found details
- AND the cause is one of `stale_index`, `name_mismatch`, `parse_error`, or `missing_history_links`
- AND recommended actions are present.