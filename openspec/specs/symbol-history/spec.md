# symbol-history Specification

## Purpose
Make `symbol` and `history` miss states actionable through explainable diagnostics.

## Requirements

### Requirement: Explain mode for missing or weak results
When `symbol` or `history` returns empty or weak output, the system MUST provide explain diagnostics containing machine-readable cause codes and recommended actions.

#### Scenario: Missing symbol returns cause and action
- GIVEN a symbol lookup with no match
- WHEN explain diagnostics are emitted
- THEN response includes `causes[]` and `recommended_actions[]`.

### Requirement: Standardized cause taxonomy
The system MUST normalize miss causes using stable codes, including at least:
- `stale_index`
- `name_mismatch`
- `parse_error`
- `missing_history_links`

#### Scenario: Stale index diagnosis
- GIVEN repository state changed after last index
- WHEN `symbol/history` miss occurs
- THEN diagnostics include `stale_index` and suggest reindex.

### Requirement: Human + JSON parity
Explain diagnostics MUST be available in JSON and human-readable output modes.

#### Scenario: CLI human output includes next steps
- GIVEN non-JSON output mode
- WHEN miss diagnostics are printed
- THEN user sees concrete next actions.
