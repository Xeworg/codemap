# Apply Progress: codemap-deadcode-precision-v1 — Slice A

## Slice A: Symbol coverage foundation

**Scope:** `packages/coding-agent/codemap/indexer/`
**Target:** Add Recv/File fields to Symbol, update parser to handle methods + init, add receiver naming.

---

## TDD Cycle Evidence

### Cycle 1: Method extraction (RED→GREEN)

| Step | Evidence |
|------|----------|
| RED test written | `TestExtractGoSymbols_IncludesMethods` — test fails because methods skipped (`d.Recv == nil` guard) |
| GREEN implemented | Removed guard in `go_parser.go`, added `symbolFromFuncDecl` path for `d.Recv != nil` with `Kind="method"`, `Recv=receiverName(...)` |
| GREEN verified | `go test ./packages/coding-agent/codemap/indexer/... -v -run "TestExtractGoSymbols"` passes |
| REFACTOR | Extracted `receiverName()` helper for testability |

### Cycle 2: Init function extraction (RED→GREEN)

| Step | Evidence |
|------|----------|
| RED test written | `TestExtractGoSymbols_IncludesInit` — test fails because init should be included with `Kind="func"`, `Name="init"` |
| GREEN implemented | No guard needed, init flows through same path as top-level funcs (d.Recv == nil) |
| GREEN verified | All 5 new tests pass |
| REFACTOR | None needed — clean path |

### Cycle 3: Pointer receiver naming (RED→GREEN)

| Step | Evidence |
|------|----------|
| RED test written | `TestExtractGoSymbols_MethodWithPointerReceiver` — `(t *T) Method()` → `Recv="*T"` |
| GREEN implemented | In `receiverName()`, detect `*ast.StarExpr` → prepend `*` |
| GREEN verified | `go test ./packages/coding-agent/codemap/indexer/...` passes |

### Cycle 4: Multiple init funcs (RED→GREEN)

| Step | Evidence |
|------|----------|
| RED test written | `TestExtractGoSymbols_MultipleInitFuncs` — multiple `init` funcs in same file |
| GREEN implemented | No special handling needed — flows through standard path |
| GREEN verified | `go test ./packages/coding-agent/codemap/indexer/...` passes |

### Cycle 5: Receiver naming variants (RED→GREEN)

| Step | Evidence |
|------|----------|
| RED test written | `TestExtractGoSymbols_ReceiverNamingVariants` — `(T)`, `(*T)` |
| GREEN implemented | `receiverName()` handles Ident (→ name), StarExpr (→ `*name`) |
| GREEN verified | `go test ./packages/coding-agent/codemap/indexer/...` passes |

---

## Completed Tasks

- [x] **A1** Expand `Symbol` model with `Recv` and `File` fields
- [x] **A2** Update `symbolFromFuncDecl` to handle methods (`d.Recv != nil`) with receiver-qualified naming
- [x] **A3** Add unit tests: `TestExtractGoSymbols_IncludesMethods`, `TestExtractGoSymbols_IncludesInit`, `TestExtractGoSymbols_MethodWithPointerReceiver`, `TestExtractGoSymbols_MultipleInitFuncs`
- [x] **A4** Add boundary test: `TestExtractGoSymbols_ReceiverNamingVariants`

---

## Files Changed

| File | Lines Added/Modified | Description |
|------|---------------------|-------------|
| `packages/coding-agent/codemap/indexer/parse_result.go` | +4 | Add `Recv`, `File` fields to `Symbol` struct |
| `packages/coding-agent/codemap/indexer/go_parser.go` | +32 | Remove `d.Recv == nil` guard, add `receiverName()` helper, method/init handling |
| `packages/coding-agent/codemap/indexer/go_parser_methods_test.go` | +178 | 5 new test functions (RED-phase tests) |

---

## Test Commands Run

```bash
# RED tests (initial failures)
go test ./packages/coding-agent/codemap/indexer/... -v -run "TestExtractGoSymbols"
# → 3 failures (methods/pointer receivers not extracted)

# GREEN (after implementation)
go test ./packages/coding-agent/codemap/indexer/... -v -run "TestExtractGoSymbols"
# → All 5 tests pass

# Full package gate
go test ./packages/coding-agent/codemap/indexer/...
# → ok

# Full suite gate
go test ./...
# → ok (all packages)
```

**All tests pass.** Existing tests updated field assignments where needed.

---

## Design Decisions

- **isPublicAPI boundary (v1):** uppercase-start only. Heuristic `isPublicAPI(name string) bool` — `unicode.IsUpper(rune(name[0]))`. No package-level heuristics in v1.
- **Receiver naming:** `*ast.Ident` → name; `*ast.StarExpr` → `*` + underlying name; package-qualified receivers → strip to last identifier.
- **Init functions:** treated as `Kind="func"`, `Name="init"`, same line range. No special Recv field.
- **No duplicate init prevention:** multiple `init` funcs each get their own symbol entry.

---

## Deviations from Design

- **File field in Symbol:** Not populated in Slice A (unused by downstream code). Will be populated in Slice B when `ParseGoFile` gains `File` context.
- **No other deviations.** Implementation follows design.md for Slice A.

---

## Remaining Tasks (Slice A)

None — all Slice A tasks complete.

---

## Workload / PR Boundary

| Metric | Value |
|--------|-------|
| Lines changed (Slice A) | ~214 total (~36 production, ~178 tests) |
| Test coverage added | 5 new test functions |
| PR scope | `packages/coding-agent/codemap/indexer/` only |
| Gate | `go test ./...` passes |

**Slice A complete. Ready to hand off to orchestrator for Slice B preparation.**