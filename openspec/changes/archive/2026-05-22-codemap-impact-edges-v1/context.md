# Explore Brief: codemap-impact-edges-v1

## Problem
`codemap impact` is a vNext pillar but currently surfaces only `calls` edges because the indexer `EdgeExtractor` emits **only** `*ast.CallExpr` edges. The `impact` command already derives risk tiers (`high`/`medium`/`low`) and confidence from edge types, yet every non-`call` edge kind (`type_use`, `references`, `imports`, `casts`) is dead code in the derivation logic — no data ever arrives. Users see monotonically `high`-risk findings and cannot prioritize review effectively.

## Goals
1. Extend `EdgeExtractor.ExtractEdges()` to emit **at least** `type_use`, `imports`, and `references` edges from file-local AST inspection.
2. Keep the existing store schema and CLI derivation logic unchanged (they already handle these edge types).
3. Ensure `impact` findings populate `medium` and `low` risk tiers deterministically.
4. Maintain fail-soft parsing and bounded AST walk (no cross-file SSA).

## Non-Goals
- Full whole-program static analysis / SSA call graph.
- Perfect interface / reflection / build-tag resolution.
- New store schema migrations (edges table already supports arbitrary `edge_type`).
- Changes to `impact` sorting, cap, or envelope structure (already implemented).
- Auto-delete or refactor execution.

## Candidate Edge Kinds
| Kind | AST Source | Risk Tier (impact.go) | Confidence Rule |
|---|---|---|---|
| `calls` | `*ast.CallExpr` | high | already emitted |
| `type_use` | `*ast.TypeSpec`, field types, param/return types | medium | high for func/type/interface |
| `references` | `*ast.Ident` in value positions referencing a var/const | medium | medium for var/const |
| `imports` | `*ast.ImportSpec` → package-level symbol usage | medium | medium |
| `casts` | `*ast.TypeAssertExpr` | medium | medium |

**Scope boundary:** file-local AST only. Cross-file package resolution and type checking are out of scope.

## Affected Files
1. `packages/coding-agent/codemap/indexer/edges.go` — extend `ExtractEdges()` with new AST node walkers.
2. `packages/coding-agent/codemap/indexer/edges_test.go` — add unit tests for each new edge kind.
3. `packages/coding-agent/codemap/cli/impact_cmd_test.go` — add integration tests that verify medium/low risk tiers appear after indexing.
4. `docs/codemap-cli-json-contract.md` — update edge-type tables if examples are added.
5. `CHANGELOG.md` — note impact edge expansion under Unreleased.

## Test Strategy
- **Unit:** `edges_test.go` — per-edge-kind fixture tests (type_use, imports, references, casts) with concrete Go source.
- **Integration:** `impact_cmd_test.go` — index a fixture repo, query impact on a symbol, assert findings include `medium`/`low` tiers.
- **Determinism:** existing `TestImpactDeterminism` must still pass.
- **Regression:** run `go test ./...` after each edge kind is added; no store or CLI contract changes should break existing tests.
- **Performance:** `perf_guardrail_test.go` should not regress (bounded AST walk).

## Risks
| Risk | Mitigation |
|---|---|
| AST walk complexity balloons | Add one node type at a time; stop if >400 changed lines |
| False-positive edges from unresolved idents | Skip unresolved refs (same fail-soft as `calls`) |
| Determinism drift from map iteration | No new map iteration in edge emission; keep stable sort in CLI |
| Impact KPI miss (<70% relevant high-risk) | Medium/low edges improve breadth, not degrade precision; capped at 50 findings |

## Estimated Review Size
- **300–600 changed lines** (indexer + tests + docs).
- Under the 400-line budget: **likely safe as a single PR** if scoped to `type_use` + `imports` first.
- If `references` + `casts` push over 400, split into two stacked PRs:
  1. `type_use` + `imports`
  2. `references` + `casts`

## Start Here
Open `packages/coding-agent/codemap/indexer/edges.go`. The `ExtractEdges()` method currently handles only `*ast.CallExpr`. Add handlers for `*ast.TypeSpec`, `*ast.ImportSpec`, and `*ast.ValueSpec` (for type references) as the highest-value bounded next step.

## Skill resolution
- `skill_resolution: paths-injected`

## Explore Addendum (pass 2)

### Prioritization update
- Prioritize edge kinds in this exact order: `type_use` → `imports` → `references` → `casts`.
- Reason: first two are lower ambiguity and highest value for unlocking medium-tier impact findings with minimal false positives.

### Ambiguity boundaries (explicit)
- `references`: only emit when identifier resolves to a known symbol in the same indexed file scope map.
- Do **not** emit `references` for language keywords, package aliases, or unresolved identifiers.
- `casts`: include type assertions (`x.(T)`) only when `T` resolves to an indexed symbol key.

### Fixture addendum
Add focused fixtures in `indexer/edges_test.go` for:
1. struct field type references (`type_use`)
2. function param/return type references (`type_use`)
3. aliased import usage (`imports`) with one positive and one ignored unresolved case
4. local var/const symbol reference (`references`) positive/negative pair
5. type assertion (`casts`) positive/negative pair

### Review-size refinement
- Base slice (`type_use` + `imports` + tests): ~180–280 LOC.
- Extended slice (`references` + `casts` + tests): +140–260 LOC.
- Combined likely range: ~320–540 LOC.
- Decision rule: if diff crosses 400, split into chained PRs immediately.
