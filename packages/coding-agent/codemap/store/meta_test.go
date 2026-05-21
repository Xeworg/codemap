package store

import (
	"context"
	"testing"
	"time"
)

func TestGetLatestSnapshotMetaReturnsLatest(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	r := NewMigrationRunner(db)
	if err := r.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	_, _ = db.ExecContext(ctx, `INSERT INTO snapshots(repo_root, head_ref, created_at) VALUES (?, ?, ?)`, "/repo", "abc111", "2026-01-01T00:00:00Z")
	_, _ = db.ExecContext(ctx, `INSERT INTO snapshots(repo_root, head_ref, created_at) VALUES (?, ?, ?)`, "/repo", "def222", time.Now().UTC().Format(time.RFC3339))

	meta, err := GetLatestSnapshotMeta(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if meta.SnapshotID != 2 {
		t.Fatalf("expected latest snapshot id 2, got %d", meta.SnapshotID)
	}
	if meta.HeadRef != "def222" {
		t.Fatalf("expected head_ref def222, got %s", meta.HeadRef)
	}
	if meta.IndexedAt == "" {
		t.Fatal("expected indexed_at to be set")
	}
}

func TestGetLatestSnapshotMetaReturnsZeroValueWhenEmpty(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	r := NewMigrationRunner(db)
	if err := r.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	meta, err := GetLatestSnapshotMeta(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if meta.SnapshotID != 0 || meta.HeadRef != "" || meta.IndexedAt != "" {
		t.Fatalf("expected zero-value meta, got %+v", meta)
	}
}
