# impact Specification

## Purpose

Define machine-consumable JSON behavior for `codemap impact` while preserving existing CLI envelope and exit-code contracts.

## Requirements

### Requirement: Impact JSON envelope v1 compatibility

The system MUST support `codemap impact --json` responses using JSON envelope v1 with top-level fields `schema_version`, `command`, `ok`, `data`, `errors`, and `meta`.

The system MUST keep `schema_version` unchanged at `"1.0"`.

#### Scenario: Impact success response uses envelope v1

- GIVEN a valid `codemap impact --json` request
- WHEN the command completes successfully
- THEN the response includes `schema_version`, `command`, `ok`, `data`, `errors`, and `meta`
- AND `schema_version` equals `"1.0"`
- AND `ok` is `true`.

### Requirement: Impact JSON determinism

For equivalent repository/index state and equivalent input, `codemap impact --json` output MUST be deterministic in envelope structure and data field shape.

#### Scenario: Equivalent runs produce stable shape

- GIVEN identical command input and unchanged repository/index state
- WHEN `codemap impact --json` is executed repeatedly
- THEN the top-level envelope structure is unchanged across runs
- AND the `data` object shape remains stable.

### Requirement: Impact exit-code stability

`codemap impact` MUST preserve stable exit-code mapping:
- `0` for success
- `1` for runtime failure
- `2` for input/validation failure
- `3` for index/data-state failure.

#### Scenario: Impact validation failure returns exit code 2

- GIVEN invalid impact command input
- WHEN command validation fails
- THEN the process exits with code `2`.
