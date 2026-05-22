# Explore — codemap-vnext-quality-intelligence

## Current State Findings
- `impact` has baseline envelope/exit-code spec but lacks richer risk/evidence behavior.
- `symbol/history` return structured envelopes, but no standardized cause taxonomy for misses.
- No dedicated `deadcode` command/spec exists yet.
- Skill runtime currently documents install/doctor flows and needs vNext workflow updates.

## Code Areas Likely Affected
- `cmd/codemap/main.go` (command wiring/help)
- `packages/coding-agent/codemap/cli/*` (handlers and output shaping)
- `packages/coding-agent/codemap/store/*` (queries and link analysis)
- `docs/codemap-cli-json-contract.md`
- `integrations/pi/skills/codemap-usage/SKILL.md`

## Risks
- Deadcode false positives if heuristics are too strict.
- Impact noise if risk scoring lacks deterministic ordering/caps.
- Explain mode becoming vague without normalized cause codes.

## Exploration Decision
Proceed with spec-first, then implement in small vertical slices: explain → impact → deadcode.
