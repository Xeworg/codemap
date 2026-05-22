# Proposal: codemap-impact-edges-v1

## Intent
Improve `codemap impact` result quality by expanding index-time edge extraction beyond calls, so risk-tiered findings (`high`/`medium`/`low`) reflect real usage patterns instead of mostly `calls`-only data.

## Scope
### In scope
1. Extend file-local AST edge extraction in `indexer/edges.go` for `type_use`, `imports`, and `references`.
2. Include `casts` edge extraction only if the slice remains within review budget.
3. Keep existing store schema and impact CLI derivation logic unchanged.
4. Add/adjust tests to validate new edge kinds and impact tier diversity.
5. Update docs/changelog to reflect behavior change.

### Out of scope
- Cross-file or whole-program SSA/type-checking analysis.
- Reflection/interface-perfect resolution.
- Envelope/sorting/cap changes in `impact` command.
- Schema migrations.

## Affected areas
- `packages/coding-agent/codemap/indexer/edges.go`
- `packages/coding-agent/codemap/indexer/edges_test.go`
- `packages/coding-agent/codemap/cli/impact_cmd_test.go`
- `docs/codemap-cli-json-contract.md` (if examples/tables need edge-type refresh)
- `CHANGELOG.md`
- `openspec/changes/codemap-impact-edges-v1/*` (phase artifacts tracking)

## Review workload estimate
- Expected: **300–600 changed lines**.
- 400-line budget risk: **medium**.

### Split recommendation
- If implementation+tests stay **<=400** lines: single PR.
- If `references` + `casts` pushes **>400**: split into stacked delivery:
  1. PR A: `type_use` + `imports` + tests
  2. PR B: `references` (+ `casts` if included) + tests/docs

## Risks and mitigations
1. **False-positive reference edges** from unresolved identifiers  
   Mitigation: fail-soft; skip unresolved refs.
2. **Complexity growth in AST walk**  
   Mitigation: add one edge kind at a time with focused tests.
3. **Determinism drift**  
   Mitigation: preserve deterministic ordering and keep determinism tests green.
4. **Limited precision ceiling without type-checker**  
   Mitigation: keep explicit non-goal and classify uncertainly where needed.

## Rollback plan
- Revert newly added non-`call` edge emitters independently.
- Keep `calls` path intact so impact remains functional at current baseline.
- Roll back docs/changelog entries together with behavior rollback.
- If partial rollout is needed, retain PR A (`type_use` + `imports`) and defer PR B (`references`/`casts`).

## Success criteria
1. `impact` findings include non-`call` edge-driven entries (`medium`/`low` tiers visible in tests).
2. Existing `impact` and `deadcode` contracts remain stable (no envelope/schema regressions).
3. `go test ./...` passes, including determinism-related tests.
4. No DB schema changes required.

## Rollout approach
1. Implement `type_use` + `imports` first and validate budget/test impact.
2. Add `references` next; include `casts` only if still within budget.
3. If budget exceeds 400 lines, switch to stacked PR split (A then B).

## Skill resolution
- `skill_resolution: paths-injected`

## Proposal Addendum (pass 2)

### Narrowed phase-1 delivery
Phase-1 should ship only:
- `type_use`
- `imports`
- corresponding unit/integration tests

This creates measurable impact-tier diversity quickly while keeping review load below threshold.

### Phase-2 conditional delivery
Phase-2 adds:
- `references`
- optional `casts`

Gate condition: proceed only if Phase-1 post-merge test signal is stable and expected diff size for Phase-2 remains reviewable.

### Additional acceptance checks
1. At least one integration test must assert presence of a non-`high` risk finding in impact output.
2. No regression in deadcode classifier tests (edge expansion must not destabilize deadcode flows).
3. Determinism holds across repeated impact command runs on same fixture.

### Updated split policy
- Prefer chained PRs by default for this change (`A: type_use+imports`, `B: references+casts`).
- Collapse into one PR only if final diff remains clearly under 400 LOC with passing tests and clean reviewer scope.
