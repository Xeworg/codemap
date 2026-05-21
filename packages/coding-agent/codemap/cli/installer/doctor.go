package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DoctorResult holds the outcome of a codemap doctor run.
type DoctorResult struct {
	Status   string        `json:"status"` // "pass", "warn", "fail"
	Checks   []DoctorCheck `json:"checks"`
	DBPath   string        `json:"db_path,omitempty"`
	DBExists bool          `json:"db_exists"`
}

// DoctorCheck is a single diagnostic check.
type DoctorCheck struct {
	Check   string `json:"check"`
	Level   string `json:"level"` // "pass", "warn", "fail"
	Message string `json:"message"`
}

// Doctor diagnoses the codemap installation and environment.
type Doctor struct {
	Installer *Installer
}

// DefaultDoctor returns a Doctor backed by the default installer.
func DefaultDoctor() *Doctor {
	return &Doctor{Installer: DefaultInstaller(".")}
}

// Run executes all diagnostic checks.
func (d *Doctor) Run() *DoctorResult {
	if d.Installer == nil {
		d.Installer = DefaultInstaller(".")
	}
	result := &DoctorResult{Checks: []DoctorCheck{}}
	hasWarn := false
	hasFail := false

	// 1) repo root
	checks := d.runChecks()
	for _, c := range checks {
		result.Checks = append(result.Checks, c)
		if c.Level == "fail" {
			hasFail = true
		} else if c.Level == "warn" {
			hasWarn = true
		}
	}

	// 2) installed skill presence
	skillCheck := d.checkInstalledSkill()
	result.Checks = append(result.Checks, skillCheck)
	if skillCheck.Level == "fail" {
		hasFail = true
	} else if skillCheck.Level == "warn" {
		hasWarn = true
	}

	// 3) installed tool presence
	toolCheck := d.checkInstalledTool()
	result.Checks = append(result.Checks, toolCheck)
	if toolCheck.Level == "fail" {
		hasFail = true
	} else if toolCheck.Level == "warn" {
		hasWarn = true
	}

	// 4) effective DB path
	dbPath := d.effectiveDBPath()
	result.DBPath = dbPath
	_, dbErr := os.Stat(dbPath)
	result.DBExists = dbErr == nil

	dbCheck := DoctorCheck{
		Check:   "default_db",
		Level:   "pass",
		Message: fmt.Sprintf("DB path: %s", dbPath),
	}
	if !result.DBExists {
		dbCheck.Level = "warn"
		dbCheck.Message += " (not created yet; run 'codemap index' to create)"
	}
	result.Checks = append(result.Checks, dbCheck)

	// Overall status
	if hasFail {
		result.Status = "fail"
	} else if hasWarn {
		result.Status = "warn"
	} else {
		result.Status = "pass"
	}

	return result
}

func (d *Doctor) runChecks() []DoctorCheck {
	var results []DoctorCheck
	inst := d.Installer

	// repo root
	if dirExists(inst.RepoRoot) {
		results = append(results, DoctorCheck{Check: "repo_root", Level: "pass", Message: fmt.Sprintf("repo root: %s", inst.RepoRoot)})
	} else {
		results = append(results, DoctorCheck{Check: "repo_root", Level: "fail", Message: "repo root not found"})
	}

	// integrations dir
	integrationsDir := filepath.Join(inst.RepoRoot, "integrations", "pi")
	if dirExists(integrationsDir) {
		results = append(results, DoctorCheck{Check: "integrations_dir", Level: "pass", Message: fmt.Sprintf("integrations dir: %s", integrationsDir)})
	} else {
		results = append(results, DoctorCheck{Check: "integrations_dir", Level: "fail", Message: "integrations/pi not found in repo root"})
	}

	// template files
	for _, t := range inst.Templates() {
		key := string(t.Kind)
		msg := fmt.Sprintf("%s template: %s", t.Kind, t.Source)
		level := "pass"
		if !fileExists(t.Source) {
			level = "fail"
			msg += " — NOT FOUND"
		}
		results = append(results, DoctorCheck{Check: key, Level: level, Message: msg})
	}

	// pi runtime base
	piExists := dirExists(inst.PiRuntimeBase)
	if piExists {
		results = append(results, DoctorCheck{Check: "pi_runtime", Level: "pass", Message: fmt.Sprintf("Pi runtime: %s", inst.PiRuntimeBase)})
	} else {
		results = append(results, DoctorCheck{
			Check:   "pi_runtime",
			Level:   "warn",
			Message: fmt.Sprintf("Pi runtime not found at %s (will be created on install)", inst.PiRuntimeBase),
		})
	}

	return results
}

