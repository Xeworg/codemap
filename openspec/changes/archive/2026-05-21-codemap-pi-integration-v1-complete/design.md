# Design — codemap-pi-integration-v1-complete

## Overview
This change finalizes Pi integration around three components:
1. **Installer core** (`codemap install`) as the single source of install/apply logic.
2. **TUI orchestration** (`codemap install --tui`) as a state-machine UI over the same core.
3. **Doctor diagnostics** (`codemap doctor`) as deterministic environment checks for operators/automation.

Goal: operationally reliable install/verify flows with machine-readable outputs and idempotent behavior.

## Architecture

### Components
- `cmd/codemap/main.go`
  - command routing (`install`, `doctor`)
  - shared repo-root resolution
  - mode switching (human/json/tui)
- `packages/coding-agent/codemap/cli/installer/install.go`
  - preflight checks
  - action planning
  - idempotent apply
  - result rendering (human + JSON)
- `packages/coding-agent/codemap/cli/installer/tui.go`
  - Bubble Tea model and state transitions
  - invokes installer core for checks/actions/apply
- `packages/coding-agent/codemap/cli/installer/doctor.go`
  - independent diagnostics model
  - PASS/WARN/FAIL semantics

### Installer core reuse
`Installer.Run()` is reused by:
- CLI mode (`codemap install`, `--dry-run`, `--json`)
- TUI mode (`codemap install --tui`) via command wrappers

No duplicate install logic is allowed in TUI; TUI is presentation/orchestration only.

## Data model

### InstallResult contract
```json
{
  "status": "dry-run|applied|up-to-date|error",
  "checks": [ ... ],
  "actions": [ ... ],
  "error": "...",
  "timestamp": "RFC3339"
}
```

### CheckResult
- `name`, `passed`, `info`
- optional `exists` for runtime-existence semantics

### ActionResult
- `kind=copy`, `source`, `target`, `changed`
- optional `skipped` when copy is not attempted due to source/read constraints

### DoctorResult contract
```json
{
  "status": "pass|warn|fail",
  "checks": [
    { "check": "...", "level": "pass|warn|fail", "message": "..." }
  ],
  "db_path": "...",
  "db_exists": true
}
```

## Flow design

### Install (CLI)
1. Resolve repo root (`-repo` explicit wins; otherwise detect nearest git root).
2. Build default installer with runtime targets.
3. Run preflight checks:
   - repo root
   - integrations dir
   - template skill/tool readability
   - runtime presence (non-blocking if absent)
4. Plan actions by source/target comparison.
5. If dry-run: return `status=dry-run`.
6. If no changes: return `status=up-to-date`.
7. Else apply copy with `MkdirAll` and return `status=applied`.

### Install (TUI)
State sequence:
- `welcome` → `checks` → `plan` → `confirm` → `result`

Rules:
- checks and actions come from installer core
- confirm `y/enter` applies
- non-confirm exits cleanly
- result reflects core status/error

### Doctor
Checks performed:
- repo/integrations/template presence
- Pi runtime presence (warn if missing)
- installed skill/tool presence in runtime targets
- effective default DB path and existence

Status aggregation:
- any fail => `fail`
- else any warn => `warn`
- else => `pass`

## JSON contracts and compatibility
- Install and doctor JSON outputs are deterministic and stable for automation.
- Existing command exit semantics remain:
  - install: `0` for applied/up-to-date/dry-run, `1` for error
  - doctor: `0` for pass/warn, `1` for fail

## Operational failure modes

1. **Missing/empty templates**
   - preflight fails (`template_*` check)
   - install aborts before apply

2. **Unreadable source file during plan**
   - action marked with `skipped` reason
   - surfaced in human/json output

3. **Pi runtime absent (first install)**
   - preflight does not fail solely for absence
   - apply creates required directories

4. **Permission error creating target dirs/files**
   - apply returns `status=error`
   - command exits non-zero

5. **Repo-root ambiguity**
   - explicit `-repo` is honored
   - fallback detects nearest `.git`

6. **DB not created yet**
   - doctor reports `default_db` as warn, with remediation (`codemap index`)

## File changes (target set)
- `cmd/codemap/main.go`
- `packages/coding-agent/codemap/cli/installer/install.go`
- `packages/coding-agent/codemap/cli/installer/tui.go`
- `packages/coding-agent/codemap/cli/installer/doctor.go`
- tests under `packages/coding-agent/codemap/cli/installer/*_test.go`

## Verification strategy
- `go test -count=1 ./...`
- `codemap install --dry-run --json`
- `codemap install --json` twice (applied then up-to-date)
- `codemap install --tui` manual smoke
- `codemap doctor` and `codemap doctor --json`

## Rollout
1. Ship installer core and doctor JSON contracts.
2. Enable TUI mode for interactive users.
3. Use doctor in CI/support scripts for environment triage.
