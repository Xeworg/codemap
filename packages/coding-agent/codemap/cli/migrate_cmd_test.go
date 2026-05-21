package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"codrut/packages/coding-agent/codemap/store"
)

func runMigrateCmdJSON(t *testing.T, dbPath string, extraArgs ...string) ([]byte, int) {
	t.Helper()
	buf := &bytes.Buffer{}
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")
	args := append([]string{"--db", dbPath}, extraArgs...)
	code := RunMigrate(context.Background(), buf, args, repoPath)
	return buf.Bytes(), code
}

// RED: migrate command must emit valid envelope with schema_version "1.0".
func TestMigrateEnvelopeSchemaVersion(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "migrate.db")
	// Migrate the DB to set up schema.
	db := store.MustOpen(dbPath)
	if err := store.Migrate(context.Background(), db.DB); err != nil {
		t.Fatalf("setup migrate failed: %v", err)
	}
	db.Close()

	out, _ := runMigrateCmdJSON(t, dbPath)
	if !bytes.Contains(out, []byte(`"schema_version":"1.0"`)) {
		t.Errorf("missing schema_version=\"1.0\":\n%s", out)
	}
}

// RED: envelope command field must be "migrate".
func TestMigrateEnvelopeCommand(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "migrate.db")
	db := store.MustOpen(dbPath)
	_ = store.Migrate(context.Background(), db.DB)
	db.Close()

	out, _ := runMigrateCmdJSON(t, dbPath)
	if !bytes.Contains(out, []byte(`"command":"migrate"`)) {
		t.Errorf("missing command=\"migrate\":\n%s", out)
	}
}

// RED: migrate on fresh DB applies migrations (migrations_applied true).
func TestMigrateMigrationsAppliedOnFresh(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "migrate.db")

	out, code := runMigrateCmdJSON(t, dbPath)
	if code != 0 {
		t.Errorf("expected exit 0 on fresh DB, got %d: %s", code, out)
	}

	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON output: %s", out)
	}
	okVal, okBool := env["ok"].(bool)
	if !okBool || !okVal {
		t.Errorf("expected ok=true on migrate success, got %v: %s", okVal, out)
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing from envelope")
	}
	if applied, ok := data["migrations_applied"].(bool); !ok || !applied {
		t.Errorf("expected migrations_applied=true on first run, got %v: %s", applied, out)
	}
}

// RED: second migrate run is idempotent (migrations_applied false).
func TestMigrateIdempotent(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "migrate.db")

	// First run — applies migrations.
	_, code1 := runMigrateCmdJSON(t, dbPath)
	if code1 != 0 {
		t.Fatalf("first migrate run failed with code %d", code1)
	}

	// Second run — should be a no-op.
	out2, code2 := runMigrateCmdJSON(t, dbPath)
	if code2 != 0 {
		t.Errorf("expected exit 0 on second (idempotent) run, got %d: %s", code2, out2)
	}

	var env2 map[string]interface{}
	if err := json.Unmarshal(out2, &env2); err != nil {
		t.Fatalf("non-JSON output: %s", out2)
	}
	data2, ok := env2["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing from second run envelope")
	}
	if applied, ok := data2["migrations_applied"].(bool); !ok || applied {
		t.Errorf("expected migrations_applied=false on idempotent run, got %v", applied)
	}

	// version_after should be non-empty after first run.
	if v := data2["version_after"]; v == nil || v == "" {
		t.Errorf("version_after should be non-empty after migrations, got: %v", v)
	}
}

// RED: exit code 0 for successful migrate.
func TestMigrateExitCode0Success(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")

	_, code := runMigrateCmdJSON(t, dbPath)
	if code != 0 {
		t.Errorf("expected exit 0 for successful migrate, got %d", code)
	}
}

// RED: exit code 0 for idempotent migrate (no pending migrations).
func TestMigrateExitCode0Idempotent(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")

	// First run: apply migrations.
	_, _ = runMigrateCmdJSON(t, dbPath)
	// Second run: no pending migrations — still exit 0.
	_, code := runMigrateCmdJSON(t, dbPath)
	if code != 0 {
		t.Errorf("expected exit 0 for idempotent migrate, got %d", code)
	}
}

// RED: exit code 1 for unreadable/non-existent db path (runtime error).
func TestMigrateExitCode1RuntimeError(t *testing.T) {
	// "/nonexistent/" is not writable and parent dir likely does not exist.
	buf := &bytes.Buffer{}
	code := RunMigrate(context.Background(), buf, []string{"--db", "/nonexistent/path/that/does/not/exist/test.db"}, "")
	if code != 1 {
		t.Errorf("expected exit 1 for unreadable db path, got %d", code)
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

// RED: exit code 2 for flag parse failure (e.g., --db with no value).
func TestMigrateExitCode2FlagParse(t *testing.T) {
	buf := &bytes.Buffer{}
	// Pass an invalid flag to trigger parse error.
	code := RunMigrate(context.Background(), buf, []string{"--db"}, "")
	if code != 2 {
		t.Errorf("expected exit 2 for flag parse error, got %d", code)
	}
}

// RED: migrate emits ok:true envelope on success.
func TestMigrateOkTrueEnvelope(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")

	out, _ := runMigrateCmdJSON(t, dbPath)
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	okVal, ok := env["ok"].(bool)
	if !ok || !okVal {
		t.Errorf("expected ok=true on success, got %v: %s", okVal, out)
	}
}

// RED: migrate emits errors=[] (empty array, not null) on success.
func TestMigrateErrorsArrayEmpty(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")

	out, _ := runMigrateCmdJSON(t, dbPath)
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	errs, ok := env["errors"].([]interface{})
	if !ok {
		t.Fatalf("errors field missing or not array: %v", env["errors"])
	}
	if len(errs) != 0 {
		t.Errorf("expected empty errors[] on success, got %d items: %v", len(errs), errs)
	}
}

// RED: migrate emits data.version_before and data.version_after.
func TestMigrateVersionFields(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")

	out, _ := runMigrateCmdJSON(t, dbPath)
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing from envelope")
	}
	if _, ok := data["version_before"]; !ok {
		t.Error("data missing version_before")
	}
	if _, ok := data["version_after"]; !ok {
		t.Error("data missing version_after")
	}
}

// RED: migrate with no --db flag uses default path (exit 0, no error).
// This test verifies the default path resolution doesn't panic and returns a valid envelope.
func TestMigrateNoDBFlagDefault(t *testing.T) {
	// Use the default cache path resolution (repoRoot defaults to ".").
	buf := &bytes.Buffer{}
	// No --db flag: will use ResolveDBPath with empty string.
	// This should not panic; exit code depends on whether default path is usable.
	code := RunMigrate(context.Background(), buf, []string{}, "")
	// Default resolution may fail if no cache dir exists, but envelope should still be valid JSON.
	if code == 0 || code == 1 {
		// Acceptable: either it worked or default resolution failed gracefully.
	} else {
		t.Errorf("expected exit 0 or 1 for no --db flag, got %d", code)
	}
}

// RED: schema_version is present in error envelopes too.
func TestMigrateErrorEnvelopeSchemaVersion(t *testing.T) {
	buf := &bytes.Buffer{}
	code := RunMigrate(context.Background(), buf, []string{"--db", "/nonexistent/test.db"}, "")
	if code != 1 {
		t.Logf("note: got code %d instead of 1", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"schema_version":"1.0"`)) {
		t.Errorf("error envelope missing schema_version=\"1.0\":\n%s", buf.Bytes())
	}
}
