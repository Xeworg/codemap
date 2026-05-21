package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDBPathExplicitFlag(t *testing.T) {
	// Explicit --db flag takes highest priority.
	path, err := ResolveDBPath("/custom/path/test.db", "/some/repo")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/custom/path/test.db" {
		t.Errorf("expected explicit path /custom/path/test.db, got %s", path)
	}
}

func TestResolveDBPathEnvVar(t *testing.T) {
	// CODEMAP_DB_PATH env var is used when no explicit flag.
	tmp := t.TempDir()
	envPath := filepath.Join(tmp, "codemap.db")
	os.Setenv("CODEMAP_DB_PATH", envPath)
	defer os.Unsetenv("CODEMAP_DB_PATH")
	path, err := ResolveDBPath("", "/some/repo")
	if err != nil {
		t.Fatal(err)
	}
	if path != envPath {
		t.Errorf("expected env path %s, got %s", envPath, path)
	}
}

func TestResolveDBPathDefault(t *testing.T) {
	// Default path uses cache dir with deterministic hash.
	path, err := ResolveDBPath("", "/some/repo")
	if err != nil {
		t.Fatal(err)
	}
	// Should be under UserCacheDir/codemap/
	cacheDir, _ := os.UserCacheDir()
	expectedPrefix := filepath.Join(cacheDir, "codemap")
	if filepath.Dir(path) != expectedPrefix {
		t.Errorf("expected parent dir %s, got %s", expectedPrefix, filepath.Dir(path))
	}
	// Should have .db extension.
	if filepath.Ext(path) != ".db" {
		t.Errorf("expected .db extension, got %s", filepath.Ext(path))
	}
	// Same repo root should produce same path.
	path2, err := ResolveDBPath("", "/some/repo")
	if err != nil {
		t.Fatal(err)
	}
	if path != path2 {
		t.Errorf("expected deterministic path, got %s vs %s", path, path2)
	}
	// Different repo roots should produce different paths.
	path3, err := ResolveDBPath("", "/other/repo")
	if err != nil {
		t.Fatal(err)
	}
	if path == path3 {
		t.Errorf("expected different paths for different repos, got same: %s", path)
	}
}

func TestResolveDBPathEnvCreatesParentDir(t *testing.T) {
	// When CODEMAP_DB_PATH is used, its parent dirs should be created.
	tmp := t.TempDir()
	envPath := filepath.Join(tmp, "subdir", "codemap", "test.db")
	os.Setenv("CODEMAP_DB_PATH", envPath)
	defer os.Unsetenv("CODEMAP_DB_PATH")
	path, err := ResolveDBPath("", "/test/repo")
	if err != nil {
		t.Fatalf("ResolveDBPath failed: %v", err)
	}
	if path != envPath {
		t.Errorf("expected %s, got %s", envPath, path)
	}
	// Verify the parent dir was created.
	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		t.Errorf("parent directory was not created: %s", filepath.Dir(path))
	}
}
