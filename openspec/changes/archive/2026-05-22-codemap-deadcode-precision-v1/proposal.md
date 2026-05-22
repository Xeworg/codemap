# Proposal: codemap-deadcode-precision-v1

## Intent
Improve `codemap deadcode` precision from effectively unusable (all symbols flagged due to missing edge data) to a trustworthy report with materially lower false positives, while keeping report-only safety and deterministic output.

## Scope
### In scope
1. Add a production edge-population path in indexing (minimum viable symbol reference/call edges) so inbound counts are meaningful.
2. Expand symbol extraction to include methods (and `init` handling) needed for deadcode precision.
3. Add precision heuristics for runtime/public-entry symbols (for example `main`, `init`, exported library API, conventional `cmd/` entrypoints) to reduce known false positives.
4. Refine deadcode classification/confidence logic to distinguish true-unused vs uncertain-with-implicit-usage.
5. Add a curated validation corpus and precision measurement harness for false-positive tracking.
6. Update docs/skill guidance to reflect real behavior and current limits.

### Non-goals
- Full whole-program static analysis/SSA call graph.
- Perfect interface/reflection/build-tag resolution in v1.
- Any auto-delete or refactor execution (deadcode remains report-only).
- Non-Go language support.

## Affected areas
- `packages/coding-agent/codemap/indexer/` (edge extraction, method/init symbol coverage)
- `packages/coding-agent/codemap/store/` (edge persistence/query usage)
- `packages/coding-agent/codemap/cli/deadcode.go` (classification/heuristics/evidence)
- `packages/coding-agent/codemap/cli/*_test.go` + indexer/store tests (precision and regressions)
- `openspec/specs/deadcode/spec.md` (if behavior contract changes)
- `integrations/pi/skills/codemap-usage/SKILL.md` (accuracy of operational guidance)

## Success metrics
1. False-positive rate on curated deadcode validation set is **<20%** (PRD target).
2. Precision baseline is reproducible and tracked in tests/report artifacts.
3. Deadcode output remains deterministic for same input/index state.
4. No source mutation side effects from deadcode command.

## Rollout plan
1. **Phase 1 (foundation):** ship production edge population + method coverage behind validation tests.
2. **Phase 2 (precision):** add runtime/public API heuristics and confidence tuning.
3. **Phase 3 (quality gate):** run curated corpus checks; gate merge on FP metric and deterministic output tests.
4. **Phase 4 (adoption):** update docs/skill notes and communicate known limits explicitly.

## Risks and mitigation
1. **Scope creep into full static analyzer**  
   Mitigation: constrain to minimal viable edge model and explicit non-goals.
2. **Metric miss (<20% FP not reached)**  
   Mitigation: establish corpus early, iterate heuristics with measurable deltas, fail fast on regressions.
3. **New false negatives from aggressive exclusions**  
   Mitigation: classify uncertain instead of suppressing, include explicit evidence tags.
4. **Performance regression during indexing**  
   Mitigation: benchmark index time/DB growth and cap edge extraction complexity.
5. **User trust gap from outdated docs**  
   Mitigation: update skill/docs in same change window as behavior updates.

## Rollback plan
- Revert deadcode precision slices independently (heuristics first, then edge extraction) if instability appears.
- Keep prior deterministic envelope/output contract intact during rollback.
- If edge extraction causes unacceptable regressions, temporarily run deadcode in conservative heuristic mode with explicit uncertainty labeling.

## Skill resolution
- `skill_resolution: paths-injected`
