# release-readiness Specification

## Purpose

Define M5 release-hardening behavior that validates and packages existing CodeMap capabilities for a versioned release, without changing core feature logic.

## Requirements

### Requirement: Smoke validation script and checklist

The project MUST provide a runnable smoke validation script or checklist that exercises `index`, `symbol`, `history`, `impact`, and `deadcode` against a curated local repository and verifies expected JSON-envelope fields.

The smoke flow MUST include deterministic pass/fail assertions and SHALL be executable from repository root.

#### Scenario: Smoke flow passes on curated fixture

- GIVEN a clean local environment with required tooling
- WHEN the smoke validation flow is executed
- THEN each required command (`index`, `symbol`, `history`, `impact`, `deadcode`) runs
- AND each step validates expected JSON-envelope fields
- AND the flow exits with success only when all assertions pass.

### Requirement: KPI sampling documentation

The project MUST include M5 KPI sampling documentation for impact relevance, explain-not-found accuracy, and deadcode false-positive rate.

The documentation MUST record sample scope, observed results, and limitations of the sample size.

#### Scenario: KPI report is auditable

- GIVEN release-hardening artifacts are prepared
- WHEN a reviewer opens the KPI sampling document
- THEN the reviewer can see which samples were used
- AND the computed KPI outcomes for impact, explain-not-found, and deadcode
- AND explicit caveats about sample size or representativeness.

### Requirement: Pi extension command-surface sync

The Pi extension integration MUST register the full intended CodeMap command surface used for vNext release workflows, including `impact`, `deadcode`, `query`, `install`, and `doctor`, in addition to previously registered commands.

#### Scenario: Extension exposes full vNext command set

- GIVEN the extension manifest/source is synchronized for release
- WHEN command registrations are inspected
- THEN `impact`, `deadcode`, `query`, `install`, and `doctor` are present
- AND registration naming and invocation wiring are consistent with existing CLI command behavior.

### Requirement: Versioned release-note cut

The changelog MUST include a dated versioned release section for this vNext cut and MUST retain an `[Unreleased]` section for subsequent work.

The release notes SHOULD summarize shipped capabilities and known limitations.

#### Scenario: Changelog reflects a release cut

- GIVEN M5 release hardening is complete
- WHEN `CHANGELOG.md` is reviewed
- THEN a dated version section exists for the release
- AND `[Unreleased]` remains present for future changes
- AND known limitations are documented.

### Requirement: Scope-creep prevention for M5

This change MUST NOT modify core decision logic for impact, deadcode, or explain-not-found behavior.

If release hardening reveals a need for core logic changes, that work SHALL be deferred to a separate follow-up change.

#### Scenario: M5 implementation remains packaging-only

- GIVEN files touched by this change are reviewed
- WHEN diffs are evaluated for core CLI logic modules
- THEN no behavior-changing edits exist in impact/deadcode/explain decision logic
- AND any discovered logic-change need is tracked as a separate follow-up change.
