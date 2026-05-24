package store

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// TestMigrationMatrix covers migration runner behavior across all DB states:
// empty, partially migrated, re-run/idempotent, and missing schema_migrations table.
func TestMigrationMatrix(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *sql.DB)
		wantErr    bool
		wantTables []string
		wantCount  map[string]int // version -> rows in schema_migrations
	}{
		{
			name: "empty_DB_runs_all_migrations",
			setup: func(t *testing.T, db *sql.DB) {
				// nothing: fresh in-memory DB
			},
			wantErr: false,
			wantTables: []string{
				"schema_migrations", "snapshots", "files", "symbols", "edges",
				"commits", "symbol_commits", "intent_notes", "parse_errors",
			},
			wantCount: map[string]int{
				"0001_init":           1,
				"0002_link_strength":  1,
				"0003_snapshot_stats": 1,
			},
		},
		{
			name: "rerun_is_idempotent",
			setup: func(t *testing.T, db *sql.DB) {
				r := NewMigrationRunner(db)
				if err := r.Migrate(ctx); err != nil {
					t.Fatalf("first migrate: %v", err)
				}
			},
			wantErr: false,
			wantTables: []string{
				"schema_migrations", "snapshots", "files", "symbols", "edges",
				"commits", "symbol_commits", "intent_notes", "parse_errors",
			},
			wantCount: map[string]int{
				"0001_init":           1,
				"0002_link_strength":  1,
				"0003_snapshot_stats": 1,
			},
		},
		{
			name: "missing_0003_record_applies_0003_only",
			setup: func(t *testing.T, db *sql.DB) {
				// Run all migrations, then manually delete 0003 record to simulate
				// a crash between record insert and completion.
				r := NewMigrationRunner(db)
				if err := r.Migrate(ctx); err != nil {
					t.Fatalf("setup migrate: %v", err)
				}
				_, err := db.ExecContext(ctx,
					`DELETE FROM schema_migrations WHERE version='0003_snapshot_stats'`)
				if err != nil {
					t.Fatalf("delete 0003 record: %v", err)
				}
			},
			wantErr: false,
			wantCount: map[string]int{
				"0001_init":          1,
				"0002_link_strength": 1,
				// Pre-check detects files_scanned column already exists and records
				// the migration without re-running ALTER. Correct recovery behavior.
				"0003_snapshot_stats": 1,
			},
		},
		{
			name: "missing_0002_record_applies_0002_and_0003",
			setup: func(t *testing.T, db *sql.DB) {
				r := NewMigrationRunner(db)
				if err := r.Migrate(ctx); err != nil {
					t.Fatalf("setup migrate: %v", err)
				}
				_, err := db.ExecContext(ctx,
					`DELETE FROM schema_migrations WHERE version='0002_link_strength'`)
				if err != nil {
					t.Fatalf("delete 0002 record: %v", err)
				}
			},
			wantErr: false,
			wantCount: map[string]int{
				"0001_init":           1,
				"0002_link_strength":  1,
				"0003_snapshot_stats": 1,
			},
		},
		{
			name: "missing_0001_record_replays_from_0001",
			setup: func(t *testing.T, db *sql.DB) {
				r := NewMigrationRunner(db)
				if err := r.Migrate(ctx); err != nil {
					t.Fatalf("setup migrate: %v", err)
				}
				_, err := db.ExecContext(ctx,
					`DELETE FROM schema_migrations WHERE version='0001_init'`)
				if err != nil {
					t.Fatalf("delete 0001 record: %v", err)
				}
			},
			wantErr: false,
			wantCount: map[string]int{
				"0001_init":           1,
				"0002_link_strength":  1,
				"0003_snapshot_stats": 1,
			},
		},
		{
			name: "missing_schema_migrations_table_falls_back_to_0001",
			setup: func(t *testing.T, db *sql.DB) {
				// DB with schema but no schema_migrations table (simulates corruption).
				// Run 0001 first to get tables, then drop schema_migrations.
				r := NewMigrationRunner(db)
				if err := r.Migrate(ctx); err != nil {
					t.Fatalf("setup migrate: %v", err)
				}
				_, err := db.ExecContext(ctx, `DROP TABLE schema_migrations`)
				if err != nil {
					t.Fatalf("drop schema_migrations: %v", err)
				}
			},
			wantErr: false,
			wantCount: map[string]int{
				"0001_init":           1,
				"0002_link_strength":  1,
				"0003_snapshot_stats": 1,
			},
		},
		{
			name: "0002_partial_column_only_no_triggers_reapplies",
			setup: func(t *testing.T, db *sql.DB) {
				r := NewMigrationRunner(db)
				if err := r.Migrate(ctx); err != nil {
					t.Fatalf("setup migrate: %v", err)
				}
				_, _ = db.ExecContext(ctx,
					`DELETE FROM schema_migrations WHERE version IN ('0002_link_strength','0003_snapshot_stats')`)
				// Column exists but triggers are gone.
				_, _ = db.ExecContext(ctx, `DROP TRIGGER IF EXISTS enforce_link_strength_enum`)
				_, _ = db.ExecContext(ctx, `DROP TRIGGER IF EXISTS enforce_link_strength_enum_update`)
			},
			wantErr: false,
			wantCount: map[string]int{
				"0001_init":           1,
				"0002_link_strength":  1,
				"0003_snapshot_stats": 1,
			},
		},
		{
			name: "0003_partial_one_column_only_reapplies",
			setup: func(t *testing.T, db *sql.DB) {
				r := NewMigrationRunner(db)
				if err := r.Migrate(ctx); err != nil {
					t.Fatalf("setup migrate: %v", err)
				}
				_, _ = db.ExecContext(ctx,
					`DELETE FROM schema_migrations WHERE version='0003_snapshot_stats'`)
				// Drop three of the four columns added by 0003.
				_, _ = db.ExecContext(ctx, `ALTER TABLE snapshots DROP COLUMN parse_errors`)
				_, _ = db.ExecContext(ctx, `ALTER TABLE snapshots DROP COLUMN symbols_found`)
				_, _ = db.ExecContext(ctx, `ALTER TABLE snapshots DROP COLUMN files_parsed`)
			},
			wantErr: false,
			wantCount: map[string]int{
				"0001_init":           1,
				"0002_link_strength":  1,
				"0003_snapshot_stats": 1,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })

			tc.setup(t, db)

			r := NewMigrationRunner(db)
			err = r.Migrate(ctx)

			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check tables exist.
			for _, name := range tc.wantTables {
				var got string
				err := db.QueryRowContext(ctx,
					`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name,
				).Scan(&got)
				if err != nil {
					t.Errorf("table %s not found: %v", name, err)
				}
			}

			// Check migration records.
			for ver, want := range tc.wantCount {
				var got int
				err := db.QueryRowContext(ctx,
					`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, ver,
				).Scan(&got)
				if err != nil && err != sql.ErrNoRows {
					t.Errorf("query migration record %s: %v", ver, err)
					continue
				}
				if got != want {
					t.Errorf("schema_migrations[%s]: want %d, got %d", ver, want, got)
				}
			}
		})
	}
}

