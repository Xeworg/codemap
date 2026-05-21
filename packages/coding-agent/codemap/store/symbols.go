package store

import (
	"context"
	"database/sql"
	"fmt"

	"codrut/packages/coding-agent/codemap/indexer"
)

// SymbolRow holds the fields of a symbol record for CLI queries.
type SymbolRow struct {
	ID        int64
	Name      string
	Kind      string
	Signature string
	StartLine int
	EndLine   int
	File      string
}

// ReplaceFileSymbols replaces all symbols and their inbound edges for a given
// file within a snapshot. It removes old rows first (transactionally) and
// inserts new ones, returning the IDs of the new symbol rows.
func ReplaceFileSymbols(ctx context.Context, db *sql.Tx, fileID int64, symbols []indexer.Symbol) ([]int64, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	// Delete old symbol_commits, edges, and symbol rows for this file.
	// We delete symbol_commits explicitly because the FK from symbol_commits
	// to symbols has no ON DELETE CASCADE (and SQLite FK enforcement is OFF).
	_, err := db.ExecContext(ctx,
		`DELETE FROM symbol_commits WHERE symbol_id IN
		 (SELECT id FROM symbols WHERE file_id = ?)`,
		fileID,
	)
	if err != nil {
		return nil, fmt.Errorf("delete symbol_commits: %w", err)
	}
	_, err = db.ExecContext(ctx,
		`DELETE FROM edges WHERE from_symbol_id IN
		 (SELECT id FROM symbols WHERE file_id = ?)`,
		fileID,
	)
	if err != nil {
		return nil, fmt.Errorf("delete edges: %w", err)
	}
	_, err = db.ExecContext(ctx,
		`DELETE FROM symbols WHERE file_id = ?`,
		fileID,
	)
	if err != nil {
		return nil, fmt.Errorf("delete symbols: %w", err)
	}

	// Insert new symbols.
	var ids []int64
	stmt, err := db.PrepareContext(ctx,
		`INSERT INTO symbols(file_id, name, kind, signature, start_line, end_line)
		 VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return nil, fmt.Errorf("prepare insert symbols: %w", err)
	}
	defer stmt.Close()

	for _, sym := range symbols {
		res, err := stmt.ExecContext(ctx,
			fileID, sym.Name, sym.Kind, sym.Signature,
			sym.StartLine, sym.EndLine,
		)
		if err != nil {
			return nil, fmt.Errorf("insert symbol %q: %w", sym.Name, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// UpsertEdge inserts an edge between two symbols if it does not already exist.
func UpsertEdge(ctx context.Context, db *sql.Tx, fromSymbolID, toSymbolID int64, edgeType string) error {
	_, err := db.ExecContext(ctx,
		`INSERT OR IGNORE INTO edges(from_symbol_id, to_symbol_id, edge_type)
		 VALUES (?, ?, ?)`,
		fromSymbolID, toSymbolID, edgeType,
	)
	return err
}

// UpsertFile records a file entry under a snapshot, returning its ID.
func UpsertFile(ctx context.Context, db *sql.Tx, repoRoot, path, lang, hash string, snapshotID int64) (int64, error) {
	res, err := db.ExecContext(ctx,
		`INSERT INTO files(path, language, hash, snapshot_id)
		 VALUES (?, ?, ?, ?)`,
		path, lang, hash, snapshotID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetSymbolByName looks up a symbol by name in the most recent snapshot.
func GetSymbolByName(ctx context.Context, db *sql.DB, name string) (*SymbolRow, error) {
	var m SnapshotMeta
	meta, err := GetLatestSnapshotMeta(ctx, db)
	if err != nil || meta.SnapshotID == 0 {
		return nil, err
	}
	m = meta
	query := `
		SELECT s.id, s.name, s.kind, s.signature, s.start_line, s.end_line, f.path
		FROM symbols s
		JOIN files f ON f.id = s.file_id
		WHERE s.name = ? AND f.snapshot_id = ?
		LIMIT 1
	`
	var sym SymbolRow
	err = db.QueryRowContext(ctx, query, name, m.SnapshotID).Scan(
		&sym.ID, &sym.Name, &sym.Kind, &sym.Signature,
		&sym.StartLine, &sym.EndLine, &sym.File,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetSymbolByName: %w", err)
	}
	return &sym, nil
}

// ReplaceFileSymbolsWithTx is a convenience wrapper that wraps ReplaceFileSymbols
// inside a transaction using the provided DB connection.
func ReplaceFileSymbolsWithTx(ctx context.Context, db *sql.DB, fileID int64, symbols []indexer.Symbol) ([]int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	ids, err := ReplaceFileSymbols(ctx, tx, fileID, symbols)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return ids, tx.Commit()
}
