# Apply Progress — codemap-critical-prd-gaps-v1

## SDD Preflight
- **Mode:** strict TDD
- **Chain strategy:** feature-branch-chain (3 slices)
- **Review budget:** 400 lines per PR
- **Execution mode:** interactive

---

## PR1: `codemap migrate` ✅ COMPLETE

### RED Phase (Task 1.1)
```bash
$ go test -count=1 ./packages/coding-agent/codemap/cli/ -run TestMigrate -v
# (expected: FAIL - RunMigrate undefined) ✅ Confirmed compile failure
```

### GREEN Phase
| Task | File | Lines | Status |
|------|------|-------|--------|
| 1.3 | `envelope.go` | +9 | `MigrateData` struct |
| 1.2 | `migrate.go` | 54 | `RunMigrate` |
| 1.4 | `main.go` | +10 | case + help wiring |
| 1.5 | `integration_test.go` | +25 | `TestMigrateEnvelopeShape` |

### Test output
```
$ go test -count=1 ./packages/coding-agent/codemap/cli/ -run TestMigrate -v
PASS (13 tests: 13 passed, 0 failed)
$ go test -count=1 ./packages/coding-agent/codemap/cli/... ./packages/coding-agent/codemap/store/...
ok (all packages)
```

### PR1 Files Changed
- `packages/coding-agent/codemap/cli/envelope.go`
- `packages/coding-agent/codemap/cli/migrate.go` (new)
- `packages/coding-agent/codemap/cli/migrate_cmd_test.go` (new)
- `packages/coding-agent/codemap/cli/integration_test.go`
- `cmd/codemap/main.go`

**PR1 line count:** ~344 lines (under 400)

---

## PR2: `codemap impact --json` ✅ COMPLETE

### RED Phase (Tasks 2.1, 2.5)
```bash
$ go test -count=1 ./packages/coding-agent/codemap/cli/ -run TestImpact -v
# (expected: FAIL - RunImpact undefined) ✅ Confirmed compile failure
```
- Verified `GetSymbolEdges` and `SymbolEdge` did NOT exist → implemented `store/edges.go`

### GREEN Phase
| Task | File | Lines | Status |
|------|------|-------|--------|
| 2.3 | `envelope.go` | +9 | `ImpactData` struct |
| 2.2 | `impact.go` | 111 | `RunImpact` |
| 2.5 | `store/edges.go` | 87 | `GetSymbolEdges`, `GetSymbolByID` |
| 2.4 | `main.go` | +6 | case + help wiring |
| 2.6 | `integration_test.go` | +53 | `TestImpactEnvelopeShapeAndDeterminism` |

**Bug fix during GREEN:** `GetLatestSnapshotMeta` returned error (not nil) on unmigrated DB → fixed `store/meta.go` to handle "no such table" gracefully, returning `SnapshotMeta{}` → impact now correctly returns exit 3 instead of 1.

**Bug fix during GREEN:** Empty slices serialized as `null` in JSON (e.g., `affected_symbols: null`). Fixed by initializing `affected = []string{}` and `evidence = []EvidenceEntry{}` before building the payload.

### Test output
```
$ go test -count=1 ./packages/coding-agent/codemap/cli/ -run TestImpact -v
PASS (17 tests: 17 passed, 0 failed)
$ go test -count=1 ./packages/coding-agent/codemap/cli/... ./packages/coding-agent/codemap/store/...
ok (all packages)
```

### PR2 Files Changed
- `packages/coding-agent/codemap/cli/envelope.go`
- `packages/coding-agent/codemap/cli/impact.go` (new)
- `packages/coding-agent/codemap/cli/impact_cmd_test.go` (new)
- `packages/coding-agent/codemap/cli/integration_test.go`
- `packages/coding-agent/codemap/store/edges.go` (new)
- `packages/coding-agent/codemap/store/meta.go`
- `cmd/codemap/main.go`

**PR2 line count:** ~370 lines (under 400)

---

## PR3: `codemap query --json` ✅ COMPLETE

### RED Phase (Tasks 3.1, 3.4)
```bash
$ go test -count=1 ./packages/coding-agent/codemap/cli/ -run TestQuery -v
# (expected: FAIL - RunQuery undefined) ✅ Confirmed compile failure
```
- `GetAllSymbols` did not exist → implemented in `store/edges.go` (also needed for query prefix fallback)

