# Delta for impact

## ADDED Requirements

### Requirement: Index-time edge extraction supports impact tier diversity

The system MUST expand file-local AST edge extraction beyond calls to emit `type_use`, `imports`, and `references` edge intents when source and target symbols are resolvable from indexed file-local context.

The system MAY emit `casts` edge intents when type-assertion targets are resolvable.

#### Scenario: Resolvable non-call usages produce edge intents

- GIVEN a parsed file containing resolvable type usage, import usage, and symbol references
- WHEN indexing extracts edges
- THEN extracted edge intents include `type_use`, `imports`, and `references` as applicable
- AND existing `calls` extraction remains supported.

### Requirement: Unresolved non-call edge candidates are fail-soft

The system MUST skip unresolved `type_use`, `imports`, `references`, and `casts` edge candidates without failing indexing.

#### Scenario: Unresolved references are skipped

- GIVEN a parsed file with non-call edge candidates that cannot be resolved
- WHEN indexing extracts edges
- THEN unresolved candidates are not emitted as edges
- AND indexing continues successfully.

## MODIFIED Requirements

### Requirement: Impact quality intelligence fields

The system MUST include quality-intelligence attributes in `codemap impact --json` results: per-finding `risk_tier` (`high`, `medium`, or `low`), `confidence`, and `evidence`.

The system MUST derive `risk_tier` from available edge types, including non-call edge types populated at index time, so that non-`high` tiers can appear when appropriate.

(Previously: Required quality-intelligence fields and allowed `risk_tier` values, but did not require non-call edge-driven tier population.)

#### Scenario: Impact finding includes risk, confidence, and evidence

- GIVEN a valid `codemap impact --json` request that returns findings
- WHEN the response is emitted
- THEN each finding includes `risk_tier`, `confidence`, and `evidence`
- AND `risk_tier` is limited to `high`, `medium`, or `low`.

#### Scenario: Mixed edge types produce tier diversity

- GIVEN indexed impact findings derived from both call and non-call edge types
- WHEN `codemap impact --json` is executed
- THEN at least one finding MAY be `high` and at least one finding MAY be non-`high` when such edges exist
- AND ordering remains deterministic for equivalent input and unchanged index state.

### Requirement: Impact default cap and deterministic ordering

The system MUST apply a default result cap when no explicit limit is provided.

The system MUST return findings in deterministic order for equivalent input and unchanged repository/index state, including when additional edge types are present.

(Previously: Required default cap and deterministic ordering without explicitly covering expanded edge-type inputs.)

#### Scenario: Impact defaults are stable across repeated runs

- GIVEN equivalent `codemap impact --json` inputs and unchanged repository/index state
- WHEN the command is executed repeatedly without an explicit limit
- THEN the number of returned findings does not exceed the default cap
- AND the ordering of findings is stable across runs.
