# migrate Specification

## Purpose

Define explicit schema migration command behavior for `codemap migrate` with stable, intentional migration execution and unchanged exit-code policy.

## Requirements

### Requirement: Explicit migration command execution

The system MUST provide `codemap migrate` as an explicit command to apply SQLite schema migrations intentionally.

#### Scenario: Migrate applies pending migrations explicitly

- GIVEN a database with unapplied schema migrations
- WHEN `codemap migrate` is executed
- THEN pending migrations are applied through the explicit migrate command path.

### Requirement: Idempotent migrate behavior

`codemap migrate` MUST be idempotent when no pending migrations exist.

#### Scenario: Migrate no-op on current schema

- GIVEN a database already at the latest migration state
- WHEN `codemap migrate` is executed
- THEN no additional schema changes are applied
- AND command outcome reports successful no-op behavior.

### Requirement: Migrate exit-code stability

`codemap migrate` MUST preserve stable exit-code mapping:
- `0` for success
- `1` for runtime failure
- `2` for input/validation failure
- `3` for index/data-state failure.

#### Scenario: Migration runtime failure returns exit code 1

- GIVEN a migration execution runtime failure
- WHEN `codemap migrate` fails at runtime
- THEN the process exits with code `1`.
