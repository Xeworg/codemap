package store

import (
	"context"
	"testing"
	"time"
)

// TestSnapshotIntegration covers BeginSnapshot, FinalizeSnapshot, and
// GetLatestSnapshotMeta against a real migrated DB.
func TestSnapshotIntegration(t *testing.T) {
	ctx := context.Background()
	db := MustTempDB(t)
	defer db.Close()
	if err := Migrate(ctx, db.DB); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// First snapshot.
	tx1, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx1: %v", err)
	}
	snap1, err := BeginSnapshot(ctx, tx1, "/repo", "refs/heads/main")
	if err != nil {
		t.Fatalf("begin snapshot 1: %v", err)
	}
	if err := FinalizeSnapshot(ctx, tx1, snap1, 10, 8, 42, 1); err != nil {
		t.Fatalf("finalize snapshot 1: %v", err)
	}
	if err := tx1.Commit(); err != nil {
		t.Fatalf("commit tx1: %v", err)
	}

	meta1, err := GetLatestSnapshotMeta(ctx, db.DB)
	if err != nil {
		t.Fatalf("get latest meta after snap1: %v", err)
	}
	if meta1.SnapshotID != snap1 {
		t.Errorf("expected SnapshotID=%d, got %d", snap1, meta1.SnapshotID)
	}
	if meta1.HeadRef != "refs/heads/main" {
		t.Errorf("expected HeadRef 'refs/heads/main', got %q", meta1.HeadRef)
	}
	// snapshot summary fields (files_scanned etc.) are stored in DB but not in SnapshotMeta;
	// FinalizeSnapshot confirmed not to error above; query separately to verify.
	var fs, fp, sf, pe int
	err = db.DB.QueryRowContext(ctx,
		"SELECT files_scanned, files_parsed, symbols_found, parse_errors FROM snapshots WHERE id=?",
		snap1).Scan(&fs, &fp, &sf, &pe)
	if err != nil {
		t.Errorf("could not verify snapshot stats: %v", err)
	}
	if fs != 10 || fp != 8 || sf != 42 || pe != 1 {
		t.Errorf("snapshot stats: expected 10/8/42/1, got %d/%d/%d/%d", fs, fp, sf, pe)
	}

	// Second snapshot — latest meta must reflect snap2.
	tx2, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx2: %v", err)
	}
	snap2, err := BeginSnapshot(ctx, tx2, "/repo", "refs/heads/feature")
	if err != nil {
		t.Fatalf("begin snapshot 2: %v", err)
	}
	if err := FinalizeSnapshot(ctx, tx2, snap2, 12, 12, 55, 0); err != nil {
		t.Fatalf("finalize snapshot 2: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatalf("commit tx2: %v", err)
	}

	meta2, err := GetLatestSnapshotMeta(ctx, db.DB)
	if err != nil {
		t.Fatalf("get latest meta after snap2: %v", err)
	}
	if meta2.SnapshotID != snap2 {
		t.Errorf("expected latest SnapshotID=%d, got %d", snap2, meta2.SnapshotID)
	}
	if meta2.HeadRef != "refs/heads/feature" {
		t.Errorf("expected HeadRef 'refs/heads/feature', got %q", meta2.HeadRef)
	}

	// indexed_at must be a valid RFC3339 timestamp and recent.
	ts, err := time.Parse(time.RFC3339, meta2.IndexedAt)
	if err != nil {
		t.Errorf("IndexedAt is not valid RFC3339: %v", err)
	}
	if time.Since(ts) > 5*time.Second {
		t.Errorf("IndexedAt timestamp is stale: %s", meta2.IndexedAt)
	}
}
