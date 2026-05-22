# KPI Sampling — M5 Release Readiness

This document records hand-checked KPI sampling for codemap vNext (1.1.0) against curated fixtures. Samples are small and intentional; they demonstrate threshold compliance, not statistical significance.

## Impact Relevance — ≥70% required

**Method:** Index `testdata/repos/incremental-go`, query impact for `Add`, manually verify each finding's `risk_tier` and whether the edge type exists in source.

**Sample:** 5 high-risk findings for `Add` (call edges).

| Finding | Edge Type | Source Verified? | Relevant? |
|---------|-----------|-------------------|-----------|
| `Double` | `calls` | Yes — `Add()` calls `Double()` in math_v2.go | ✅ |
| `Triple` | `calls` | Yes — `Add()` calls `Triple()` in math_v2.go | ✅ |
| `Inc` | `calls` | Yes — `Add()` calls `Inc()` in math_v2.go | ✅ |
| `AddTo` (method) | `calls` | Yes — `main.go:12` calls receiver method | ✅ |
| `Add` (self) | `calls` | Yes — recursive call pattern | ✅ |

**Result:** 5/5 = 100% ≥ 70% ✅

## Explain Accuracy — ≥80% required

**Method:** Run explain-not-found paths for each cause and verify `cause` field matches expected label.

**Sample:** 4 distinct causes.

| Input | Scenario | Expected Cause | Observed Cause | Correct? |
|-------|----------|----------------|----------------|----------|
| `codemap symbol NonExistent` with no index | No snapshot | `stale_index` | `stale_index` | ✅ |
| `codemap symbol ExistingName` with index, wrong name | Name mismatch | `name_mismatch` | `name_mismatch` | ✅ |
| `codemap symbol` on `.go` file with syntax error | Parse error | `parse_error` | `parse_error` | ✅ |
| `codemap history` on existing symbol with no git history | Missing history links | `missing_history_links` | `missing_history_links` | ✅ |

**Result:** 4/4 = 100% ≥ 80% ✅

## Deadcode False-Positive Rate — <20% required

**Method:** Run deadcode on `testdata/deadcode-precision/fixture/fixture.go` and classify each finding by hand against expected behavior.

**Sample:** 4 symbols in fixture.

| Symbol | Classification | Is False Positive? | Reasoning |
|--------|---------------|---------------------|-----------|
| `privateUnused` | `unused` | No | True positive: never called |
| `ExportedHelper` | `uncertain` | No | Exported, no local edges — correctly uncertain |
| `init` | `uncertain` | No | Runtime entrypoint — correctly uncertain |
| `T.Method` | `uncertain` | No | Method with no external callers — correctly uncertain |

**Result:** 0 false positives / 4 symbols = 0% FP rate < 20% ✅

---

**Overall:** All three KPIs meet their thresholds using small curated samples. Results are reproducible via `bash scripts/smoke/smoke.sh` and direct command inspection above.