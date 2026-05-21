package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"codrut/packages/coding-agent/codemap/indexer"
	"codrut/packages/coding-agent/codemap/store"
)

func runQueryCmdJSON(t *testing.T, dbPath, queryTerm string) ([]byte, int) {
	t.Helper()
	buf := &bytes.Buffer{}
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")
	args := []string{"--db", dbPath}
	if queryTerm != "" {
		args = append(args, queryTerm)
	}
	code := RunQuery(context.Background(), buf, args, repoPath)
	return buf.Bytes(), code
}

// RED: query emits valid envelope with schema_version "1.0".
func TestQueryEnvelopeSchemaVersion(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query.db")
	indexBuf := &bytes.Buffer{}
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")
	RunIndex(context.Background(), indexBuf, []string{"--db", dbPath}, repoPath)

	out, _ := runQueryCmdJSON(t, dbPath, "Valid")
	if !bytes.Contains(out, []byte(`"schema_version":"1.0"`)) {
		t.Errorf("missing schema_version=\"1.0\":\n%s", out)
	}
}

// RED: query envelope command field must be "query".
func TestQueryEnvelopeCommand(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runQueryCmdJSON(t, dbPath, "Valid")
	if !bytes.Contains(out, []byte(`"command":"query"`)) {
		t.Errorf("missing command=\"query\":\n%s", out)
	}
}

// RED: query with no index returns exit 3.
func TestQueryExitCode3NoIndex(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "no_index.db")

	_, code := runQueryCmdJSON(t, dbPath, "Valid")
	if code != 3 {
		t.Errorf("expected exit 3 for no-index state, got %d", code)
	}
}

// RED: query with missing argument returns exit 2.
func TestQueryExitCode2MissingArg(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	_, code := runQueryCmdJSON(t, dbPath, "")
	if code != 2 {
		t.Errorf("expected exit 2 for missing query term, got %d", code)
	}
}

// RED: query with valid DB + index returns exit 0.
func TestQueryExitCode0Success(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	_, code := runQueryCmdJSON(t, dbPath, "Valid")
	if code != 0 {
		t.Errorf("expected exit 0 for successful query, got %d", code)
	}
}

