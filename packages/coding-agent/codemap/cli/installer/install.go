package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TemplateKind identifies the type of integration artifact.
type TemplateKind string

const (
	TemplateSkill     TemplateKind = "skill"
	TemplateExtension TemplateKind = "extension"
)

// TemplateRef describes a template file in the repo and its runtime destination.
type TemplateRef struct {
	Source      string // absolute path in repo
	Destination string // absolute path in Pi runtime
	Kind        TemplateKind
}

// InstallResult holds the outcome of an install run.
type InstallResult struct {
	Status    string         `json:"status"` // "applied", "up-to-date", "error", "dry-run"
	Checks    []CheckResult  `json:"checks"`
	Actions   []ActionResult `json:"actions"`
	Warnings  []string       `json:"warnings,omitempty"` // e.g. PATH not including ~/.local/bin
	Error     string         `json:"error,omitempty"`
	Timestamp string         `json:"timestamp"`
}

// CheckResult describes a pre-flight check.
type CheckResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Exists  *bool  `json:"exists,omitempty"` // present for runtime checks; nil for others
	Info    string `json:"info,omitempty"`
	Skipped string `json:"skipped,omitempty"` // reason skipped, if not run
}

// ActionResult describes an action taken or planned.
type ActionResult struct {
	Kind    string `json:"kind"` // "copy", "install_binary"
	Source  string `json:"source"`
	Target  string `json:"target"`
	Changed bool   `json:"changed"`
	Skipped string `json:"skipped,omitempty"` // reason skipped, if not copied
}

// Installer sets up codemap integration into Pi runtime.
type Installer struct {
	// RepoRoot is the root of the codemap project (where integrations/ lives).
	RepoRoot string
	// PiRuntimeBase is the base directory of the Pi agent runtime (~/.pi).
	PiRuntimeBase string
	// SkillTargetDir is where skills are installed.
	SkillTargetDir string
	// ExtensionTargetDir is where Pi extensions are installed.
	ExtensionTargetDir string
	// BinaryTargetDir is where the codemap binary is installed (default: ~/.local/bin).
	BinaryTargetDir string
	// BinaryName is the name of the codemap binary (default: codemap).
	BinaryName string
	// DryRun makes checks and reports planned actions without applying.
	DryRun bool
	// JSONOutput switches output to machine-readable JSON.
	JSONOutput bool
}

// DefaultInstaller returns an Installer with sensible defaults.
func DefaultInstaller(repoRoot string) *Installer {
	home, _ := os.UserHomeDir()
	piBase := filepath.Join(home, ".pi", "agent")
	return &Installer{
		RepoRoot:           repoRoot,
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
		BinaryTargetDir:    filepath.Join(home, ".local", "bin"),
		BinaryName:         "codemap",
	}
}

// Validate checks that Installer fields are safe and returns an error if not.
// Invalid BinaryName with path traversal or unsafe characters is rejected.
func (i *Installer) Validate() error {
	if err := validateBinaryName(i.BinaryName); err != nil {
		return fmt.Errorf("BinaryName: %w", err)
	}
	return nil
}