func (d *Doctor) checkInstalledSkill() DoctorCheck {
	skillDst := filepath.Join(d.Installer.SkillTargetDir, "codemap-usage", "SKILL.md")
	if fileExists(skillDst) {
		return DoctorCheck{Check: "installed_skill", Level: "pass", Message: "skill installed at: " + skillDst}
	}
	return DoctorCheck{
		Check:   "installed_skill",
		Level:   "warn",
		Message: fmt.Sprintf("skill not installed at %s (run 'codemap install' to install)", skillDst),
	}
}

func (d *Doctor) checkInstalledTool() DoctorCheck {
	toolDst := filepath.Join(d.Installer.ToolTargetDir, "codemap-tool.json")
	if fileExists(toolDst) {
		return DoctorCheck{Check: "installed_tool", Level: "pass", Message: "tool installed at: " + toolDst}
	}
	return DoctorCheck{
		Check:   "installed_tool",
		Level:   "warn",
		Message: fmt.Sprintf("tool not installed at %s (run 'codemap install' to install)", toolDst),
	}
}

// effectiveDBPath returns the actual DB path that codemap will use.
func (d *Doctor) effectiveDBPath() string {
	dbPath := os.Getenv("CODEMAP_DB_PATH")
	if dbPath != "" {
		return dbPath
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	absRoot := d.Installer.RepoRoot
	if abs, err := filepath.Abs(absRoot); err == nil {
		absRoot = abs
	}
	return filepath.Join(cacheDir, "codemap", hashRepo(absRoot)+".db")
}

// hashRepo returns a short hash of the repo root for DB naming.
func hashRepo(repoRoot string) string {
	h := sha256.Sum256([]byte(repoRoot))
	return hex.EncodeToString(h[:16])
}

// Print renders a human-readable doctor report.
func (r *DoctorResult) Print() string {
	var lines []string
	lines = append(lines, "CodeMap Doctor\n")
	var statusIcon string
	switch r.Status {
	case "pass":
		statusIcon = "✅ PASS"
	case "warn":
		statusIcon = "⚠ WARN"
	case "fail":
		statusIcon = "❌ FAIL"
	}
	lines = append(lines, fmt.Sprintf("Overall: %s\n", statusIcon))
	lines = append(lines, "─── Checks ───")
	for _, c := range r.Checks {
		var icon string
		switch c.Level {
		case "pass":
			icon = "✅"
		case "warn":
			icon = "⚠"
		case "fail":
			icon = "❌"
		default:
			icon = "?"
		}
		lines = append(lines, fmt.Sprintf("%s %-20s %s", icon, c.Check, c.Message))
	}
	if r.DBPath != "" {
		lines = append(lines, fmt.Sprintf("\nDefault DB: %s", r.DBPath))
		if r.DBExists {
			lines = append(lines, " ✅ exists")
		} else {
			lines = append(lines, " ⚠ not created yet")
		}
	}
	return strings.Join(lines, "\n")
}

// JSON returns machine-readable JSON output.
func (r *DoctorResult) JSON() string {
	b, _ := json.MarshalIndent(r, "", "  ")
	return string(b)
}
