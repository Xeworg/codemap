# Apply Progress: codemap-impact-edges-v1

> Last updated: 2026-05-22

## Summary

| Phase | Status | Tasks |
|-------|--------|-------|
| Phase 1 — `type_use` + `imports` | ✅ COMPLETE | P1–P6 + P5 integration test |
| Phase 2 — `references` + `casts` | ✅ COMPLETE | P7–P11 |

## Phase 1 — `type_use` + `imports` (PR A)

**Detail:** see [apply-progress/phase-1.md](apply-progress/phase-1.md)

## Phase 2 — `references` + `casts` (PR B)

**Detail:** see [apply-progress/phase-2.md](apply-progress/phase-2.md)

## Risks

| Risk | Status |
|------|--------|
| Phase 2 pushes total diff >400 LOC | ✅ Within budget; ~140 extra edges + tests |
| deadcode regression from edge expansion | ✅ Regression gate all packages pass |
| Determinism drift | ✅ Stable emission order + determinism tests green |

## Skill Resolution

`skill_resolution: paths-injected`
