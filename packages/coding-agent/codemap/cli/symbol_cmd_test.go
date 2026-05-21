package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func runSymbolCmdJSON(t *testing.T, dbPath, symbolArg string) ([]byte, int) {
	t.Helper()
	buf := &bytes.Buffer{}
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")
	args := []string{"--db", dbPath, "--json", symbolArg}
	code := RunSymbol(context.Background(), buf, args, repoPath)
	return buf.Bytes(), code
}

func TestSymbolCmdJSONEnvelopeSchemaVersion(t *testing.T) {
	// RED: schema_version must be "1.0"
	// Need populated DB first.
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	indexBuf := &bytes.Buffer{}
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")
	RunIndex(context.Background(), indexBuf, []string{"--db", db}, repoPath)

	out, _ := runSymbolCmdJSON(t, db, "Valid")
	if !bytes.Contains(out, []byte(`"schema_version":"1.0"`)) {
		t.Errorf("missing schema_version=\"1.0\":\n%s", out)
	}
}

func TestSymbolCmdJSONEnvelopeMetaFields(t *testing.T) {
	// RED: meta must have snapshot_id, head_ref, indexed_at, is_stale
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runSymbolCmdJSON(t, db, "Valid")
	for _, field := range []string{`"snapshot_id"`, `"head_ref"`, `"indexed_at"`, `"is_stale"`} {
		if !bytes.Contains(out, []byte(field)) {
			t.Errorf("missing meta field %s:\n%s", field, out)
		}
	}
}

func TestSymbolCmdJSONDataEvidencePresent(t *testing.T) {
	// RED: data.evidence must be present
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runSymbolCmdJSON(t, db, "Valid")
	if !bytes.Contains(out, []byte(`"evidence"`)) {
		t.Errorf("missing evidence field:\n%s", out)
	}
}

func TestSymbolCmdJSONConfidenceEnum(t *testing.T) {
	// RED: confidence must be one of high|medium|low
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runSymbolCmdJSON(t, db, "Valid")
	// Parse JSON and check confidence field.
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON output:\n%s", out)
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data field missing or not object")
	}
	if conf, ok := data["confidence"]; ok {
		switch conf {
		case "high", "medium", "low":
			// OK
		default:
			t.Errorf("confidence %q not in [high,medium,low]", conf)
		}
	}
}

func TestSymbolCmdJSONTopLevelEnvelope(t *testing.T) {
	// RED: top-level envelope must have schema_version, ok, data, errors, meta
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runSymbolCmdJSON(t, db, "Valid")
	for _, field := range []string{`"schema_version"`, `"ok"`, `"data"`, `"errors"`, `"meta"`} {
		if !bytes.Contains(out, []byte(field)) {
			t.Errorf("missing top-level field %s:\n%s", field, out)
		}
	}
}

func TestSymbolCmdExitCode0Success(t *testing.T) {
	// RED: exit 0 for valid symbol query
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	_, code := runSymbolCmdJSON(t, db, "Valid")
	if code != 0 {
		t.Errorf("expected exit 0 for valid symbol, got %d", code)
	}
}

func TestSymbolCmdExitCode2ValidationError(t *testing.T) {
	// RED: exit 2 for missing/invalid symbol argument
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	// Empty DB — no index done.
	_, code := runSymbolCmdJSON(t, db, "")
	if code != 2 {
		t.Errorf("expected exit 2 for validation error, got %d", code)
	}
}

func TestSymbolCmdExitCode3NotFound(t *testing.T) {
	// RED: exit 3 when symbol not found (data-state: index exists, symbol absent)
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	_, code := runSymbolCmdJSON(t, db, "NonExistentSymbolXYZ")
	if code != 3 {
		t.Errorf("expected exit 3 for not-found symbol, got %d", code)
	}
}

func TestSymbolCmdExitCode1RuntimeError(t *testing.T) {
	// RED: exit 1 for runtime error (bad DB)
	buf := &bytes.Buffer{}
	args := []string{"--db", "/nonexistent/test.db", "--json", "foo"}
	code := RunSymbol(context.Background(), buf, args, "")
	if code != 1 {
		t.Errorf("expected exit 1 for runtime error, got %d", code)
	}
}

func TestSymbolCmdReturnsSymbolData(t *testing.T) {
	// RED: valid symbol returns data with symbol name and kind
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runSymbolCmdJSON(t, db, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON:\n%s", out)
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing")
	}
	// Should have symbol info in data
	if _, ok := data["name"]; !ok {
		t.Error("data missing name field")
	}
}

func TestSymbolCmdExitCode2JSONEnvelope(t *testing.T) {
	// validation errors emit JSON envelope with ok:false and errors[]
	// No --db flag; uses default cache path. Trigger validation by omitting symbol arg.
	buf := &bytes.Buffer{}
	code := RunSymbol(context.Background(), buf, []string{}, "")
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.Bytes())
	}
	if ok, _ := env["ok"].(bool); ok {
		t.Errorf("expected ok=false on validation error, got ok=true")
	}
	if errs, ok := env["errors"].([]interface{}); !ok || len(errs) == 0 {
		t.Errorf("expected non-empty errors[] on validation error, got %v", env["errors"])
	}
	if _, ok := env["schema_version"]; !ok {
		t.Errorf("expected schema_version in error envelope")
	}
}

func TestSymbolCmdExitCode3JSONEnvelope(t *testing.T) {
	// GREEN: not-found errors emit JSON envelope with ok:false and errors[]
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))
	buf := &bytes.Buffer{}
	code := RunSymbol(context.Background(), buf, []string{"--db", db, "DoesNotExist"}, "")
	if code != 3 {
		t.Errorf("expected exit 3, got %d", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.Bytes())
	}
	if ok, _ := env["ok"].(bool); ok {
		t.Errorf("expected ok=false on not-found, got ok=true")
	}
	if errs, ok := env["errors"].([]interface{}); !ok || len(errs) == 0 {
		t.Errorf("expected non-empty errors[] on not-found, got %v", env["errors"])
	}
}

func TestSymbolCmdExitCode1JSONEnvelope(t *testing.T) {
	// GREEN: runtime errors emit JSON envelope with ok:false and errors[]
	buf := &bytes.Buffer{}
	code := RunSymbol(context.Background(), buf, []string{"--db", "/nonexistent/test.db", "foo"}, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.Bytes())
	}
	if ok, _ := env["ok"].(bool); ok {
		t.Errorf("expected ok=false on runtime error, got ok=true")
	}
	if errs, ok := env["errors"].([]interface{}); !ok || len(errs) == 0 {
		t.Errorf("expected non-empty errors[] on runtime error, got %v", env["errors"])
	}
}
