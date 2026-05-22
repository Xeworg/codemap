package codemap_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"codrut/packages/coding-agent/codemap/store"
)

// file is captured at init time so helpers can use it in package-level vars.
var file = func() string {
	_, f, _, _ := runtime.Caller(0)
	return f
}()

// moduleRoot returns the absolute path to the codemap package root directory.
// file is .../testdata/deadcode-precision/deadcode_precision_test.go.
// dir(dir(file)) = .../testdata/  (testdata/ is the module root for codemap package)
func moduleRoot() string {
	return filepath.Dir(filepath.Dir(file))
}

// repoRoot returns the repository root (5 levels up from the test file).
// file:     .../testdata/deadcode-precision/deadcode_precision_test.go
//
//	dir1:    .../testdata/deadcode-precision/
//	dir2:    .../testdata/
//	dir3:    .../packages/coding-agent/codemap/
//	dir4:    .../packages/coding-agent/
//	dir5:    .../packages/
//	dir6:    .../  (= repo root, parent of packages/)
func repoRoot() string {
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))))
}

// fixturePath is the regression fixture Go package directory.
var fixturePath = filepath.Join(filepath.Dir(file), "fixture")

// buildCodemap builds a codemap binary from the current source tree and
// returns the path to the binary. The binary includes the deadcode command.
func buildCodemap(t *testing.T) string {
	t.Helper()
	codemapBinary := filepath.Join(t.TempDir(), "codemap")
	buildCmd := exec.Command("go", "build", "-o", codemapBinary, "./cmd/codemap")
	buildCmd.Dir = repoRoot()
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build codemap: %v\n%s", err, out)
	}
	return codemapBinary
}

// TestDeadcode_PrecisionFixture_GoFiles verifies the fixture directory contains
// valid Go source files so the regression test has a meaningful target.
func TestDeadcode_PrecisionFixture_GoFiles(t *testing.T) {
	entries, err := os.ReadDir(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var hasGo bool
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".go") {
			hasGo = true
			break
		}
	}
	if !hasGo {
		t.Errorf("fixture directory has no .go files")
	}
}

// TestDeadcode_PrecisionRegression runs codemap index + deadcode on the fixture
// and asserts the precision heuristics hold.
func TestDeadcode_PrecisionRegression(t *testing.T) {
	codemapBinary := buildCodemap(t)

	absFixturePath, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatal(err)
	}

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "deadcode_precision.db")

	// Create DB and migrate.
	db := store.MustOpen(dbPath)
	if err := store.Migrate(context.Background(), db.DB); err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Run: codemap --repo <absFixturePath> index --db <tmp>/deadcode_precision.db
	indexArgs := []string{"--repo", absFixturePath, "index", "--db", dbPath}
	indexCmd := exec.Command(codemapBinary, indexArgs...)
	if out, err := indexCmd.CombinedOutput(); err != nil {
		t.Fatalf("codemap index failed: %v\n%s", err, out)
	}

	// Run: codemap --repo <absFixturePath> deadcode --db <tmp>/deadcode_precision.db
	dcArgs := []string{"--repo", absFixturePath, "deadcode", "--db", dbPath}
	dcCmd := exec.Command(codemapBinary, dcArgs...)
	dcOut, err := dcCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("codemap deadcode failed: %v\n%s", err, dcOut)
	}

	// Parse JSON envelope.
	var env map[string]interface{}
	if err := json.Unmarshal(dcOut, &env); err != nil {
		t.Fatalf("non-JSON deadcode output: %s", dcOut)
	}
	if env["ok"] != true {
		t.Fatalf("deadcode returned ok=false: %s", dcOut)
	}

	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data field in envelope")
	}
	findings, ok := data["findings"].([]interface{})
	if !ok {
		t.Fatal("missing or non-array findings in envelope")
	}

	// Build symbol-name → finding map.
	findingsMap := make(map[string]map[string]interface{})
	for _, f := range findings {
		fm := f.(map[string]interface{})
		name := fm["symbol_name"].(string)
		findingsMap[name] = fm
	}

	// D1 assertions.

	// ExportedHelper: MUST NOT be classified "unused" with high confidence.
	if h, ok := findingsMap["ExportedHelper"]; ok {
		class := h["classification"].(string)
		conf := h["confidence"].(string)
		if class == "unused" && conf == "high" {
			t.Errorf("ExportedHelper: got unused+high, want uncertain")
		}
	}

	// privateUnused: SHOULD be classified "unused" or "likely-unused".
	// Lowercase name means it's not public API, so it gets the kind-based
	// confidence: "func" → high.
	if h, ok := findingsMap["privateUnused"]; ok {
		class := h["classification"].(string)
		if class != "unused" && class != "likely-unused" {
			t.Errorf("privateUnused: got %q, want unused or likely-unused", class)
		}
	}

	// init: MUST NOT be classified "unused".
	if h, ok := findingsMap["init"]; ok {
		class := h["classification"].(string)
		if class == "unused" {
			t.Errorf("init: got unused, want uncertain")
		}
	}

	// T.Method: MUST NOT be classified "unused" with high confidence.
	if h, ok := findingsMap["T.Method"]; ok {
		class := h["classification"].(string)
		conf := h["confidence"].(string)
		if class == "unused" && conf == "high" {
			t.Errorf("T.Method: got unused+high, want uncertain or likely-unused")
		}
	}
}