// validateBinaryName ensures the binary name is safe to use in paths.
// Rejects: empty, contains path separators, starts with dot, >64 chars,
// or characters outside [a-zA-Z0-9._-].
func validateBinaryName(name string) error {
	if name == "" {
		return fmt.Errorf("cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("exceeds 64 characters")
	}
	if name[0] == '.' {
		return fmt.Errorf("cannot start with a dot")
	}
	for _, c := range name {
		if c == '/' || c == '\\' || c == filepath.Separator {
			return fmt.Errorf("contains path separator")
		}
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '.' && c != '_' && c != '-' {
			return fmt.Errorf("contains unsafe character: %c", c)
		}
	}
	return nil
}

// Templates returns the list of template files to install and their runtime targets.
func (i *Installer) Templates() []TemplateRef {
	skillSrc := filepath.Join(i.RepoRoot, "integrations", "pi", "skills", "codemap-usage", "SKILL.md")
	skillDst := filepath.Join(i.SkillTargetDir, "codemap-usage", "SKILL.md")
	extSrc := filepath.Join(i.RepoRoot, "integrations", "pi", "extensions", "codemap-extension.ts")
	extDst := filepath.Join(i.ExtensionTargetDir, "codemap-extension.ts")
	return []TemplateRef{
		{Source: skillSrc, Destination: skillDst, Kind: TemplateSkill},
		{Source: extSrc, Destination: extDst, Kind: TemplateExtension},
	}
}

// Run executes pre-flight checks and (if not dry-run) applies install actions.
func (i *Installer) Run() *InstallResult {
	result := &InstallResult{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Validate installer fields before proceeding
	if err := i.Validate(); err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result
	}

	// Preflight checks
	checks := i.runChecks()
	result.Checks = checks
	for _, c := range checks {
		if !c.Passed {
			result.Status = "error"
			result.Error = "pre-flight check failed: " + c.Name
			return result
		}
	}

	// Gather actions (planned or applied)
	actions := i.planActions()
	result.Actions = actions

	for _, a := range actions {
		if a.Kind == "install_binary" && strings.HasPrefix(a.Skipped, "build failed:") {
			result.Status = "error"
			result.Error = a.Skipped
			return result
		}
	}

	hasChange := false
	for _, a := range actions {
		if a.Changed {
			hasChange = true
			break
		}
	}

	if i.DryRun {
		result.Status = "dry-run"
		return result
	}

	if !hasChange {
		result.Status = "up-to-date"
		return result
	}

	// Apply
	err := i.apply(actions)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result
	}

	result.Status = "applied"

	// Check PATH for binary target dir
	if pathWarning := checkPATHWarning(i.BinaryTargetDir); pathWarning != "" {
		result.Warnings = append(result.Warnings, pathWarning)
	}

	return result
}

// runChecks performs pre-flight checks.
// It only fails for real blockers: unreadable template sources, or home dir inaccessible.
// Missing Pi runtime directory is reported as a notice (passed=true) so first installs proceed.
func (i *Installer) runChecks() []CheckResult {
	var results []CheckResult

	// Check repo root exists and has integrations dir
	integrationsDir := filepath.Join(i.RepoRoot, "integrations", "pi")
	results = append(results, CheckResult{
		Name:   "repo_root",
		Passed: dirExists(i.RepoRoot),
		Info:   i.RepoRoot,
	})
	results = append(results, CheckResult{
		Name:   "integrations_dir",
		Passed: dirExists(integrationsDir),
		Info:   integrationsDir,
	})

	// Check each template exists in source (real blocker: can't read source)
	for _, t := range i.Templates() {
		if src, err := os.ReadFile(t.Source); err != nil {
			results = append(results, CheckResult{
				Name:   "template_" + string(t.Kind),
				Passed: false,
				Info:   t.Source,
			})
		} else if len(src) == 0 {
			// Empty source is a blocker: nothing to copy
			results = append(results, CheckResult{
				Name:   "template_" + string(t.Kind),
				Passed: false,
				Info:   t.Source + " (empty file)",
			})
		} else {
			results = append(results, CheckResult{
				Name:   "template_" + string(t.Kind),
				Passed: true,
				Info:   t.Source,
			})
		}
	}

	// Check Pi runtime: report existence explicitly but do not block on absence.
	// Block only if home directory is unreadable (real permission/path issue).
	_, homeErr := os.UserHomeDir()
	piRuntimeExists := dirExists(i.PiRuntimeBase)
	runtimePassed := true
	runtimeInfo := i.PiRuntimeBase
	var existsVal bool = piRuntimeExists

	if homeErr != nil {
		// Cannot determine home dir — this is a real blocker
		runtimePassed = false
		runtimeInfo = "home directory unreadable: " + homeErr.Error()
		existsVal = false
	}

	results = append(results, CheckResult{
		Name:   "pi_runtime",
		Passed: runtimePassed,
		Exists: &existsVal,
		Info:   runtimeInfo,
	})

	// Check binary source exists (go.mod + cmd/codemap/main.go)
	// This is a non-blocking check: warn if missing but allow install to proceed.
	goMod := filepath.Join(i.RepoRoot, "go.mod")
	binarySrc := filepath.Join(i.RepoRoot, "cmd", i.BinaryName)
	mainPath := filepath.Join(binarySrc, "main.go")
	if _, err := os.Stat(goMod); err != nil {
		results = append(results, CheckResult{
			Name:    "binary_source",
			Passed:  true,
			Info:    "go.mod not found — binary install will be skipped",
			Skipped: "binary_source_unavailable",
		})
	} else if _, err := os.Stat(mainPath); err != nil {
		results = append(results, CheckResult{
			Name:    "binary_source",
			Passed:  true,
			Info:    "cmd/" + i.BinaryName + "/main.go not found — binary install will be skipped",
			Skipped: "binary_source_unavailable",
		})
	} else {
		results = append(results, CheckResult{
			Name:   "binary_source",
			Passed: true,
			Info:   binarySrc,
		})
	}

	return results
}

