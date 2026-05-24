package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// DefaultCacheTTL is the default time-to-live for cached impact entries (24 hours).
const DefaultCacheTTL = 24 * time.Hour

// SymbolHit represents a single hop in a blast-radius traversal.
type SymbolHit struct {
	SymbolID  int64
	Depth     int
	EdgePath  string // comma-separated list of edge types from source to this symbol
	RiskScore float64
}

// FileHit represents a file-level aggregation of symbol hits.
type FileHit struct {
	FileID    int64
	Depth     int
	EdgeCount int
	RiskScore float64
}

// BlastRadiusQuery runs a recursive CTE to find all symbols reachable from symbolID
// up to maxDepth hops away, filtered optionally by edgeTypes.
// It returns SymbolHit entries sorted by depth then risk score.
func BlastRadiusQuery(ctx context.Context, db *sql.DB, symbolID int64, maxDepth int, edgeTypes []string) ([]SymbolHit, error) {
	if maxDepth < 1 {
		return nil, nil
	}
	if maxDepth > 10 {
		maxDepth = 10
	}

	// Build edge type filter clause for the recursive part.
	var typeFilter string
	var extraArgs []interface{}
	if len(edgeTypes) > 0 {
		placeholders := make([]string, len(edgeTypes))
		for i, et := range edgeTypes {
			placeholders[i] = "?"
			extraArgs = append(extraArgs, et)
		}
		typeFilter = fmt.Sprintf("  AND e.edge_type IN (%s)\n", strings.Join(placeholders, ","))
	}

	query := fmt.Sprintf(`
	WITH RECURSIVE
	  frontier(from_id, depth, path, risk) AS (
		-- Base case: start from the source symbol.
		SELECT s.id, 0, '', 0.0
		FROM symbols s
		WHERE s.id = ?
		UNION ALL
		-- Recursive case: follow outgoing edges and accumulate risk.
		SELECT e.to_symbol_id, f.depth + 1, f.path || ',' || e.edge_type,
		       f.risk +
		       CASE e.edge_type
		         WHEN 'calls' THEN 1.0
		         WHEN 'type_use' THEN 0.7
		         WHEN 'references' THEN 0.7
		         WHEN 'imports' THEN 0.5
		         WHEN 'casts' THEN 0.5
		         WHEN 'subtype' THEN 0.5
		         WHEN 'exports' THEN 0.5
		         ELSE 0.3
		       END
		FROM edges e
		JOIN frontier f ON e.from_symbol_id = f.from_id
		WHERE f.depth < ?
		  %s
	  )
	SELECT f.from_id, f.depth, f.path, f.risk
	FROM frontier f
	WHERE f.depth > 0
	ORDER BY f.depth ASC, f.risk DESC;
	`, typeFilter)

	args := append([]interface{}{symbolID, maxDepth}, extraArgs...)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("BlastRadiusQuery: %w", err)
	}
	defer rows.Close()

	var hits []SymbolHit
	for rows.Next() {
		var h SymbolHit
		var path sql.NullString
		if err := rows.Scan(&h.SymbolID, &h.Depth, &path, &h.RiskScore); err != nil {
			return nil, fmt.Errorf("scan blast radius row: %w", err)
		}
		if path.Valid {
			h.EdgePath = path.String
		}
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("blast radius rows: %w", err)
	}
	return hits, nil
}

