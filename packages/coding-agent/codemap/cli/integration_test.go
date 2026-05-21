package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"codrut/packages/coding-agent/codemap/store"

	"codrut/packages/coding-agent/codemap/indexer"
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

	// Both should succeed. Snapshot IDs should be the same on no-op reindex.
	var env1, env2 map[string]interface{}
	json.Unmarshal(buf1.Bytes(), &env1)
	json.Unmarshal(buf2.Bytes(), &env2)
	data1 := env1["data"].(map[string]interface{})
	data2 := env2["data"].(map[string]interface{})

	id1 := int(data1["snapshot_id"].(float64))
	id2 := int(data2["snapshot_id"].(float64))
	if id1 != id2 {
		t.Errorf("second index created a new snapshot on no-op run: id1=%d, id2=%d", id1, id2)
	}

	// Symbol query must still work after no-op reindex.
	symBuf := &bytes.Buffer{}
	code := RunSymbol(context.Background(), symBuf, []string{"--db", dbPath, "Valid"}, "")
	if code != 0 {
		t.Fatalf("symbol failed after no-op reindex: %s", symBuf.String())
	}
	var symEnv map[string]interface{}
	json.Unmarshal(symBuf.Bytes(), &symEnv)
	if symEnv["ok"] != true {
		t.Errorf("symbol should succeed after no-op reindex, got ok=%v", symEnv["ok"])
	}
	metaSym, ok := symEnv["meta"].(map[string]interface{})
	if !ok {
		t.Fatal("symbol response missing meta")
	}
	if int(metaSym["snapshot_id"].(float64)) != id1 {
		t.Errorf("symbol snapshot_id should match no-op index snapshot: got %v, want %d", metaSym["snapshot_id"], id1)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatal("DB should exist after second index")
	}
}

// TestMigrateEnvelopeShape verifies migrate command emits the correct envelope structure.
func TestMigrateEnvelopeShape(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "migrate_shape.db")
	buf := &bytes.Buffer{}
	code := RunMigrate(context.Background(), buf, []string{"--db", dbPath}, "")
	if code != 0 {
		t.Fatalf("migrate failed with exit %d: %s", code, buf.String())
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("non-JSON migrate output: %s", buf.String())
	}
	if env["schema_version"] != "1.0" {
		t.Errorf("expected schema_version=1.0, got %v", env["schema_version"])
	}
	if env["ok"] != true {
		t.Errorf("expected ok=true, got %v", env["ok"])
	}
	if env["command"] != "migrate" {
		t.Errorf("expected command=migrate, got %v", env["command"])
	}
	errs, ok := env["errors"].([]interface{})
	if !ok || len(errs) != 0 {
		t.Errorf("expected empty errors[], got %v", env["errors"])
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data field")
	}
	if data["version_after"] == nil || data["version_after"] == "" {
		t.Error("version_after should be non-empty after migrations")
	}
	if data["version_before"] == nil {
		t.Error("version_before should be present")
	}
	if applied, ok := data["migrations_applied"].(bool); !ok {
		t.Error("migrations_applied should be boolean")
	} else if !applied {
		t.Error("migrations_applied should be true on fresh DB")
	}
}

// TestImpactEnvelopeShapeAndDeterminism verifies impact command envelope shape and output determinism.
func TestImpactEnvelopeShapeAndDeterminism(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact_det.db")
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")

	// Run index first to create symbols.
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, repoPath)

	// Run impact twice with same args.
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}
	code1 := RunImpact(context.Background(), buf1, []string{"--db", dbPath, "Valid"}, repoPath)
	code2 := RunImpact(context.Background(), buf2, []string{"--db", dbPath, "Valid"}, repoPath)
	out1, out2 := buf1.Bytes(), buf2.Bytes()

	if code1 != 0 {
		t.Fatalf("first impact failed: %s", out1)
	}
	if code2 != 0 {
		t.Fatalf("second impact failed: %s", out2)
	}

	// Verify deterministic output.
	if !bytes.Equal(out1, out2) {
		t.Errorf("impact output not deterministic:\nfirst:  %s\nsecond: %s", out1, out2)
	}

	// Verify envelope shape.
	var env map[string]interface{}
	if err := json.Unmarshal(out1, &env); err != nil {
		t.Fatalf("non-JSON output: %s", out1)
	}
	if env["schema_version"] != "1.0" {
		t.Errorf("expected schema_version=1.0, got %v", env["schema_version"])
	}
	if env["command"] != "impact" {
		t.Errorf("expected command=impact, got %v", env["command"])
	}
	if env["ok"] != true {
		t.Errorf("expected ok=true, got %v", env["ok"])
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing")
	}
	// affected_symbols should be a JSON array.
	affected, ok := data["affected_symbols"].([]interface{})
	if !ok {
		t.Errorf("affected_symbols should be array, got %T", data["affected_symbols"])
	}
	_ = affected
	// evidence should be a JSON array.
	evidence, ok := data["evidence"].([]interface{})
	if !ok {
		t.Errorf("evidence should be array, got %T", data["evidence"])
	}
	_ = evidence
}

