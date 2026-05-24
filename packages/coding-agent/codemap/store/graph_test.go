package store

import (
	"context"
	"database/sql"
	"testing"
)

func TestWarmCacheForSnapshot(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, sql := range []string{
		// Minimal schema.
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY, name TEXT, file_id INTEGER)`,
		`CREATE TABLE files (id INTEGER PRIMARY KEY, snapshot_id INTEGER)`,
		`CREATE TABLE edges (from_symbol_id INTEGER, to_symbol_id INTEGER, edge_type TEXT)`,
		`CREATE TABLE snapshots (id INTEGER PRIMARY KEY)`,
		// Cache table.
		`CREATE TABLE symbol_impact_cache (source_symbol_id INTEGER, target_symbol_id INTEGER, depth INTEGER, edge_path TEXT, risk_score REAL, updated_at INTEGER)`,
		// Insert snapshot.
		`INSERT INTO snapshots VALUES (1)`,
		// Two files in the snapshot.
		`INSERT INTO files VALUES (1, 1), (2, 1)`,
		// Three symbols: 1 calls 2, 2 calls 3.
		`INSERT INTO symbols VALUES (1, 'A', 1), (2, 'B', 1), (3, 'C', 2)`,
		`INSERT INTO edges VALUES (1, 2, 'calls'), (2, 3, 'type_use')`,
	} {
		if _, err := db.Exec(sql); err != nil {
			t.Fatal(err)
		}
	}

	WarmCacheForSnapshotSync(ctx, db, 3, 3, 3)

	// Symbol 1 (A) is top: 1 inbound, vs 0 for others.
	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM symbol_impact_cache WHERE source_symbol_id = 1`,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Error("expected cache entries for symbol 1 after warm")
	}
}

func TestWarmCacheForSnapshotEmptyDB(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Should not panic on empty DB.
	WarmCacheForSnapshot(ctx, db, 10, 3, 3)
}

func TestWarmCacheForSnapshotContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, sql := range []string{
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY, name TEXT, file_id INTEGER)`,
		`CREATE TABLE files (id INTEGER PRIMARY KEY, snapshot_id INTEGER)`,
		`CREATE TABLE edges (from_symbol_id INTEGER, to_symbol_id INTEGER, edge_type TEXT)`,
		`CREATE TABLE snapshots (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE symbol_impact_cache (source_symbol_id INTEGER, target_symbol_id INTEGER, depth INTEGER, edge_path TEXT, risk_score REAL, updated_at INTEGER)`,
		`INSERT INTO snapshots VALUES (1)`,
		`INSERT INTO files VALUES (1, 1)`,
		`INSERT INTO symbols VALUES (1, 'A', 1)`,
		`INSERT INTO edges VALUES (1, 1, 'calls')`,
	} {
		if _, err := db.Exec(sql); err != nil {
			t.Fatal(err)
		}
	}

	// Should not panic when context is cancelled before work begins.
	WarmCacheForSnapshot(ctx, db, 5, 3, 3)
}

func TestInvalidateCacheForSymbol(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE symbol_impact_cache (source_symbol_id INTEGER, target_symbol_id INTEGER, depth INTEGER, edge_path TEXT, risk_score REAL, updated_at INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO symbol_impact_cache VALUES (1, 2, 1, '', 1.0, ` + nowUnix() + `)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO symbol_impact_cache VALUES (3, 1, 1, '', 1.0, ` + nowUnix() + `)`); err != nil {
		t.Fatal(err)
	}

	err = InvalidateCacheForSymbol(ctx, db, 1)
	if err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM symbol_impact_cache WHERE source_symbol_id = 1 OR target_symbol_id = 1`,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("want 0 remaining entries, got %d", count)
	}
}

func TestWriteCachedImpact(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE symbol_impact_cache (source_symbol_id INTEGER, target_symbol_id INTEGER, depth INTEGER, edge_path TEXT, risk_score REAL, updated_at INTEGER)`); err != nil {
		t.Fatal(err)
	}

	hits := []SymbolHit{
		{SymbolID: 2, Depth: 1, EdgePath: ",calls", RiskScore: 1.0},
		{SymbolID: 3, Depth: 2, EdgePath: ",calls,type_use", RiskScore: 1.7},
	}
	err = WriteCachedImpact(ctx, db, 1, hits)
	if err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM symbol_impact_cache WHERE source_symbol_id = 1`,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("want 2 cache entries, got %d", count)
	}

	// Overwrite: write new hits for same source, old entries should be gone.
	hits2 := []SymbolHit{
		{SymbolID: 4, Depth: 1, EdgePath: ",imports", RiskScore: 0.5},
	}
	err = WriteCachedImpact(ctx, db, 1, hits2)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM symbol_impact_cache WHERE source_symbol_id = 1`,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("after overwrite: want 1 entry, got %d", count)
	}
}

func TestTopNByConnectivity(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, sql := range []string{
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY, name TEXT, file_id INTEGER)`,
		`CREATE TABLE files (id INTEGER PRIMARY KEY, snapshot_id INTEGER)`,
		`CREATE TABLE edges (from_symbol_id INTEGER, to_symbol_id INTEGER, edge_type TEXT)`,
		`CREATE TABLE snapshots (id INTEGER PRIMARY KEY)`,
		`INSERT INTO snapshots VALUES (1)`,
		`INSERT INTO files VALUES (1, 1), (2, 1)`,
		`INSERT INTO symbols VALUES (1, 'A', 1), (2, 'B', 1), (3, 'C', 2)`,
		// B has 2 inbound, C has 1 inbound.
		`INSERT INTO edges VALUES (1, 2, 'calls'), (3, 2, 'calls'), (1, 3, 'calls')`,
	} {
		if _, err := db.Exec(sql); err != nil {
			t.Fatal(err)
		}
	}

	ids, err := TopNByConnectivity(ctx, db, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("want 2 ids, got %d", len(ids))
	}
	if ids[0] != 2 {
		t.Errorf("top-1: want symbol 2, got %d", ids[0])
	}
	if ids[1] != 3 {
		t.Errorf("top-2: want symbol 3, got %d", ids[1])
	}
}

// nowUnix returns a timestamp far in the future so cache entries appear fresh.
func nowUnix() string {
	return "9999999999"
}
