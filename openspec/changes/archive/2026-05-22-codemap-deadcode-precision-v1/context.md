# SDD Change Context: codemap-deadcode-precision-v1

## Initialization Summary
- Reused existing `openspec/config.yaml`.
- Confirmed strict TDD is enabled (`sdd.strict_tdd: true`).
- Confirmed canonical test runner is active Go test:
  - `testing.runner.type: go-test`
  - `testing.runner.command: go test ./...`

## Repository Context
- Primary stack: Go module (`go.mod` present at repository root).
- Existing related spec available: `openspec/specs/deadcode/spec.md`.
- OpenSpec changes workspace: `openspec/changes/`.

## Phase Guardrails (from config)
- Spec before implementation.
- Design required for non-trivial changes.
- Tasks before apply.
- Verify before merge.

## Notes
- `.atl/skill-registry.md` exists.
- Engram memory tools are not available in this session; persistence is file-based under `openspec/`.
