package store

import (
	"context"
	"database/sql"
	"time"
)

// Migrate is a convenience wrapper that creates a MigrationRunner and runs all pending migrations.
func Migrate(ctx context.Context, db *sql.DB) error {
	return NewMigrationRunner(db).Migrate(ctx)
}

// BeginSnapshot creates a new snapshot record and returns its ID.
func BeginSnapshot(ctx context.Context, tx *sql.Tx, repoRoot, headRef string) (int64, error) {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO snapshots(repo_root, head_ref, created_at) VALUES (?, ?, ?)`,
		repoRoot,
		headRef,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// FinalizeSnapshot updates the parse summary fields on a snapshot record.
func FinalizeSnapshot(ctx context.Context, tx *sql.Tx, snapshotID int64, filesScanned, filesParsed, symbolsFound, parseErrors int) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE snapshots
		 SET files_scanned = ?, files_parsed = ?, symbols_found = ?, parse_errors = ?
		 WHERE id = ?`,
		filesScanned, filesParsed, symbolsFound, parseErrors, snapshotID,
	)
	return err
}

// GetSnapshotByID returns the snapshot record for a given ID.
func GetSnapshotByID(ctx context.Context, db *sql.DB, snapshotID int64) (SnapshotMeta, error) {
	return GetLatestSnapshotMeta(ctx, db) // currently only latest is needed
}
