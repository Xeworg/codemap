# Tasks — codemap-critical-prd-gaps-v1

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~520–700 (7 files new, 2 modified, tests) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR1: migrate | PR2: impact | PR3: query |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

```
Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High
```

---

## PR1 — `codemap migrate`

**Goal:** Expose `store.NewMigrationRunner(db).Migrate(ctx)` as `codemap migrate`. Idempotent. No new DB state required.

### Task 1.1 — RED: tests for `codemap migrate` exit codes and envelope

**Files:** `packages/coding-agent/codemap/cli/migrate_cmd_test.go` (new)

Cover:
- `migrate --db <empty-db>` → exit `0`, envelope has `ok:true`, `schema_version:"1.0"`, `data.migrations_applied` > 0.
- `migrate --db <db-with-migrations-applied>` → exit `0`, `data.migrations_applied == 0`, idempotent.
- `migrate --db <unreadable-path>` → exit `1`.
- `--db` flag omitted and repo has no cache dir → exit `0` (uses default, creates dir).
- `schema_version == "1.0"` in envelope top-level.
- `command == "migrate"` in envelope.

Use `MustTempDB(t)` from store test helpers; call `store.Migrate(ctx, db)` before each test to seed a clean state.

### Task 1.2 — GREEN: `packages/coding-agent/codemap/cli/migrate.go`

Implement `RunMigrate(ctx context.Context, w io.Writer, args []string, repoRoot string) int`:

```
flag parsing:
  --db string  (optional, default "")
  -h / --help  → help then exit 0

validation:
  if --db missing → use ResolveDBPath("", repoRoot)
  if --db path unreadable → write error envelope, return 1

open DB:
  if Open fails → write error envelope, return 1

run:
  version_before, _ := NewMigrationRunner(db).CurrentSchemaVersion(ctx)
  if err := NewMigrationRunner(db).Migrate(ctx); err != nil → write error, return 1
  version_after, _ := NewMigrationRunner(db).CurrentSchemaVersion(ctx)

report:
  applied := version_before == "none" || version_after != version_before
  migrateData := MigrateData{
    MigrationsApplied: applied,
    VersionBefore:     version_before,
    VersionAfter:      version_after,
  }
  meta := EmptyMeta()  // no snapshot needed for migrate
  emit envelope, return 0
```

Exit codes: `2` on flag parse failure, `1` on DB/runtime failure, `0` on success.

### Task 1.3 — GREEN: add `MigrateData` struct to `envelope.go`

```go
type MigrateData struct {
  MigrationsApplied bool   `json:"migrations_applied"`
  VersionBefore     string `json:"version_before"`
  VersionAfter      string `json:"version_after"`
}
```

Ensure field order is stable (alphabetical in struct tags is fine; Go struct field order is deterministic).

### Task 1.4 — RED: integrate migrate into `main.go`

**File:** `cmd/codemap/main.go`

Add to `switch cmd`:
```go
case "migrate":
    exitCode = runWithHelp(ctx, stdout, stderr, "migrate", subargs, repoRoot,
        func() int { return cli.RunMigrate(ctx, stdout, subargs, repoRoot) })
```

Add help text in `helpFor("migrate", w)`:
```
Usage: codemap migrate [flags]
Run pending schema migrations on the database.

Flags:
  -db path    Path to SQLite database (optional; default: ~/.cache/codemap/<hash>.db)

Example:
  codemap migrate -db myrepo.db
  codemap migrate               # uses default cache path
```

### Task 1.5 — GREEN: cross-command compatibility test

**File:** `packages/coding-agent/codemap/cli/integration_test.go` (extend)

Add `TestMigrateEnvelopeShape`:
- Run `RunMigrate` against a temp DB.
- Unmarshal JSON → check top-level `schema_version`, `ok`, `command`, `errors == []`.
- Verify `data.version_after` is non-empty string.

---

## PR2 — `codemap impact --json`

**Goal:** Return envelope with `affected_symbols` + `evidence` sorted deterministically. Requires indexed state.

