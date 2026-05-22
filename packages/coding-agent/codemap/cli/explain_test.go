package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codrut/packages/coding-agent/codemap/indexer"
	"codrut/packages/coding-agent/codemap/store"
)

// RED Phase 3 — Symbol/History Explain-Not-Found Tests
// Tasks 3.1 & 3.2: verify structured explain_not_found in not-found envelopes.

// NOTE: runSymbolCmdJSON and runHistoryCmdJSON are defined in symbol_cmd_test.go
// and history_cmd_test.go respectively. This file uses those helpers.

// TestSymbol_NotFound_ReturnsExplainWithCause verifies that querying a non-existent
// symbol returns ok=false with an explain_not_found payload containing a valid cause.
func TestSymbol_NotFound_ReturnsExplainWithCause(t *testing.T) {
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, code := runSymbolCmdJSON(t, db, "NonExistentSymbolXYZ")
	if code != 3 {
		t.Errorf("expected exit 3 for not-found, got %d", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON output: %s", out)
	}
	if env["ok"] != false {
		t.Errorf("expected ok=false for not-found, got %v", env["ok"])
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing or not object")
	}
	enf, ok := data["explain_not_found"].(map[string]interface{})
	if !ok {
		t.Fatalf("explain_not_found missing from not-found response: %s", out)
	}
	cause, ok := enf["cause"].(string)
	if !ok || cause == "" {
		t.Errorf("explain_not_found.cause missing or empty: %v", enf["cause"])
	}
	// Cause must be one of the valid values.
	validCauses := []string{"stale_index", "name_mismatch", "parse_error", "missing_history_links"}
	found := false
	for _, c := range validCauses {
		if cause == c {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("cause %q not in valid set %v", cause, validCauses)
	}
	// recommended_actions must be non-empty array.
	actions, ok := enf["recommended_actions"].([]interface{})
	if !ok {
		t.Fatalf("recommended_actions missing or not array: %v", enf["recommended_actions"])
	}
	if len(actions) == 0 {
		t.Error("recommended_actions must be non-empty")
	}
}

// TestSymbol_NotFound_CauseIsDeterministic verifies that the same not-found query
// always returns the same cause (no randomness in derivation).
func TestSymbol_NotFound_CauseIsDeterministic(t *testing.T) {
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	var firstCause string
	for i := 0; i < 5; i++ {
		out, _ := runSymbolCmdJSON(t, db, "NonExistentSymbolXYZ")
		var env map[string]interface{}
		json.Unmarshal(out, &env)
		data := env["data"].(map[string]interface{})
		enf := data["explain_not_found"].(map[string]interface{})
		cause := enf["cause"].(string)
		if i == 0 {
			firstCause = cause
		} else if cause != firstCause {
			t.Errorf("iteration %d: cause changed from %q to %q (non-deterministic)", i, firstCause, cause)
		}
	}
}

// TestSymbol_NotFound_RecommendedActionsNonEmpty verifies that recommended_actions
// contains actionable guidance (not generic text).
func TestSymbol_NotFound_RecommendedActionsNonEmpty(t *testing.T) {
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runSymbolCmdJSON(t, db, "NonExistentSymbolXYZ")
	var env map[string]interface{}
	json.Unmarshal(out, &env)
	data := env["data"].(map[string]interface{})
	enf := data["explain_not_found"].(map[string]interface{})
	actions := enf["recommended_actions"].([]interface{})
	if len(actions) == 0 {
		t.Fatal("recommended_actions must be non-empty")
	}
	// Each action should be a non-empty string.
	for i, a := range actions {
		s, ok := a.(string)
		if !ok || strings.TrimSpace(s) == "" {
			t.Errorf("action[%d] is empty or not string: %v", i, a)
		}
	}
}

// TestSymbol_NotFound_HasMetaFields verifies that even on not-found, the envelope
// contains valid meta fields.
func TestSymbol_NotFound_HasMetaFields(t *testing.T) {
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runSymbolCmdJSON(t, db, "NonExistentSymbolXYZ")
	var env map[string]interface{}
	json.Unmarshal(out, &env)
	meta, ok := env["meta"].(map[string]interface{})
	if !ok {
		t.Fatal("meta missing from envelope")
	}
	for _, field := range []string{"snapshot_id", "head_ref", "indexed_at", "is_stale"} {
		if meta[field] == nil {
			t.Errorf("meta missing field: %s", field)
		}
	}
}

// TestSymbol_NotFound_HasSchemaVersion verifies schema_version is present in
// explain-not-found error envelope.
func TestSymbol_NotFound_HasSchemaVersion(t *testing.T) {
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runSymbolCmdJSON(t, db, "NonExistentSymbolXYZ")
	if !bytes.Contains(out, []byte(`"schema_version":"1.0"`)) {
		t.Errorf("explain-not-found envelope missing schema_version: %s", out)
	}
}

// TestHistory_NotFound_ReturnsExplainWithCause verifies that querying a non-existent
// symbol returns ok=false with structured explain_not_found.
func TestHistory_NotFound_ReturnsExplainWithCause(t *testing.T) {
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, code := runHistoryCmdJSON(t, db, "NonExistentSymbolXYZ")
	if code != 3 {
		t.Errorf("expected exit 3 for not-found, got %d", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON output: %s", out)
	}
	if env["ok"] != false {
		t.Errorf("expected ok=false for not-found, got %v", env["ok"])
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing or not object")
	}
	enf, ok := data["explain_not_found"].(map[string]interface{})
	if !ok {
		t.Fatalf("explain_not_found missing from not-found response: %s", out)
	}
	cause, ok := enf["cause"].(string)
	if !ok || cause == "" {
		t.Errorf("explain_not_found.cause missing or empty: %v", enf["cause"])
	}
	validCauses := []string{"stale_index", "name_mismatch", "parse_error", "missing_history_links"}
	found := false
	for _, c := range validCauses {
		if cause == c {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("cause %q not in valid set %v", cause, validCauses)
	}
	actions, ok := enf["recommended_actions"].([]interface{})
	if !ok {
		t.Fatalf("recommended_actions missing or not array: %v", enf["recommended_actions"])
	}
	if len(actions) == 0 {
		t.Error("recommended_actions must be non-empty")
	}
}

// TestHistory_NotFound_RecommendedActionsValid verifies history-specific recommended actions.
func TestHistory_NotFound_RecommendedActionsValid(t *testing.T) {
	tmp := t.TempDir()
	db := filepath.Join(tmp, "test.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", db}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runHistoryCmdJSON(t, db, "NonExistentSymbolXYZ")
	var env map[string]interface{}
	json.Unmarshal(out, &env)
	data := env["data"].(map[string]interface{})
	enf := data["explain_not_found"].(map[string]interface{})
	actions := enf["recommended_actions"].([]interface{})
	if len(actions) == 0 {
		t.Fatal("recommended_actions must be non-empty")
	}
	for i, a := range actions {
		s, ok := a.(string)
		if !ok || strings.TrimSpace(s) == "" {
			t.Errorf("action[%d] is empty or not string: %v", i, a)
		}
	}
}

func TestSymbol_NotFound_StaleIndex(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "stale.db")
	db := store.MustOpen(dbPath)
	defer db.Close()
	if err := store.Migrate(context.Background(), db.DB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	_, err := db.DB.ExecContext(context.Background(), `INSERT INTO snapshots(repo_root, head_ref, created_at) VALUES (?, ?, ?)`, tmp, "HEAD", time.Now().UTC().Add(-48*time.Hour).Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insert stale snapshot: %v", err)
	}

	out, code := runSymbolCmdJSON(t, dbPath, "Nope")
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	var env map[string]interface{}
	_ = json.Unmarshal(out, &env)
	cause := env["data"].(map[string]interface{})["explain_not_found"].(map[string]interface{})["cause"]
	if cause != "stale_index" {
		t.Fatalf("expected stale_index, got %v", cause)
	}
}

func TestSymbol_NotFound_NameMismatch(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "fresh.db")
	db := store.MustOpen(dbPath)
	defer db.Close()
	if err := store.Migrate(context.Background(), db.DB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	_, err := db.DB.ExecContext(context.Background(), `INSERT INTO snapshots(repo_root, head_ref, created_at) VALUES (?, ?, ?)`, tmp, "HEAD", time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	out, _ := runSymbolCmdJSON(t, dbPath, "Nope")
	var env map[string]interface{}
	_ = json.Unmarshal(out, &env)
	cause := env["data"].(map[string]interface{})["explain_not_found"].(map[string]interface{})["cause"]
	if cause != "name_mismatch" {
		t.Fatalf("expected name_mismatch, got %v", cause)
	}
}

func TestHistory_NotFound_MissingHistoryLinks(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "history.db")
	db := store.MustOpen(dbPath)
	defer db.Close()
	ctx := context.Background()
	if err := store.Migrate(ctx, db.DB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	snapID, err := store.BeginSnapshot(ctx, tx, tmp, "HEAD")
	if err != nil {
		t.Fatalf("begin snapshot: %v", err)
	}
	fileID, err := store.UpsertFile(ctx, tx, tmp, "pkg/a.go", "go", "h1", snapID)
	if err != nil {
		t.Fatalf("upsert file: %v", err)
	}
	_, err = store.ReplaceFileSymbols(ctx, tx, fileID, []indexer.Symbol{{Name: "OnlySymbol", Kind: "func", Signature: "func()", StartLine: 1, EndLine: 2}})
	if err != nil {
		t.Fatalf("replace symbols: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	out, code := runHistoryCmdJSON(t, dbPath, "OnlySymbol")
	if code != 0 {
		t.Fatalf("expected exit 0 for symbol without history, got %d; out=%s", code, out)
	}
	var env map[string]interface{}
	_ = json.Unmarshal(out, &env)
	cause := env["data"].(map[string]interface{})["explain_not_found"].(map[string]interface{})["cause"]
	if cause != "missing_history_links" {
		t.Fatalf("expected missing_history_links, got %v", cause)
	}
}
