# Proposal: codemap-vnext-release-v1

## Intent
Complete vNext **M5 release hardening** without changing core feature logic. This change packages and validates already-delivered capabilities (index/symbol/history/impact/deadcode/explain) for release readiness.

## Scope (M5-only)
1. Add a smoke script/checklist that validates end-to-end command behavior and JSON envelope expectations.
2. Add KPI sampling documentation (impact relevance, explain-not-found accuracy, deadcode false-positive rate) using curated fixtures/samples.
3. Sync Pi extension command surface with existing CLI commands (add missing tool registrations).
4. Cut release notes/version section in `CHANGELOG.md` (promote from `[Unreleased]` to a dated release section and keep a new `[Unreleased]` header).
5. Optional: lightweight CLI `--version` output if needed for release UX, with minimal implementation.

## Explicitly Out of Scope
- Any new impact/deadcode/explain feature logic.
- Schema or JSON contract changes.
- DB/migration behavior changes.
- CI/CD pipeline redesign.
- Multi-language support.
- Auto-fix/refactor behavior.
- Performance tuning/benchmarking work.

## Affected Areas
- `scripts/` (new smoke script/checklist)
- `docs/` (KPI sampling report and release-facing notes)
- `integrations/pi/extensions/codemap-extension.ts` (command-surface sync)
- `CHANGELOG.md` (release cut)
- Optional/minimal: `cmd/codemap/main.go` (`--version` only if required)

## Risks and Mitigations
1. **Scope creep into feature logic**
   - Mitigation: reject changes touching impact/deadcode/explain decision logic in this change.
2. **Smoke script environment fragility**
   - Mitigation: use existing local fixtures/repositories and deterministic JSON assertions.
3. **Extension/CLI drift recurrence**
   - Mitigation: explicitly map extension registrations to current CLI command list.
4. **KPI overclaiming from small samples**
   - Mitigation: document sample size and limitations explicitly.

## Anti-Scope-Creep Guardrails
- If implementation requires modifying core behavior in:
  - `packages/coding-agent/codemap/cli/impact.go`
  - `packages/coding-agent/codemap/cli/deadcode.go`
  - `packages/coding-agent/codemap/cli/explain.go`
  stop and defer to a new follow-up change.
- Keep this change release-packaging and validation only.

## Review Workload Estimate
- Estimated changed lines: **200–350 LOC**
- 400-line budget risk: **Low**
- Recommended delivery: **single PR** (split only if optional `--version` + docs expansion pushes beyond budget).

## Rollback Plan
- Revert smoke/KPI/docs/extension/release-note changes as a unit.
- If optional `--version` is added and causes review noise, drop it and proceed with docs/validation-only release cut.

## Success Criteria
- Smoke script/checklist passes on target fixture repo(s).
- KPI sampling document is added and references concrete command outputs.
- Pi extension registers the full intended command surface.
- CHANGELOG contains a dated release section for vNext and known limitations.
- `go test ./...` remains green.

## Skill Resolution
- `skill_resolution: none` (no injected skill file paths provided in this delegated execution)
