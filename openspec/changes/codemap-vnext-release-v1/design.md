# Design: codemap-vnext-release-v1

## Scope and Guardrails

This change is **release hardening only** (M5). It must not change core logic in:
- `packages/coding-agent/codemap/cli/impact.go`
- `packages/coding-agent/codemap/cli/deadcode.go`
- `packages/coding-agent/codemap/cli/explain.go`

If release checks expose behavior issues, capture follow-up work in a new change instead of patching core logic here.

## Implementation Plan

### 1) Smoke script/checklist

## Goal
Provide deterministic end-to-end validation for: `index`, `symbol`, `history`, `impact`, `deadcode`.

## Files
- `scripts/smoke-codemap-vnext.sh` (new)
- `docs/smoke-codemap-vnext.md` (new, short operator checklist)

## Design
- Build local binary from repo root (`go build ./cmd/codemap`).
- Use curated repo fixture: `packages/coding-agent/codemap/testdata/repos/incremental-go`.
- Use temp DB path and trap cleanup.
- Run required commands with `--json` where applicable.
- Validate JSON envelope deterministically with `jq` (fallback: minimal grep checks if jq unavailable).
- Exit non-zero on first assertion failure.

## Assertions
Per command verify at least:
- `ok == true` for successful cases.
- Envelope keys exist: `schema_version`, `command`, `ok`, `data`, `errors`, `meta`.
- Command-specific checks:
  - `index`: `data.files_scanned >= 1`
  - `symbol`: `data.name` non-empty
  - `history`: `data.symbol_name` matches input
  - `impact`: findings array present; if non-empty, each finding has `risk_tier`
  - `deadcode`: findings array present; each classification in allowed enum

### 2) KPI sampling artifact

## Goal
Document auditable KPI sample outcomes for:
- impact relevance
- explain-not-found accuracy
- deadcode false-positive rate

## Files
- `docs/kpi-sampling-m5.md` (new)

## Design
- Use small curated samples only; explicitly state sample size and caveats.
- Record method, sample inputs, observed outputs, and computed percentage.
- Keep reproducibility commands in the doc (copy/paste ready).

## KPI method
- **Impact relevance**: manually validate a small set of high-risk findings from fixture output.
- **Explain accuracy**: run known not-found cases and map to expected cause labels.
- **Deadcode FP rate**: use deadcode precision fixture and classify false positives by manual truth table.

### 3) Pi extension command-surface sync

## Goal
Register missing vNext commands in extension wiring.

## Files
- `integrations/pi/extensions/codemap-extension.ts` (modify)

## Design
Add registrations consistent with current naming/wiring pattern for:
- `codemap_impact`
- `codemap_deadcode`
- `codemap_query`
- `codemap_install`
- `codemap_doctor`

Preserve existing registration style and argument pass-through behavior.

### 4) Changelog release cut

## Goal
Cut versioned release section while keeping ongoing unreleased area.

## Files
- `CHANGELOG.md` (modify)

## Design
- Keep top `## [Unreleased]` section.
- Add dated section `## [1.1.0] - 2026-05-22` (or release-day date).
- Summarize shipped capabilities (impact/deadcode/explain improvements, extension sync, smoke+KPI artifacts).
- Include known limitations from spec/proposal.

## Optional item
- `cmd/codemap/main.go`: add `--version` only if requested during apply; otherwise defer to keep diff small.

## Deterministic Validation Steps

1. `go test ./...` (must pass before and after edits).
2. Run smoke script from repo root; require zero exit code.
3. Re-run smoke script once to confirm stable deterministic pass/fail behavior.
4. Verify extension command list by static inspection (`rg "codemap_(index|symbol|history|impact|deadcode|query|install|doctor)" integrations/pi/extensions/codemap-extension.ts`).
5. Verify changelog has both `[Unreleased]` and versioned section.
6. Spot-check that prohibited core logic files were not modified.

## Rollout and Review Budget

Estimated changes: **200–350 LOC**, low risk for 400-line budget.

Recommended rollout: **single PR** with 4 commits:
1. smoke script + checklist
2. KPI sampling doc
3. extension sync
4. changelog release cut

If diff grows >400 LOC, split into two stacked PRs:
- PR A: smoke + KPI docs
- PR B: extension sync + changelog

## Test/Verification Contract

Minimum green gates:
- `go test ./...`
- `bash scripts/smoke-codemap-vnext.sh`
- Manual checklist confirms extension command surface and changelog release cut
- Scope guard check confirms no edits in impact/deadcode/explain core decision logic files

## Risks and Mitigations

- **Environment drift (jq/tooling missing):** provide fallback checks and explicit prerequisites in checklist.
- **KPI overclaiming:** include caveats and small-sample disclaimer in KPI doc.
- **Extension mismatch:** use explicit command matrix and static grep verification.
- **Scope creep pressure:** enforce prohibited-files check in verification steps.

## skill_resolution
- `none` (no injected skill paths in this delegated task)
