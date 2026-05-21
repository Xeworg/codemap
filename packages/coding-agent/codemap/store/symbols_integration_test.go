package store

import (
	"context"
	"testing"

	"codrut/packages/coding-agent/codemap/indexer"
)

// TestSymbolsIntegration covers UpsertFile and ReplaceFileSymbols against a real
// migrated DB: initial symbols replaced by a different set, old rows removed.
func TestSymbolsIntegration(t *testing.T) {
	ctx := context.Background()
	db := MustTempDB(t)
	defer db.Close()
	if err := Migrate(ctx, db.DB); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Create snapshot.
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	snapID, err := BeginSnapshot(ctx, tx, "/repo", "refs/heads/main")
	if err != nil {
		t.Fatalf("begin snapshot: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Insert file.
	tx2, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx2: %v", err)
	}
	fileID, err := UpsertFile(ctx, tx2, "/repo", "pkg/foo.go", "go", "abc123", snapID)
	if err != nil {
		t.Fatalf("upsert file: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatalf("commit tx2: %v", err)
	}

	// Replace with initial symbol set.
	initial := []indexer.Symbol{
		{Name: "TypeA", Kind: "type", Signature: "type TypeA", StartLine: 1, EndLine: 5},
	}
	ids1, err := ReplaceFileSymbolsWithTx(ctx, db.DB, fileID, initial)
	if err != nil {
		t.Fatalf("replace symbols initial: %v", err)
	}
	if len(ids1) != 1 {
		t.Fatalf("expected 1 id from replace, got %d", len(ids1))
	}

	// Verify initial symbol present.
	sym, err := GetSymbolByName(ctx, db.DB, "TypeA")
	if err != nil {
		t.Fatalf("get TypeA: %v", err)
	}
	if sym == nil {
		t.Fatal("TypeA should be present after initial replace")
	}
	if sym.Kind != "type" {
		t.Errorf("expected kind 'type', got %q", sym.Kind)
	}

	// Replace with new symbol set.
	replacement := []indexer.Symbol{
		{Name: "TypeB", Kind: "type", Signature: "type TypeB", StartLine: 10, EndLine: 20},
		{Name: "FuncX", Kind: "func", Signature: "func FuncX", StartLine: 22, EndLine: 30},
	}
	ids2, err := ReplaceFileSymbolsWithTx(ctx, db.DB, fileID, replacement)
	if err != nil {
		t.Fatalf("replace symbols replacement: %v", err)
	}
	if len(ids2) != 2 {
		t.Fatalf("expected 2 ids after replacement, got %d", len(ids2))
	}

	// Old symbol must be gone.
	old, err := GetSymbolByName(ctx, db.DB, "TypeA")
	if err != nil {
		t.Fatalf("get TypeA after replacement: %v", err)
	}
	if old != nil {
		t.Errorf("TypeA should be gone after replacement, got: %+v", old)
	}

	// New symbols must be present.
	for _, want := range []struct{ name, kind string }{
		{"TypeB", "type"},
		{"FuncX", "func"},
	} {
		sym, err := GetSymbolByName(ctx, db.DB, want.name)
		if err != nil {
			t.Fatalf("get %s: %v", want.name, err)
		}
		if sym == nil {
			t.Errorf("%s missing after replacement", want.name)
			continue
		}
		if sym.Kind != want.kind {
			t.Errorf("expected kind %q for %s, got %q", want.kind, want.name, sym.Kind)
		}
	}
}
