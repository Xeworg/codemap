# Apply Progress — codemap-pi-integration-v1-complete

## Scope
Installer/doctor closure, docs contract sync, CI baseline, and history usability follow-up.

## TDD Cycle Evidence

| Slice | RED | GREEN | REFACTOR | Evidence |
|---|---|---|---|---|
| Installer/doctor command wiring and diagnostics | Added/extended failing coverage around doctor/install behavior and DB path parity | Implemented `doctor` command wiring, repo-root resolution, deterministic doctor JSON/human output and DB path parity with runtime | Minor helper cleanup and nil guards | `cmd/codemap/main.go`, `packages/coding-agent/codemap/cli/installer/{doctor.go,doctor_test.go}` |
| Installer idempotency/runtime behavior | Existing tests and review findings identified edge gaps | Kept idempotent apply/up-to-date behavior and non-blocking first-runtime checks | No behavior regression; output contract preserved | `packages/coding-agent/codemap/cli/installer/install.go`, installer tests |
| Docs/skill contract drift | Review identified mismatch: install/doctor absent in docs/skill contract | Added install/doctor contract sections and usage guidance | Simplified help/docs consistency by removing undocumented `--yes` help flag | `docs/codemap-cli-json-contract.md`, `integrations/pi/skills/codemap-usage/SKILL.md`, `cmd/codemap/main.go` |
| CI baseline | Manual-only verification risk | Added minimal CI workflow for test+build on push/PR | Kept workflow minimal and deterministic | `.github/workflows/ci.yml` |
| History evidence usability | `history` returned `no_history` after fresh index in common flow | Populated synthetic `commits` + `symbol_commits` during index and added CLI regression test | Kept existing schema/helpers; minimized changes | `packages/coding-agent/codemap/cli/index.go`, `packages/coding-agent/codemap/cli/history_cmd_test.go` |

## Commands/Verification
- `go test -count=1 ./...`
- `go build ./cmd/codemap`
- `./codemap install --dry-run --json`
- `./codemap install --json`
- `./codemap install --json` (rerun up-to-date)
- `./codemap doctor`
- `./codemap doctor --json`
- `./codemap index`
- `./codemap symbol RunIndex`

## Notes
- `--yes` compatibility was removed from this change scope and from command help because it was undocumented/unsupported in runtime behavior.
- TUI mode remains available (`--tui`) and is manually operable; deeper automated TUI testing is tracked separately.