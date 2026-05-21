# Sync Report — codemap-critical-prd-gaps-v1

## Status: synced

All domain specs from the change were successfully merged into canonical `openspec/specs/`.

---

## Domains Synced

| Domain | Canonical File | Action |
|--------|---------------|--------|
| `impact` | `openspec/specs/impact/spec.md` | Created (new canonical spec) |
| `migrate` | `openspec/specs/migrate/spec.md` | Created (new canonical spec) |
| `query` | `openspec/specs/query/spec.md` | Created (new canonical spec) |

---

## Canonical Files Updated

- `openspec/specs/impact/spec.md`
- `openspec/specs/migrate/spec.md`
- `openspec/specs/query/spec.md`

---

## Requirements Synced

### impact (3 requirements added)

- Impact JSON envelope v1 compatibility
- Impact JSON determinism
- Impact exit-code stability

### migrate (3 requirements added)

- Explicit migration command execution
- Idempotent migrate behavior
- Migrate exit-code stability

### query (3 requirements added)

- Query JSON envelope v1 compatibility
- Query machine-parseable deterministic output
- Query exit-code stability

---

## Active Same-Domain Collisions

None. No other active change touches `impact`, `migrate`, or `query` domains.

---

## Destructive Sync Approvals / Blockers

- **Destructive changes:** None. No REMOVED or large MODIFIED deltas were present.
- **Approval required:** No.

---

## Validation Commands / Checks Performed

```bash
# Verify canonical files exist and match change spec checksums
$ sha256sum openspec/changes/codemap-critical-prd-gaps-v1/specs/impact/spec.md openspec/specs/impact/spec.md
54db73bcc16d0a30433285f5d8706bf62d51802c88e51cca0d36971168eddcb1  openspec/changes/codemap-critical-prd-gaps-v1/specs/impact/spec.md
54db73bcc16d0a30433285f5d8706bf62d51802c88e51cca0d36971168eddcb1  openspec/specs/impact/spec.md

$ sha256sum openspec/changes/codemap-critical-prd-gaps-v1/specs/migrate/spec.md openspec/specs/migrate/spec.md
b659bda03f6a9810594914392d94ae7f667a41a4cbb2339c66853dc1ed57b842  openspec/changes/codemap-critical-prd-gaps-v1/specs/migrate/spec.md
b659bda03f6a9810594914392d94ae7f667a41a4cbb2339c66853dc1ed57b842  openspec/specs/migrate/spec.md

$ sha256sum openspec/changes/codemap-critical-prd-gaps-v1/specs/query/spec.md openspec/specs/query/spec.md
70320c46319d4d1be2dbfff3f7d6218604a32f40c55b6d71bb8eebbbbd2b37f9  openspec/changes/codemap-critical-prd-gaps-v1/specs/query/spec.md
70320c46319d4d1be2dbfff3f7d6218604a32f40c55b6d71bb8eebbbbd2b37f9  openspec/specs/query/spec.md
```

All checksums match. Canonical files are byte-identical to change specs.

---

## Next Recommended Phase

`sdd-archive` — the change is clean, verified, and synced. Ready for archive.
