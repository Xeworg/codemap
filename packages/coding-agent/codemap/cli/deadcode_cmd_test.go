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

// RED Phase 3 — Deadcode Command Tests
// Task 3.1: verify deadcode command wiring, response format, no-mutation.

func runDeadcodeCmdJSON(t *testing.T, dbPath string, extraArgs ...string) ([]byte, int) {
	t.Helper()
	buf := &bytes.Buffer{}
	repoPath := filepath.Join("..", "testdata", "repos", "parse-mixed")
	args := append([]string{"--db", dbPath}, extraArgs...)
	code := RunDeadcode(context.Background(), buf, args, repoPath)
	return buf.Bytes(), code
}

// TestDeadcode_Wiring verifies the deadcode command exists and returns valid JSON.
func TestDeadcode_Wiring(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "deadcode.db")

	// Create DB with snapshot and some symbols.
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
	// Insert a symbol with no edges.
	_, err = store.ReplaceFileSymbols(context.Background(), tx, fileID, []indexer.Symbol{
		{Name: "UnusedFunc", Kind: "func", Signature: "func UnusedFunc()", StartLine: 1, EndLine: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	out, code := runDeadcodeCmdJSON(t, dbPath)
	if code != 0 {
		t.Errorf("expected exit 0, got %d: %s", code, out)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("non-JSON output: %s", out)
	}
	if env["command"] != "deadcode" {
		t.Errorf("expected command=deadcode, got %v", env["command"])
	}
}

// TestDeadcode_FindingsHaveRequiredFields verifies each finding has all required fields.
func TestDeadcode_FindingsHaveRequiredFields(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "deadcode.db")

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
		{Name: "DeadSymbol", Kind: "func", Signature: "func DeadSymbol()", StartLine: 1, EndLine: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	out, _ := runDeadcodeCmdJSON(t, dbPath)
	var env map[string]interface{}
	json.Unmarshal(out, &env)
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing")
	}
	findings, ok := data["findings"].([]interface{})
	if !ok {
		t.Fatalf("findings missing or not array: %T", data["findings"])
	}
	if len(findings) == 0 {
		t.Skip("no deadcode findings expected with no edges in test data")
	}
	for _, f := range findings {
		fm := f.(map[string]interface{})
		for _, field := range []string{"symbol_name", "classification", "suggestion", "confidence", "evidence"} {
			if fm[field] == nil {
				t.Errorf("DeadcodeFinding missing required field: %s", field)
			}
		}
	}
}

// TestDeadcode_ClassificationsValid verifies all classification values are valid.
func TestDeadcode_ClassificationsValid(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "deadcode.db")

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
		{Name: "NoEdges", Kind: "func", Signature: "func NoEdges()", StartLine: 1, EndLine: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	out, _ := runDeadcodeCmdJSON(t, dbPath)
	var env map[string]interface{}
	json.Unmarshal(out, &env)
	data := env["data"].(map[string]interface{})
	findings := data["findings"].([]interface{})
	for _, f := range findings {
		fm := f.(map[string]interface{})
		class := fm["classification"].(string)
		if !IsValidDeadcodeClassification(class) {
			t.Errorf("invalid classification %q", class)
		}
	}
}

// TestDeadcode_SuggestionsValid verifies all suggestion values are valid.
func TestDeadcode_SuggestionsValid(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "deadcode.db")

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
		{Name: "NoEdges", Kind: "func", Signature: "func NoEdges()", StartLine: 1, EndLine: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	out, _ := runDeadcodeCmdJSON(t, dbPath)
	var env map[string]interface{}
	json.Unmarshal(out, &env)
	data := env["data"].(map[string]interface{})
	findings := data["findings"].([]interface{})
	for _, f := range findings {
		fm := f.(map[string]interface{})
		sug := fm["suggestion"].(string)
		if !IsValidDeadcodeSuggestion(sug) {
			t.Errorf("invalid suggestion %q", sug)
		}
	}
}

// TestDeadcode_DeterministicOrder verifies deadcode output is deterministic.
func TestDeadcode_DeterministicOrder(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "deadcode_det.db")

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
		{Name: "NoEdgesA", Kind: "func", Signature: "func NoEdgesA()", StartLine: 1, EndLine: 5},
		{Name: "NoEdgesB", Kind: "type", Signature: "type NoEdgesB int", StartLine: 6, EndLine: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	out1, _ := runDeadcodeCmdJSON(t, dbPath)
	out2, _ := runDeadcodeCmdJSON(t, dbPath)
	if !bytes.Equal(out1, out2) {
		t.Errorf("deadcode output not deterministic:\nfirst:  %s\nsecond: %s", out1, out2)
	}
}

// TestDeadcode_NoMutation verifies the source DB is unchanged after running deadcode.
func TestDeadcode_NoMutation(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "deadcode_mutation.db")

	// Create DB with snapshot and symbols.
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
		{Name: "NoEdges", Kind: "func", Signature: "func NoEdges()", StartLine: 1, EndLine: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Record mtime and size before.
	infoBefore, err := os.Stat(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	sizeBefore := infoBefore.Size()
	mtimeBefore := infoBefore.ModTime()

	// Run deadcode.
	runDeadcodeCmdJSON(t, dbPath)

	// Check after.
	infoAfter, err := os.Stat(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if infoAfter.Size() != sizeBefore {
		t.Errorf("DB size changed: before=%d, after=%d", sizeBefore, infoAfter.Size())
	}
	if !infoAfter.ModTime().Equal(mtimeBefore) {
		t.Errorf("DB mtime changed: before=%v, after=%v", mtimeBefore, infoAfter.ModTime())
	}
}

// TestDeadcode_ExitCode0OnSuccess verifies exit 0 when DB is readable.
func TestDeadcode_ExitCode0OnSuccess(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "deadcode_exit.db")

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
		{Name: "NoEdges", Kind: "func", Signature: "func NoEdges()", StartLine: 1, EndLine: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.Close()

	_, code := runDeadcodeCmdJSON(t, dbPath)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

// TestDeadcode_ExitCode1OnRuntimeError verifies exit 1 for unreadable DB.
func TestDeadcode_ExitCode1OnRuntimeError(t *testing.T) {
	buf := &bytes.Buffer{}
	code := RunDeadcode(context.Background(), buf, []string{"--db", "/nonexistent/deadcode.db"}, "")
	if code != 1 {
		t.Errorf("expected exit 1 for unreadable DB, got %d", code)
	}
}

// TestDeadcode_ExitCode3WhenNoIndex verifies exit 3 when no snapshot exists.
func TestDeadcode_ExitCode3WhenNoIndex(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "no_index.db")

	// Empty DB - no migrations.
	_, code := runDeadcodeCmdJSON(t, dbPath)
	if code != 3 {
		t.Errorf("expected exit 3 for no-index state, got %d", code)
	}
}

func TestClassify_WithInboundEdges_ClassifiesUncertain(t *testing.T) {
	class, _, _ := classifyDeadcode(1, "func", "Private", "pkg/file.go")
	if class != "uncertain" {
		t.Fatalf("expected uncertain, got %q", class)
	}
}

func TestClassify_MainFunc_NoEdges_ClassifiesUncertain(t *testing.T) {
	class, _, _ := classifyDeadcode(0, "func", "main", "cmd/app/main.go")
	if class != "uncertain" {
		t.Fatalf("expected uncertain, got %q", class)
	}
	evidence := deadcodeEvidence(0, "main", "cmd/app/main.go")
	if !hasEvidenceType(evidence, EvidenceImplicitRuntime) {
		t.Fatalf("expected evidence %q", EvidenceImplicitRuntime)
	}
}

func TestClassify_InitFunc_NoEdges_ClassifiesUncertain(t *testing.T) {
	class, _, _ := classifyDeadcode(0, "func", "init", "pkg/init.go")
	if class != "uncertain" {
		t.Fatalf("expected uncertain, got %q", class)
	}
}

func TestClassify_ExportedNoEdges_Uncertain(t *testing.T) {
	class, _, _ := classifyDeadcode(0, "func", "Exported", "pkg/file.go")
	if class != "uncertain" {
		t.Fatalf("expected uncertain, got %q", class)
	}
	evidence := deadcodeEvidence(0, "Exported", "pkg/file.go")
	if !hasEvidenceType(evidence, EvidencePublicAPISurface) {
		t.Fatalf("expected evidence %q", EvidencePublicAPISurface)
	}
}

func TestClassify_PrivateFuncNoEdges_Unused(t *testing.T) {
	class, _, _ := classifyDeadcode(0, "func", "private", "pkg/file.go")
	if class != "unused" {
		t.Fatalf("expected unused, got %q", class)
	}
}

func TestClassify_MethodNoEdges_NotHighConfidenceUnused(t *testing.T) {
	class, _, confidence := classifyDeadcode(0, "method", "method", "pkg/file.go")
	if class == "unused" && confidence == "high" {
		t.Fatalf("method should not be high-confidence unused")
	}
}

func TestEvidence_Composable(t *testing.T) {
	evidence := deadcodeEvidence(0, "main", "cmd/app/main.go")
	if !hasEvidenceType(evidence, EvidenceNoInboundEdges) {
		t.Fatalf("expected evidence %q", EvidenceNoInboundEdges)
	}
	if !hasEvidenceType(evidence, EvidenceImplicitRuntime) {
		t.Fatalf("expected evidence %q", EvidenceImplicitRuntime)
	}
	if hasEvidenceType(evidence, EvidencePublicAPISurface) {
		t.Fatalf("main should not have public_api_surface (lowercase start)")
	}
}

func TestEvidence_PublicAPIComposes(t *testing.T) {
	evidence := deadcodeEvidence(0, "Exported", "pkg/file.go")
	if !hasEvidenceType(evidence, EvidenceNoInboundEdges) {
		t.Fatalf("expected evidence %q", EvidenceNoInboundEdges)
	}
	if !hasEvidenceType(evidence, EvidencePublicAPISurface) {
		t.Fatalf("expected evidence %q", EvidencePublicAPISurface)
	}
	if hasEvidenceType(evidence, EvidenceImplicitRuntime) {
		t.Fatalf("Exported should not have implicit_runtime_entry")
	}
}

func TestEvidence_InboundComposes(t *testing.T) {
	evidence := deadcodeEvidence(1, "Used", "pkg/file.go")
	if !hasEvidenceType(evidence, EvidenceInboundEdges) {
		t.Fatalf("expected evidence %q", EvidenceInboundEdges)
	}
}

func hasEvidenceType(evidence []EvidenceEntry, typ string) bool {
	for _, e := range evidence {
		if e.Type == typ {
			return true
		}
	}
	return false
}
