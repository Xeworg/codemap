# Tasks: codemap-deadcode-precision-v1

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 800–1100 (total across slices) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | Slice A → Slice B → Slice C → Slice D |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

```text
Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium
```

**Rationale:** Four slices map cleanly to four PRs. Each slice touches distinct packages and has clear start/finish/verification boundaries. Stacked-to-main avoids long-lived feature branches and keeps integration pressure minimal. If slices A–C land cleanly, slice D (docs/regression fixtures) can be fast-tracked. Explicit decision gate before apply because the total line count may exceed 400 lines in some slices depending on test fixture volume.

---

## Slice A: Symbol coverage foundation

**Scope:** `packages/coding-agent/codemap/indexer/`

**Target:** Add methods + `init` to `ExtractGoSymbols`, update `Symbol` model, add receiver-qualified naming.

### A1 — Expand Symbol model for methods

**File:** `packages/coding-agent/codemap/indexer/parse_result.go`

Add `Recv` and `File` fields to `Symbol` so the classifier can reason about receiver context and file provenance:

```go
type Symbol struct {
    Name      string
    Kind      string // "func", "type", "var", "const", "method"
    Signature string
    Recv      string // receiver type name, e.g. "T" or "*T", empty if top-level func
    File      string
    StartLine int
    EndLine   int
}
```

**Verification:** All existing `go_parser_test.go` tests pass (update field assignments in the file builder where needed).

### A2 — Update symbolFromFuncDecl to handle methods

**File:** `packages/coding-agent/codemap/indexer/go_parser.go`

- In the `*ast.FuncDecl` case, **remove** the `if d.Recv == nil` guard.
- For top-level funcs (`d.Recv == nil`): set `Kind = "func"`, `Recv = ""`.
- For methods (`d.Recv != nil`): set `Kind = "method"` and `Recv = receiverName(d.Recv)`.
- Handle `init` as `Kind = "func"`, `Name = "init"`, no special `Recv`.

Receiver name: for a `*ast.Ident` receiver, use the identifier; for a `*ast.StarExpr`, prefix with `*`. This produces names like `T`, `*T`.

### A3 — Add unit tests for method and init extraction

**File:** `packages/coding-agent/codemap/indexer/go_parser_test.go`

Add `TestExtractGoSymbols_IncludesMethods`, `TestExtractGoSymbols_IncludesInit`, `TestExtractGoSymbols_MethodWithPointerReceiver`, `TestExtractGoSymbols_MultipleInitFuncs`.

Each test:
- Has concrete Go source with methods and/or `init`.
- Asserts correct `Kind`, `Recv`, and `Name` fields.
- Verifies no duplicate symbols, no regression on existing top-level funcs.

### A4 — Boundary test: receiver name edge cases

**File:** `packages/coding-agent/codemap/indexer/go_parser_test.go`

`TestExtractGoSymbols_ReceiverNamingVariants`:
- `(T)` → `Recv = "T"`
- `(*T)` → `Recv = "*T"`
- `(pkg.T)` → `Recv = "T"` (strip package prefix at AST level; full qualification is out of scope for v1)

---

## Slice B: Edge extraction + persistence

**Scope:** `packages/coding-agent/codemap/indexer/` + `packages/coding-agent/codemap/store/`

**Target:** AST-based call-edge extraction, store helpers, index transaction wiring.

### B1 — Define Edge types and extractor interface

**File:** `packages/coding-agent/codemap/indexer/edges.go` (new)

```go
// SymbolKey uniquely identifies a symbol within a file for edge resolution.
type SymbolKey struct {
    File  string // source file (for disambiguation)
    Name  string
    Recv  string // empty for top-level, receiver-qualified for methods
}

// EdgeIntent represents a single resolved call/reference edge between two symbols.
type EdgeIntent struct {
    From SymbolKey
    To   SymbolKey
    Kind string // "call", "ref", "type_use"
}

// EdgeExtractor processes a single parsed file and emits EdgeIntents.
type EdgeExtractor struct {
    file    *ast.File
    fset    *token.FileSet
    syms    map[string]SymbolKey // name -> key (file-local resolver)
}

// NewEdgeExtractor(file, fset, symbols) → *EdgeExtractor
// ExtractEdges() → []EdgeIntent
```

Scope: only `calls` (AST `*ast.CallExpr` callee resolution). `ref` and `type_use` are deferred to v1.1.

### B2 — SymbolKey map construction in EdgeExtractor

**File:** `packages/coding-agent/codemap/indexer/edges.go`

- Walk `f.Decls` to build `syms` map: `Name` → `SymbolKey{File, Name, Recv}`.
- Key by simple `Name` for top-level symbols; `Recv + "." + Name` for methods.
- Fail-soft: if a name collides, prefer the first seen (deterministic by AST order).

### B3 — Call resolution in ExtractEdges

**File:** `packages/coding-agent/codemap/indexer/edges.go`

- Walk `*ast.CallExpr` nodes in the file AST.
- Resolve callee: `*ast.Ident.Name` → lookup in `syms`; `*ast.SelectorExpr` → lookup `X.Name + "." + Sel.Name`.
- If resolved, emit `EdgeIntent{From: callSite, To: target, Kind: "call"}`.
- Track source position via `fset.Position(node.Pos())` for debugging/logging (no edge attribute needed).