### Task 2.1 — RED: tests for `codemap impact` exit codes and determinism

**File:** `packages/coding-agent/codemap/cli/impact_cmd_test.go` (new)

Cover:
- `impact MySymbol` with no index → exit `3`, envelope `ok:false`, error in `errors[]`.
- `impact` with missing symbol arg → exit `2`, `errors` contains "symbol name required".
- `impact --db <path>` with valid DB + indexed state → exit `0`, envelope `ok:true`.
- `schema_version == "1.0"`, `command == "impact"`.
- `data.affected_symbols` is a JSON array (sorted by name → file).
- `data.evidence` is a JSON array.
- **Determinism**: call `RunImpact` twice with same DB → compare JSON bytes → must be identical.
- **No snapshot guard**: if `GetLatestSnapshotMeta` returns `SnapshotID == 0` → exit `3`.

Setup: use `MustTempDB(t)` + `store.Migrate` + `store.BeginSnapshot` + insert a symbol + an edge → commit.

### Task 2.2 — GREEN: `packages/coding-agent/codemap/cli/impact.go`

Implement `RunImpact(ctx context.Context, w io.Writer, args []string, repoRoot string) int`:

```
flag parsing: --db, -h/--help
validation: symbolArg required → exit 2 on empty
db open → exit 1 on failure
meta := GetLatestSnapshotMeta → exit 3 if SnapshotID==0

sym := GetSymbolByName → exit 3 if nil
edges := GetSymbolEdges(ctx, db, sym.ID)  // existing store function
related := collect all related symbol IDs from edges

Build stable set:
  affected := []string{}
  for _, e := range edges {
    rsym := GetSymbolByID(e.RelatedSymbolID) → append rsym.Name
  }
  sort.Strings(affected)  // deterministic

evidence := []EvidenceEntry{}
for _, e := range edges {
  evidence = append(evidence, EvidenceEntry{
    Type:        "symbol_link",
    Description: fmt.Sprintf("linked via %s (strength: %s)", e.LinkType, e.LinkStrength),
    Source:      rsym.File,
  })
}
sort.SliceStable(evidence, func(i,j) bool { return evidence[i].Description < evidence[j].Description })

data := ImpactData{
  TargetSymbol: sym.Name,
  AffectedSymbols: affected,
  Evidence: evidence,
}

envelope with meta (snapshot, head_ref, indexed_at, is_stale)
return 0
```

Exit codes: `2` flag/input, `1` runtime, `3` no-index/no-symbol.

### Task 2.3 — GREEN: add `ImpactData` struct to `envelope.go`

```go
type ImpactData struct {
  TargetSymbol     string          `json:"target_symbol"`
  AffectedSymbols  []string        `json:"affected_symbols"`
  Evidence         []EvidenceEntry `json:"evidence"`
}
```

### Task 2.4 — RED: integrate impact into `main.go`

**File:** `cmd/codemap/main.go`

Add case:
```go
case "impact":
    exitCode = runWithHelp(ctx, stdout, stderr, "impact", subargs, repoRoot,
        func() int { return cli.RunImpact(ctx, stdout, subargs, repoRoot) })
```

Add help in `helpFor("impact", w)`:
```
Usage: codemap impact [flags] <symbol>

Show symbols that depend on or relate to the given symbol, with evidence.

Flags:
  -db path    Path to SQLite database (optional; default: ~/.cache/codemap/<hash>.db)

Example:
  codemap impact MyFunction
  codemap impact --db myrepo.db MyFunction
```

### Task 2.5 — RED: verify `GetSymbolEdges` exists in store

**File:** `packages/coding-agent/codemap/store/*.go`

If `GetSymbolEdges` does not exist, implement it:
```go
func GetSymbolEdges(ctx context.Context, db *sql.DB, symbolID int64) ([]SymbolEdge, error)
```
- Query `edges` table WHERE `from_symbol_id=? OR to_symbol_id=?`
- Return `SymbolEdge{FromSymbolID, ToSymbolID, LinkType, LinkStrength}` rows.

