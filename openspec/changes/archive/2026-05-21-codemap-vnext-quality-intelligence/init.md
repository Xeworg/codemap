# Init — codemap-vnext-quality-intelligence

## Change ID
`codemap-vnext-quality-intelligence`

## Context
- Repository: `codrut`
- Branch: `feat/codemap-vnext-sdd`
- Base PRD: `docs/prd-vnext.md`
- Execution mode: interactive (`openspec/config.yaml`)
- Strict TDD: active (`go test ./...`)

## Objective
Deliver vNext improvements for:
1. `codemap impact` reliability and risk-tier output
2. explain-why-not-found in `symbol/history`
3. `codemap deadcode` report + suggestions

## Constraints
- Preserve CLI JSON envelope v1.0
- Keep reviewer workload under 400 changed lines per work unit where possible
- Update Pi skill docs as part of release hardening