// planActions computes what would change.
// If a source file cannot be read, the action is marked as skipped with a reason
// so the error surfaces clearly in output instead of being silently dropped.
func (i *Installer) planActions() []ActionResult {
	var actions []ActionResult
	// Template copy actions
	for _, t := range i.Templates() {
		changed, skipReason := needsCopy(t.Source, t.Destination)
		actions = append(actions, ActionResult{
			Kind:    "copy",
			Source:  t.Source,
			Target:  t.Destination,
			Changed: changed,
			Skipped: skipReason,
		})
	}
	// Binary install action
	// Build to a temp file for comparison, reuse it in apply() if needed
	binarySrc := filepath.Join(i.RepoRoot, "cmd", i.BinaryName)
	binaryDst := filepath.Join(i.BinaryTargetDir, i.BinaryName)
	binaryChanged, binarySkip, tmpBin := i.planBinaryInstall(binarySrc, binaryDst)
	actions = append(actions, ActionResult{
		Kind:    "install_binary",
		Source:  tmpBin,
		Target:  binaryDst,
		Changed: binaryChanged,
		Skipped: binarySkip,
	})
	return actions
}

// apply copies templates to runtime targets and builds/installs the binary.
func (i *Installer) apply(actions []ActionResult) error {
	for _, a := range actions {
		if a.Kind == "install_binary" {
			if a.Changed {
				// Only build during apply — never during dry-run planning
				if err := i.installBinary(a.Source, a.Target); err != nil {
					return fmt.Errorf("install binary: %w", err)
				}
			}
			continue
		}
		// copy action — ensure parent dirs exist
		if err := os.MkdirAll(filepath.Dir(a.Target), 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", filepath.Dir(a.Target), err)
		}
		// Copy only if changed or target does not exist
		if a.Changed {
			if err := copyFile(a.Source, a.Target); err != nil {
				return fmt.Errorf("copy %s -> %s: %w", a.Source, a.Target, err)
			}
		}
	}
	return nil
}

// planBinaryInstall determines if the binary needs to be (re)installed.
// It returns (needs install, skip reason, tempBin for apply to reuse).
//
// Dry-run safety: never builds during planning when the binary already exists.
// Never writes to the target destination during planning.
func (i *Installer) planBinaryInstall(binarySrc, binaryDst string) (bool, string, string) {
	mainPath := filepath.Join(binarySrc, "main.go")
	if !fileExists(mainPath) {
		return false, "binary source not available in repo", ""
	}

	if !fileExists(binaryDst) {
		if i.DryRun {
			return false, "dry-run — binary missing, would be installed", ""
		}
		return true, "", ""
	}

	// Destination exists. In dry-run skip rebuild check.
	if i.DryRun {
		return false, "dry-run — binary exists, skipping rebuild check", ""
	}

	// Non-dry-run with existing binary: build to a temp file to compare.
	tmpBin := binaryDst + ".install-check"
	out, err := i.runGoBuild(tmpBin)
	if err != nil {
		os.Remove(tmpBin)
		return false, "build failed: " + strings.TrimSpace(out), ""
	}
	defer func() { os.Remove(tmpBin) }()

	// Compare new build against installed binary
	dstData, err := os.ReadFile(binaryDst)
	if err != nil {
		return true, "", "" // unreadable — reinstall
	}
	newData, err := os.ReadFile(tmpBin)
	if err != nil {
		return true, "", "" // unreadable — reinstall
	}
	if string(dstData) != string(newData) {
		return true, "", "" // different — install will rebuild during apply
	}
	return false, "binary up-to-date", "" // identical — nothing to do
}

