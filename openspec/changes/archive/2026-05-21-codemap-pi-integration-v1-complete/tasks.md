# Tasks — codemap-pi-integration-v1-complete

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~260–420 |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR1 (doctor + installer hardening) → PR2 (operator docs + contracts) → PR3 (final verify + archive prep) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

## PR1 — Installer/Doctor closure under strict TDD
Dependencies: existing install core + TUI + doctor baseline

- [x] 1. **RED: installer/doctor behavioral gaps tests**
  - Add/extend failing tests in:
    - `packages/coding-agent/codemap/cli/installer/install_test.go`
    - `packages/coding-agent/codemap/cli/installer/doctor_test.go`
  - Cover:
    - explicit `-repo` resolution for `install`
    - missing-runtime first install success
    - unreadable template/source warning surfacing
    - deterministic doctor PASS/WARN/FAIL aggregation

- [x] 2. **GREEN: implement installer/doctor hardening**
  - Implement minimal changes in:
    - `cmd/codemap/main.go`
    - `packages/coding-agent/codemap/cli/installer/install.go`
    - `packages/coding-agent/codemap/cli/installer/doctor.go`
  - Keep JSON contracts deterministic (`install --json`, `doctor --json`).

- [x] 3. **TRIANGULATE + REFACTOR: idempotent rerun matrix**
  - Add/extend tests for rerun states (dry-run → applied → up-to-date).
  - Refactor shared check rendering/helpers only if covered by passing tests.

## PR2 — Operator docs + integration contract lock
Dependencies: PR1

- [x] 4. **RED: docs contract drift checks (manual + fixture assertions where present)**
  - Validate documented flags and output fields against executable behavior for:
    - `codemap install`
    - `codemap doctor`
    - DB default/override precedence

- [x] 5. **GREEN: update operator docs to match runtime behavior**
  - Update/add:
    - `docs/codemap-cli-json-contract.md`
    - `docs/codemap-performance-baseline.md` (only if references changed)
    - `integrations/pi/skills/codemap-usage/SKILL.md` (if command flow wording needs sync)
  - Ensure docs include copyable examples:
    - install dry-run/apply/tui
    - doctor human/json
    - index/symbol/history with implicit default DB.

- [x] 6. **TRIANGULATE + REFACTOR: install artifact consistency checks**
  - Verify and adjust integration templates if needed:
    - `integrations/pi/tools/codemap-tool.json`
    - `integrations/pi/skills/codemap-usage/SKILL.md`
  - Keep repo/runtime sync behavior unchanged and idempotent.

## PR3 — Final verification checklist + archive readiness
Dependencies: PR2

- [x] 7. **Strict verification checklist (must pass before archive)**
  - Execute and capture outputs:
    - `go test -count=1 ./...`
    - `go build ./cmd/codemap`
    - `codemap install --dry-run --json`
    - `codemap install --json` (apply)
    - `codemap install --json` (up-to-date rerun)
    - `codemap doctor`
    - `codemap doctor --json`
    - `codemap index` (no `--db`)
    - `codemap symbol <known-symbol>` (no `--db`)

- [x] 8. **Write final verify artifact**
  - Create/update:
    - `openspec/changes/codemap-pi-integration-v1-complete/verify-report.md`
  - Include pass/fail table, command evidence, and any residual non-blocking risks.

- [x] 9. **Archive readiness criteria (must all be true)**
  - `proposal.md`, `spec.md`, `design.md`, `tasks.md`, `verify-report.md` present and coherent.
  - No failing tests/build in latest verification run.
  - Installer and doctor JSON contracts documented and matching outputs.
  - Integration templates (`skill` + `tool`) present and versioned in repo.
  - Change is ready for `openspec` archive/sync.