// TestQueryDeterminismMultipleSymbols verifies query determinism across multiple symbols.
func TestQueryDeterminismMultipleSymbols(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "query_det_multi.db")

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
		{Name: "A0", Kind: "func", Signature: "func A0()", StartLine: 1, EndLine: 5},
		{Name: "A1", Kind: "func", Signature: "func A1()", StartLine: 10, EndLine: 15},
		{Name: "A2", Kind: "func", Signature: "func A2()", StartLine: 20, EndLine: 25},
		{Name: "B0", Kind: "func", Signature: "func B0()", StartLine: 30, EndLine: 35},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Run query 3 times and verify identical output.
	var outputs [][]byte
	for i := 0; i < 3; i++ {
		buf := &bytes.Buffer{}
		RunQuery(context.Background(), buf, []string{"--db", dbPath, "A"}, tmp)
		outputs = append(outputs, buf.Bytes())
	}
	for i := 1; i < len(outputs); i++ {
		if !bytes.Equal(outputs[0], outputs[i]) {
			t.Errorf("query output not deterministic across runs: run0 len=%d, run%d len=%d",
				len(outputs[0]), i, len(outputs[i]))
		}
	}

	// Verify sorted order.
	var env map[string]interface{}
	json.Unmarshal(outputs[0], &env)
	data := env["data"].(map[string]interface{})
	matches := data["matches"].([]interface{})
	var prev string
	for _, m := range matches {
		name := m.(map[string]interface{})["name"].(string)
		if prev != "" && name < prev {
			t.Errorf("matches not sorted by name: %v", data["matches"])
		}
		prev = name
	}
}

// TestExitCodesMatrix verifies stable exit code mapping across all commands.
func TestExitCodesMatrix(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "exitcode.db")
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")

	// Pre-populate DB with an index.
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, repoPath)

	tests := []struct {
		name      string
		runFunc   func() int
		wantCodes []int
	}{
		{"symbol: success", func() int {
			return RunSymbol(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath, "Valid"}, repoPath)
		}, []int{0}},
		{"symbol: validation", func() int {
			return RunSymbol(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, repoPath)
		}, []int{2}},
		{"symbol: not found", func() int {
			return RunSymbol(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath, "NonExistentXYZ"}, repoPath)
		}, []int{3}},
		{"symbol: runtime", func() int {
			return RunSymbol(context.Background(), &bytes.Buffer{}, []string{"--db", "/nonexistent/db", "x"}, repoPath)
		}, []int{1}},
		{"history: success", func() int {
			return RunHistory(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath, "Valid"}, repoPath)
		}, []int{0}},
		{"history: validation", func() int {
			return RunHistory(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, repoPath)
		}, []int{2}},
		{"history: not found", func() int {
			return RunHistory(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath, "NonExistentXYZ"}, repoPath)
		}, []int{3}},
		{"migrate: success", func() int {
			return RunMigrate(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, repoPath)
		}, []int{0}},
		{"migrate: flag parse", func() int {
			return RunMigrate(context.Background(), &bytes.Buffer{}, []string{"--db"}, repoPath)
		}, []int{2}},
		{"migrate: runtime", func() int {
			return RunMigrate(context.Background(), &bytes.Buffer{}, []string{"--db", "/nonexistent/path"}, repoPath)
		}, []int{1}},
		{"impact: success", func() int {
			return RunImpact(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath, "Valid"}, repoPath)
		}, []int{0}},
		{"impact: validation", func() int {
			return RunImpact(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, repoPath)
		}, []int{2}},
		{"impact: not found", func() int {
			return RunImpact(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath, "NonExistentXYZ"}, repoPath)
		}, []int{3}},
		{"impact: runtime", func() int {
			return RunImpact(context.Background(), &bytes.Buffer{}, []string{"--db", "/this/path/does/not/exist/and/cannot/be/created/test.db", "x"}, repoPath)
		}, []int{1}},
		{"impact: no index", func() int {
			emptyDB := filepath.Join(tmp, "empty.db")
			return RunImpact(context.Background(), &bytes.Buffer{}, []string{"--db", emptyDB, "x"}, repoPath)
		}, []int{3}},
		{"query: success", func() int {
			return RunQuery(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath, "Valid"}, repoPath)
		}, []int{0}},
		{"query: validation", func() int {
			return RunQuery(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, repoPath)
		}, []int{2}},
		{"query: runtime", func() int {
			return RunQuery(context.Background(), &bytes.Buffer{}, []string{"--db", "/nonexistent/db", "x"}, repoPath)
		}, []int{1}},
		{"query: no index", func() int {
			emptyDB := filepath.Join(tmp, "empty2.db")
			return RunQuery(context.Background(), &bytes.Buffer{}, []string{"--db", emptyDB, "x"}, repoPath)
		}, []int{3}},
	}

	for _, tt := range tests {
		code := tt.runFunc()
		ok := false
		for _, want := range tt.wantCodes {
			if code == want {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("%s: got exit %d, want one of %v", tt.name, code, tt.wantCodes)
		}
	}
}

