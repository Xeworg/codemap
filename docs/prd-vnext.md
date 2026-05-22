# PRD — CodeMap vNext

## 1) Goal
Ship a balanced vNext release that improves change safety and codebase hygiene through three pillars:
1. Reliable impact analysis for PR planning (`codemap impact`)
2. Clear diagnostics when lookups fail (`symbol/history` explain mode)
3. Dead code detection with actionable guidance (`codemap deadcode`)

## 2) Problem
Current workflows still rely on manual guesswork for:
- Understanding downstream impact before code changes
- Debugging missing symbol/history results
- Detecting newly introduced dead code

This creates review risk, regressions, and technical debt accumulation.

## 3) Scope (In)

### A. `codemap impact` (v1, CLI-first)
- Input: symbol name (and optional repo/db flags)
- Output: impacted symbols/files grouped by risk tier (`high|medium|low`)
- Evidence included (e.g., references, call links, file links, history links)
- JSON envelope aligned with existing CLI contract

### B. Explain-Why-Not-Found mode (`symbol/history`)
- New explain path when a query returns empty or weak result
- Return probable root causes and next-step recommendations
- Causes include: stale/missing index, naming mismatch, parse errors, missing links
- Works in JSON output and human-readable mode

### C. `codemap deadcode` (v1 = report + suggestions)
- Detect likely dead symbols (starting with no inbound references)
- Classify findings: `unused | likely-unused | uncertain`
- Output actionable suggestion per finding: `remove | deprecate | justify`
- Include confidence + evidence
- Respect exclusions: generated files, known entrypoints, explicit allowlist

## 4) Scope (Out)
- Hard CI gate by default for dead code
- Full inter-procedural whole-program precision guarantees
- Automatic code deletion/refactors

## 5) Success Metrics (KPIs)
- ≥70% of sampled high-risk impact results validated as relevant
- ≥80% of “not found” cases provide a correct primary cause
- Deadcode false-positive rate under 20% on curated sample set
- User-reported time-to-triage reduced for symbol/history failures

## 6) User Stories
- As a developer, I can run `codemap impact <symbol>` and know what to review first.
- As a developer, when `symbol/history` fails, I get concrete reasons and next actions.
- As a developer, I can identify likely dead code and decide safely what to do.

## 7) Functional Requirements
1. CLI commands must preserve JSON envelope schema consistency.
2. Impact results must include risk tier and evidence list.
3. Explain mode must produce machine-readable cause codes + human message.
4. Deadcode findings must include symbol, location, classification, suggestion, confidence.
5. All commands support `-repo` global flag behavior consistently.

## 8) Risks & Mitigations
- False positives in deadcode/impact → confidence scoring + `uncertain` bucket + exclusions.
- Reviewer overload from noisy output → deterministic sorting and capped default result set.
- Ambiguous diagnostics in explain mode → standardized cause taxonomy and recommended actions.

## 9) Milestones
- M1: Data model + JSON contracts for impact/explain/deadcode
- M2: `impact` v1 implementation + tests
- M3: explain-not-found implementation in `symbol/history` + tests
- M4: `deadcode` report+suggestions + tests
- M5: docs, examples, smoke validation, skill update, release notes

## 10) Release Decision
Ship when:
- All three pillars are implemented with tests
- KPI sampling is run and documented
- CLI docs updated with examples and troubleshooting
- Pi skill docs updated for `impact`, explain-not-found, and `deadcode` workflows
