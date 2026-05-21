# query Specification

## Purpose

Define deterministic machine-consumable JSON behavior for `codemap query` under the existing CLI response and exit-code contract.

## Requirements

### Requirement: Query JSON envelope v1 compatibility

The system MUST support `codemap query --json` responses using JSON envelope v1 with top-level fields `schema_version`, `command`, `ok`, `data`, `errors`, and `meta`.

The system MUST keep `schema_version` unchanged at `"1.0"`.

#### Scenario: Query success response uses envelope v1

- GIVEN a valid `codemap query --json` request
- WHEN the command completes successfully
- THEN the response includes `schema_version`, `command`, `ok`, `data`, `errors`, and `meta`
- AND `schema_version` equals `"1.0"`
- AND `ok` is `true`.

### Requirement: Query machine-parseable deterministic output

`codemap query --json` MUST return machine-parseable query results in a deterministic `data` shape for equivalent inputs and unchanged index state.

#### Scenario: Query results are parseable with stable shape

- GIVEN a valid query against an unchanged index
- WHEN `codemap query --json` returns results
- THEN results are present in a machine-parseable `data` structure
- AND equivalent repeated runs preserve the same `data` structure.

### Requirement: Query exit-code stability

`codemap query` MUST preserve stable exit-code mapping:
- `0` for success
- `1` for runtime failure
- `2` for input/validation failure
- `3` for index/data-state failure.

#### Scenario: Query data-state failure returns exit code 3

- GIVEN query execution depends on unavailable or invalid index/data state
- WHEN command execution fails due to data/index state
- THEN the process exits with code `3`.
