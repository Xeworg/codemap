# Slice D: Regression fixtures + docs — COMPLETED

## Status: PASS ✅

## Changes

### New files
- `packages/coding-agent/codemap/testdata/deadcode-precision/fixture/fixture.go`: Go package with exported API (`ExportedHelper`), private unused (`privateUnused`), private used (`privateUsed` via main), `init` function, and method `T.Method`.
- `packages/coding-agent/codemap/testdata/deadcode-precision/deadcode_precision_test.go`: end-to-end regression test that builds codemap from source, runs index + deadcode on the fixture, and asserts precision heuristics.
- `packages/coding-agent/codemap/docs/deadcode.md`: operational docs covering evidence tiers, heuristic boundaries, safe-action guarantee, and known limitations.
- `CHANGELOG.md` (new): added unreleased section with deadcode precision v1 changelog entry.

## Evidence

```
go test ./packages/coding-agent/codemap/testdata/deadcode-precision/...
--- PASS: TestDeadcode_PrecisionFixture_GoFiles
--- PASS: TestDeadcode_PrecisionRegression

go test ./... (full suite)
ok  	codrut/packages/coding-agent/codemap/cli
ok  	codrut/packages/coding-agent/codemap/cli/installer
ok  	codrut/packages/coding-agent/codemap/git
ok  	codrut/packages/coding-agent/codemap/indexer
ok  	codrut/packages/coding-agent/codemap/store
```

## TDD cycle

- RED (D1): fixture had two issues — (1) `package main` + `func main()` in testdata caused two packages collision in the same directory; (2) `PrivateUnused` (uppercase) was exported API and got `uncertain` instead of the expected `unused`. Both fixed by renaming to lowercase `privateUnused`.
- RED (D1): `filepath.Dir` relative vs absolute path confusion in test helper. Fixed by using absolute path via `filepath.Abs` at runtime.
- RED (D1): installed PATH binary lacked `deadcode` command. Fixed by building from source with `go build ./cmd/codemap` using correct `repoRoot()` computed from test-file location (6 levels up from test file).
- GREEN (D1): all D1 assertions pass after fixes.
- GREEN (D2): docs written and verified for structural completeness.
- GREEN (D3): changelog entry added with exact text from tasks.md.

## D1 assertions verified

| Symbol | Expected | Result |
|--------|----------|--------|
| `ExportedHelper` | NOT `unused+high` | ✅ uncertain |
| `init` | NOT `unused` | ✅ uncertain |
| `privateUnused` | `unused` or `likely-unused` | ✅ `unused` |
| `T.Method` | NOT `unused+high` | ✅ uncertain |
| `privateUsed` | (used by main, no edges expected from fixture perspective) | ✅ uncertain or absent |

## Open risks

- Regression test builds codemap from source on each run (~0.3s). Acceptable for correctness.
- `docs/deadcode.md` uses the existing `docs/` location convention. If the project later moves to a single docs root, this file should migrate accordingly.