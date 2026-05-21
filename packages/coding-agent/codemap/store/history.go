package store

import (
	"context"
	"database/sql"
	"fmt"
)

// SymbolHistoryEntry represents a commit-symbol link with strength and metadata.
type SymbolHistoryEntry struct {
	CommitHash   string
	CommitAuthor string
	CommitDate   string
	ChangeType   string
	LinkStrength string // "strong", "medium", or "weak"
}

// GetSymbolHistory returns commit links for a symbol, ordered by link_strength
// descending, then recency descending. It filters out rows with invalid enum values.
func GetSymbolHistory(ctx context.Context, db *sql.DB, symbolID int64) ([]SymbolHistoryEntry, error) {
	query := `
		SELECT sc.commit_hash, c.author, c.date, sc.change_type, sc.link_strength
		FROM symbol_commits sc
		JOIN commits c ON c.hash = sc.commit_hash
		WHERE sc.symbol_id = ?
		  AND sc.link_strength IN ('strong', 'medium', 'weak')
		ORDER BY
		  CASE sc.link_strength
		    WHEN 'strong' THEN 3
		    WHEN 'medium' THEN 2
		    WHEN 'weak'   THEN 1
		  END DESC,
		  c.date DESC
	`
	rows, err := db.QueryContext(ctx, query, symbolID)
	if err != nil {
		return nil, fmt.Errorf("GetSymbolHistory query: %w", err)
	}
	defer rows.Close()

	var entries []SymbolHistoryEntry
	for rows.Next() {
		var e SymbolHistoryEntry
		if err := rows.Scan(&e.CommitHash, &e.CommitAuthor, &e.CommitDate, &e.ChangeType, &e.LinkStrength); err != nil {
			return nil, fmt.Errorf("scan history row: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return entries, rows.Err()
}

// UpsertSymbolCommit inserts or ignores a commit-symbol link with explicit strength.
// Enum enforcement is done by the DB trigger; this function is a thin wrapper.
func UpsertSymbolCommit(ctx context.Context, db *sql.Tx, symbolID int64, commitHash, linkStrength, changeType string) error {
	_, err := db.ExecContext(ctx,
		`INSERT OR IGNORE INTO symbol_commits(commit_hash, symbol_id, change_type, link_strength)
		 VALUES (?, ?, ?, ?)`,
		commitHash, symbolID, changeType, linkStrength,
	)
	return err
}

// EnsureCommit inserts a commit record if it does not already exist.
func EnsureCommit(ctx context.Context, db *sql.Tx, hash, author, date, message string) error {
	_, err := db.ExecContext(ctx,
		`INSERT OR IGNORE INTO commits(hash, author, date, message) VALUES (?, ?, ?, ?)`,
		hash, author, date, message,
	)
	return err
}

// EnsureCommitForDB inserts a commit using a *sql.DB (opens its own transaction).
func EnsureCommitForDB(ctx context.Context, db *sql.DB, hash, author, date, message string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := EnsureCommit(ctx, tx, hash, author, date, message); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
