package store

import (
	"context"
	"testing"

	"codrut/packages/coding-agent/codemap/indexer"
)

// TestUpsertEdges_Basic verifies that edges are persisted and can be queried.
func TestUpsertEdges_Basic(t *testing.T) {
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
		{Name: "Caller", Kind: "func", Signature: "func()", StartLine: 1, EndLine: 3},
		{Name: "Helper", Kind: "func", Signature: "func()", StartLine: 5, EndLine: 7},
	})
	if err != nil {
		t.Fatal(err)
	}
	callerID, helperID := ids[0], ids[1]

	// Insert an edge: Caller → Helper.
	err = UpsertEdges(context.Background(), tx, []ResolvedEdge{
		{FromSymbolID: callerID, ToSymbolID: helperID, EdgeType: "call"},
	})
	if err != nil {
		t.Fatalf("UpsertEdges failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// Query via existing GetInboundEdges helper (uses db.DB, not the committed tx).
	inbound, err := GetInboundEdges(context.Background(), db.DB, helperID)
	if err != nil {
		t.Fatalf("GetInboundEdges failed: %v", err)
	}
	if len(inbound) != 1 {
		t.Errorf("expected 1 inbound edge to Helper, got %d", len(inbound))
	}
	if inbound[0].FromSymbolID != callerID {
		t.Errorf("expected inbound from Caller (%d), got %d", callerID, inbound[0].FromSymbolID)
	}
}

// TestUpsertEdges_SkipsZeroIDs verifies that edges with zero IDs are skipped
// and do not cause errors.
func TestUpsertEdges_SkipsZeroIDs(t *testing.T) {
	db := MustTempDB(t)
	if err := Migrate(context.Background(), db.DB); err != nil {
		t.Fatal(err)
	}

	// Insert file + symbol using ReplaceFileSymbolsWithTx (no open tx left behind).
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
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	ids, err := ReplaceFileSymbolsWithTx(context.Background(), db.DB, fileID, []indexer.Symbol{
		{Name: "A", Kind: "func", Signature: "func()", StartLine: 1, EndLine: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	aID := ids[0]

	// Insert edges in a fresh transaction.
	tx2, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	err = UpsertEdges(context.Background(), tx2, []ResolvedEdge{
		{FromSymbolID: aID, ToSymbolID: 999, EdgeType: "call"}, // to=999 doesn't exist
		{FromSymbolID: 0, ToSymbolID: aID, EdgeType: "call"},   // from=0 → skip
		{FromSymbolID: aID, ToSymbolID: 0, EdgeType: "call"},   // to=0 → skip
	})
	if err != nil {
		t.Fatalf("UpsertEdges should not error on skip: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatal(err)
	}

	// A has no inbound edges (the only inserted edge pointed to 999, not to A).
	inbound, err := GetInboundEdges(context.Background(), db.DB, aID)
	if err != nil {
		t.Fatalf("GetInboundEdges failed: %v", err)
	}
	if len(inbound) != 0 {
		t.Errorf("expected 0 inbound edges to A, got %d", len(inbound))
	}
}

// TestUpsertEdges_ReplacesOldEdges verifies that re-indexing a file
// (same file path across snapshots) replaces old edges with new ones.
// This mirrors the CLI index pipeline where UpsertFile creates a new file
// row per snapshot and ReplaceFileSymbols clears+re-inserts symbols for
// that file, leaving previous snapshot symbols untouched.
func TestUpsertEdges_ReplacesOldEdges(t *testing.T) {
	db := MustTempDB(t)
	if err := Migrate(context.Background(), db.DB); err != nil {
		t.Fatal(err)
	}

	// Snapshot 1: Caller → OldHelper.
	tx1, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	snapID1, err := BeginSnapshot(context.Background(), tx1, t.TempDir(), "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	fileID1, err := UpsertFile(context.Background(), tx1, t.TempDir(), "main.go", "go", "abc", snapID1)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx1.Commit(); err != nil {
		t.Fatal(err)
	}
	ids1, err := ReplaceFileSymbolsWithTx(context.Background(), db.DB, fileID1, []indexer.Symbol{
		{Name: "Caller", Kind: "func", Signature: "func()", StartLine: 1, EndLine: 3},
		{Name: "OldHelper", Kind: "func", Signature: "func()", StartLine: 5, EndLine: 7},
	})
	if err != nil {
		t.Fatal(err)
	}
	tx1b, _ := db.BeginTx(context.Background(), nil)
	_ = UpsertEdges(context.Background(), tx1b, []ResolvedEdge{
		{FromSymbolID: ids1[0], ToSymbolID: ids1[1], EdgeType: "call"},
	})
	_ = tx1b.Commit()

	// Snapshot 2: re-index same file, Caller → NewHelper (OldHelper gone).
	tx2, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	snapID2, err := BeginSnapshot(context.Background(), tx2, t.TempDir(), "HEAD2")
	if err != nil {
		t.Fatal(err)
	}
	fileID2, err := UpsertFile(context.Background(), tx2, t.TempDir(), "main.go", "go", "def", snapID2)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatal(err)
	}
	ids2, err := ReplaceFileSymbolsWithTx(context.Background(), db.DB, fileID2, []indexer.Symbol{
		{Name: "Caller", Kind: "func", Signature: "func()", StartLine: 1, EndLine: 3},
		{Name: "NewHelper", Kind: "func", Signature: "func()", StartLine: 9, EndLine: 11},
	})
	if err != nil {
		t.Fatal(err)
	}
	tx2b, _ := db.BeginTx(context.Background(), nil)
	_ = UpsertEdges(context.Background(), tx2b, []ResolvedEdge{
		{FromSymbolID: ids2[0], ToSymbolID: ids2[1], EdgeType: "call"},
	})
	_ = tx2b.Commit()

	// Direct query: fileID1 has Caller + OldHelper (snapshot 1 symbols intact).
	// fileID2 has Caller + NewHelper (snapshot 2 symbols).
	rows1, err := db.QueryContext(context.Background(),
		`SELECT name FROM symbols WHERE file_id = ? ORDER BY name`, fileID1)
	if err != nil {
		t.Fatal(err)
	}
	var snap1Names []string
	for rows1.Next() {
		var name string
		rows1.Scan(&name)
		snap1Names = append(snap1Names, name)
	}
	rows1.Close()
	if len(snap1Names) != 2 {
		t.Errorf("snapshot 1: expected 2 symbols, got %d: %v", len(snap1Names), snap1Names)
	}

	rows2, err := db.QueryContext(context.Background(),
		`SELECT name FROM symbols WHERE file_id = ? ORDER BY name`, fileID2)
	if err != nil {
		t.Fatal(err)
	}
	var snap2Names []string
	for rows2.Next() {
		var name string
		rows2.Scan(&name)
		snap2Names = append(snap2Names, name)
	}
	rows2.Close()
	if len(snap2Names) != 2 {
		t.Errorf("snapshot 2: expected 2 symbols, got %d: %v", len(snap2Names), snap2Names)
	}

	// Verify NewHelper is in snapshot 2.
	found := false
	for _, n := range snap2Names {
		if n == "NewHelper" {
			found = true
			break
		}
	}
	if !found {
		t.Error("NewHelper should exist in snapshot 2")
	}
	// Verify OldHelper is in snapshot 1 (NOT replaced by snapshot 2's re-index).
	found = false
	for _, n := range snap1Names {
		if n == "OldHelper" {
			found = true
			break
		}
	}
	if !found {
		t.Error("OldHelper should still exist in snapshot 1")
	}
}