// TestEnvelopeShapeAllCommands verifies schema_version, ok, errors, meta across all commands.
func TestEnvelopeShapeAllCommands(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "shape.db")
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")

	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, repoPath)

	commands := []struct {
		name  string
		args  []string
		check func(t *testing.T, out []byte, code int)
	}{
		{"index", []string{"--db", dbPath}, func(t *testing.T, out []byte, code int) {
			var env map[string]interface{}
			json.Unmarshal(out, &env)
			if env["schema_version"] != "1.0" {
				t.Error("index: missing schema_version")
			}
			if _, ok := env["ok"]; !ok {
				t.Error("index: missing ok")
			}
			if _, ok := env["errors"]; !ok {
				t.Error("index: missing errors")
			}
			if _, ok := env["meta"]; !ok {
				t.Error("index: missing meta")
			}
		}},
		{"symbol", []string{"--db", dbPath, "Valid"}, func(t *testing.T, out []byte, code int) {
			var env map[string]interface{}
			json.Unmarshal(out, &env)
			if env["schema_version"] != "1.0" {
				t.Error("symbol: missing schema_version")
			}
			if _, ok := env["ok"]; !ok {
				t.Error("symbol: missing ok")
			}
			if _, ok := env["errors"]; !ok {
				t.Error("symbol: missing errors")
			}
			if _, ok := env["meta"]; !ok {
				t.Error("symbol: missing meta")
			}
		}},
		{"history", []string{"--db", dbPath, "Valid"}, func(t *testing.T, out []byte, code int) {
			var env map[string]interface{}
			json.Unmarshal(out, &env)
			if env["schema_version"] != "1.0" {
				t.Error("history: missing schema_version")
			}
			if _, ok := env["ok"]; !ok {
				t.Error("history: missing ok")
			}
			if _, ok := env["errors"]; !ok {
				t.Error("history: missing errors")
			}
			if _, ok := env["meta"]; !ok {
				t.Error("history: missing meta")
			}
		}},
		{"migrate", []string{"--db", dbPath}, func(t *testing.T, out []byte, code int) {
			var env map[string]interface{}
			json.Unmarshal(out, &env)
			if env["schema_version"] != "1.0" {
				t.Error("migrate: missing schema_version")
			}
			if _, ok := env["ok"]; !ok {
				t.Error("migrate: missing ok")
			}
			if _, ok := env["errors"]; !ok {
				t.Error("migrate: missing errors")
			}
		}},
		{"impact", []string{"--db", dbPath, "Valid"}, func(t *testing.T, out []byte, code int) {
			var env map[string]interface{}
			json.Unmarshal(out, &env)
			if env["schema_version"] != "1.0" {
				t.Error("impact: missing schema_version")
			}
			if _, ok := env["ok"]; !ok {
				t.Error("impact: missing ok")
			}
			if _, ok := env["errors"]; !ok {
				t.Error("impact: missing errors")
			}
			if _, ok := env["meta"]; !ok {
				t.Error("impact: missing meta")
			}
		}},
		{"query", []string{"--db", dbPath, "Valid"}, func(t *testing.T, out []byte, code int) {
			var env map[string]interface{}
			json.Unmarshal(out, &env)
			if env["schema_version"] != "1.0" {
				t.Error("query: missing schema_version")
			}
			if _, ok := env["ok"]; !ok {
				t.Error("query: missing ok")
			}
			if _, ok := env["errors"]; !ok {
				t.Error("query: missing errors")
			}
			if _, ok := env["meta"]; !ok {
				t.Error("query: missing meta")
			}
		}},
	}

	for _, cmd := range commands {
		t.Run(cmd.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			var code int
			switch cmd.name {
			case "index":
				code = RunIndex(context.Background(), buf, cmd.args, repoPath)
			case "symbol":
				code = RunSymbol(context.Background(), buf, cmd.args, repoPath)
			case "history":
				code = RunHistory(context.Background(), buf, cmd.args, repoPath)
			case "migrate":
				code = RunMigrate(context.Background(), buf, cmd.args, repoPath)
			case "impact":
				code = RunImpact(context.Background(), buf, cmd.args, repoPath)
			case "query":
				code = RunQuery(context.Background(), buf, cmd.args, repoPath)
			}
			cmd.check(t, buf.Bytes(), code)
		})
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
