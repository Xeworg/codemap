package store

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

// Open opens or creates a SQLite database at the given path.
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{DB: db}, nil
}

// DB wraps sql.DB with typed helpers.
type DB struct {
	*sql.DB
}

// MustOpen is like Open but panics on error. Use only in tests.
func MustOpen(path string) *DB {
	db, err := Open(path)
	if err != nil {
		panic("MustOpen: " + err.Error())
	}
	return db
}

// MustTempDB creates a temporary in-memory DB. Use only in tests.
func MustTempDB(t T) *DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	return &DB{DB: db}
}

// T is the interface satisfied by *testing.T.
type T interface {
	Helper()
	Fatal(args ...interface{})
}

// Exists reports whether path points to an existing file.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
