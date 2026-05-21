# codemap-pi-integration Specification

## Purpose

Define deterministic installation and diagnostics behavior for integrating CodeMap with Pi harness workflows.

## Requirements

### Requirement: Install command modes

The system MUST provide `codemap install` with `--dry-run`, `--json`, and `--tui` modes.

#### Scenario: Dry-run reports planned actions without applying

- GIVEN integration templates are present in the repository
- WHEN `codemap install --dry-run --json` is executed
- THEN the command returns `status: "dry-run"`
- AND the response includes preflight checks and planned copy actions
- AND no runtime files are modified.

### Requirement: Artifact sync idempotency

The installer MUST sync the versioned skill and tool templates from the repository into Pi runtime targets.

Repeated install runs MUST be idempotent: unchanged artifacts SHALL report up-to-date and SHALL NOT be recopied.

#### Scenario: Re-running install is up-to-date

- GIVEN a successful install has already synced artifacts
- WHEN `codemap install --json` is executed again without template changes
- THEN the command returns `status: "up-to-date"`
- AND each action reports `changed: false`.

### Requirement: First-install behavior without pre-existing runtime

The installer MUST succeed when Pi runtime directories do not yet exist.

The system MUST create required runtime parent directories during apply.

#### Scenario: First install creates runtime paths

- GIVEN Pi runtime skill/tool target directories are absent
- WHEN `codemap install` is executed
- THEN preflight does not fail solely due to missing runtime directories
- AND apply creates required directories
- AND templates are copied successfully.

### Requirement: Doctor diagnostics output

The system MUST provide `codemap doctor` with human-readable output and `--json` output.

Doctor JSON output MUST include overall status, per-check results, default DB path, and DB existence state.

#### Scenario: Doctor returns machine-readable diagnostics

- GIVEN repository templates and runtime integration artifacts are discoverable
- WHEN `codemap doctor --json` is executed
- THEN output includes `status`, `checks[]`, `db_path`, and `db_exists`
- AND check levels use deterministic PASS/WARN/FAIL semantics.

### Requirement: Default DB resolution and overrides

CodeMap CLI commands MUST resolve DB path using this precedence:
1. explicit `--db` flag
2. `CODEMAP_DB_PATH` environment variable
3. default cache path outside the repository.

The default DB path MUST be deterministic per repository identity.

#### Scenario: Command resolves DB without explicit flag

- GIVEN `--db` is omitted and `CODEMAP_DB_PATH` is unset
- WHEN a CodeMap command requiring DB access is executed
- THEN the command uses the deterministic default cache DB path
- AND the path is outside repository working directories.
