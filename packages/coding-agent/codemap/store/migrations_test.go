package store

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestMigrateCreatesExpectedTablesAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	r := NewMigrationRunner(db)

	if err := r.Migrate(ctx); err != nil {
		t.Fatalf("first migrate failed: %v", err)
	}
	if err := r.Migrate(ctx); err != nil {
		t.Fatalf("second migrate failed: %v", err)
	}

	expected := []string{
		"schema_migrations", "snapshots", "files", "symbols", "edges",
		"commits", "symbol_commits", "intent_notes", "parse_errors",
	}
	for _, name := range expected {
		name := name
		t.Run(name, func(t *testing.T) {
			var got string
			err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&got)
			if err != nil {
				t.Fatalf("table %s not found: %v", name, err)
			}
		})
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version='0001_init'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 migration row for 0001_init, got %d", count)
	}

	// 0002_link_strength
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version='0002_link_strength'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 migration row for 0002_link_strength, got %d", count)
	}

	// link_strength column in symbol_commits
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(symbol_commits)`)
	if err != nil {
		t.Fatal(err)
	}
	hasLinkStrength := false
	for rows.Next() {
		var cid int
		var cname, ctype string
		var notnull, pk int
		var dflt interface{}
		_ = rows.Scan(&cid, &cname, &ctype, &notnull, &dflt, &pk)
		if cname == "link_strength" {
			hasLinkStrength = true
			break
		}
	}
	rows.Close()
	if !hasLinkStrength {
		t.Fatal("symbol_commits table missing link_strength column")
	}

	// Triggers exist (IF NOT EXISTS makes them idempotent but they must be present).
	triggers := []string{"enforce_link_strength_enum", "enforce_link_strength_enum_update"}
	for _, name := range triggers {
		var got string
		err := db.QueryRowContext(ctx,
			`SELECT name FROM sqlite_master WHERE type='trigger' AND name=?`, name).Scan(&got)
		if err != nil {
			t.Fatalf("trigger %s not found: %v", name, err)
		}
	}
}
