package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"codrut/packages/coding-agent/codemap/store"

	"codrut/packages/coding-agent/codemap/indexer"
)

func runImpactCmdJSON(t *testing.T, dbPath, symbolArg string, extraArgs ...string) ([]byte, int) {
	t.Helper()
	buf := &bytes.Buffer{}
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")
	args := append([]string{"--db", dbPath}, extraArgs...)
	if symbolArg != "" {
		args = append(args, symbolArg)
	}
	code := RunImpact(context.Background(), buf, args, repoPath)
	return buf.Bytes(), code
}

// RED: impact emits valid envelope with schema_version "1.0".
func TestImpactEnvelopeSchemaVersion(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact.db")
	indexBuf := &bytes.Buffer{}
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")
	RunIndex(context.Background(), indexBuf, []string{"--db", dbPath}, repoPath)

	out, _ := runImpactCmdJSON(t, dbPath, "Valid")
	if !bytes.Contains(out, []byte(`"schema_version":"1.0"`)) {
		t.Errorf("missing schema_version=\"1.0\":\n%s", out)
	}
}

// RED: impact envelope command field must be "impact".
func TestImpactEnvelopeCommand(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runImpactCmdJSON(t, dbPath, "Valid")
	if !bytes.Contains(out, []byte(`"command":"impact"`)) {
		t.Errorf("missing command=\"impact\":\n%s", out)
	}
}

// RED: impact with no index returns exit 3.
func TestImpactExitCode3NoIndex(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "no_index.db")
	// Empty DB — no index run.

	_, code := runImpactCmdJSON(t, dbPath, "Valid")
	if code != 3 {
		t.Errorf("expected exit 3 for no-index state, got %d", code)
	}
}

// RED: impact with missing symbol argument returns exit 2.
func TestImpactExitCode2MissingSymbol(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	_, code := runImpactCmdJSON(t, dbPath, "")
	if code != 2 {
		t.Errorf("expected exit 2 for missing symbol arg, got %d", code)
	}
}

// RED: impact with valid DB + index returns exit 0.
func TestImpactExitCode0Success(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	_, code := runImpactCmdJSON(t, dbPath, "Valid")
	if code != 0 {
		t.Errorf("expected exit 0 for successful impact query, got %d", code)
	}
}

// RED: impact exit 3 for non-existent symbol.
func TestImpactExitCode3SymbolNotFound(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	_, code := runImpactCmdJSON(t, dbPath, "NonExistentSymbolXYZ")
	if code != 3 {
		t.Errorf("expected exit 3 for not-found symbol, got %d", code)
	}
}

