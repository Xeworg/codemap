package store

import (
	"context"
	"database/sql"
	"fmt"
)

// SymbolEdge represents a directed edge between two symbols in the code graph.
type SymbolEdge struct {
	FromSymbolID int64
	ToSymbolID   int64
	EdgeType     string
}

// GetSymbolEdges returns all edges incident to a given symbol (inbound + outbound).
func GetSymbolEdges(ctx context.Context, db *sql.DB, symbolID int64) ([]SymbolEdge, error) {
	query := `
		SELECT from_symbol_id, to_symbol_id, edge_type
		FROM edges
		WHERE from_symbol_id = ? OR to_symbol_id = ?
	`
	rows, err := db.QueryContext(ctx, query, symbolID, symbolID)
	if err != nil {
		return nil, fmt.Errorf("GetSymbolEdges query: %w", err)
	}
	defer rows.Close()

	var edges []SymbolEdge
	for rows.Next() {
		var e SymbolEdge
		if err := rows.Scan(&e.FromSymbolID, &e.ToSymbolID, &e.EdgeType); err != nil {
			return nil, fmt.Errorf("scan edge row: %w", err)
		}
		edges = append(edges, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return edges, nil
}

// GetInboundEdges returns all edges that point TO a given symbol (inbound only).
func GetInboundEdges(ctx context.Context, db *sql.DB, symbolID int64) ([]SymbolEdge, error) {
	query := `
		SELECT from_symbol_id, to_symbol_id, edge_type
		FROM edges
		WHERE to_symbol_id = ?
	`
	rows, err := db.QueryContext(ctx, query, symbolID)
	if err != nil {
		return nil, fmt.Errorf("GetInboundEdges query: %w", err)
	}
	defer rows.Close()

	var edges []SymbolEdge
	for rows.Next() {
		var e SymbolEdge
		if err := rows.Scan(&e.FromSymbolID, &e.ToSymbolID, &e.EdgeType); err != nil {
			return nil, fmt.Errorf("scan edge row: %w", err)
		}
		edges = append(edges, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return edges, nil
}

// GetSymbolsWithZeroInboundEdges returns symbols from the most recent snapshot
// that have no incoming edges (i.e., nothing references them).
func GetSymbolsWithZeroInboundEdges(ctx context.Context, db *sql.DB) ([]SymbolRow, error) {
	meta, err := GetLatestSnapshotMeta(ctx, db)
	if err != nil || meta.SnapshotID == 0 {
		return nil, nil
	}
	// Subquery: symbols that DO have inbound edges.
	// Outer query: symbols in current snapshot that are NOT in that set.
	query := `
		SELECT s.id, s.name, s.kind, s.signature, s.start_line, s.end_line, f.path
		FROM symbols s
		JOIN files f ON f.id = s.file_id
		WHERE f.snapshot_id = ?
		  AND s.id NOT IN (
		      SELECT DISTINCT e.to_symbol_id
		      FROM edges e
		  )
		ORDER BY s.name, f.path
	`
	rows, err := db.QueryContext(ctx, query, meta.SnapshotID)
	if err != nil {
		return nil, fmt.Errorf("GetSymbolsWithZeroInboundEdges query: %w", err)
	}
	defer rows.Close()

	var symbols []SymbolRow
	for rows.Next() {
		var sym SymbolRow
		if err := rows.Scan(&sym.ID, &sym.Name, &sym.Kind, &sym.Signature,
			&sym.StartLine, &sym.EndLine, &sym.File); err != nil {
			return nil, fmt.Errorf("scan symbol row: %w", err)
		}
		symbols = append(symbols, sym)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return symbols, nil
}

// GetAllSymbols returns all symbols from the most recent snapshot, ordered deterministically.
func GetAllSymbols(ctx context.Context, db *sql.DB) ([]SymbolRow, error) {
	meta, err := GetLatestSnapshotMeta(ctx, db)
	if err != nil || meta.SnapshotID == 0 {
		return nil, nil
	}
	query := `
		SELECT s.id, s.name, s.kind, s.signature, s.start_line, s.end_line, f.path
		FROM symbols s
		JOIN files f ON f.id = s.file_id
		WHERE f.snapshot_id = ?
		ORDER BY s.name, f.path
	`
	rows, err := db.QueryContext(ctx, query, meta.SnapshotID)
	if err != nil {
		return nil, fmt.Errorf("GetAllSymbols query: %w", err)
	}
	defer rows.Close()

	var symbols []SymbolRow
	for rows.Next() {
		var sym SymbolRow
		if err := rows.Scan(&sym.ID, &sym.Name, &sym.Kind, &sym.Signature,
			&sym.StartLine, &sym.EndLine, &sym.File); err != nil {
			return nil, fmt.Errorf("scan symbol row: %w", err)
		}
		symbols = append(symbols, sym)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return symbols, rows.Err()
}

// GetSymbolByID returns a symbol row by its ID, or nil if not found.
func GetSymbolByID(ctx context.Context, db *sql.DB, id int64) (*SymbolRow, error) {
	query := `
		SELECT s.id, s.name, s.kind, s.signature, s.start_line, s.end_line, f.path
		FROM symbols s
		JOIN files f ON f.id = s.file_id
		WHERE s.id = ?
		LIMIT 1
	`
	var sym SymbolRow
	err := db.QueryRowContext(ctx, query, id).Scan(
		&sym.ID, &sym.Name, &sym.Kind, &sym.Signature,
		&sym.StartLine, &sym.EndLine, &sym.File,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetSymbolByID: %w", err)
	}
	return &sym, nil
}
