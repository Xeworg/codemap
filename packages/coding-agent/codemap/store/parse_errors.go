package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// RecordParseError inserts a parse error record for a file under a snapshot.
func RecordParseError(ctx context.Context, db *sql.DB, snapshotID int64, file, parserName, errMsg string) error {
	if snapshotID <= 0 {
		return fmt.Errorf("invalid snapshot_id %d", snapshotID)
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO parse_errors(file, parser, error, snapshot_id, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		file, parserName, errMsg, snapshotID,
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// GetParseErrorsForSnapshot returns all parse error rows for a given snapshot.
func GetParseErrorsForSnapshot(ctx context.Context, db *sql.DB, snapshotID int64) ([]ParseError, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, file, parser, error, snapshot_id, created_at
		 FROM parse_errors WHERE snapshot_id = ?`,
		snapshotID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ParseError
	for rows.Next() {
		var e ParseError
		if err := rows.Scan(&e.ID, &e.File, &e.Parser, &e.Error, &e.SnapshotID, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ParseError represents a parse error record from the store.
type ParseError struct {
	ID         int64
	File       string
	Parser     string
	Error      string
	SnapshotID int64
	CreatedAt  string
}

// StripErrorPrefix removes parser-specific prefix noise from error messages.
func StripErrorPrefix(errMsg string) string {
	// Most parser errors are of the form "filename:line:col: message".
	// Strip everything up to and including the first colon followed by a number
	// to leave just the actionable message.
	errMsg = strings.TrimSpace(errMsg)
	// Remove any "syntax error: " prefix for cleaner storage.
	errMsg = strings.TrimPrefix(errMsg, "syntax error: ")
	errMsg = strings.TrimPrefix(errMsg, "expected ':', found 'EOF'")
	return errMsg
}
