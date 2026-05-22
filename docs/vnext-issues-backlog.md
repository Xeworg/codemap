# CodeMap vNext — Issues Backlog

Base PRD: `docs/prd-vnext.md`

## Issue 1 — M1: Define contracts and taxonomies
**Type**: design/architecture  
**Goal**: Lock JSON contracts and shared enums before implementation.

### Scope
- Define JSON schema for `impact` response.
- Define JSON schema for `deadcode` response.
- Define explain-not-found taxonomy for `symbol/history`.
- Define shared enums: `risk_tier`, `confidence`, `not_found_cause`, `suggestion`.

### Subtasks
- [ ] Update `docs/codemap-cli-json-contract.md` with new command sections.
- [ ] Add explain-not-found response examples (success + failure variants).
- [ ] Align envelope semantics (`ok`, `data`, `errors`, `meta`) with existing commands.
- [ ] Add migration note for clients consuming older schema.

### Acceptance Criteria
- [ ] Contracts documented with at least one valid sample per command.
- [ ] Cause taxonomy is deterministic and machine-readable.
- [ ] No contract ambiguity on empty results.

---

## Issue 2 — M3: Implement explain-not-found for `symbol/history`
**Type**: feature  
**Goal**: Make failed lookups actionable.

### Scope
- Add explain path when `symbol` or `history` has empty/weak result.
- Return `causes[]` and `recommended_actions[]`.
- Support both JSON output and human-readable output.

### Subtasks
- [ ] Implement cause detection hooks in CLI/query flow.
- [ ] Add cause codes (e.g., stale index, name mismatch, parse errors, missing links).
- [ ] Add integration tests for representative failure cases.
- [ ] Add command help updates/examples.

### Acceptance Criteria
- [ ] Empty/weak lookups produce structured explain output.
- [ ] Suggested actions map correctly to causes.
- [ ] Tests cover at least 4 distinct causes.

---

## Issue 3 — M2: Implement `codemap impact` v1 (CLI-first)
**Type**: feature  
**Goal**: Provide reliable impact analysis before changes/PRs.

### Scope
- New CLI command: `codemap impact <symbol>`.
- Return impacted symbols/files with `high|medium|low` risk tiers.
- Include evidence for each impact item.

### Subtasks
- [ ] Implement command wiring and handler.
- [ ] Add dependency/reference query logic.
- [ ] Implement risk scoring heuristics.
- [ ] Add deterministic sorting and default result cap.
- [ ] Add unit + integration tests.

### Acceptance Criteria
- [ ] Command returns valid JSON envelope and deterministic output.
- [ ] Risk tiers and evidence present for each item.
- [ ] Smoke-tested on project repo.

---

## Issue 4 — M4: Implement `codemap deadcode` v1 (report + suggestions)
**Type**: feature  
**Goal**: Detect likely dead code without hard enforcement.

### Scope
- New CLI command: `codemap deadcode`.
- Classification: `unused | likely-unused | uncertain`.
- Suggestions: `remove | deprecate | justify`.
- Exclusions: generated files, known entrypoints, allowlist.

### Subtasks
- [ ] Implement zero-inbound-reference baseline detection.
- [ ] Implement classification and confidence scoring.
- [ ] Implement suggestion mapping logic.
- [ ] Add exclusion/allowlist support.
- [ ] Add curated fixtures and tests for false-positive control.

### Acceptance Criteria
- [ ] Output includes symbol, location, class, suggestion, confidence, evidence.
- [ ] Exclusions behave deterministically.
- [ ] False-positive rate measured and documented on curated sample.

---

## Issue 5 — M5: Docs, skill update, smoke, release notes
**Type**: docs/release  
**Goal**: Ensure adoption and operability.

### Scope
- Update user docs and JSON contract docs.
- Update Pi skill docs/workflows for new commands.
- Add smoke commands and expected outputs.
- Publish release notes.

### Subtasks
- [ ] Update `docs/codemap-cli-json-contract.md` examples for all new features.
- [ ] Update `integrations/pi/skills/codemap-usage/SKILL.md` with:
  - `codemap impact` workflow
  - explain-not-found troubleshooting flow
  - `codemap deadcode` workflow and interpretation
- [ ] Validate installer artifact includes updated skill content.
- [ ] Add smoke script/checklist for `index/symbol/history/impact/deadcode`.
- [ ] Draft release notes with migration/compatibility notes.

### Acceptance Criteria
- [ ] Skill and docs reflect actual command behavior.
- [ ] Smoke checklist passes in a clean environment.
- [ ] Release notes include known limitations and next steps.

---

## Suggested Implementation Order
1. Issue 1 (M1)
2. Issue 2 (M3)
3. Issue 3 (M2)
4. Issue 4 (M4)
5. Issue 5 (M5)