If `SymbolEdge` type does not exist, define it in `store/models.go` or `store/edges.go`.

### Task 2.6 — GREEN: cross-command compatibility test

**File:** `packages/coding-agent/codemap/cli/integration_test.go` (extend)

Add `TestImpactEnvelopeShapeAndDeterminism`:
- Index a repo with symbols + edges.
- Run `RunImpact` twice → JSON bytes identical.
- Unmarshal → check `data.affected_symbols` is sorted, `data.evidence` count matches edge count.

---

## PR3 — `codemap query --json`

**Goal:** Deterministic symbol/history lookup returning envelope with `QueryData`. Falls back to prefix match on exact miss.

### Task 3.1 — RED: tests for `codemap query` exit codes and determinism

**File:** `packages/coding-agent/codemap/cli/query_cmd_test.go` (new)

Cover:
- `query MyFunc` with no index → exit `3`.
- `query` missing arg → exit `2`, error contains "query term required".
- `query --db <valid-db>` with indexed state → exit `0`, envelope `ok:true`.
- `schema_version == "1.0"`, `command == "query"`.
- `data.matches` is a JSON array of objects with `name`, `kind`, `file`, `signature`.
- **Exact-first**: query with exact symbol name returns that symbol first.
- **Prefix fallback**: query with partial name returns all symbols whose name starts with term (sorted by name).
- **Determinism**: call `RunQuery` twice with same args → identical JSON bytes.
- Longest-prefix-first ordering when prefix fallback returns multiple results.

Setup: use `MustTempDB(t)` + migrate + beginSnapshot + insert multiple symbols (e.g., `Foo`, `FooBar`, `FooBaz`) + commit.

### Task 3.2 — GREEN: `packages/coding-agent/codemap/cli/query.go`

Implement `RunQuery(ctx context.Context, w io.Writer, args []string, repoRoot string) int`:

```
flag parsing: --db, --json (accepted, ignored, JSON default), -h/--help
validation: query term required → exit 2 on empty
db open → exit 1 on failure
meta := GetLatestSnapshotMeta → exit 3 if SnapshotID==0

exact := GetSymbolByName(term)
var matches []QueryMatch
if exact != nil {
    matches = append(matches, QueryMatch{Name: exact.Name, Kind: exact.Kind, File: exact.File, Signature: exact.Signature})
}

// prefix fallback
prefix := term + "%"
allSymbols := GetAllSymbols(ctx, db)  // new helper: SELECT * FROM symbols
for _, s := range allSymbols {
    if strings.HasPrefix(s.Name, term) && s.Name != term {
        matches = append(matches, QueryMatch{Name: s.Name, Kind: s.Kind, File: s.File, Signature: s.Signature})
    }
}

// deterministic sort: by name, then file
sort.Slice(matches, func(i,j int) bool {
    if matches[i].Name != matches[j].Name {
        return matches[i].Name < matches[j].Name
    }
    return matches[i].File < matches[j].File
})

data := QueryData{
    Query:    term,
    Matches:  matches,
    Count:    len(matches),
}

envelope with meta → return 0
```

Exit codes: `2` input, `1` runtime, `3` no-index.

### Task 3.3 — GREEN: add `QueryData` struct + `QueryMatch` to `envelope.go`

```go
type QueryData struct {
    Query   string        `json:"query"`
    Matches []QueryMatch  `json:"matches"`
    Count   int           `json:"count"`
}

type QueryMatch struct {
    Name      string `json:"name"`
    Kind      string `json:"kind"`
    File      string `json:"file"`
    Signature string `json:"signature,omitempty"`
}
```

### Task 3.4 — RED: implement `GetAllSymbols` in store (if not present)

**File:** `packages/coding-agent/codemap/store/symbols.go` or new file

```go
func GetAllSymbols(ctx context.Context, db *sql.DB) ([]Symbol, error)
```
- `SELECT * FROM symbols ORDER BY name, file` (deterministic sort at DB level).
- Return `[]Symbol` or error.

### Task 3.5 — RED: integrate query into `main.go`