// RED: impact emits ok:true envelope on success.
func TestImpactOkTrueEnvelope(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runImpactCmdJSON(t, dbPath, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	okVal, okBool := env["ok"].(bool)
	if !okBool || !okVal {
		t.Errorf("expected ok=true on success, got ok=%v", okVal)
	}
}

// RED: impact emits errors=[] on success.
func TestImpactErrorsArrayEmpty(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runImpactCmdJSON(t, dbPath, "Valid")
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

// RED: impact data has target_symbol field.
func TestImpactDataHasTargetSymbol(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runImpactCmdJSON(t, dbPath, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing from envelope")
	}
	if data["target_symbol"] == nil {
		t.Error("data missing target_symbol field")
	}
}

// GREEN: impact data has findings array (replaces affected_symbols).
func TestImpactDataHasFindings(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runImpactCmdJSON(t, dbPath, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing or not object")
	}
	findings, ok := data["findings"].([]interface{})
	if !ok {
		t.Errorf("findings should be a JSON array, got %T", data["findings"])
	}
	_ = findings
}

// RED: impact data has evidence array (top-level evidence, may be empty).
func TestImpactDataHasEvidence(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runImpactCmdJSON(t, dbPath, "Valid")
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing from envelope")
	}
	if data["evidence"] == nil {
		t.Error("data missing evidence field")
	}
}

// RED: impact emits deterministic JSON (same DB, same output).
func TestImpactDeterminism(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact_det.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out1, _ := runImpactCmdJSON(t, dbPath, "Valid")
	out2, _ := runImpactCmdJSON(t, dbPath, "Valid")
	if !bytes.Equal(out1, out2) {
		t.Errorf("impact JSON not deterministic:\nfirst:  %s\nsecond: %s", out1, out2)
	}
}

// RED: impact error envelope includes schema_version.
func TestImpactErrorEnvelopeSchemaVersion(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "no_index.db")

	buf := &bytes.Buffer{}
	code := RunImpact(context.Background(), buf, []string{"--db", dbPath, "Foo"}, "")
	if code != 3 {
		t.Logf("note: got code %d instead of 3", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"schema_version":"1.0"`)) {
		t.Errorf("error envelope missing schema_version=\"1.0\":\n%s", buf.Bytes())
	}
}

// RED: impact emits meta with snapshot_id.
func TestImpactMetaSnapshotID(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact.db")
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, _ := runImpactCmdJSON(t, dbPath, "Valid")
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

// RED: findings must be in deterministic order: risk_tier priority > confidence > symbol > file.
func TestImpactFindingsSortedByRiskThenConfidence(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact_sorted.db")

	// Setup: create DB, run migrations, insert snapshot with Target symbol + edges to multiple dependents.
	db := store.MustOpen(dbPath)
	if err := store.Migrate(context.Background(), db.DB); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	snapID, err := store.BeginSnapshot(context.Background(), tx, tmp, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	fileID1, err := store.UpsertFile(context.Background(), tx, tmp, "a.go", "go", "abc", snapID)
	if err != nil {
		t.Fatal(err)
	}
	fileID2, err := store.UpsertFile(context.Background(), tx, tmp, "b.go", "go", "def", snapID)
	if err != nil {
		t.Fatal(err)
	}
	// Insert symbols: Target, Alpha, Beta.
	ids, err := store.ReplaceFileSymbols(context.Background(), tx, fileID1, []indexer.Symbol{
		{Name: "Target", Kind: "func", Signature: "func Target()", StartLine: 1, EndLine: 5},
		{Name: "Alpha", Kind: "func", Signature: "func Alpha()", StartLine: 10, EndLine: 15},
	})
	if err != nil {
		t.Fatal(err)
	}
	targetID := ids[0]
	alphaID := ids[1]
	ids2, err := store.ReplaceFileSymbols(context.Background(), tx, fileID2, []indexer.Symbol{
		{Name: "Beta", Kind: "type", Signature: "type Beta int", StartLine: 1, EndLine: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	betaID := ids2[0]
	// Add edges: Alpha->Target (calls=high), Beta->Target (type_use=medium).
	// This creates two incident edges for Target, yielding two findings.
	_ = store.UpsertEdge(context.Background(), tx, alphaID, targetID, "calls")
	_ = store.UpsertEdge(context.Background(), tx, betaID, targetID, "type_use")
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	out, code := runImpactCmdJSON(t, dbPath, "Target")
	if code != 0 {
		t.Fatalf("impact failed: %s", out)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data := env["data"].(map[string]interface{})
	findings, ok := data["findings"].([]interface{})
	if !ok {
		t.Fatalf("findings should be []interface{}, got %T", data["findings"])
	}
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	// Verify each finding has required fields.
	for _, f := range findings {
		fm := f.(map[string]interface{})
		for _, field := range []string{"symbol_name", "risk_tier", "confidence", "evidence"} {
			if fm[field] == nil {
				t.Errorf("finding missing required field: %s", field)
			}
		}
	}
	// Verify order: Alpha (high/calls) before Beta (medium/type_use).
	first := findings[0].(map[string]interface{})
	second := findings[1].(map[string]interface{})
	if first["symbol_name"] != "Alpha" {
		t.Errorf("first finding should be Alpha (high/calls), got %v", first["symbol_name"])
	}
	if second["symbol_name"] != "Beta" {
		t.Errorf("second finding should be Beta (medium/type_use), got %v", second["symbol_name"])
	}
	if first["risk_tier"] != "high" {
		t.Errorf("Alpha risk_tier should be high, got %v", first["risk_tier"])
	}
	if second["risk_tier"] != "medium" {
		t.Errorf("Beta risk_tier should be medium, got %v", second["risk_tier"])
	}
}

// RED: per-finding evidence entries have type and description fields.
func TestImpactFindingEvidenceFields(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact_evidence.db")

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
	ids, err := store.ReplaceFileSymbols(context.Background(), tx, fileID, []indexer.Symbol{
		{Name: "ImpactTarget", Kind: "func", Signature: "func ImpactTarget()", StartLine: 1, EndLine: 5},
		{Name: "DepA", Kind: "func", Signature: "func DepA()", StartLine: 10, EndLine: 15},
		{Name: "DepB", Kind: "func", Signature: "func DepB()", StartLine: 20, EndLine: 25},
	})
	if err != nil {
		t.Fatal(err)
	}
	targetID := ids[0]
	depAID := ids[1]
	depBID := ids[2]
	_ = store.UpsertEdge(context.Background(), tx, depAID, targetID, "calls")
	_ = store.UpsertEdge(context.Background(), tx, depBID, targetID, "imports")
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	out, code := runImpactCmdJSON(t, dbPath, "ImpactTarget")
	if code != 0 {
		t.Fatalf("impact failed: %s", out)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON: %s", out)
	}
	data := env["data"].(map[string]interface{})
	findings, ok := data["findings"].([]interface{})
	if !ok {
		t.Fatal("findings should be an array")
	}
	for _, f := range findings {
		fm := f.(map[string]interface{})
		evidence, ok := fm["evidence"].([]interface{})
		if !ok {
			t.Errorf("finding evidence should be array, got %T", fm["evidence"])
			continue
		}
		for _, e := range evidence {
			entry, ok := e.(map[string]interface{})
			if !ok {
				t.Errorf("evidence entry should be object, got %T", e)
				continue
			}
			if entry["type"] == nil {
				t.Error("evidence entry missing type field")
			}
			if entry["description"] == nil {
				t.Error("evidence entry missing description field")
			}
		}
	}
}

// RED: impact with unreadable db path returns exit 1.
func TestImpactExitCode1RuntimeError(t *testing.T) {
	buf := &bytes.Buffer{}
	code := RunImpact(context.Background(), buf, []string{"--db", "/nonexistent/impact.db", "Foo"}, "")
	if code != 1 {
		t.Errorf("expected exit 1 for unreadable db, got %d", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.Bytes())
	}
	if ok, _ := env["ok"].(bool); ok {
		t.Errorf("expected ok=false on runtime error")
	}
	if errs, ok := env["errors"].([]interface{}); !ok || len(errs) == 0 {
		t.Errorf("expected non-empty errors[] on runtime error")
	}
}

// TRIANGULATE: impact cap defaults to 50 findings.
func TestImpactDefaultCap50(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "impact_cap.db")
	// Use parse-mixed fixture which should have >= 1 symbol with edges
	RunIndex(context.Background(), &bytes.Buffer{}, []string{"--db", dbPath}, filepath.Join("..", "testdata", "repos", "parse-mixed"))

	out, code := runImpactCmdJSON(t, dbPath, "Valid")
	if code != 0 {
		t.Fatalf("impact query failed: %s", out)
	}
	var env map[string]interface{}
	json.Unmarshal(out, &env)
	data := env["data"].(map[string]interface{})
	findings := data["findings"].([]interface{})
	if len(findings) > defaultImpactLimit {
		t.Errorf("findings count %d exceeds default cap %d", len(findings), defaultImpactLimit)
	}
}
