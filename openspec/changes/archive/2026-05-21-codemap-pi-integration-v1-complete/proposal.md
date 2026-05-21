# Proposal — codemap-pi-integration-v1-complete

## Intent
Close the remaining finish-line gaps for CodeMap Pi integration so teams can reliably install, verify, and operate CodeMap from Pi harness workflows without manual setup drift.

## Scope (minimal)
- Finalize `codemap install` operational behavior (CLI + TUI paths already implemented) with release-ready checks.
- Finalize `codemap doctor` diagnostics behavior and expected PASS/WARN/FAIL semantics.
- Finalize operator documentation for install, doctor, default DB behavior, overrides, and troubleshooting.
- Produce final verification evidence and archive-readiness artifacts.

## Out of scope
- New indexing features beyond existing MVP behavior.
- Multi-language parsing.
- Global mandatory enforcement policies outside current project integration artifacts.

## Current baseline
Already implemented in repo:
- Installer core + TUI (`codemap install`, `--dry-run`, `--json`, `--tui`, `--yes` compatibility).
- First-install runtime creation when Pi runtime is absent.
- `codemap doctor` command with human and JSON diagnostics.
- Versioned integration artifacts:
  - `integrations/pi/skills/codemap-usage/SKILL.md`
  - `integrations/pi/tools/codemap-tool.json`

## Remaining work to close
1. Lock and document final operator workflow (install → doctor → index/symbol/history).
2. Validate end-to-end idempotency and diagnostics in final verify report.
3. Ensure OpenSpec artifacts for closure are complete and archive-ready.

## Affected areas
- `cmd/codemap/main.go`
- `packages/coding-agent/codemap/cli/installer/*`
- `docs/*` (Pi integration/operator docs)
- `openspec/changes/codemap-pi-integration-v1-complete/*`

## Risks
- Environment-specific Pi runtime path differences may cause install confusion.
- False-negative diagnostics if runtime exists but permissions are restricted.
- Documentation drift from actual command behavior.

## Mitigations
- Keep checks explicit and machine-readable in `--json` modes.
- Include first-install and idempotent rerun verification evidence.
- Tie docs to tested command outputs and examples.

## Rollback
- Revert integration command/router additions and installer package changes if regressions appear.
- Remove synced runtime artifacts from `~/.pi/agent/{skills,tools}` and re-run install from stable commit.

## Success criteria
- `codemap install` works in dry-run/apply/up-to-date paths, including first-install on missing runtime.
- `codemap doctor` reports deterministic PASS/WARN/FAIL and valid JSON output.
- Documented workflow is accurate and reproducible.
- Verification evidence is captured and OpenSpec change is ready for archive.