// TestMigrationRunnerCurrentVersion covers CurrentSchemaVersion across states.
func TestMigrationRunnerCurrentVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("empty_DB_returns_none", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })

		r := NewMigrationRunner(db)
		ver, err := r.CurrentSchemaVersion(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if ver != "none" {
			t.Errorf("want none, got %s", ver)
		}
	})

	t.Run("returns_latest_after_migrate", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })

		r := NewMigrationRunner(db)
		if err := r.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		ver, err := r.CurrentSchemaVersion(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if ver != "0004_graph_cache" {
			t.Errorf("want 0004_graph_cache, got %s", ver)
		}
	})
}

// TestMigrationLinkStrengthEnumTriggers verifies the enum enforcement triggers.
func TestMigrationLinkStrengthEnumTriggers(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	r := NewMigrationRunner(db)
	if err := r.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	// Insert a commit first (needed by symbol_commits FK).
	_, err = db.ExecContext(ctx,
		`INSERT INTO commits(hash, author, date, message) VALUES ('abc123','a','2026-01-01','c')`)
	if err != nil {
		t.Fatal(err)
	}

	// Insert a symbol (needs a file, which needs a snapshot).
	var snapshotID int64
	err = db.QueryRowContext(ctx,
		`INSERT INTO snapshots(repo_root, head_ref, created_at) VALUES ('.','ref','2026-01-01T00:00:00Z') RETURNING id`,
	).Scan(&snapshotID)
	if err != nil {
		t.Fatal(err)
	}

	var fileID int64
	err = db.QueryRowContext(ctx,
		`INSERT INTO files(path, language, hash, snapshot_id) VALUES ('a.go','go','x',?) RETURNING id`,
		snapshotID,
	).Scan(&fileID)
	if err != nil {
		t.Fatal(err)
	}

	var symbolID int64
	err = db.QueryRowContext(ctx,
		`INSERT INTO symbols(file_id, name, kind, signature, start_line, end_line)
		 VALUES (?, 'Foo', 'func', 'func()', 1, 1) RETURNING id`,
		fileID,
	).Scan(&symbolID)
	if err != nil {
		t.Fatal(err)
	}

	// Valid link_strength values should succeed.
	valid := []string{"strong", "medium", "weak"}
	for _, v := range valid {
		_, err = db.ExecContext(ctx,
			`INSERT INTO symbol_commits(symbol_id, commit_hash, change_type, link_strength)
			 VALUES (?, 'abc123', 'modify', ?)`,
			symbolID, v,
		)
		if err != nil {
			t.Errorf("link_strength=%q: want success, got %v", v, err)
		}
		// Clean up for next iteration.
		_, _ = db.ExecContext(ctx, `DELETE FROM symbol_commits WHERE symbol_id=?`, symbolID)
	}

	// Invalid link_strength should be rejected.
	invalid := []string{"INVALID", "Strong", "", "STRONG"}
	for _, v := range invalid {
		_, err = db.ExecContext(ctx,
			`INSERT INTO symbol_commits(symbol_id, commit_hash, change_type, link_strength)
			 VALUES (?, 'abc123', 'modify', ?)`,
			symbolID, v,
		)
		if err == nil {
			t.Errorf("link_strength=%q: want rejection, got nil error", v)
		}
		// Also check the error message mentions the trigger constraint.
		if err != nil && !strings.Contains(err.Error(), "link_strength") {
			t.Errorf("link_strength=%q: got error but unexpected message: %v", v, err)
		}
	}
}

// TestMigrationRunnerIsAppliedErrors tests isApplied handling of unexpected errors.
func TestMigrationRunnerIsAppliedErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("closed_db_returns_error_from_isApplied", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		// Close immediately so subsequent ops fail.
		_ = db.Close()

		r := NewMigrationRunner(db)
		_, err = r.isApplied(ctx, "0001_init")
		if err == nil {
			t.Error("want error on closed db, got nil")
		}
	})
}
