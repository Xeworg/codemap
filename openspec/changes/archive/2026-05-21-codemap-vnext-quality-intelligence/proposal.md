# Proposal — codemap-vnext-quality-intelligence

## Intent
Deliver CodeMap vNext quality intelligence so developers can plan safer PRs, diagnose missing symbol/history results quickly, and identify likely dead code with actionable guidance.

## Scope
### In scope
1. Enrich `codemap impact` with risk tiers (`high|medium|low`), confidence, evidence, deterministic ordering, and default result cap.
2. Add explain-not-found behavior for `codemap symbol` and `codemap history` with cause taxonomy (`stale_index|name_mismatch|parse_error|missing_history_links`) and recommended actions in JSON/human output.
3. Add `codemap deadcode` command returning findings with symbol/location/classification (`unused|likely-unused|uncertain`), suggestion (`remove|deprecate|justify`), confidence, and evidence.
4. Update CLI JSON contract docs and Pi skill workflows; complete smoke validation.

### Out of scope
- Mandatory CI deadcode gating.
- Automatic code deletion/refactoring.
- Full whole-program precision guarantees.

## Affected Areas
- `cmd/codemap/main.go` (command routing/help for `deadcode`)
- `packages/coding-agent/codemap/cli/envelope.go` (new explain/deadcode/impact payload structs)
- `packages/coding-agent/codemap/cli/symbol.go`, `history.go`, `impact.go`, new `deadcode.go`
- `packages/coding-agent/codemap/store/edges.go`, `parse_errors.go` (read paths for evidence/causes)
- `docs/codemap-cli-json-contract.md`
- `integrations/pi/skills/codemap-usage/SKILL.md`

## Risks
- False positives in deadcode classification.
- Impact output noise that overloads review.
- Vague explain diagnostics without strict cause/action mapping.
- Contract drift between implementation and docs.

## Rollback Plan
- Keep changes slice-based by phase (contracts → explain → impact → deadcode).
- If a slice regresses behavior, revert the slice commit and preserve prior stable commands/envelope.
- Treat deadcode as report-only; disable command wiring if confidence quality is unacceptable.

## Success Criteria
- Envelope compatibility remains stable across commands.
- `symbol/history` misses return actionable structured causes and recommendations.
- `impact` output is deterministic and includes risk/confidence/evidence.
- `deadcode` returns classified findings with suggestions and confidence/evidence.
- Docs and skill match runtime behavior; smoke checks pass for index/symbol/history/impact/deadcode.
