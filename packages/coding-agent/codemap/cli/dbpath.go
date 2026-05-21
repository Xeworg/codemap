package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

// DefaultDBPath returns the default database path for a given repo root.
// The path is placed in the user's cache directory (~/.cache/codemap/ by default)
// to keep the index outside the project and out of version control.
// The filename is a deterministic hash of the absolute repo root path.
func DefaultDBPath(repoRoot string) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		// Fallback: use temp dir if cache dir is unavailable.
		cacheDir = os.TempDir()
	}
	// Canonicalize the repo root for deterministic hashing.
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		absRoot = repoRoot
	}
	hash := sha256.Sum256([]byte(absRoot))
	filename := hex.EncodeToString(hash[:16]) + ".db"
	dbPath := filepath.Join(cacheDir, "codemap", filename)
	// Ensure the parent directory exists.
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return "", err
	}
	return dbPath, nil
}

// ResolveDBPath returns the effective DB path given an optional --db flag value,
// CODEMAP_DB_PATH env var, and the repo root for computing the default.
// Priority: explicit --db > CODEMAP_DB_PATH env > computed default.
func ResolveDBPath(explicitDB, repoRoot string) (string, error) {
	if explicitDB != "" {
		return explicitDB, nil
	}
	if envPath := os.Getenv("CODEMAP_DB_PATH"); envPath != "" {
		if err := os.MkdirAll(filepath.Dir(envPath), 0755); err != nil {
			return "", err
		}
		return envPath, nil
	}
	return DefaultDBPath(repoRoot)
}
