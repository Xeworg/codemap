package store

import (
	"context"
	"testing"

	"codrut/packages/coding-agent/codemap/indexer"
)

// TestHistoryIntegration covers EnsureCommitForDB, UpsertSymbolCommit, and
// GetSymbolHistory against a real migrated DB.
func TestHistoryIntegration(t *testing.T) {
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

	// Insert file + symbol.
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
	symIDs, err := ReplaceFileSymbolsWithTx(ctx, db.DB, fileID, []indexer.Symbol{
		{Name: "MyFunc", Kind: "func", Signature: "func MyFunc()", StartLine: 1, EndLine: 5},
	})
	if err != nil {
		t.Fatalf("replace symbols: %v", err)
	}
	if len(symIDs) != 1 {
		t.Fatalf("expected 1 symbol id, got %d", len(symIDs))
	}
	symID := symIDs[0]

	// Insert two commits with different dates via EnsureCommitForDB.
	if err := EnsureCommitForDB(ctx, db.DB, "111111", "author1", "2026-01-01T10:00:00Z", "init"); err != nil {
		t.Fatalf("ensure commit 1: %v", err)
	}
	if err := EnsureCommitForDB(ctx, db.DB, "222222", "author2", "2026-04-01T10:00:00Z", "add feature"); err != nil {
		t.Fatalf("ensure commit 2: %v", err)
	}

	// Link symbol to both commits with different strengths.
	tx3, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx3: %v", err)
	}
	if err := UpsertSymbolCommit(ctx, tx3, symID, "111111", "weak", "modify"); err != nil {
		t.Fatalf("upsert symbol commit weak: %v", err)
	}
	if err := UpsertSymbolCommit(ctx, tx3, symID, "222222", "strong", "add"); err != nil {
		t.Fatalf("upsert symbol commit strong: %v", err)
	}
	if err := tx3.Commit(); err != nil {
		t.Fatalf("commit tx3: %v", err)
	}

	// Get history — ordered by strength desc.
	entries, err := GetSymbolHistory(ctx, db.DB, symID)
	if err != nil {
		t.Fatalf("get symbol history: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(entries))
	}

	e := entries[0]
	if e.LinkStrength != "strong" {
		t.Errorf("first entry should be 'strong', got %q", e.LinkStrength)
	}
	if e.CommitHash != "222222" {
		t.Errorf("first entry commit should be 222222, got %s", e.CommitHash)
	}
}

// TestReplaceFileSymbolsDeletesCommits verifies that ReplaceFileSymbols also
// removes symbol_commits for the deleted symbols. Without this, re-indexing
// a file leaves orphaned symbol_commits rows in the DB.
func TestReplaceFileSymbolsDeletesCommits(t *testing.T) {
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

	// Insert file + symbol.
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

	symIDs, err := ReplaceFileSymbolsWithTx(ctx, db.DB, fileID, []indexer.Symbol{
		{Name: "MyFunc", Kind: "func", Signature: "func MyFunc()", StartLine: 1, EndLine: 5},
	})
	if err != nil {
		t.Fatalf("replace symbols: %v", err)
	}
	symID := symIDs[0]

	// Insert a commit and link the symbol to it.
	if err := EnsureCommitForDB(ctx, db.DB, "333333", "author", "2026-05-01T10:00:00Z", "link symbol"); err != nil {
		t.Fatalf("ensure commit: %v", err)
	}
	tx3, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx3: %v", err)
	}
	if err := UpsertSymbolCommit(ctx, tx3, symID, "333333", "strong", "add"); err != nil {
		t.Fatalf("upsert symbol commit: %v", err)
	}
	if err := tx3.Commit(); err != nil {
		t.Fatalf("commit tx3: %v", err)
	}

	// Verify the link exists.
	var countBefore int
	row := db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM symbol_commits WHERE symbol_id = ?", symID)
	if err := row.Scan(&countBefore); err != nil {
		t.Fatalf("count before: %v", err)
	}
	if countBefore != 1 {
		t.Fatalf("expected 1 symbol_commit before re-index, got %d", countBefore)
	}

	// Re-index: call ReplaceFileSymbols again with a different symbol set.
	_, err = ReplaceFileSymbolsWithTx(ctx, db.DB, fileID, []indexer.Symbol{
		{Name: "NewFunc", Kind: "func", Signature: "func NewFunc()", StartLine: 1, EndLine: 5},
	})
	if err != nil {
		t.Fatalf("replace symbols (re-index): %v", err)
	}

	// The old symbol_commits row must be gone.
	var countAfter int
	row = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM symbol_commits WHERE symbol_id = ?", symID)
	if err := row.Scan(&countAfter); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if countAfter != 0 {
		t.Errorf("expected 0 symbol_commits for deleted symbol, got %d (orphaned rows remain)", countAfter)
	}
}

// TestEnsureCommitIdempotent verifies EnsureCommitForDB is safe to call twice.
func TestEnsureCommitIdempotent(t *testing.T) {
	ctx := context.Background()
	db := MustTempDB(t)
	defer db.Close()
	if err := Migrate(ctx, db.DB); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Insert commit.
	if err := EnsureCommitForDB(ctx, db.DB, "aaaaaa", "author", "2026-05-01T10:00:00Z", "fix bug"); err != nil {
		t.Fatalf("ensure commit 1: %v", err)
	}
	// Call again — must not error (INSERT OR IGNORE).
	if err := EnsureCommitForDB(ctx, db.DB, "aaaaaa", "author", "2026-05-01T10:00:00Z", "fix bug"); err != nil {
		t.Errorf("ensure commit idempotent: %v", err)
	}
}
