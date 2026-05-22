# Apply Progress: codemap-deadcode-precision-v1 — ALL SLICES COMPLETE

**Change:** codemap-deadcode-precision-v1
**Status:** ✅ ALL SLICES COMPLETE
**Mode:** Strict TDD
**Test runner:** `go test ./...`
**Date:** 2026-05-22

---

## Summary

All four slices of `codemap-deadcode-precision-v1` are complete:

| Slice | Scope | Status |
|-------|-------|--------|
| A | Symbol model + method/init extraction | ✅ COMPLETE |
| B | Edge extraction + persistence + store + CLI wiring | ✅ COMPLETE |
| C | Inbound-aware classifier + evidence tiers + heuristics | ✅ COMPLETE |
| D | Regression fixtures + docs + changelog | ✅ COMPLETE |

---

## Slice D: Regression fixtures + docs

**Status:** COMPLETE ✅

### New files
- `packages/coding-agent/codemap/testdata/deadcode-precision/fixture/fixture.go` — Go package with exported API, private unused/used, init, and method.
- `packages/coding-agent/codemap/testdata/deadcode-precision/deadcode_precision_test.go` — End-to-end regression test (builds codemap from source, runs index + deadcode, asserts heuristics).
- `packages/coding-agent/codemap/docs/deadcode.md` — Operational docs for deadcode command.
- `CHANGELOG.md` (new) — Changelog with unreleased deadcode precision v1 entry.

### TDD RED→GREEN evidence
- RED (D1): `package main` + `func main()` in fixture directory caused package collision. Fixed by separating `package fixture` (no `main`) into a subdirectory.
- RED (D1): `PrivateUnused` (uppercase, exported) got `uncertain` instead of `unused`. Fixed by renaming to lowercase `privateUnused`.
- RED (D1): installed PATH binary lacked `deadcode` command. Fixed by building from source with `go build ./cmd/codemap` using correct `repoRoot()` (6 levels up from test file).
- GREEN (D1–D3): all tests pass, docs written, changelog entry added.

### D1 assertions verified
| Symbol | Expected | Result |
|--------|----------|--------|
| `ExportedHelper` | NOT `unused+high` | ✅ uncertain |
| `init` | NOT `unused` | ✅ uncertain |
| `privateUnused` | `unused` or `likely-unused` | ✅ `unused` |
| `T.Method` | NOT `unused+high` | ✅ uncertain |

---

## Test Commands Run

```bash
go test ./...
ok  	codrut/packages/coding-agent/codemap/cli
ok  	codrut/packages/coding-agent/codemap/cli/installer
ok  	codrut/packages/coding-agent/codemap/git
ok  	codrut/packages/coding-agent/codemap/indexer
ok  	codrut/packages/coding-agent/codemap/store
ok  	codrut/packages/coding-agent/codemap/testdata/deadcode-precision
```

---

## Files Changed (Slice D)

```
packages/coding-agent/codemap/testdata/deadcode-precision/fixture/fixture.go         [NEW]
packages/coding-agent/codemap/testdata/deadcode-precision/deadcode_precision_test.go [NEW]
packages/coding-agent/codemap/docs/deadcode.md                                      [NEW]
CHANGELOG.md                                                                       [NEW]
openspec/changes/codemap-deadcode-precision-v1/apply-progress/slice-d.md            [NEW]
```

---

## Change Complete

All 4 slices (A–D) for `codemap-deadcode-precision-v1` are implemented, tested, and documented.

Next: archive the change per SDD protocol.