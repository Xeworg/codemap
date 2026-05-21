# Archive Report — codemap-critical-prd-gaps-v1

## Status: PASS

All preconditions met. Verification and sync are PASS. No blockers, collisions, or destructive deltas.

---

## Artifacts Read

| Artifact | Path | Status |
|----------|------|--------|
| Proposal | `openspec/changes/codemap-critical-prd-gaps-v1/proposal.md` | Present |
| Design | `openspec/changes/codemap-critical-prd-gaps-v1/design.md` | Present |
| Tasks | `openspec/changes/codemap-critical-prd-gaps-v1/tasks.md` | Present |
| Verify Report | `openspec/changes/codemap-critical-prd-gaps-v1/verify-report.md` | PASS |
| Sync Report | `openspec/changes/codemap-critical-prd-gaps-v1/sync-report.md` | synced |
| Config | `openspec/config.yaml` | Present |

---

## Domains Synced

| Domain | Canonical File | Action |
|--------|---------------|--------|
| `impact` | `openspec/specs/impact/spec.md` | Created (new canonical spec) |
| `migrate` | `openspec/specs/migrate/spec.md` | Created (new canonical spec) |
| `query` | `openspec/specs/query/spec.md` | Created (new canonical spec) |

---

## Requirements Synced

### ADDED

- **impact**
  - Impact JSON envelope v1 compatibility
  - Impact JSON determinism
  - Impact exit-code stability
- **migrate**
  - Explicit migration command execution
  - Idempotent migrate behavior
  - Migrate exit-code stability
- **query**
  - Query JSON envelope v1 compatibility
  - Query machine-parseable deterministic output
  - Query exit-code stability

### MODIFIED

None.

### REMOVED

None.

---

## Active Same-Domain Change Warnings

None. No other active change touches `impact`, `migrate`, or `query` domains.

---

## Destructive Merge Approvals / Blockers

- **Destructive changes:** None.
- **Approval required:** No.

---

## Archived Path

`openspec/changes/archive/2026-05-21-codemap-critical-prd-gaps-v1/`

---

## Memory Observation IDs

N/A — artifact store mode is `openspec` (Engram unavailable in this session).
