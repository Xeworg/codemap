# Tasks: codemap-vnext-release-v1

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 200–350 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR (smoke + extension + changelog + KPI docs) |
| Delivery strategy | single-pr |
| Chain strategy | none |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: none
400-line budget risk: Low
```

**Rationale:** All deliverables (smoke script, Pi extension sync, KPI doc, release cut, doc spot-checks) are additive and low-risk. No core logic changes to impact/deadcode/explain. Single PR with distinct file groups per deliverable is well within review budget.

---

## Task Ordering

```
R1 → R2 → R3 → R4 → R5 → R6 → R7 → R8
```

**Gate:** `go test ./...` must be green at R4 and R8. Smoke script must pass at R6.

---

## R1 — Create smoke script directory and scaffold

**File:** `scripts/smoke/smoke.sh` (new)

Create a bash smoke script that:
- Builds `codemap` from source (`go build ./cmd/codemap`).
- Uses `testdata/repos/incremental-go` as target repo.
- Uses a temp DB path (`$TMPDIR/smoke-$$.db`).
- Runs each of the 5 commands with JSON assertions via `grep` or `jq`.
- Returns exit 0 only if all assertions pass.
- Cleans up the temp DB on exit.

**Verification:** `bash scripts/smoke/smoke.sh` exits 0 from repo root.

---

## R2 — Add smoke assertions for index, symbol, history

**File:** `scripts/smoke/smoke.sh` (update)

Extend smoke.sh with steps:

| Step | Command | Assert |
|------|---------|--------|
| 1 | `codemap --repo <repo> index --db <tmp.db>` | `ok == true`, `files_scanned >= 2` |
| 2 | `codemap --repo <repo> symbol --db <tmp.db> Add` | `ok == true`, `kind == "func"` |
| 3 | `codemap --repo <repo> history --db <tmp.db> Add` | `ok == true`, `evidence` is array |

**Verification:** Smoke steps 1–3 pass individually.

---

## R3 — Add smoke assertions for impact, deadcode, query

**File:** `scripts/smoke/smoke.sh` (update)

Extend smoke.sh with steps:

| Step | Command | Assert |
|------|---------|--------|
| 4 | `codemap --repo <repo> impact --db <tmp.db> Add` | `ok == true`, findings contain `risk_tier` |
| 5 | `codemap --repo <repo> deadcode --db <tmp.db>` | `ok == true`, findings contain `classification` |
| 6 | `codemap --repo <repo> query --db <tmp.db> Add` | `ok == true`, results array present |

**Verification:** Smoke steps 4–6 pass individually.

---

## R4 — Add smoke assertions for migrate, doctor, install

**File:** `scripts/smoke/smoke.sh` (update)

Extend smoke.sh with steps:

| Step | Command | Assert |
|------|---------|--------|
| 7 | `codemap --repo <repo> migrate --db <tmp.db>` | `ok == true` (idempotent) |
| 8 | `codemap doctor --json` | `ok == true`, status ∈ {pass,warn} |
| 9 | `codemap install --dry-run --json` | `ok == true`, status present |

**Verification:** Smoke steps 7–9 pass. Run `go test ./...` — all packages green.

---

## R5 — Sync Pi extension command surface

**File:** `integrations/pi/extensions/codemap-extension.ts`

Add missing tool registrations so the extension exposes the full CLI surface:

```typescript
"codemap_impact": { ... },
"codemap_deadcode": { ... },
"codemap_query": { ... },
"codemap_install": { ... },
"codemap_doctor": { ... },
```

- Follow the existing pattern for `codemap_index`, `codemap_symbol`, `codemap_history`.
- Map each to the correct CLI subcommand and argument structure.
- Ensure `description` fields are present and accurate.

**Verification:** TypeScript compiles without errors. Extension still registers existing tools.

---

## R6 — Create KPI sampling documentation

**File:** `docs/kpi-sampling-m5.md` (new)

Document the KPI sampling method for M5 using existing fixtures:

| KPI | Threshold | Method | Result |
|-----|-----------|--------|--------|
| Impact relevance | ≥70% | Manually verify 5–10 high-risk findings from `testdata/repos/incremental-go`; count correct |
| Explain accuracy | ≥80% | Run 4 explain-not-found cases (stale_index, name_mismatch, parse_error, missing_history_links); count correct |
| Deadcode FP rate | <20% | Evaluate `testdata/deadcode-precision/fixture/fixture.go`; count false positives |

- Keep the document under 30 lines.
- Reference the fixtures used by name.
- Record actual results (even if hand-checked).

**Verification:** `docs/kpi-sampling-m5.md` exists and contains entries for all 3 KPIs.

---

## R7 — Cut release notes in CHANGELOG

**File:** `CHANGELOG.md`

Replace the `[Unreleased]` header with a versioned release cut:

```markdown
## [1.1.0] — 2026-05-22

