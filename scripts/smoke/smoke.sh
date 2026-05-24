#!/bin/bash
# codemap vNext smoke validation script
# Run from repository root: bash scripts/smoke/smoke.sh
set -e

REPO="${REPO:-packages/coding-agent/codemap/testdata/repos/incremental-go}"
TMPDB="${TMPDIR:-/tmp}/smoke-codemap-$$"
BINARY="${BINARY:-./codemap}"

mkdir -p "$TMPDB"

cleanup() {
  rm -rf "$TMPDB"
}
trap cleanup EXIT

echo "=== Building codemap ==="
go build -o "$BINARY" ./cmd/codemap

echo "=== Smoke: index ==="
$BINARY --repo "$REPO" index --db "$TMPDB/codemap.db" | grep -q '"ok":true'
echo "PASS"

echo "=== Smoke: symbol ==="
$BINARY --repo "$REPO" symbol --db "$TMPDB/codemap.db" Add | grep -q '"ok":true'
echo "PASS"

echo "=== Smoke: history ==="
$BINARY --repo "$REPO" history --db "$TMPDB/codemap.db" Add | grep -q '"ok":true'
echo "PASS"

echo "=== Smoke: impact ==="
$BINARY --repo "$REPO" impact --db "$TMPDB/codemap.db" Add | grep -q '"ok":true'
echo "PASS"

echo "=== Smoke: deadcode ==="
$BINARY --repo "$REPO" deadcode --db "$TMPDB/codemap.db" | grep -q '"ok":true'
echo "PASS"

echo "=== Smoke: query ==="
$BINARY --repo "$REPO" query --db "$TMPDB/codemap.db" Add | grep -q '"ok":true'
echo "PASS"

echo "=== Smoke: migrate ==="
$BINARY --repo "$REPO" migrate --db "$TMPDB/codemap.db" | grep -q '"ok":true'
echo "PASS"

echo "=== Smoke: doctor ==="
OUT=$($BINARY doctor --json)
echo "$OUT" | grep -q '"status": "pass"'
echo "PASS"

echo "=== Smoke: install dry-run ==="
OUT=$($BINARY install --dry-run --json)
echo "$OUT" | grep -q '"status": "dry-run"'
echo "PASS"

echo "=== Smoke: graph-query ==="
OUT=$($BINARY --repo "$REPO" graph-query --db "$TMPDB/codemap.db" Add)
echo "$OUT" | grep -q '"ok":true'
echo "$OUT" | grep -q '"command":"graph-query"'
echo "PASS"

echo "=== Smoke: graph-query with depth ==="
OUT=$($BINARY --repo "$REPO" graph-query --db "$TMPDB/codemap.db" --depth 1 Add)
echo "$OUT" | grep -q '"ok":true'
echo "PASS"

echo "=== Smoke: graph-query with no-cache ==="
OUT=$($BINARY --repo "$REPO" graph-query --db "$TMPDB/codemap.db" --no-cache Add)
echo "$OUT" | grep -q '"ok":true'
echo "PASS"

echo "=== Smoke: impact --depth 1 ==="
OUT=$($BINARY --repo "$REPO" impact --db "$TMPDB/codemap.db" --depth 1 Add)
echo "$OUT" | grep -q '"ok":true'
echo "PASS"

echo "=== Smoke: impact --no-cache ==="
OUT=$($BINARY --repo "$REPO" impact --db "$TMPDB/codemap.db" --no-cache Add)
echo "$OUT" | grep -q '"ok":true'
echo "PASS"

echo ""
echo "=== ALL SMOKE TESTS PASSED ==="