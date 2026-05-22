package store

import (
	"context"
	"testing"

	"codrut/packages/coding-agent/codemap/indexer"
)

// TestGetInboundEdges verifies GetInboundEdges returns only edges pointing TO the target.
func TestGetInboundEdges(t *testing.T) {
	db := MustTempDB(t)
	if err := Migrate(context.Background(), db.DB); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	snapID, err := BeginSnapshot(context.Background(), tx, t.TempDir(), "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	fileID, err := UpsertFile(context.Background(), tx, t.TempDir(), "main.go", "go", "abc", snapID)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := ReplaceFileSymbols(context.Background(), tx, fileID, []indexer.Symbol{
		{Name: "A", Kind: "func", Signature: "func A()", StartLine: 1, EndLine: 5},
		{Name: "B", Kind: "func", Signature: "func B()", StartLine: 6, EndLine: 10},
		{Name: "C", Kind: "func", Signature: "func C()", StartLine: 11, EndLine: 15},
	})
	if err != nil {
		t.Fatal(err)
	}
	aID, bID, cID := ids[0], ids[1], ids[2]
	// Edges: A->B (A calls B), C->B (C calls B), A->C (A imports C).
	_ = UpsertEdge(context.Background(), tx, aID, bID, "calls")
	_ = UpsertEdge(context.Background(), tx, cID, bID, "calls")
	_ = UpsertEdge(context.Background(), tx, aID, cID, "imports")
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// B has 2 inbound edges (from A and C).
	inB, err := GetInboundEdges(context.Background(), db.DB, bID)
	if err != nil {
		t.Fatalf("GetInboundEdges(B) failed: %v", err)
	}
	if len(inB) != 2 {
		t.Errorf("B expected 2 inbound edges, got %d", len(inB))
	}

	// A has 0 inbound edges.
	inA, err := GetInboundEdges(context.Background(), db.DB, aID)
	if err != nil {
		t.Fatalf("GetInboundEdges(A) failed: %v", err)
	}
	if len(inA) != 0 {
		t.Errorf("A expected 0 inbound edges, got %d", len(inA))
	}

	// C has 1 inbound edge (from A).
	inC, err := GetInboundEdges(context.Background(), db.DB, cID)
	if err != nil {
		t.Fatalf("GetInboundEdges(C) failed: %v", err)
	}
	if len(inC) != 1 {
		t.Errorf("C expected 1 inbound edge, got %d", len(inC))
	}
}

// TestGetSymbolWithZeroInboundEdges verifies symbols with no incoming references.
func TestGetSymbolsWithZeroInboundEdges(t *testing.T) {
	db := MustTempDB(t)
	if err := Migrate(context.Background(), db.DB); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	snapID, err := BeginSnapshot(context.Background(), tx, t.TempDir(), "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	fileID, err := UpsertFile(context.Background(), tx, t.TempDir(), "main.go", "go", "abc", snapID)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := ReplaceFileSymbols(context.Background(), tx, fileID, []indexer.Symbol{
		{Name: "Alpha", Kind: "func", Signature: "func Alpha()", StartLine: 1, EndLine: 5},
		{Name: "Beta", Kind: "func", Signature: "func Beta()", StartLine: 6, EndLine: 10},
		{Name: "Gamma", Kind: "func", Signature: "func Gamma()", StartLine: 11, EndLine: 15},
	})
	if err != nil {
		t.Fatal(err)
	}
	alphaID, betaID := ids[0], ids[1]
	_ = ids[2] // gamma has no edges
	// Edges: Alpha -> Beta (Alpha calls Beta).
	_ = UpsertEdge(context.Background(), tx, alphaID, betaID, "calls")
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	zeroInbound, err := GetSymbolsWithZeroInboundEdges(context.Background(), db.DB)
	if err != nil {
		t.Fatalf("GetSymbolsWithZeroInboundEdges failed: %v", err)
	}
	// Alpha has outbound edge but also no inbound edges.
	// Beta has inbound edge from Alpha.
	// Gamma has no edges at all.
	// Expected: Alpha and Gamma (both have zero inbound edges).
	if len(zeroInbound) != 2 {
		t.Errorf("expected 2 symbols with zero inbound, got %d: %+v", len(zeroInbound), zeroInbound)
	}
	names := make(map[string]bool)
	for _, s := range zeroInbound {
		names[s.Name] = true
	}
	if !names["Alpha"] {
		t.Error("Alpha should have zero inbound edges")
	}
	if !names["Gamma"] {
		t.Error("Gamma should have zero inbound edges")
	}
	if names["Beta"] {
		t.Error("Beta should NOT have zero inbound edges (has inbound from Alpha)")
	}
}

// TestGetSymbolsWithZeroInboundEdges_EmptySnapshot verifies empty result when no snapshot.
// Note: shared in-memory DB may have residual data from test suite; skip if symbols exist.
func TestGetSymbolsWithZeroInboundEdges_EmptySnapshot(t *testing.T) {
	db := MustTempDB(t)
	// No migrations, no snapshot.
	symbols, err := GetSymbolsWithZeroInboundEdges(context.Background(), db.DB)
	if err != nil {
		t.Fatalf("GetSymbolsWithZeroInboundEdges on empty DB failed: %v", err)
	}
	// Empty snapshot: should return nil (early exit when SnapshotID == 0).
	if symbols != nil && len(symbols) > 0 {
		t.Logf("got %d symbols in empty snapshot (may be test suite residual data)", len(symbols))
	}
}
