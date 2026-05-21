package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TemplateKind identifies the type of integration artifact.
type TemplateKind string

const (
	TemplateSkill TemplateKind = "skill"
	TemplateTool  TemplateKind = "tool"
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
	Error     string         `json:"error,omitempty"`
	Timestamp string         `json:"timestamp"`
}

// CheckResult describes a pre-flight check.
type CheckResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Info   string `json:"info,omitempty"`
}

// ActionResult describes an action taken or planned.
type ActionResult struct {
	Kind    string `json:"kind"` // "copy"
	Source  string `json:"source"`
	Target  string `json:"target"`
	Changed bool   `json:"changed"`
}

// Installer sets up codemap integration into Pi runtime.
type Installer struct {
	// RepoRoot is the root of the codemap project (where integrations/ lives).
	RepoRoot string
	// PiRuntimeBase is the base directory of the Pi agent runtime (~/.pi).
	PiRuntimeBase string
	// SkillTargetDir is where skills are installed.
	SkillTargetDir string
	// ToolTargetDir is where tools are installed.
	ToolTargetDir string
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
		RepoRoot:       repoRoot,
		PiRuntimeBase:  piBase,
		SkillTargetDir: filepath.Join(piBase, "skills"),
		ToolTargetDir:  filepath.Join(piBase, "tools"),
	}
}

// Templates returns the list of template files to install and their runtime targets.
func (i *Installer) Templates() []TemplateRef {
	skillSrc := filepath.Join(i.RepoRoot, "integrations", "pi", "skills", "codemap-usage", "SKILL.md")
	skillDst := filepath.Join(i.SkillTargetDir, "codemap-usage", "SKILL.md")
	toolSrc := filepath.Join(i.RepoRoot, "integrations", "pi", "tools", "codemap-tool.json")
	toolDst := filepath.Join(i.ToolTargetDir, "codemap-tool.json")
	return []TemplateRef{
		{Source: skillSrc, Destination: skillDst, Kind: TemplateSkill},
		{Source: toolSrc, Destination: toolDst, Kind: TemplateTool},
	}
}

// Run executes pre-flight checks and (if not dry-run) applies install actions.
func (i *Installer) Run() *InstallResult {
	result := &InstallResult{
		Timestamp: strings.ReplaceAll(fmt.Sprintf("%v", os.Getenv("GOTOOLCHAIN")), " ", "T"),
	}
	if result.Timestamp == "" {
		result.Timestamp = "now"
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
	return result
}

// runChecks performs pre-flight checks.
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

	// Check each template exists in source
	for _, t := range i.Templates() {
		results = append(results, CheckResult{
			Name:   "template_" + string(t.Kind),
			Passed: fileExists(t.Source),
			Info:   t.Source,
		})
	}

	// Check Pi runtime dir is accessible: exists, or parent exists (installer creates via MkdirAll).
	// We use a simple heuristic: the home dir component must be reachable.
	// This avoids false failures when running from a test tmpdir that lacks a real ~/.pi.
	piParent := filepath.Dir(i.PiRuntimeBase)
	homeDir := filepath.Dir(piParent)
	checkPassed := dirExists(i.PiRuntimeBase) || (dirExists(piParent) && dirExists(homeDir))
	results = append(results, CheckResult{
		Name:   "pi_runtime",
		Passed: checkPassed,
		Info:   i.PiRuntimeBase,
	})

	return results
}

// planActions computes what would change.
func (i *Installer) planActions() []ActionResult {
	var actions []ActionResult
	for _, t := range i.Templates() {
		changed := needsCopy(t.Source, t.Destination)
		actions = append(actions, ActionResult{
			Kind:    "copy",
			Source:  t.Source,
			Target:  t.Destination,
			Changed: changed,
		})
	}
	return actions
}

// apply copies templates to runtime targets idempotently.
func (i *Installer) apply(actions []ActionResult) error {
	// Ensure parent dirs exist
	for _, a := range actions {
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
		lines = append(lines, fmt.Sprintf("  %s %s -> %s", icon, a.Source, a.Target))
	}
	return strings.Join(lines, "\n")
}

// JSON returns machine-readable JSON output.
func (r *InstallResult) JSON() string {
	b, _ := json.MarshalIndent(r, "", "  ")
	return string(b)
}

// --- helpers ---

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func needsCopy(src, dst string) bool {
	if !fileExists(dst) {
		return true
	}
	srcData, err := os.ReadFile(src)
	if err != nil {
		return false
	}
	dstData, err := os.ReadFile(dst)
	if err != nil {
		return true
	}
	return string(srcData) != string(dstData)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
