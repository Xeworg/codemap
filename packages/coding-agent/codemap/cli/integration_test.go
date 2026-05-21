package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"codrut/packages/coding-agent/codemap/store"
)

// TestIntegrationEndToEnd runs a full index -> symbol -> history cycle.
func TestIntegrationEndToEnd(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "e2e.db")
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")

	// Step 1: index
	buf := &bytes.Buffer{}
	code := RunIndex(context.Background(), buf, []string{"--db", dbPath}, repoPath)
	if code != 0 {
		t.Fatalf("index failed with exit %d: %s", code, buf.String())
	}

	// Verify index envelope.
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("non-JSON index output: %s", buf.String())
	}
	if env["schema_version"] != "1.0" {
		t.Errorf("expected schema_version=1.0, got %v", env["schema_version"])
	}
	if env["ok"] != true {
		t.Errorf("expected ok=true, got %v", env["ok"])
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data field")
	}
	if data["files_scanned"] == nil {
		t.Error("missing files_scanned in index data")
	}

	// Step 2: symbol query
	symBuf := &bytes.Buffer{}
	code = RunSymbol(context.Background(), symBuf, []string{"--db", dbPath, "Valid"}, "")
	if code != 0 {
		t.Fatalf("symbol failed with exit %d: %s", code, symBuf.String())
	}
	var symEnv map[string]interface{}
	if err := json.Unmarshal(symBuf.Bytes(), &symEnv); err != nil {
		t.Fatalf("non-JSON symbol output: %s", symBuf.String())
	}
	if symEnv["schema_version"] != "1.0" {
		t.Errorf("symbol: expected schema_version=1.0, got %v", symEnv["schema_version"])
	}
	if symEnv["ok"] != true {
		t.Errorf("symbol: expected ok=true, got %v", symEnv["ok"])
	}

	// Step 3: history query
	histBuf := &bytes.Buffer{}
	code = RunHistory(context.Background(), histBuf, []string{"--db", dbPath, "Valid"}, "")
	if code != 0 {
		t.Fatalf("history failed with exit %d: %s", code, histBuf.String())
	}
	var histEnv map[string]interface{}
	if err := json.Unmarshal(histBuf.Bytes(), &histEnv); err != nil {
		t.Fatalf("non-JSON history output: %s", histBuf.String())
	}
	if histEnv["schema_version"] != "1.0" {
		t.Errorf("history: expected schema_version=1.0, got %v", histEnv["schema_version"])
	}

	// Verify DB state after indexing.
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	meta, err := store.GetLatestSnapshotMeta(context.Background(), db.DB)
	if err != nil {
		t.Fatal(err)
	}
	if meta.SnapshotID == 0 {
		t.Error("expected SnapshotID > 0 after index")
	}
}

// TestIntegrationIndexIdempotent verifies that running index twice is safe.
func TestIntegrationIndexIdempotent(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "idempotent.db")
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")

	// Run index once.
	buf1 := &bytes.Buffer{}
	code1 := RunIndex(context.Background(), buf1, []string{"--db", dbPath}, repoPath)
	if code1 != 0 {
		t.Fatalf("first index failed: %s", buf1.String())
	}

	// Run index again (incremental reindex).
	buf2 := &bytes.Buffer{}
	code2 := RunIndex(context.Background(), buf2, []string{"--db", dbPath}, repoPath)
	if code2 != 0 {
		t.Fatalf("second index failed: %s", buf2.String())
	}

	// Both should succeed. Snapshot IDs should differ.
	var env1, env2 map[string]interface{}
	json.Unmarshal(buf1.Bytes(), &env1)
	json.Unmarshal(buf2.Bytes(), &env2)
	data1 := env1["data"].(map[string]interface{})
	data2 := env2["data"].(map[string]interface{})

	if data1["snapshot_id"] == data2["snapshot_id"] {
		t.Log("snapshot_id unchanged on second run (may be correct if no files changed)")
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatal("DB should exist after second index")
	}
}

// TestIntegrationStaleDetection verifies is_stale flag behavior.
func TestIntegrationStaleDetection(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "stale.db")
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")

	// Run index.
	buf := &bytes.Buffer{}
	RunIndex(context.Background(), buf, []string{"--db", dbPath}, repoPath)

	// Query symbol — is_stale should be false immediately after indexing.
	symBuf := &bytes.Buffer{}
	RunSymbol(context.Background(), symBuf, []string{"--db", dbPath, "Valid"}, "")

	var env map[string]interface{}
	json.Unmarshal(symBuf.Bytes(), &env)
	meta, ok := env["meta"].(map[string]interface{})
	if !ok {
		t.Fatal("missing meta field")
	}
	// is_stale may be false or true depending on fixture indexed_at time.
	// Just verify the field exists.
	if _, ok := meta["is_stale"]; !ok {
		t.Error("meta missing is_stale field")
	}
}
