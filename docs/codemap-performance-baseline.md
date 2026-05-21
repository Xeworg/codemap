# CodeMap Performance Baseline

## How to Run

### Guardrail tests
```bash
go test -count=1 ./packages/coding-agent/codemap/indexer -run Test.*Perf -v
```

### Benchmarks
```bash
# Full benchmark suite (3s per benchmark)
go test -bench . -run ^$ -benchtime=3s ./packages/coding-agent/codemap/indexer

# Individual benchmark
go test -bench=BenchmarkRunIndexFullScan -run ^$ -benchtime=5s ./packages/coding-agent/codemap/indexer
```

---

## Current Baseline Numbers

Measured on: **12th Gen Intel Core i7-12700H**  
Fixture: `testdata/repos/incremental-go` (3 .go source files, 1 vendor file excluded)

| Benchmark | ns/op | Notes |
|---|---|---|
| `BenchmarkRunIndexFullScan` | ~58,000 ns/op (~58 µs) | Cold scan, no PrevFiles |
| `BenchmarkRunIndexUnchangedReindex` | ~33,000 ns/op (~33 µs) | All files skipped; minimal diff cost |
| `BenchmarkRunIndexOneFileChanged` | ~702,000 ns/op (~702 µs) | Includes file mutation + parse of changed file |
| `BenchmarkDiscoverFiles` | ~28,000 ns/op (~28 µs) | File walk cost only |

### Guardrail test results
```
TestPerfIncrementalScalingRegression     PASS  — unchanged < full scan; changed >= unchanged
TestPerfFileDiscoveryExcludesVendor      PASS  — no vendor paths in candidates
TestPerfParseErrorDoesNotAbort           PASS  — parse error counted, run completes
```

---

## Key Observations

1. **Unchanged reindex is ~44% faster than full scan** (~33 µs vs ~58 µs). Incremental diff logic is effective even on a tiny fixture. The delta grows proportionally with repo size.

2. **One-file-change reindex is ~12× more expensive than unchanged reindex** (~702 µs vs ~33 µs). This includes file mutation inside the benchmark loop, which inflates the number. Real-world changed reindex is expected to be closer to full scan (parse cost dominates).

3. **Vendor exclusion is working correctly**: `TestPerfFileDiscoveryExcludesVendor` passes, ensuring large `vendor/` directories do not inflate scan cost.

4. **Fail-soft contract holds**: `TestPerfParseErrorDoesNotAbort` confirms that a single broken `.go` file does not abort the run.

---

## Regression Thresholds

The guardrail tests enforce relative guarantees:

| Guard | Condition |
|---|---|
| Incremental scaling | `unchanged_processed < full_scanned` |
| Change cost | `changed_processed >= unchanged_processed` |
| Full scan cost | `changed_processed < full_scanned` |
| Vendor exclusion | zero vendor paths in candidates |

These are **deterministic** (file-count based, not wall-clock based) to avoid flakiness in CI.

---

## Scalability Expectations

The fixture is minimal (3 files). Real Go repos scale as follows:

| Repo size | Expected full-scan cost | Expected unchanged-reindex cost |
|---|---|---|
| ~100 .go files | ~1–5 ms | ~0.2–1 ms |
| ~1,000 .go files | ~10–50 ms | ~2–10 ms |
| ~10,000 .go files | ~100–500 ms | ~20–100 ms |

Cost is dominated by file I/O and Go AST parsing. Hash computation (`sha256`) is negligible.

---

## Adding New Benchmarks

Benchmarks use `b.TempDir()` to ensure isolation. To benchmark a new scenario:

```go
func BenchmarkMyScenario(b *testing.B) {
    repo := setupPerfBench(b, "my-repo-fixture")
    req := IndexRequest{RepoRoot: repo, ...}
    ctx := context.Background()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = RunIndex(ctx, req)
    }
}
```

Add fixtures under `packages/coding-agent/codemap/testdata/repos/`.

---

## Non-goals

- Strict wall-clock thresholds (too flaky for CI).
- Memory allocation benchmarks (MVP cost is dominated by I/O, not allocation).
- Multi-language parsing performance (Go-only MVP).