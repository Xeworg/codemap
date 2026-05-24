package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"codrut/packages/coding-agent/codemap/migrations"
)

// MigrationRunner applies schema migrations in version order.
// It is idempotent: already-applied migrations are skipped.
type MigrationRunner struct {
	db *sql.DB
}

// NewMigrationRunner returns a runner for the given db connection.
func NewMigrationRunner(db *sql.DB) *MigrationRunner {
	return &MigrationRunner{db: db}
}

// Migrate runs all pending migrations in order.
// It is idempotent: already-applied migrations are skipped.
// For migrations that alter tables (0002, 0003), it also pre-checks
// schema state to handle crash-after-alter but before recording safely.
func (m *MigrationRunner) Migrate(ctx context.Context) error {
	migrations := []struct {
		version string
		sql     string
		pre     func(ctx context.Context) (bool, error) // nil = always run
	}{
		{"0001_init", migrations.Init0001, nil},
		{"0002_link_strength", migrations.Init0002, m.checkLinkStrengthMissing},
		{"0003_snapshot_stats", migrations.Init0003, m.checkSnapshotStatsMissing},
		{"0004_graph_cache", migrations.Init0004, nil},
	}

	for _, mig := range migrations {
		applied, err := m.isApplied(ctx, mig.version)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", mig.version, err)
		}
		if applied {
			continue
		}

		// Pre-check for schema-level idempotency: if the migration's
		// schema changes are already present, record it and skip applying.
		if mig.pre != nil {
			present, err := mig.pre(ctx)
			if err != nil {
				return fmt.Errorf("pre-check migration %s: %w", mig.version, err)
			}
			if present {
				if err := m.recordMigration(ctx, mig.version); err != nil {
					return fmt.Errorf("record migration %s: %w", mig.version, err)
				}
				continue
			}
		}

		if err := m.execStatements(ctx, mig.sql); err != nil {
			return fmt.Errorf("apply migration %s: %w", mig.version, err)
		}

		if err := m.recordMigration(ctx, mig.version); err != nil {
			return fmt.Errorf("record migration %s: %w", mig.version, err)
		}
	}
	return nil
}

// checkLinkStrengthMissing returns true only when BOTH the link_strength column
// and both enum-enforcement triggers exist, meaning the migration is fully applied.
func (m *MigrationRunner) checkLinkStrengthMissing(ctx context.Context) (bool, error) {
	col, err := m.columnExists(ctx, "symbol_commits", "link_strength")
	if err != nil || !col {
		return col, err
	}
	trigger1, err := m.triggerExists(ctx, "enforce_link_strength_enum")
	if err != nil || !trigger1 {
		return trigger1, err
	}
	trigger2, err := m.triggerExists(ctx, "enforce_link_strength_enum_update")
	if err != nil || !trigger2 {
		return trigger2, err
	}
	return true, nil
}

// checkSnapshotStatsMissing returns true only when ALL four snapshot stat columns
// exist, meaning the migration is fully applied.
func (m *MigrationRunner) checkSnapshotStatsMissing(ctx context.Context) (bool, error) {
	for _, col := range []string{"files_scanned", "files_parsed", "symbols_found", "parse_errors"} {
		exists, err := m.columnExists(ctx, "snapshots", col)
		if err != nil || !exists {
			return exists, err
		}
	}
	return true, nil
}

// columnExists checks whether a column exists in a table.
func (m *MigrationRunner) columnExists(ctx context.Context, table, column string) (bool, error) {
	rows, err := m.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var cname, ctype string
		var notnull, pk int
		var dflt interface{}
		if err := rows.Scan(&cid, &cname, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if cname == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// triggerExists checks whether a trigger with the given name exists in the DB.
func (m *MigrationRunner) triggerExists(ctx context.Context, name string) (bool, error) {
	var got string
	err := m.db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='trigger' AND name=?`,
		name,
	).Scan(&got)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *MigrationRunner) isApplied(ctx context.Context, version string) (bool, error) {
	// Fast path: query schema_migrations if the table exists.
	var count int
	err := m.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`,
		version,
	).Scan(&count)
	if err == nil {
		return count > 0, nil
	}
	// Table may not exist yet (first migration before 0001_init is applied).
	errStr := err.Error()
	if strings.Contains(errStr, "no such table") || strings.Contains(errStr, "SQL logic error") {
		return false, nil
	}
	return false, err
}

func (m *MigrationRunner) execStatements(ctx context.Context, sqlScript string) error {
	// Pass the full script to SQLite so it can parse multi-statement blocks
	// (including CREATE TRIGGER ... END; which contains internal semicolons).
	// Tolerate "duplicate column name" / "duplicate trigger name" errors since they
	// mean the schema element already exists after a partial prior run.
	_, err := m.db.ExecContext(ctx, sqlScript)
	if err != nil {
		errStr := err.Error()
		if !strings.Contains(errStr, "duplicate column name") &&
			!strings.Contains(errStr, "duplicate trigger name") {
			return err
		}
	}
	return nil
}

func (m *MigrationRunner) recordMigration(ctx context.Context, version string) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES (?, ?)`,
		version,
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// CurrentSchemaVersion returns the highest applied migration version.
// Returns "none" if no migrations have been applied or the schema_migrations table
// does not exist yet (empty/fresh DB).
func (m *MigrationRunner) CurrentSchemaVersion(ctx context.Context) (string, error) {
	var version string
	err := m.db.QueryRowContext(ctx,
		`SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`,
	).Scan(&version)
	if err == sql.ErrNoRows {
		return "none", nil
	}
	if err != nil {
		// Table may not exist on a fresh/empty DB.
		errStr := err.Error()
		if strings.Contains(errStr, "no such table") || strings.Contains(errStr, "SQL logic error") {
			return "none", nil
		}
		return "", err
	}
	return version, nil
}
