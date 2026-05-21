package store

import (
	"context"
	"testing"
)

// TestParseErrorsIntegration covers RecordParseError and GetParseErrorsForSnapshot
// against a real migrated DB.
func TestParseErrorsIntegration(t *testing.T) {
	ctx := context.Background()
	db := MustTempDB(t)
	defer db.Close()
	if err := Migrate(ctx, db.DB); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Snapshot 1.
	tx1, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx1: %v", err)
	}
	snap1, err := BeginSnapshot(ctx, tx1, "/repo", "refs/heads/main")
	if err != nil {
		t.Fatalf("begin snapshot 1: %v", err)
	}
	if err := tx1.Commit(); err != nil {
		t.Fatalf("commit tx1: %v", err)
	}

	// Record two parse errors under snap1.
	if err := RecordParseError(ctx, db.DB, snap1, "foo/broken.go", "go/ast", "syntax error"); err != nil {
		t.Fatalf("record parse error 1: %v", err)
	}
	if err := RecordParseError(ctx, db.DB, snap1, "bar/invalid.go", "go/ast", "expected ',', found '}'"); err != nil {
		t.Fatalf("record parse error 2: %v", err)
	}

	// Snapshot 2 (no errors).
	tx2, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx2: %v", err)
	}
	snap2, err := BeginSnapshot(ctx, tx2, "/repo", "refs/heads/main")
	if err != nil {
		t.Fatalf("begin snapshot 2: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatalf("commit tx2: %v", err)
	}

	// Get errors for snap1 — must return exactly 2.
	errs1, err := GetParseErrorsForSnapshot(ctx, db.DB, snap1)
	if err != nil {
		t.Fatalf("get parse errors snap1: %v", err)
	}
	if len(errs1) != 2 {
		t.Errorf("expected 2 errors for snap1, got %d", len(errs1))
	}

	// Get errors for snap2 — must be empty.
	errs2, err := GetParseErrorsForSnapshot(ctx, db.DB, snap2)
	if err != nil {
		t.Fatalf("get parse errors snap2: %v", err)
	}
	if len(errs2) != 0 {
		t.Errorf("expected 0 errors for snap2, got %d", len(errs2))
	}
}