### GREEN Phase
| Task | File | Lines | Status |
|------|------|-------|--------|
| 3.3 | `envelope.go` | +16 | `QueryData`, `QueryMatch` |
| 3.2 | `query.go` | 106 | `RunQuery` |
| 3.5 | `main.go` | +6 | case + help wiring |
| 3.6 | `integration_test.go` | +195 | `TestQueryDeterminismMultipleSymbols`, `TestExitCodesMatrix`, `TestEnvelopeShapeAllCommands` |

### Test output
```
$ go test -count=1 ./packages/coding-agent/codemap/cli/ -run TestQuery -v
PASS (18 tests: 18 passed, 0 failed)
$ go test -count=1 ./packages/coding-agent/codemap/cli/... ./packages/coding-agent/codemap/store/...
ok (all packages)
```

### PR3 Files Changed
- `packages/coding-agent/codemap/cli/envelope.go`
- `packages/coding-agent/codemap/cli/query.go` (new)
- `packages/coding-agent/codemap/cli/query_cmd_test.go` (new)
- `packages/coding-agent/codemap/cli/integration_test.go`
- `cmd/codemap/main.go`

**PR3 line count:** ~390 lines (under 400)

---

## Shared deliverables ✅ ALL COMPLETE

### Task S.1 — Envelope compatibility across all commands
`TestEnvelopeShapeAllCommands` verifies `schema_version`, `ok`, `errors`, `meta` for all 6 commands (index, symbol, history, migrate, impact, query) ✅

### Task S.2 — Exit code matrix
`TestExitCodesMatrix` table-driven test for all 6 commands × 4 exit codes (0, 1, 2, 3) ✅

### Task S.3 — helpRoot updated
`codemap migrate`, `codemap impact`, `codemap query` all listed in `helpRoot` output ✅

---

## TDD Cycle Evidence

### PR1 — migrate
| Cycle | Action | Result |
|-------|--------|--------|
| RED | Write 13 tests in `migrate_cmd_test.go` | Build fails: `RunMigrate undefined` |
| GREEN | Implement `RunMigrate`, `MigrateData`, wire `main.go` | 13 tests PASS |
| REFACTOR | No refactoring needed (thin impl, follows existing patterns) | Full suite PASS |

### PR2 — impact
| Cycle | Action | Result |
|-------|--------|--------|
| RED | Write 17 tests in `impact_cmd_test.go` | Build fails: `RunImpact undefined` |
| GREEN | Implement `RunImpact`, `ImpactData`, `store/edges.go`, fix `meta.go` + empty slices | 17 tests PASS |
| REFACTOR | Fix empty slice → `[]T{}` initialization for JSON null fix | Full suite PASS |

### PR3 — query
| Cycle | Action | Result |
|-------|--------|--------|
| RED | Write 18 tests in `query_cmd_test.go` | Build fails: `RunQuery undefined` |
| GREEN | Implement `RunQuery`, `QueryData`, `QueryMatch`, wire `main.go` | 18 tests PASS |
| REFACTOR | Add `TestQueryDeterminismMultipleSymbols`, `TestExitCodesMatrix`, `TestEnvelopeShapeAllCommands` | Full suite PASS |

---

## Final verification

```bash
$ go test -count=1 ./packages/coding-agent/codemap/cli/...
ok      codrut/packages/coding-agent/codemap/cli        0.637s
ok      codrut/packages/coding-agent/codemap/cli/installer   0.004s

$ go test -count=1 ./packages/coding-agent/codemap/store/...
ok      codrut/packages/coding-agent/codemap/store     0.070s

$ go build ./cmd/codemap/...
# (no output — build successful)
```

All 48 tests across 3 command test suites + 2 integration tests + 2 store tests pass.

---

## Residue risks

| Risk | Severity | Status |
|------|----------|--------|
| Envelope drift from v1 contract | Low | Mitigated: `TestEnvelopeShapeAllCommands` |
| Exit-code regressions | Low | Mitigated: `TestExitCodesMatrix` |
| Non-deterministic ordering | Low | Mitigated: sort.Strings/sort.Slice on all variable-length fields |
| Empty slice → null in JSON | Low | Fixed: initialize `[]T{}` before encoding |
| `GetLatestSnapshotMeta` on unmigrated DB | Low | Fixed: handle "no such table" gracefully |
| Impact/Query DB path resolution | Low | Note: `DefaultDBPath` creates dirs recursively; tests use absolute /nonexistent paths only for runtime errors |

---

*Last updated: all 3 PRs complete, full suite green*