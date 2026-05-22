# Delta for impact

## ADDED Requirements

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