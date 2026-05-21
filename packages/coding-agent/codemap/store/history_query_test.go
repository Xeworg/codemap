package store

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

// --- HistoryQuery tests (require in-memory SQLite) ---

func newTestDB(t *testing.T) (*sql.DB, func()) {
	f, err := os.CreateTemp("", "codemap_history_test_*.db")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	f.Close()
	db, err := sql.Open("sqlite", f.Name())
	if err != nil {
		os.Remove(f.Name())
		t.Fatalf("sql.Open: %v", err)
	}
	cleanup := func() {
		db.Close()
		os.Remove(f.Name())
	}
	return db, cleanup
}

func schemaForTest(t *testing.T, db *sql.DB) {
	schema := `
	CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL);
	CREATE TABLE IF NOT EXISTS snapshots (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_root TEXT NOT NULL, head_ref TEXT NOT NULL, created_at TEXT NOT NULL);
	CREATE TABLE IF NOT EXISTS commits (hash TEXT PRIMARY KEY, author TEXT, date TEXT, message TEXT);
	CREATE TABLE IF NOT EXISTS symbols (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL, name TEXT NOT NULL, kind TEXT NOT NULL, signature TEXT, start_line INTEGER NOT NULL, end_line INTEGER NOT NULL);
	CREATE TABLE IF NOT EXISTS symbol_commits (id INTEGER PRIMARY KEY AUTOINCREMENT, symbol_id INTEGER NOT NULL, commit_hash TEXT NOT NULL, change_type TEXT, link_strength TEXT NOT NULL, FOREIGN KEY(symbol_id) REFERENCES symbols(id), FOREIGN KEY(commit_hash) REFERENCES commits(hash));
	`
	_, err := db.Exec(schema)
	if err != nil {
		t.Fatalf("schema setup: %v", err)
	}
}

func TestGetSymbolHistoryRequiresLinkStrength(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	schemaForTest(t, db)

	// Insert snapshot, file, symbol.
	_, err := db.Exec(`INSERT INTO snapshots(id, repo_root, head_ref, created_at) VALUES (1, '/tmp', 'abc', '2024-01-01')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO commits(hash, author, date, message) VALUES ('h1','a','2024-01-01','msg')`)
	if err != nil {
		t.Fatal(err)
	}
	// No symbol yet — query should return empty.

	entries, err := GetSymbolHistory(context.Background(), db, int64(99))
	if err != nil {
		t.Fatalf("GetSymbolHistory returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for non-existent symbol, got %d", len(entries))
	}
}

func TestGetSymbolHistoryWithEntries(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	schemaForTest(t, db)

	_, err := db.Exec(`INSERT INTO snapshots(id, repo_root, head_ref, created_at) VALUES (1, '/tmp', 'abc', '2024-01-01')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO commits(hash, author, date, message) VALUES ('h1','a','2024-01-01','msg1'), ('h2','b','2024-02-01','msg2')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO symbols(id, file_id, name, kind, signature, start_line, end_line) VALUES (1, 1, 'Foo', 'func', 'func()', 10, 20)`)
	if err != nil {
		t.Fatal(err)
	}

	// Link symbol to two commits with different strengths.
	_, err = db.Exec(`INSERT INTO symbol_commits(symbol_id, commit_hash, change_type, link_strength) VALUES (1, 'h1', 'modify', 'weak'), (1, 'h2', 'modify', 'strong')`)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := GetSymbolHistory(context.Background(), db, int64(1))
	if err != nil {
		t.Fatalf("GetSymbolHistory: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// Entries should be ordered strong desc → weak desc.
	if entries[0].LinkStrength != "strong" {
		t.Errorf("first entry should be strong, got %v", entries[0].LinkStrength)
	}
	if entries[1].LinkStrength != "weak" {
		t.Errorf("second entry should be weak, got %v", entries[1].LinkStrength)
	}
}

func TestGetSymbolHistoryEnforcesEnumValues(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	schemaForTest(t, db)

	_, err := db.Exec(`INSERT INTO snapshots(id, repo_root, head_ref, created_at) VALUES (1, '/tmp', 'abc', '2024-01-01')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO commits(hash, author, date, message) VALUES ('h1','a','2024-01-01','msg')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO symbols(id, file_id, name, kind, signature, start_line, end_line) VALUES (1, 1, 'Foo', 'func', 'func()', 10, 20)`)
	if err != nil {
		t.Fatal(err)
	}
	// Insert with invalid enum value.
	_, err = db.Exec(`INSERT INTO symbol_commits(symbol_id, commit_hash, change_type, link_strength) VALUES (1, 'h1', 'modify', 'invalid_strength')`)
	if err != nil {
		t.Fatal(err)
	}

	// GetSymbolHistory should either skip invalid values or return error.
	// The MVP contract requires allowed values only; we enforce at insert time.
	entries, err := GetSymbolHistory(context.Background(), db, int64(1))
	if err != nil {
		t.Fatalf("GetSymbolHistory should not error on invalid DB value (fail-soft at query): %v", err)
	}
	// Should skip invalid strength entries.
	for _, e := range entries {
		if e.LinkStrength != "strong" && e.LinkStrength != "medium" && e.LinkStrength != "weak" {
			t.Errorf("query returned invalid link_strength: %q", e.LinkStrength)
		}
	}
}

func TestSymbolHistoryQuerySQLite(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	schemaForTest(t, db)

	// Setup: 3 commits, 1 symbol, 3 links with different strengths and dates.
	_, err := db.Exec(`INSERT INTO snapshots(id, repo_root, head_ref, created_at) VALUES (1, '/tmp', 'abc', '2024-01-01')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		INSERT INTO commits(hash, author, date, message) VALUES
		('ca', 'a', '2024-01-01', 'first'),
		('cb', 'b', '2024-02-01', 'second'),
		('cc', 'c', '2024-03-01', 'third')
	`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO symbols(id, file_id, name, kind, signature, start_line, end_line) VALUES (5, 1, 'Bar', 'type', 'type Bar', 1, 30)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		INSERT INTO symbol_commits(symbol_id, commit_hash, change_type, link_strength) VALUES
		(5, 'ca', 'add', 'weak'),
		(5, 'cb', 'modify', 'medium'),
		(5, 'cc', 'modify', 'strong')
	`)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := GetSymbolHistory(context.Background(), db, int64(5))
	if err != nil {
		t.Fatalf("GetSymbolHistory: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	// Order: strong(c3), medium(c2), weak(c1) — and within same strength, recency desc.
	if entries[0].CommitHash != "cc" || entries[1].CommitHash != "cb" || entries[2].CommitHash != "ca" {
		t.Errorf("wrong order: %v", commitHashes(entries))
	}
}

func TestSymbolHistoryQueryEmpty(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	schemaForTest(t, db)

	entries, err := GetSymbolHistory(context.Background(), db, int64(1))
	if err != nil {
		t.Fatalf("GetSymbolHistory on empty DB: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries on empty DB, got %d", len(entries))
	}
}

// symbolID removed — use int64 directly

func commitHashes(entries []SymbolHistoryEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.CommitHash
	}
	return out
}
