# Archive Report: codemap-impact-edges-v1

| Field | Value |
|-------|-------|
| **Status** | PASS |
| **Change ID** | codemap-impact-edges-v1 |
| **Archive Date** | 2026-05-22 |
| **Archived Path** | `openspec/changes/archive/2026-05-22-codemap-impact-edges-v1/` |

## Artifacts Read

- `openspec/changes/codemap-impact-edges-v1/proposal.md`
- `openspec/changes/codemap-impact-edges-v1/specs/impact/spec.md`
- `openspec/changes/codemap-impact-edges-v1/design.md`
- `openspec/changes/codemap-impact-edges-v1/tasks.md`
- `openspec/changes/codemap-impact-edges-v1/verify-report.md`
- `openspec/config.yaml`

## Verification Status

- **Verify report**: PASS (all P1–P11 complete, `go test ./...` green)
- **Blockers**: None
- **Destructive merge**: None (no REMOVED requirements)

## Domains Synced

| Domain | Action | Canonical Path |
|--------|--------|----------------|
| `impact` | Sync ADDED + MODIFIED requirements | `openspec/specs/impact/spec.md` |

## Requirements Summary

| Operation | Requirement Name |
|-----------|------------------|
| **ADDED** | Index-time edge extraction supports impact tier diversity |
| **ADDED** | Unresolved non-call edge candidates are fail-soft |
| **MODIFIED** | Impact quality intelligence fields |
| **MODIFIED** | Impact default cap and deterministic ordering |
| **REMOVED** | — |

## Active Same-Domain Change Warnings

- None. No other active change under `openspec/changes/*/specs/impact/` at archive time.

## Destructive Merge Approvals

- Not applicable (no REMOVED requirements, no large MODIFIED blocks that delete scenarios).

## Post-Archive Test Status

```
$ go test ./...
# → all packages green (verified before and after move)
```

## Notes

- Archive-time sync fallback was used because no prior `sync-report.md` existed; parent task explicitly approved canonical sync.
- Phase 1 apply-progress drift (P5 listed as deferred) was corrected before archive.
- Total diff: ~841 insertions across 12 files including indexer edges, tests, CLI impact tests, docs, and changelog.
