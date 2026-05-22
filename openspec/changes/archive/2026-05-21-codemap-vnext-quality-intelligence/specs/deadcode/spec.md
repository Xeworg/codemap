# deadcode Specification

## Purpose

Define report-only dead code intelligence behavior for `codemap deadcode` in deterministic, machine-consumable output.

## Requirements

### Requirement: Deadcode finding structure

The system MUST return deadcode findings with symbol identity and location details, classification, suggestion, confidence, and evidence.

Classifications MUST be limited to `unused`, `likely-unused`, or `uncertain`.

Suggestions MUST be limited to `remove`, `deprecate`, or `justify`.

#### Scenario: Deadcode result includes required fields

- GIVEN a valid `codemap deadcode --json` request with findings available
- WHEN the response is emitted
- THEN each finding includes symbol identity and location
- AND each finding includes classification, suggestion, confidence, and evidence
- AND classification and suggestion values are restricted to allowed enums.

### Requirement: Deadcode report-only safety

The system MUST provide deadcode as report-only output and SHALL NOT perform automatic code deletion or refactoring.

#### Scenario: Deadcode command does not mutate source

- GIVEN a repository and a valid `codemap deadcode` invocation
- WHEN the command completes
- THEN no source files are modified by deadcode analysis
- AND output is limited to findings and guidance.

### Requirement: Deadcode deterministic output

For equivalent repository/index state and equivalent input, deadcode output MUST be deterministic in schema shape and finding ordering.

#### Scenario: Repeated deadcode runs are stable

- GIVEN unchanged repository/index state and equivalent deadcode input
- WHEN `codemap deadcode --json` is run repeatedly
- THEN the `data` shape remains stable
- AND finding ordering remains stable.