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

### Requirement: Impact quality intelligence fields

The system MUST include quality-intelligence attributes in `codemap impact --json` results: per-finding `risk_tier` (`high`, `medium`, or `low`), `confidence`, and `evidence`.

#### Scenario: Impact finding includes risk, confidence, and evidence

- GIVEN a valid `codemap impact --json` request that returns findings
- WHEN the response is emitted
- THEN each finding includes `risk_tier`, `confidence`, and `evidence`
- AND `risk_tier` is limited to `high`, `medium`, or `low`.

### Requirement: Impact default cap and deterministic ordering

The system MUST apply a default result cap when no explicit limit is provided.

The system MUST return findings in deterministic order for equivalent input and unchanged repository/index state.

#### Scenario: Impact defaults are stable across repeated runs

- GIVEN equivalent `codemap impact --json` inputs and unchanged repository/index state
- WHEN the command is executed repeatedly without an explicit limit
- THEN the number of returned findings does not exceed the default cap
- AND the ordering of findings is stable across runs.