### B4 — Return EdgeIntents from indexer pipeline

**File:** `packages/coding-agent/codemap/indexer/` (update relevant files)

- `ParseResult` gains `Edges []EdgeIntent`.
- `ParseGoFile` calls `NewEdgeExtractor` + `ExtractEdges` and appends to result.
- `FileEntry` in `indexer.go` (or wherever it's defined) gains `Edges []EdgeIntent` so the indexer pass can propagate them to `RunIndex` result.
- `RunIndex` result gains `EdgesFound int` counter.

### B5 — Store helper: batch edge upsert

**File:** `packages/coding-agent/codemap/store/edges.go`

Add:
```go
// UpsertEdges persists a batch of resolved edges, given symbol ID maps.
// Fails soft: logs unresolved keys, skips edge insert.
func UpsertEdges(ctx context.Context, db *sql.Tx, resolvedEdges []ResolvedEdge) error
```

`ResolvedEdge` holds `FromSymbolID, ToSymbolID, EdgeType`.

### B6 — Update index.go transaction wiring

**File:** `packages/coding-agent/codemap/cli/index.go`

- After `ReplaceFileSymbols`, build `name→symbolID` map for the file's new symbols.
- Convert each `EdgeIntent` to a `ResolvedEdge` by resolving `From` and `To` via the name map.
- Call `store.UpsertEdges` with resolved edges.
- Log/skipped count for unresolved edges (no error).

### B7 — Integration test: method call creates inbound edge

**File:** `packages/coding-agent/codemap/indexer/edges_test.go` (new)

Fixture:
```go
package test

type T struct{}

func (t T) Method() {}

func Caller() {
    var x T
    x.Method()
}
```

- Parse file, extract edges.
- Assert exactly one edge: `Caller → T.Method`, `Kind = "call"`.
- Assert `T.Method` has `Recv = "T"`.

### B8 — Integration test: reindex replaces edges correctly

**File:** `packages/coding-agent/codemap/store/edges_test.go` (new)

- Insert file + symbols + edges in snapshot 1.
- Reindex same file (different hash) in snapshot 2.
- Verify old edges are gone, new edges exist (via `GetInboundEdges`).

---

## Slice C: Deadcode precision classifier

**Scope:** `packages/coding-agent/codemap/cli/` + `packages/coding-agent/codemap/store/`

**Target:** Inbound-aware deadcode classification, evidence tiers, heuristic entrypoint detection.

### C1 — Store helper: symbols with inbound counts

**File:** `packages/coding-agent/codemap/store/edges.go`

Add:
```go
// GetAllSymbolsWithInboundCounts returns all symbols from latest snapshot
// with their inbound edge counts pre-computed.
func GetAllSymbolsWithInboundCounts(ctx context.Context, db *sql.DB) ([]SymbolWithInbound, error)

// SymbolWithInbound holds a symbol row plus its computed inbound edge count.
type SymbolWithInbound struct {
    SymbolRow
    InboundCount int
}
```

### C2 — Heuristic entrypoint predicates

**File:** `packages/coding-agent/codemap/cli/deadcode.go`

Add as unexported helpers:
```go
// isRuntimeEntrypoint returns true for main/init entrypoints.
func isRuntimeEntrypoint(name, file string) bool

// isPublicAPI returns true for exported symbols that may be used externally.
// Heuristic: name starts with uppercase letter.
func isPublicAPI(name string) bool

// isEntrypointFile returns true for files matching cmd/ entrypoint patterns.
func isEntrypointFile(file string) bool
```

### C3 — Refactor classifyDeadcode with evidence tiers

**File:** `packages/coding-agent/codemap/cli/deadcode.go`

Replace `classifyDeadcode` with:
```go
func classifyDeadcode(inbound int, kind, name, file string) (classification, suggestion, confidence string)
```

- `inbound > 0`: `classification = "uncertain"`, `suggestion = "review"`, `confidence = "low"`, evidence: `inbound_edges`
- `inbound == 0` + `isRuntimeEntrypoint(name, file)`: `classification = "uncertain"`, `confidence = "low"`, evidence: `implicit_runtime_entry`
- `inbound == 0` + `isPublicAPI(name)`: `classification = "uncertain"`, `confidence = "low"`, evidence: `public_api_surface`
- `inbound == 0` + no heuristics: `classification = "unused"`, `suggestion = "remove"`, confidence by kind (func/type → high, var/const → medium, method → medium).

Build evidence slice composably from applicable heuristic matches.

### C4 — Evidence type constants

**File:** `packages/coding-agent/codemap/cli/envelope.go` (or `deadcode.go`)

Add evidence type constants:
```go
const (
    EvidenceNoInboundEdges    = "no_inbound_edges"
    EvidenceInboundEdges      = "inbound_edges"
    EvidenceImplicitRuntime   = "implicit_runtime_entry"
    EvidencePublicAPISurface  = "public_api_surface"
)
```

### C5 — Update deadcode.go to use inbound-aware query

**File:** `packages/coding-agent/codemap/cli/deadcode.go`

- Replace `GetSymbolsWithZeroInboundEdges` + per-symbol edge count with `GetAllSymbolsWithInboundCounts`.
- Compute evidence slice from heuristic predicates.
- Remove per-symbol `GetSymbolEdges` call (single query replaces it).

### C6 — Deadcode unit tests

**File:** `packages/coding-agent/codemap/cli/deadcode_cmd_test.go` (or `deadcode_test.go`)

Add test cases:
- `TestClassify_WithInboundEdges_ClassifiesUncertain`: symbol with `inbound > 0` → `classification != "unused"`.
- `TestClassify_MainFunc_NoEdges_ClassifiesUncertain`: `func main()` with no stored edges → `classification = "uncertain"`, evidence includes `implicit_runtime_entry`.
- `TestClassify_InitFunc_NoEdges_ClassifiesUncertain`: `func init()` → `uncertain`.
- `TestClassify_ExportedNoEdges_Uncertain`: exported func with no edges → `uncertain`, evidence includes `public_api_surface`.
- `TestClassify_PrivateFuncNoEdges_Unused`: unexported func, no edges → `unused`.
- `TestClassify_MethodNoEdges_UnusedOrUncertain`: method with no edges → NOT `unused` with high confidence (defer to method owner analysis).
- `TestEvidence_Composable`: verify multiple evidence entries can coexist for one finding.

### C7 — Determinism regression test

**File:** `packages/coding-agent/codemap/cli/deadcode_test.go` (new)

Run `classifyDeadcode` on a fixed input tuple 10×. Assert identical `classification`, `suggestion`, `confidence`, and evidence slice on every run.

---

## Slice D: Precision regression corpus + docs

**Scope:** `packages/coding-agent/codemap/` (fixtures, docs)

**Target:** Curated precision test fixtures, operational documentation, changelog note.

### D1 — Precision regression fixture

**File:** `packages/coding-agent/codemap/testdata/deadcode-precision/` (new)

Create fixture Go package with:
- Exported API (`func ExportedHelper()`) — should be uncertain even if no local edges.
- `main` function — must be uncertain.
- `init` function(s) — must be uncertain.
- Private unused function (`func privateUnused()`) — should be unused.
- Private used function (`func privateUsed()` called by `main`) — should be uncertain.
- Method with no external callers — should be uncertain (method has owner context).

Add `deadcode_precision_test.go` that runs `codemap index` + `codemap deadcode --json` on the fixture and asserts:
- `ExportedHelper` is NOT classified `unused` with high confidence.
- `main` is NOT classified `unused`.
- `privateUnused` IS classified `unused` or `likely-unused`.

### D2 — Update CLI operational docs

**File:** `packages/coding-agent/codemap/README.md` or `docs/deadcode.md` (add if missing)

Document:
- The three evidence tiers and what they mean for the reviewer.
- The heuristic boundaries (runtime entrypoints, public API surface).
- The guaranteed safe actions (only `unused` + high confidence → suggestion `remove`).
- Any known limitations or precision caveats.

### D3 — Changelog entry

**File:** `CHANGELOG.md` or `packages/coding-agent/codemap/CHANGELOG.md`

Add entry under the unreleased section:
```markdown
## [Unreleased]

### Changed

- **deadcode precision v1**: `codemap deadcode` now uses explicit symbol
  edges (calls resolved at index time) plus heuristic entrypoint detection.
  Exported functions and runtime entrypoints (`main`, `init`) are classified
  `uncertain` when no explicit inbound edges are found, reducing false-positive
  rate for commonly-used patterns. Symbol coverage expanded to include methods
  and `init` functions.
```

---

## Dependency ordering

```
A1 → A2 → A3 → A4  (Slice A: symbol model + parser tests)
         ↓
         B1 → B2 → B3 → B4 → B5 → B6 → B7 → B8  (Slice B: edges)
                  ↓
                  C1 → C2 → C3 → C4 → C5 → C6 → C7  (Slice C: classifier)
                           ↓
                           D1 → D2 → D3  (Slice D: fixtures + docs)
```

**Gate:** `go test ./...` must pass at the end of each slice before the next begins.

**Rollback boundary per slice:** If `go test ./...` fails after completing a slice, revert to the last commit on the previous slice and re-evaluate the failing test scope.

---

## Risks and mitigation

| Risk | Mitigation |
|------|-----------|
| Edge resolution undercount (partial AST resolution) | Classify heuristic-only cases as `uncertain`; never `unused` + high confidence |
| Performance regression from edge extraction walk | Bound AST walk to current file; no cross-file SSA; add perf guardrail test in Slice B |
| Edge cleanup regression on reindex | `ReplaceFileSymbols` already deletes edges; `UpsertEdges` uses `INSERT OR IGNORE`; verify with B8 integration test |
| Determinism drift from non-deterministic map iteration | Use `sort.Slice` on symbol slice before edge resolution; enforce via C7 test |
| Classifier heuristic over-breadening | Review `isPublicAPI` scope — only uppercase start, no package-level heuristics in v1 |