**File:** `cmd/codemap/main.go`

Add case:
```go
case "query":
    exitCode = runWithHelp(ctx, stdout, stderr, "query", subargs, repoRoot,
        func() int { return cli.RunQuery(ctx, stdout, subargs, repoRoot) })
```

Add help in `helpFor("query", w)`:
```
Usage: codemap query [flags] <term>

Look up symbols by exact name or prefix. Returns deterministic JSON.

Flags:
  -db path    Path to SQLite database (optional; default: ~/.cache/codemap/<hash>.db)
  --json      Output JSON envelope (default, implicit)

Example:
  codemap query MyFunction
  codemap query --db myrepo.db Foo
```

### Task 3.6 — RED: determinism regression test

**File:** `packages/coding-agent/codemap/cli/integration_test.go` (extend)

Add `TestQueryDeterminismMultipleSymbols`:
- Index a repo with symbols `A0`, `A1`, `A2`, `B0` etc.
- Run `RunQuery` 3 times with same args → compare JSON bytes → must be identical.
- Check `data.matches` is sorted by name, then file.

Add `TestQueryPrefixFallbackOrdering`:
- Insert symbols: `Foo`, `FooBar`, `FooBaz`, `Food`, `Fool`.
- Run `query Foo` → verify `matches[0].name == "Foo"` (exact first).
- Verify `FooBar`, `FooBaz`, `Food`, `Fool` follow in sorted order.
- Verify total count = 5.

---

## Shared deliverables (all PRs)

### Task S.1 — Envelope compatibility across all commands

After each PR, verify all three commands (`index`, `symbol`, `history` — existing) and new commands all produce:
- `schema_version == "1.0"` (top-level string, not null).
- `ok` boolean present.
- `errors` array (may be empty, never null in JSON).
- `meta` object with `snapshot_id`, `head_ref`, `indexed_at`, `is_stale`.

Update `integration_test.go` with a table-driven test covering all commands.

### Task S.2 — Exit code consistency matrix

Add to `integration_test.go`:
```
TestExitCodes
  index:   0 (success), 1 (runtime), 2 (flag parse)
  symbol:  0 (success), 1 (runtime), 2 (validation), 3 (no index)
  history: 0 (success), 1 (runtime), 2 (validation), 3 (no index)
  migrate: 0 (success), 1 (runtime), 2 (flag parse)
  impact:  0 (success), 1 (runtime), 2 (validation), 3 (no index)
  query:   0 (success), 1 (runtime), 2 (validation), 3 (no index)
```

### Task S.3 — Update `helpRoot` in `main.go`

After all three PRs, add the new commands to `helpRoot` output:
```
  codemap index       Scan and index a Go repository
  codemap symbol      Query a symbol by name
  codemap history     Query commit history for a symbol
  codemap migrate     Run pending schema migrations
  codemap impact      Show symbols impacted by a given symbol
  codemap query       Look up symbols by name or prefix
```

---

## Implementation order dependency

```
PR1 (migrate)
  └─ Task 1.1 RED → Task 1.2 GREEN → Task 1.3 GREEN → Task 1.4 RED → Task 1.5 GREEN

PR2 (impact) [requires PR1 store layer, uses same patterns]
  └─ Task 2.1 RED → Task 2.5 RED (verify GetSymbolEdges) → Task 2.2 GREEN → Task 2.3 GREEN → Task 2.4 RED → Task 2.6 GREEN

PR3 (query) [requires PR1 & PR2 infrastructure]
  └─ Task 3.1 RED → Task 3.4 RED (GetAllSymbols) → Task 3.2 GREEN → Task 3.3 GREEN → Task 3.5 RED → Task 3.6 RED

Shared: S.1 → S.2 → S.3 (after PR3)
```

---

## Verification command

After each PR, run:
```bash
go test -count=1 ./packages/coding-agent/codemap/cli/...
go test -count=1 ./packages/coding-agent/codemap/store/...
```

All tests must pass green before merge. Determinism tests must be non-flaky (deterministic DB seed + stable sort).