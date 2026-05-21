package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codrut/packages/coding-agent/codemap/cli"
)

func osHint() string {
	home, _ := os.UserHomeDir()
	return home
}

func TestDoctorResultPass(t *testing.T) {
	doc := &Doctor{
		Installer: &Installer{
			RepoRoot:       ".",
			PiRuntimeBase:  filepath.Join(osHint(), ".pi", "agent"),
			SkillTargetDir: filepath.Join(osHint(), ".pi", "agent", "skills"),
			ToolTargetDir:  filepath.Join(osHint(), ".pi", "agent", "tools"),
		},
	}
	result := doc.Run()
	// Status may be pass/warn depending on runtime state; checks must exist
	if len(result.Checks) == 0 {
		t.Fatal("no checks returned")
	}
	// DB path must be populated
	if result.DBPath == "" {
		t.Error("db_path should be populated")
	}
}

func TestDoctorResultJSON(t *testing.T) {
	doc := &Doctor{Installer: DefaultInstaller(".")}
	result := doc.Run()
	json := result.JSON()
	if !strings.Contains(json, `"status"`) {
		t.Error("JSON missing status field")
	}
	if !strings.Contains(json, `"checks"`) {
		t.Error("JSON missing checks field")
	}
}

func TestDoctorResultPrint(t *testing.T) {
	doc := &Doctor{Installer: DefaultInstaller(".")}
	result := doc.Run()
	output := result.Print()
	if output == "" {
		t.Error("Print returned empty output")
	}
	// Should contain overall status
	if !strings.Contains(output, "PASS") && !strings.Contains(output, "WARN") && !strings.Contains(output, "FAIL") {
		t.Error("Print missing overall status")
	}
}

func TestDoctorCheckLevels(t *testing.T) {
	checks := []DoctorCheck{
		{Check: "test", Level: "pass", Message: "ok"},
		{Check: "test2", Level: "warn", Message: "warning"},
		{Check: "test3", Level: "fail", Message: "fail"},
	}
	for _, c := range checks {
		if c.Level != "pass" && c.Level != "warn" && c.Level != "fail" {
			t.Errorf("invalid level: %s", c.Level)
		}
	}
}

func TestDoctorPrintFormat(t *testing.T) {
	doc := &Doctor{Installer: DefaultInstaller(".")}
	result := doc.Run()
	var buf bytes.Buffer
	buf.WriteString(result.Print())
	out := buf.String()
	// Each check should appear in output
	for _, c := range result.Checks {
		if !strings.Contains(out, c.Check) {
			t.Errorf("check %q not found in output", c.Check)
		}
	}
}

func TestEffectiveDBPath(t *testing.T) {
	doc := &Doctor{Installer: DefaultInstaller(".")}
	path := doc.effectiveDBPath()
	if path == "" {
		t.Error("effectiveDBPath returned empty")
	}
	// Should contain .cache/codemap
	if !strings.Contains(path, ".cache") || !strings.Contains(path, "codemap") {
		t.Error("effectiveDBPath does not contain expected cache path")
	}
}

func TestHashRepo(t *testing.T) {
	h1 := hashRepo("/tmp/repo")
	h2 := hashRepo("/tmp/repo")
	h3 := hashRepo("/tmp/other")
	if h1 != h2 {
		t.Error("same repo should produce same hash")
	}
	if h1 == h3 {
		t.Error("different repos should produce different hashes")
	}
	if len(h1) != 32 {
		t.Errorf("expected 32-char hash, got %d", len(h1))
	}
}

func TestEffectiveDBPathMatchesDefault(t *testing.T) {
	doc := &Doctor{Installer: DefaultInstaller(".")}
	got := doc.effectiveDBPath()
	want, err := cli.DefaultDBPath(".")
	if err != nil {
		t.Fatalf("DefaultDBPath error: %v", err)
	}
	if got != want {
		t.Errorf("effectiveDBPath mismatch\ngot:  %s\nwant: %s", got, want)
	}
}