// installBinary installs the built binary to the target directory.
// Builds a temp binary and performs atomic rename when possible.
func (i *Installer) installBinary(binarySrc, binaryDst string) error {
	if err := os.MkdirAll(i.BinaryTargetDir, 0755); err != nil {
		return fmt.Errorf("create bin dir %s: %w", i.BinaryTargetDir, err)
	}

	var tmpBin string
	if binarySrc != "" && fileExists(binarySrc) {
		// Reuse pre-built temp from planning
		tmpBin = binarySrc
	} else {
		// Build fresh if no pre-built temp available
		tmpBin = binaryDst + ".tmp-install"
		out, err := i.runGoBuild(tmpBin)
		if err != nil {
			return fmt.Errorf("go build: %s", strings.TrimSpace(out))
		}
		defer os.Remove(tmpBin)
	}

	// Atomic rename; fallback to copy across filesystem boundaries
	if err := os.Rename(tmpBin, binaryDst); err != nil {
		data, err := os.ReadFile(tmpBin)
		if err != nil {
			return fmt.Errorf("read temp binary: %w", err)
		}
		if err := os.WriteFile(binaryDst, data, 0755); err != nil {
			return fmt.Errorf("write binary %s: %w", binaryDst, err)
		}
		if tmpBin != binarySrc {
			os.Remove(tmpBin)
		}
	}
	return nil
}

// runGoBuild runs `go build -o outputPath ./cmd/binaryName` from RepoRoot and returns combined output + error.
func (i *Installer) runGoBuild(outputPath string) (string, error) {
	pkg := "./cmd/" + i.BinaryName
	cmd := exec.Command("go", "build", "-o", outputPath, pkg)
	cmd.Dir = i.RepoRoot
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Print prints a human-readable result.
func (r *InstallResult) Print() string {
	var lines []string
	status := r.Status
	if status == "applied" {
		status = "installed"
	}
	lines = append(lines, fmt.Sprintf("CodeMap Pi installer — %s", status))
	if r.Error != "" {
		lines = append(lines, fmt.Sprintf("Error: %s", r.Error))
	}
	lines = append(lines, "Checks:")
	for _, c := range r.Checks {
		icon := "✅"
		if !c.Passed {
			icon = "❌"
		}
		lines = append(lines, fmt.Sprintf("  %s %s  (%s)", icon, c.Name, c.Info))
	}
	lines = append(lines, "Actions:")
	for _, a := range r.Actions {
		icon := "—"
		if a.Changed {
			icon = "→"
		}
		line := fmt.Sprintf("  %s %s -> %s", icon, a.Source, a.Target)
		if a.Skipped != "" {
			line += "  ⚠ " + a.Skipped
		}
		lines = append(lines, line)
	}
	if len(r.Warnings) > 0 {
		lines = append(lines, "Warnings:")
		for _, w := range r.Warnings {
			lines = append(lines, fmt.Sprintf("  ⚠ %s", w))
		}
	}
	return strings.Join(lines, "\n")
}

// JSON returns machine-readable JSON output.
func (r *InstallResult) JSON() string {
	b, _ := json.MarshalIndent(r, "", "  ")
	return string(b)
}

// --- helpers ---

// checkPATHWarning returns a warning message if targetDir is not in PATH.
func checkPATHWarning(targetDir string) string {
	pathEnv := os.Getenv("PATH")
	entries := filepath.SplitList(pathEnv)
	for _, entry := range entries {
		absEntry, err := filepath.Abs(entry)
		if err != nil {
			continue
		}
		absTarget, err := filepath.Abs(targetDir)
		if err != nil {
			continue
		}
		if absEntry == absTarget {
			return ""
		}
	}
	return fmt.Sprintf("%s not in PATH — add 'export PATH=\"$HOME/.local/bin:$PATH\"' to your shell config", targetDir)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func needsCopy(src, dst string) (bool, string) {
	if !fileExists(dst) {
		return true, ""
	}
	srcData, err := os.ReadFile(src)
	if err != nil {
		// Source unreadable: surface error so install output is explicit.
		return false, "source unreadable: " + err.Error()
	}
	dstData, err := os.ReadFile(dst)
	if err != nil {
		return true, ""
	}
	return string(srcData) != string(dstData), ""
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