// BlastRadiusForFile aggregates symbol-level blast radius to file level.
func BlastRadiusForFile(ctx context.Context, db *sql.DB, fileID int64, maxDepth int) ([]FileHit, error) {
	hits, err := BlastRadiusQuery(ctx, db, fileID, maxDepth, nil)
	if err != nil {
		return nil, err
	}

	// Collect target symbol IDs.
	symIDs := make([]int64, 0, len(hits))
	for _, h := range hits {
		symIDs = append(symIDs, h.SymbolID)
	}
	if len(symIDs) == 0 {
		return nil, nil
	}

	// Map symbol IDs to their file IDs.
	type symFile struct {
		SymbolID int64
		FileID   int64
	}
	var symFiles []symFile
	query := `SELECT id, file_id FROM symbols WHERE id IN (` +
		strings.Join(strings.Split(strings.Repeat("?", len(symIDs)), ""), ",") + `)`
	args := make([]interface{}, len(symIDs))
	for i, id := range symIDs {
		args[i] = id
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("BlastRadiusForFile map: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sf symFile
		if err := rows.Scan(&sf.SymbolID, &sf.FileID); err != nil {
			return nil, fmt.Errorf("scan sym->file: %w", err)
		}
		symFiles = append(symFiles, sf)
	}

	// Build symbolID -> fileID map.
	symToFile := make(map[int64]int64, len(symFiles))
	for _, sf := range symFiles {
		symToFile[sf.SymbolID] = sf.FileID
	}

	// Aggregate by file.
	fileMap := make(map[int64]*FileHit)
	for _, h := range hits {
		fid, ok := symToFile[h.SymbolID]
		if !ok {
			continue
		}
		fh, ok := fileMap[fid]
		if !ok {
			fh = &FileHit{FileID: fid, Depth: h.Depth}
			fileMap[fid] = fh
		}
		fh.EdgeCount++
		if h.RiskScore > fh.RiskScore {
			fh.RiskScore = h.RiskScore
		}
		if h.Depth < fh.Depth {
			fh.Depth = h.Depth
		}
	}

	var result []FileHit
	for _, fh := range fileMap {
		result = append(result, *fh)
	}
	return result, nil
}

// TopNByConnectivity returns the IDs of the top-N symbols with the most inbound edges.
// These are used as "hot symbols" for cache pre-warming.
func TopNByConnectivity(ctx context.Context, db *sql.DB, n int) ([]int64, error) {
	if n < 1 {
		return nil, nil
	}
	query := `
		SELECT s.id
		FROM symbols s
		JOIN files f ON f.id = s.file_id
		WHERE f.snapshot_id = (SELECT MAX(id) FROM snapshots)
		ORDER BY (
			SELECT COUNT(*) FROM edges e WHERE e.to_symbol_id = s.id
		) DESC
		LIMIT ?
	`
	rows, err := db.QueryContext(ctx, query, n)
	if err != nil {
		return nil, fmt.Errorf("TopNByConnectivity: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan top-n id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetCachedImpact returns cached blast-radius hits for a symbol if the cache
// entry is fresh (within ttl). It returns (hits, true, nil) on a cache hit,
// (nil, false, nil) on a miss, and (nil, false, err) on an error.
func GetCachedImpact(ctx context.Context, db *sql.DB, symbolID int64, maxDepth int, ttl time.Duration) ([]SymbolHit, bool, error) {
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	cutoff := time.Now().Add(-ttl).Unix()
	query := `
		SELECT target_symbol_id, depth, edge_path, risk_score
		FROM symbol_impact_cache
		WHERE source_symbol_id = ? AND depth <= ?
		  AND updated_at > ?
	`
	rows, err := db.QueryContext(ctx, query, symbolID, maxDepth, cutoff)
	if err != nil {
		return nil, false, fmt.Errorf("GetCachedImpact: %w", err)
	}
	defer rows.Close()

	var hits []SymbolHit
	hasData := false
	for rows.Next() {
		hasData = true
		var h SymbolHit
		var path sql.NullString
		if err := rows.Scan(&h.SymbolID, &h.Depth, &path, &h.RiskScore); err != nil {
			return nil, false, fmt.Errorf("scan cached hit: %w", err)
		}
		if path.Valid {
			h.EdgePath = path.String
		}
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	if !hasData {
		return nil, false, nil
	}
	return hits, true, nil
}

// WriteCachedImpact stores blast-radius results in the cache, replacing any
// existing entries for the same (source_symbol_id, depth) pair.
func WriteCachedImpact(ctx context.Context, db *sql.DB, symbolID int64, hits []SymbolHit) error {
	if len(hits) == 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WriteCachedImpact begin: %w", err)
	}
	defer tx.Rollback()

	// Group hits by depth so we can delete per depth.
	depthMap := make(map[int][]SymbolHit)
	for _, h := range hits {
		depthMap[h.Depth] = append(depthMap[h.Depth], h)
	}

	now := time.Now().Unix()

	del, err := tx.PrepareContext(ctx,
		`DELETE FROM symbol_impact_cache WHERE source_symbol_id = ? AND depth = ?`)
	if err != nil {
		return fmt.Errorf("WriteCachedImpact prepare del: %w", err)
	}
	defer del.Close()

	ins, err := tx.PrepareContext(ctx,
		`INSERT INTO symbol_impact_cache(source_symbol_id, target_symbol_id, depth, edge_path, risk_score, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("WriteCachedImpact prepare ins: %w", err)
	}
	defer ins.Close()

	// Delete all existing entries for this source (replace strategy).
	_, err = tx.ExecContext(ctx,
		`DELETE FROM symbol_impact_cache WHERE source_symbol_id = ?`, symbolID)
	if err != nil {
		return fmt.Errorf("delete cache: %w", err)
	}
	for _, h := range hits {
		var path interface{}
		if h.EdgePath != "" {
			path = h.EdgePath
		}
		if _, err := ins.ExecContext(ctx, symbolID, h.SymbolID, h.Depth, path, h.RiskScore, now); err != nil {
			return fmt.Errorf("insert cache hit: %w", err)
		}
	}

	return tx.Commit()
}

// InvalidateCacheForSymbol removes all cached impact entries for a symbol
// as source or target, to be called when a symbol's definition or its
// outgoing edges may have changed.
func InvalidateCacheForSymbol(ctx context.Context, db *sql.DB, symbolID int64) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM symbol_impact_cache WHERE source_symbol_id = ? OR target_symbol_id = ?`,
		symbolID, symbolID,
	)
	return err
}

// WarmCacheForSnapshot computes and writes impact cache entries for the top-N
// hot symbols (by inbound connectivity) in the most recent snapshot.
// It uses up to maxConcurrency goroutines and fails softly (logs errors, never
// returns an error that would abort an index run).
// Call this after FinalizeSnapshot + tx.Commit in the index pipeline.
func WarmCacheForSnapshot(ctx context.Context, db *sql.DB, topN int, maxDepth int, maxConcurrency int) {
	if topN < 1 {
		topN = 100
	}
	if maxDepth < 1 {
		maxDepth = 3
	}
	if maxConcurrency < 1 {
		maxConcurrency = 3
	}

	ids, err := TopNByConnectivity(ctx, db, topN)
	if err != nil {
		slog.Warn("WarmCacheForSnapshot: TopNByConnectivity", "err", err)
		return
	}
	if len(ids) == 0 {
		return
	}

	// Bounded concurrency: limit to maxConcurrency goroutines at a time.
	sem := semaphore.NewWeighted(int64(maxConcurrency))
	var wg sync.WaitGroup
	for _, symID := range ids {
		symID := symID // capture loop variable
		if err := sem.Acquire(ctx, 1); err != nil {
			return // context cancelled — stop warming
		}
		wg.Add(1)
		go func() {
			defer func() {
				sem.Release(1)
				wg.Done()
			}()
			hits, err := BlastRadiusQuery(ctx, db, symID, maxDepth, nil)
			if err != nil {
				slog.Warn("WarmCacheForSnapshot: BlastRadiusQuery",
					"symbolID", symID, "err", err)
				return
			}
			if len(hits) > 0 {
				if err := WriteCachedImpact(ctx, db, symID, hits); err != nil {
					slog.Warn("WarmCacheForSnapshot: WriteCachedImpact",
						"symbolID", symID, "err", err)
				}
			}
		}()
	}
	wg.Wait()
}

// WarmCacheForSnapshotSync is the synchronous variant of WarmCacheForSnapshot.
// Use this in tests to ensure the function completes before the DB is closed.
func WarmCacheForSnapshotSync(ctx context.Context, db *sql.DB, topN int, maxDepth int, maxConcurrency int) {
	if topN < 1 {
		topN = 100
	}
	if maxDepth < 1 {
		maxDepth = 3
	}
	if maxConcurrency < 1 {
		maxConcurrency = 3
	}

	ids, err := TopNByConnectivity(ctx, db, topN)
	if err != nil {
		slog.Warn("WarmCacheForSnapshotSync: TopNByConnectivity", "err", err)
		return
	}
	if len(ids) == 0 {
		return
	}

	// Process sequentially in test context — no goroutines needed.
	for _, symID := range ids {
		select {
		case <-ctx.Done():
			return
		default:
		}
		hits, err := BlastRadiusQuery(ctx, db, symID, maxDepth, nil)
		if err != nil {
			slog.Warn("WarmCacheForSnapshotSync: BlastRadiusQuery",
				"symbolID", symID, "err", err)
			continue
		}
		if len(hits) > 0 {
			if err := WriteCachedImpact(ctx, db, symID, hits); err != nil {
				slog.Warn("WarmCacheForSnapshotSync: WriteCachedImpact",
					"symbolID", symID, "err", err)
			}
		}
	}
}
