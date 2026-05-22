# Design: codemap-impact-edges-v1

## Scope and phase plan

This change expands index-time edge extraction for `impact` quality, in two phases:

- **Phase 1 (required):** `type_use` + `imports`
- **Phase 2 (conditional):** `references` + optional `casts`

Out of scope: SSA, cross-file type-checking, reflection-perfect resolution, schema changes.

---

## Current baseline

- `indexer/edges.go` emits only `call` edges from `*ast.CallExpr`.
- `impact` already derives `risk_tier`/`confidence` from edge type, but non-call paths are data-starved.
- Store schema already supports arbitrary `edge_type`; no migration needed.

---

## Architecture changes

### 1) Edge extraction expansion (`packages/coding-agent/codemap/indexer/edges.go`)

Keep `EdgeExtractor` as single-file AST walker and extend `ExtractEdges()` to append additional `EdgeIntent` kinds.

#### Phase 1 emitters

1. **`type_use`**
   - Sources: `*ast.TypeSpec`, struct field types, func param/result types, var declarations with explicit type.
   - Resolution: extract terminal type identifier (`T`, `*T`, `pkg.T` -> `T`) and resolve via `syms` map.
   - Emit: `EdgeIntent{Kind: "type_use"}` when both `From` and `To` resolve.

2. **`imports`**
   - Sources: `*ast.ImportSpec` + usage of imported alias/identifier in selectors.
   - Resolution: alias map from import specs; link selector use to file-local symbol context when resolvable.
   - Emit only resolvable import-driven intents; skip unresolved alias/package cases.

#### Phase 2 emitters

3. **`references`**
   - Sources: value-position `*ast.Ident` reads/writes.
   - Guardrails: exclude declarations, keywords, package aliases, unresolved names.
   - Emit when identifier resolves to known symbol key.

4. **`casts`** (optional)
   - Sources: `*ast.TypeAssertExpr` (and explicit conversion call patterns only if trivial/low-risk).
   - Emit only when asserted/converted type resolves.

### 2) Resolution strategy

- Reuse/extend symbol map building, but keep deterministic first-wins behavior.
- Introduce tiny helper resolvers in `edges.go`:
  - `resolveTypeExpr(ast.Expr) (SymbolKey, bool)`
  - `resolveIdentRef(*ast.Ident) (SymbolKey, bool)`
  - `enclosingSymbol(ast.Node) SymbolKey` (reuse call-site approach for edge source)
- Unresolved candidates are **fail-soft skipped**.

### 3) Determinism guarantees

- Traverse AST in source order (`ast.Inspect` / decl-order loops only).
- Avoid map-iteration-driven edge emission order.
- Keep append order stable by extracting in fixed phase order:
  1) calls, 2) type_use, 3) imports, 4) references, 5) casts.
- If dedup needed, use ordered seen-set keyed by `from|to|kind` while preserving first-seen order.

### 4) No CLI/store contract changes

- `store.UpsertEdges` unchanged.
- `impact` command behavior unchanged structurally; improved tier diversity emerges from richer edge data.
- Envelope schema unchanged.

---

## Data flow

1. Parser builds symbols for file.
2. `EdgeExtractor` emits `[]EdgeIntent` with mixed kinds.
3. Index pipeline resolves intents to symbol IDs and upserts edges.
4. `impact` reads incident edges and derives risk tier/confidence.
5. Non-call edges now enable `medium`/`low` tier findings where applicable.

---

## File-level implementation plan

### Phase 1
- `packages/coding-agent/codemap/indexer/edges.go`
  - Add `type_use` extractor helpers.
  - Add import alias scan + resolvable import edge extraction.
- `packages/coding-agent/codemap/indexer/edges_test.go`
  - Add tests for type_use and imports (positive/negative unresolved).
- `packages/coding-agent/codemap/cli/impact_cmd_test.go`
  - Add integration assertion: presence of at least one non-`high` finding when fixture contains non-call relations.

### Phase 2
- `packages/coding-agent/codemap/indexer/edges.go`
  - Add guarded references extraction.
  - Add optional casts extraction.
- `packages/coding-agent/codemap/indexer/edges_test.go`
  - Add references and casts tests.
- `packages/coding-agent/codemap/cli/impact_cmd_test.go`
  - Extend fixture assertions for tier diversity and determinism.

### Docs/changelog (final step)
- `docs/codemap-cli-json-contract.md` (examples/tables if needed)
- `CHANGELOG.md`

---

## Fail-soft rules (explicit)

Skip (do not error) when:
- target symbol not resolvable from file-local context,
- candidate is ambiguous across unsupported contexts,
- node kind requires cross-file/type-check data.

Never fail `index` due to unresolved non-call candidates.

---

## Test plan boundaries

### Unit (indexer)
- `type_use` extraction from:
  - struct fields,
  - func params/results,
  - var explicit type.
- `imports` extraction:
  - alias import used (positive),
  - unresolved/unused alias (negative).
- Phase 2:
  - `references` resolvable ident (positive) + declaration/unresolved (negative),
  - `casts` resolvable assertion (positive) + unresolved type (negative).

### Integration (CLI impact)
- Index fixture with both call and non-call relations.
- Assert output includes:
  - valid envelope fields,
  - at least one non-`high` tier finding when expected,
  - deterministic ordering across repeated runs.

### Regression
- `go test ./...` pass required after each phase.
- Existing deadcode and determinism tests must remain green.

---

## Rollout and review control

- Default delivery: **stacked PRs**
  - PR A: phase 1 (`type_use` + `imports`)
  - PR B: phase 2 (`references` + optional `casts`)
- If total diff stays clearly <400 lines with clean scope, may collapse.
- Stop/split immediately if review budget risk increases.

---

## Risks and mitigations

1. **False positives in references**
   - Mitigation: strict resolvable-only rule; skip declarations and unresolved ids.
2. **Order nondeterminism**
   - Mitigation: fixed extraction order + stable append path + determinism tests.
3. **Complexity drift**
   - Mitigation: phase gates with explicit LOC/test checkpoints.

---

## Acceptance mapping to spec delta

- ADDED: non-call edge extraction (`type_use`, `imports`, `references`, optional `casts`) -> implemented in extractor + tests.
- ADDED: unresolved non-call candidates fail-soft -> skip logic + negative tests.
- MODIFIED: impact tier diversity supported by indexed edge types -> integration tests asserting non-`high` tiers when present.
- MODIFIED: deterministic ordering under expanded edge types -> repeat-run determinism test coverage.

## Skill resolution
- `skill_resolution: paths-injected`