// RED: query emits ok:true envelope on success.
func TestQueryOkTrueEnvelope(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runQueryCmdJSON(t, dbPath, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	okVal, ok := env["ok"].(bool)
	if !ok || !okVal {
		t.Errorf("expected ok=true on success, got ok=%v", okVal)
	}
}

// RED: query emits errors=[] on success.
func TestQueryErrorsArrayEmpty(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runQueryCmdJSON(t, dbPath, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	errs, ok := env["errors"].([]interface{})
	if !ok {
		t.Fatalf("errors field missing or not array: %v", env["errors"])
	}
	if len(errs) != 0 {
		t.Errorf("expected empty errors[] on success, got %d items", len(errs))
	}
}

// RED: query data has query field (echoed term).
func TestQueryDataHasQuery(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runQueryCmdJSON(t, dbPath, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing from envelope")
	}
	if data["query"] == nil {
		t.Error("data missing query field")
	}
	if data["query"] != "Valid" {
		t.Errorf("query field should echo the search term, got %v", data["query"])
	}
}

// RED: query data has matches array.
func TestQueryDataHasMatches(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runQueryCmdJSON(t, dbPath, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data := env["data"].(map[string]interface{})
	matches, ok := data["matches"].([]interface{})
	if !ok {
		t.Errorf("matches should be a JSON array, got %T", data["matches"])
	}
	// Should have at least the symbol "Valid".
	if len(matches) == 0 {
		t.Error("expected at least one match for 'Valid'")
	}
}

// RED: query data has count field.
func TestQueryDataHasCount(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runQueryCmdJSON(t, dbPath, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data := env["data"].(map[string]interface{})
	if data["count"] == nil {
		t.Error("data missing count field")
	}
}

// RED: query emits deterministic JSON.
func TestQueryDeterminism(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query_det.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out1, _ := runQueryCmdJSON(t, dbPath, "Valid")
	out2, _ := runQueryCmdJSON(t, dbPath, "Valid")
	if !bytes.Equal(out1, out2) {
		t.Errorf("query JSON not deterministic:\nfirst:  %s\nsecond: %s", out1, out2)
	}
}

// RED: query error envelope includes schema_version.
func TestQueryErrorEnvelopeSchemaVersion(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "no_index.db")

	buf := &bytes.Buffer{}
	code := RunQuery(context.Background(), buf, []string{"--db", dbPath, "Foo"}, "")
	if code != 3 {
		t.Logf("note: got code %d instead of 3", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"schema_version":"1.0"`)) {
		t.Errorf("error envelope missing schema_version=\"1.0\":\n%s", buf.Bytes())
	}
}

// RED: query meta has snapshot_id.
func TestQueryMetaSnapshotID(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runQueryCmdJSON(t, dbPath, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	meta, ok := env["meta"].(map[string]interface{})
	if !ok {
		t.Fatal("meta missing from envelope")
	}
	if meta["snapshot_id"] == nil {
		t.Error("meta missing snapshot_id")
	}
}

// RED: query matches have required fields (name, kind, file).
func TestQueryMatchFields(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query_match.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runQueryCmdJSON(t, dbPath, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data := env["data"].(map[string]interface{})
	matches, ok := data["matches"].([]interface{})
	if !ok || len(matches) == 0 {
		t.Skip("no matches to validate")
	}
	match := matches[0].(map[string]interface{})
	for _, field := range []string{"name", "kind", "file"} {
		if match[field] == nil {
			t.Errorf("match missing %s field", field)
		}
	}
}

// RED: query with unreadable db path returns exit 1.
func TestQueryExitCode1RuntimeError(t *testing.T) {
	buf := &bytes.Buffer{}
	code := RunQuery(context.Background(), buf, []string{"--db", "/nonexistent/query.db", "Foo"}, "")
	if code != 1 {
		t.Errorf("expected exit 1 for unreadable db, got %d", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.Bytes())
	}
	okVal, ok := env["ok"].(bool)
	if !ok || okVal {
		t.Errorf("expected ok=false on runtime error")
	}
	if env["errors"] == nil {
		t.Errorf("expected errors[] on runtime error")
	}
}

// RED: prefix fallback — partial name returns matching symbols.
func TestQueryPrefixFallback(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query_prefix.db")

	db := store.MustOpen(dbPath)
	if err := store.Migrate(context.Background(), db.DB); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	snapID, err := store.BeginSnapshot(context.Background(), tx, tmp, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	fileID, err := store.UpsertFile(context.Background(), tx, tmp, "main.go", "go", "abc", snapID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ReplaceFileSymbols(context.Background(), tx, fileID, []indexer.Symbol{
		{Name: "Foo", Kind: "func", Signature: "func Foo()", StartLine: 1, EndLine: 5},
		{Name: "FooBar", Kind: "func", Signature: "func FooBar()", StartLine: 10, EndLine: 15},
		{Name: "FooBaz", Kind: "func", Signature: "func FooBaz()", StartLine: 20, EndLine: 25},
		{Name: "Food", Kind: "func", Signature: "func Food()", StartLine: 30, EndLine: 35},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	out, code := runQueryCmdJSON(t, dbPath, "Foo")
	if code != 0 {
		t.Fatalf("query failed: %s", out)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data := env["data"].(map[string]interface{})
	matches, ok := data["matches"].([]interface{})
	if !ok {
		t.Fatalf("matches should be array, got %T", data["matches"])
	}
	if len(matches) < 4 {
		t.Errorf("expected at least 4 matches for prefix 'Foo', got %d: %v", len(matches), data["matches"])
	}
}

// RED: exact match is first in prefix results.
func TestQueryExactMatchFirst(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query_exact.db")

	db := store.MustOpen(dbPath)
	if err := store.Migrate(context.Background(), db.DB); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	snapID, err := store.BeginSnapshot(context.Background(), tx, tmp, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	fileID, err := store.UpsertFile(context.Background(), tx, tmp, "main.go", "go", "abc", snapID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ReplaceFileSymbols(context.Background(), tx, fileID, []indexer.Symbol{
		{Name: "Foo", Kind: "func", Signature: "func Foo()", StartLine: 1, EndLine: 5},
		{Name: "FooBar", Kind: "func", Signature: "func FooBar()", StartLine: 10, EndLine: 15},
		{Name: "FooBaz", Kind: "func", Signature: "func FooBaz()", StartLine: 20, EndLine: 25},
		{Name: "Food", Kind: "func", Signature: "func Food()", StartLine: 30, EndLine: 35},
		{Name: "Fool", Kind: "func", Signature: "func Fool()", StartLine: 40, EndLine: 45},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	out, code := runQueryCmdJSON(t, dbPath, "Foo")
	if code != 0 {
		t.Fatalf("query failed: %s", out)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data := env["data"].(map[string]interface{})
	matches, ok := data["matches"].([]interface{})
	if !ok {
		t.Fatal("matches should be array")
	}
	if len(matches) == 0 {
		t.Fatal("expected at least one match")
	}
	first := matches[0].(map[string]interface{})
	if first["name"] != "Foo" {
		t.Errorf("expected 'Foo' (exact match) first, got %v", first["name"])
	}
	// Total count should be 5 (Foo + FooBar + FooBaz + Food + Fool).
	count, ok := data["count"].(float64)
	if !ok {
		t.Error("count should be number")
	}
	if int(count) != 5 {
		t.Errorf("expected count=5 for prefix 'Foo', got %v", count)
	}
}

// RED: matches are sorted deterministically.
func TestQueryMatchesSorted(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query_sorted.db")

	db := store.MustOpen(dbPath)
	if err := store.Migrate(context.Background(), db.DB); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	snapID, err := store.BeginSnapshot(context.Background(), tx, tmp, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	fileID, err := store.UpsertFile(context.Background(), tx, tmp, "main.go", "go", "abc", snapID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ReplaceFileSymbols(context.Background(), tx, fileID, []indexer.Symbol{
		{Name: "Zebra", Kind: "func", Signature: "func Zebra()", StartLine: 1, EndLine: 5},
		{Name: "Alpha", Kind: "func", Signature: "func Alpha()", StartLine: 10, EndLine: 15},
		{Name: "Beta", Kind: "func", Signature: "func Beta()", StartLine: 20, EndLine: 25},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	out, code := runQueryCmdJSON(t, dbPath, "A") // prefix match on "A"
	if code != 0 {
		t.Fatalf("query failed: %s", out)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data := env["data"].(map[string]interface{})
	matches, ok := data["matches"].([]interface{})
	if !ok {
		t.Fatal("matches should be array")
	}
	var prev string
	for _, m := range matches {
		name := m.(map[string]interface{})["name"].(string)
		if prev != "" && name < prev {
			t.Errorf("matches not sorted: %v", data["matches"])
		}
		prev = name
	}
}
