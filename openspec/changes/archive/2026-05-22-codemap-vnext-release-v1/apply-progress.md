# Apply Progress: codemap-vnext-release-v1

## Status: COMPLETE

## RED → GREEN Evidence

| Attempt | Problem | Fix |
|---------|---------|-----|
| RED 1 | Smoke script failed on `grep -q '"ok":true'` for `doctor` and `install` commands | These commands output `{"status":"pass"}` and `{"status":"dry-run"}` respectively, not `"ok"`. Fixed grep patterns to use `'"status": "pass"'` and `'"status": "dry-run"'` with space after colon. |

**GREEN at:** `bash scripts/smoke/smoke.sh` — ALL TESTS PASSED; `go test ./...` — all packages green.

## Tasks Completed (R1–R8)

- [x] **R1** — Smoke script scaffold (`scripts/smoke/smoke.sh`)
- [x] **R2** — Assertions for `index`, `symbol`, `history`
- [x] **R3** — Assertions for `impact`, `deadcode`, `query`
- [x] **R4** — Assertions for `migrate`, `doctor`, `install`; `go test ./...` gate
- [x] **R5** — Pi extension sync: added `codemap_impact`, `codemap_deadcode`, `codemap_query`, `codemap_install`, `codemap_doctor`
- [x] **R6** — KPI sampling doc (`docs/kpi-sampling-m5.md`) with 3 KPIs documented
- [x] **R7** — CHANGELOG release cut `## [1.1.0] — 2026-05-22`
- [x] **R8** — SKILL.md and contract docs spot-verified; no drift found

## Files Changed

| File | Change |
|------|--------|
| `scripts/smoke/smoke.sh` | New: end-to-end smoke validation for all 9 commands |
| `docs/kpi-sampling-m5.md` | New: KPI sampling evidence for impact, explain, deadcode |
| `CHANGELOG.md` | Modified: added `## [1.1.0] — 2026-05-22` section with Added/Changed/Known Limitations/Next Steps |
| `integrations/pi/extensions/codemap-extension.ts` | Modified: added 5 new tool registrations (impact, deadcode, query, install, doctor) |

## Scope Guard

**No core logic edits** in:
- `packages/coding-agent/codemap/cli/impact.go` ✅
- `packages/coding-agent/codemap/cli/deadcode.go` ✅
- `packages/coding-agent/codemap/cli/explain.go` ✅

All changes are additive: new smoke script, KPI doc, changelog release cut, extension registrations.

## Test Commands Run

```bash
go test ./...            # → all packages green
bash scripts/smoke/smoke.sh  # → ALL SMOKE TESTS PASSED
```