### Added
- **smoke script**: `scripts/smoke/smoke.sh` for end-to-end validation of `index`, `symbol`, `history`, `impact`, `deadcode`, `query`, `migrate`, `doctor`, `install`.
- **Pi extension sync**: Extension now registers all CLI commands (`impact`, `deadcode`, `query`, `install`, `doctor`).
- **KPI sampling documentation**: `docs/kpi-sampling-m5.md` captures impact relevance, explain accuracy, and deadcode false-positive sampling.

### Changed
- **deadcode precision v1**: Symbols now classified with inbound-edge-aware confidence and heuristic entrypoint detection. Exported symbols and runtime entrypoints (`main`, `init`) are `uncertain` when no explicit inbound edges are found.
- **impact tier diversity**: `codemap impact` now surfaces `medium` and `low` risk findings via `type_use`, `imports`, `references`, and `casts` edges extracted at index time.

### Fixed
- *(any bugs fixed in prior changes)*

### Known Limitations
- Go-only. No multi-language support.
- File-local AST resolution; cross-module impact may be incomplete.
- Deadcode heuristics may over-flag exported symbols consumed by external repos.
- No auto-fix; commands are advisory only.

### Next Steps
- Interface/reflection edge support for deadcode v1.1.
- Cross-file SSA for impact precision.
- TUI/explorer mode enhancements.

---

## [Unreleased]
```

**Verification:** `CHANGELOG.md` has a `## [1.1.0] — 2026-05-22` header above an empty `[Unreleased]` section.

---

## R8 — Spot-verify skill and contract docs

**Files:** `integrations/pi/skills/codemap-usage/SKILL.md`, `docs/codemap-cli-json-contract.md`

- Verify `SKILL.md` workflows for `impact` and `deadcode` match the actual CLI command shapes and output fields.
- Verify `docs/codemap-cli-json-contract.md` examples for `impact` and `deadcode` have correct JSON field names and structure.
- Fix any discrepancies found (field name drift, missing response fields, outdated examples).

**Verification:** No factual errors in SKILL.md impact/deadcode sections; contract examples match envelope types.

---

## Gate Summary

| Gate | Condition |
|------|-----------|
| R4 gate | `go test ./...` passes |
| R6 gate | `docs/kpi-sampling-m5.md` exists with 3 KPI entries |
| R7 gate | CHANGELOG has versioned `[1.1.0]` header |
| R8 gate | Docs spot-checks complete, no drift found |
| Final gate | Smoke script (`bash scripts/smoke/smoke.sh`) exits 0 |

---

## Risks and mitigation

| Risk | Mitigation |
|------|-----------|
| Smoke script fails on CI due to missing `jq` | Support both `jq` assertions and `grep`-fallback JSON checking |
| Pi extension TypeScript errors | Compile check before commit; keep existing tools unchanged |
| Doc drift between skill/contract and code | Manual spot-check at R8; fix only what is clearly wrong |
| KPI document too thin to be credible | Use actual hand-checked results on real fixtures; document sample scope explicitly |
| Scope creep (adding logic to impact/deadcode/explain) | Anti-scope-creep guard in design: no edits to core CLI logic in this change |

---

## Rollback

- Revert `scripts/smoke/smoke.sh`: removes smoke coverage only.
- Revert `integrations/pi/extensions/codemap-extension.ts`: removes new tool registrations only.
- Revert `CHANGELOG.md`: reverts to `[Unreleased]` only.
- Revert `docs/kpi-sampling-m5.md`: removes KPI doc only.
- R8 doc spot-checks: no rollback needed if only factual corrections.

Core logic in `cli/`, `store/`, `indexer/` is untouched; regression gate is `go test ./...` which remains green throughout.