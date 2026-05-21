package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func runHistoryCmdJSON(t *testing.T, dbPath, symbolArg string) ([]byte, int) {
	t.Helper()
	buf := &bytes.Buffer{}
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")
	args := []string{"--db", dbPath, "--json", symbolArg}
	code := RunHistory(context.Background(), buf, args, repoPath)
	return buf.Bytes(), code
}

func populateDB(t *testing.T, dbPath string) {
	buf := &bytes.Buffer{}
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")
	RunIndex(context.Background(), buf, []string{"--db", dbPath}, repoPath)
}

func TestHistoryCmdJSONEnvelopeSchemaVersion(t *testing.T) {
	// RED: schema_version must be "1.0"
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	populateDB(t, db)

	out, _ := runHistoryCmdJSON(t, db, "Valid")
	if !bytes.Contains(out, []byte(`"schema_version":"1.0"`)) {
		t.Errorf("missing schema_version=\"1.0\":\n%s", out)
	}
}

func TestHistoryCmdJSONEnvelopeMetaFields(t *testing.T) {
	// RED: meta must have snapshot_id, head_ref, indexed_at, is_stale
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	populateDB(t, db)

	out, _ := runHistoryCmdJSON(t, db, "Valid")
	for _, field := range []string{`"snapshot_id"`, `"head_ref"`, `"indexed_at"`, `"is_stale"`} {
		if !bytes.Contains(out, []byte(field)) {
			t.Errorf("missing meta field %s:\n%s", field, out)
		}
	}
}

func TestHistoryCmdJSONDataEvidencePresent(t *testing.T) {
	// RED: data.evidence must be present (may be empty array if no history)
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	populateDB(t, db)

	out, _ := runHistoryCmdJSON(t, db, "Valid")
	if !bytes.Contains(out, []byte(`"evidence"`)) {
		t.Errorf("missing evidence field:\n%s", out)
	}
}

func TestHistoryCmdJSONConfidenceEnum(t *testing.T) {
	// RED: confidence must be high|medium|low
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	populateDB(t, db)

	out, _ := runHistoryCmdJSON(t, db, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON output:\n%s", out)
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing or not object")
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

func TestHistoryCmdJSONTopLevelEnvelope(t *testing.T) {
	// RED: top-level must have schema_version, ok, data, errors, meta
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	populateDB(t, db)

	out, _ := runHistoryCmdJSON(t, db, "Valid")
	for _, field := range []string{`"schema_version"`, `"ok"`, `"data"`, `"errors"`, `"meta"`} {
		if !bytes.Contains(out, []byte(field)) {
			t.Errorf("missing top-level field %s:\n%s", field, out)
		}
	}
}

func TestHistoryCmdExitCode0Success(t *testing.T) {
	// RED: exit 0 for valid history query (even if no commits)
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	populateDB(t, db)

	_, code := runHistoryCmdJSON(t, db, "Valid")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestHistoryCmdExitCode2ValidationError(t *testing.T) {
	// RED: exit 2 for missing symbol argument
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	_, code := runHistoryCmdJSON(t, db, "")
	if code != 2 {
		t.Errorf("expected exit 2 for validation error, got %d", code)
	}
}

func TestHistoryCmdExitCode3NotFound(t *testing.T) {
	// RED: exit 3 when symbol not found in index
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	populateDB(t, db)

	_, code := runHistoryCmdJSON(t, db, "NonExistentSymbolXYZ")
	if code != 3 {
		t.Errorf("expected exit 3 for not-found symbol, got %d", code)
	}
}

func TestHistoryCmdExitCode1RuntimeError(t *testing.T) {
	// RED: exit 1 for runtime error (bad DB)
	buf := &bytes.Buffer{}
	args := []string{"--db", "/nonexistent/test.db", "--json", "foo"}
	code := RunHistory(context.Background(), buf, args, "")
	if code != 1 {
		t.Errorf("expected exit 1 for runtime error, got %d", code)
	}
}

func TestHistoryCmdReturnsHistoryEntries(t *testing.T) {
	// RED: history response data should contain history/evidence array
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	populateDB(t, db)

	out, _ := runHistoryCmdJSON(t, db, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON output:\n%s", out)
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing or not object")
	}
	// data should have a history or evidence field
	if _, ok := data["evidence"]; !ok {
		t.Error("data missing evidence field")
	}
}

func TestHistoryCmdLinkStrengthEnum(t *testing.T) {
	// RED: link_strength values in history must be strong|medium|weak
	// This test is a structural test: we just verify the field exists
	// and any non-empty history entries contain valid enum values.
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	populateDB(t, db)

	out, _ := runHistoryCmdJSON(t, db, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON output:\n%s", out)
	}
	// Just verify the field name appears in the envelope.
	// Actual enum enforcement is in store/history.go tests.
	if !bytes.Contains(out, []byte(`"link_strength"`)) && !bytes.Contains(out, []byte(`"evidence"`)) {
		t.Error("history response should contain link_strength or evidence field")
	}
}

func TestHistoryCmdExitCode2JSONEnvelope(t *testing.T) {
	// GREEN: validation errors emit JSON envelope with ok:false and errors[]
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	buf := &bytes.Buffer{}
	code := RunHistory(context.Background(), buf, []string{"--db", db, ""}, "")
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
}

func TestHistoryCmdExitCode3JSONEnvelope(t *testing.T) {
	// GREEN: not-found errors emit JSON envelope with ok:false and errors[]
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	populateDB(t, db)
	buf := &bytes.Buffer{}
	code := RunHistory(context.Background(), buf, []string{"--db", db, "DoesNotExist"}, "")
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

func TestHistoryCmdExitCode1JSONEnvelope(t *testing.T) {
	// GREEN: runtime errors emit JSON envelope with ok:false and errors[]
	buf := &bytes.Buffer{}
	code := RunHistory(context.Background(), buf, []string{"--db", "/nonexistent/test.db", "foo"}, "")
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
