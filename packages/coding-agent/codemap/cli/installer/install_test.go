package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplates(t *testing.T) {
	tmp := t.TempDir()
	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	toolDir := filepath.Join(tmp, "integrations", "pi", "tools")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(toolDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill content"), 0644)
	os.WriteFile(filepath.Join(toolDir, "codemap-tool.json"), []byte("tool content"), 0644)

	home := t.TempDir()
	piBase := filepath.Join(home, ".pi", "agent")
	i := &Installer{
		RepoRoot:       tmp,
		PiRuntimeBase:  piBase,
		SkillTargetDir: filepath.Join(piBase, "skills"),
		ToolTargetDir:  filepath.Join(piBase, "tools"),
	}

	templates := i.Templates()
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
	if templates[0].Kind != TemplateSkill {
		t.Errorf("first template should be skill, got %v", templates[0].Kind)
	}
	if templates[1].Kind != TemplateTool {
		t.Errorf("second template should be tool, got %v", templates[1].Kind)
	}
}

func TestRunChecksRepoNotFound(t *testing.T) {
	home := t.TempDir()
	piBase := filepath.Join(home, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	i := &Installer{RepoRoot: "/nonexistent", PiRuntimeBase: piBase, SkillTargetDir: "", ToolTargetDir: ""}
	checks := i.runChecks()
	if checks[0].Passed {
		t.Error("repo_root should fail for nonexistent path")
	}
}

func TestRunChecksRuntimeMissing(t *testing.T) {
	// piBase intentionally not created — simulating first-time install

	integrations := filepath.Join(t.TempDir(), "integrations", "pi", "skills", "codemap-usage")
	toolDir := filepath.Join(t.TempDir(), "integrations", "pi", "tools")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(toolDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill"), 0644)
	os.WriteFile(filepath.Join(toolDir, "codemap-tool.json"), []byte("tool"), 0644)

	piBase := filepath.Join(t.TempDir(), ".pi", "agent")
	i := &Installer{
		RepoRoot:       t.TempDir(),
		PiRuntimeBase:  piBase,
		SkillTargetDir: filepath.Join(piBase, "skills"),
		ToolTargetDir:  filepath.Join(piBase, "tools"),
	}
	checks := i.runChecks()
	// pi_runtime check should pass (exists=false) so first install is allowed
	var rt CheckResult
	for _, c := range checks {
		if c.Name == "pi_runtime" {
			rt = c
			break
		}
	}
	if rt.Name == "" {
		t.Fatal("pi_runtime check not found")
	}
	if !rt.Passed {
		t.Errorf("pi_runtime should pass even when missing (installer creates it), got: %v", rt)
	}
	if rt.Exists == nil || *rt.Exists {
		t.Errorf("pi_runtime exists should be false, got: %v", rt.Exists)
	}
}

func TestRunChecksRuntimeExists(t *testing.T) {
	tmp := t.TempDir()
	home := t.TempDir()
	piBase := filepath.Join(home, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	toolDir := filepath.Join(tmp, "integrations", "pi", "tools")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(toolDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill"), 0644)
	os.WriteFile(filepath.Join(toolDir, "codemap-tool.json"), []byte("tool"), 0644)

	i := &Installer{
		RepoRoot:       tmp,
		PiRuntimeBase:  piBase,
		SkillTargetDir: filepath.Join(piBase, "skills"),
		ToolTargetDir:  filepath.Join(piBase, "tools"),
	}
	checks := i.runChecks()
	if len(checks) != 5 {
		t.Errorf("expected 5 checks, got %d: %v", len(checks), checks)
	}
	var rt CheckResult
	for _, c := range checks {
		if c.Name == "pi_runtime" {
			rt = c
			break
		}
	}
	if rt.Name == "" {
		t.Fatal("pi_runtime check not found")
	}
	if !rt.Passed {
		t.Errorf("pi_runtime should pass when directory exists, got: %v", rt)
	}
	if rt.Exists == nil || !*rt.Exists {
		t.Errorf("pi_runtime exists should be true, got: %v", rt.Exists)
	}
}

func TestDryRunNoChanges(t *testing.T) {
	tmp := t.TempDir()
	home := t.TempDir()
	piBase := filepath.Join(home, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	toolDir := filepath.Join(tmp, "integrations", "pi", "tools")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(toolDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill content"), 0644)
	os.WriteFile(filepath.Join(toolDir, "codemap-tool.json"), []byte("tool content"), 0644)

	skillDst := filepath.Join(piBase, "skills", "codemap-usage", "SKILL.md")
	toolDst := filepath.Join(piBase, "tools", "codemap-tool.json")
	os.MkdirAll(filepath.Dir(skillDst), 0755)
	os.MkdirAll(filepath.Dir(toolDst), 0755)
	os.WriteFile(skillDst, []byte("skill content"), 0644)
	os.WriteFile(toolDst, []byte("tool content"), 0644)

	i := &Installer{
		RepoRoot:       tmp,
		PiRuntimeBase:  piBase,
		SkillTargetDir: filepath.Join(piBase, "skills"),
		ToolTargetDir:  filepath.Join(piBase, "tools"),
		DryRun:         true,
	}
	result := i.Run()
	if result.Status != "dry-run" {
		t.Errorf("expected dry-run, got %s", result.Status)
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
	for _, a := range result.Actions {
		if a.Changed {
			t.Errorf("no action should be marked changed in dry-run")
		}
	}
}

func TestExplicitRepoPath(t *testing.T) {
	tmp := t.TempDir()
	home := t.TempDir()
	piBase := filepath.Join(home, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	// Create a "foreign" repo at /nonexistent but use explicit repoRoot
	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	toolDir := filepath.Join(tmp, "integrations", "pi", "tools")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(toolDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill explicit"), 0644)
	os.WriteFile(filepath.Join(toolDir, "codemap-tool.json"), []byte("tool explicit"), 0644)

	i := &Installer{
		RepoRoot:       tmp, // explicit path, not "."
		PiRuntimeBase:  piBase,
		SkillTargetDir: filepath.Join(piBase, "skills"),
		ToolTargetDir:  filepath.Join(piBase, "tools"),
	}
	result := i.Run()
	if result.Status != "applied" {
		t.Fatalf("expected applied with explicit repoRoot, got %s: %s", result.Status, result.Error)
	}
	skillDst := filepath.Join(piBase, "skills", "codemap-usage", "SKILL.md")
	if data, _ := os.ReadFile(skillDst); string(data) != "skill explicit" {
		t.Errorf("expected 'skill explicit', got %q", string(data))
	}
}

func TestApplyIdempotent(t *testing.T) {
	tmp := t.TempDir()
	home := t.TempDir()
	piBase := filepath.Join(home, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	toolDir := filepath.Join(tmp, "integrations", "pi", "tools")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(toolDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill v1"), 0644)
	os.WriteFile(filepath.Join(toolDir, "codemap-tool.json"), []byte("tool v1"), 0644)

	i := &Installer{
		RepoRoot:       tmp,
		PiRuntimeBase:  piBase,
		SkillTargetDir: filepath.Join(piBase, "skills"),
		ToolTargetDir:  filepath.Join(piBase, "tools"),
	}

	r1 := i.Run()
	if r1.Status != "applied" {
		t.Fatalf("first run should apply, got %s: %s", r1.Status, r1.Error)
	}

	r2 := i.Run()
	if r2.Status != "up-to-date" {
		t.Errorf("second run should be up-to-date, got %s: %s", r2.Status, r2.Error)
	}

	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill v2"), 0644)
	r3 := i.Run()
	if r3.Status != "applied" {
		t.Errorf("modified source should re-apply, got %s: %s", r3.Status, r3.Error)
	}

	data, _ := os.ReadFile(filepath.Join(piBase, "skills", "codemap-usage", "SKILL.md"))
	if string(data) != "skill v2" {
		t.Errorf("expected 'skill v2', got %q", string(data))
	}
}

func TestApplyCreatesRuntimeOnFirstInstall(t *testing.T) {
	tmp := t.TempDir()
	home := t.TempDir()
	piBase := filepath.Join(home, ".pi", "agent")
	// Deliberately NOT creating piBase — simulate first install

	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	toolDir := filepath.Join(tmp, "integrations", "pi", "tools")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(toolDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill fresh"), 0644)
	os.WriteFile(filepath.Join(toolDir, "codemap-tool.json"), []byte("tool fresh"), 0644)

	i := &Installer{
		RepoRoot:       tmp,
		PiRuntimeBase:  piBase,
		SkillTargetDir: filepath.Join(piBase, "skills"),
		ToolTargetDir:  filepath.Join(piBase, "tools"),
	}

	r1 := i.Run()
	if r1.Status != "applied" {
		t.Fatalf("first install should apply (runtime created), got %s: %s", r1.Status, r1.Error)
	}

	r2 := i.Run()
	if r2.Status != "up-to-date" {
		t.Errorf("second run should be up-to-date, got %s: %s", r2.Status, r2.Error)
	}

	skillDst := filepath.Join(piBase, "skills", "codemap-usage", "SKILL.md")
	toolDst := filepath.Join(piBase, "tools", "codemap-tool.json")
	if data, err := os.ReadFile(skillDst); err != nil || string(data) != "skill fresh" {
		t.Errorf("skill should be installed, got data=%q err=%v", data, err)
	}
	if data, err := os.ReadFile(toolDst); err != nil || string(data) != "tool fresh" {
		t.Errorf("tool should be installed, got data=%q err=%v", data, err)
	}
}

func TestDryRunShowsChanges(t *testing.T) {
	tmp := t.TempDir()
	home := t.TempDir()
	piBase := filepath.Join(home, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	toolDir := filepath.Join(tmp, "integrations", "pi", "tools")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(toolDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill fresh"), 0644)
	os.WriteFile(filepath.Join(toolDir, "codemap-tool.json"), []byte("tool fresh"), 0644)

	i := &Installer{
		RepoRoot:       tmp,
		PiRuntimeBase:  piBase,
		SkillTargetDir: filepath.Join(piBase, "skills"),
		ToolTargetDir:  filepath.Join(piBase, "tools"),
		DryRun:         true,
	}
	result := i.Run()
	if result.Status != "dry-run" {
		t.Errorf("expected dry-run, got %s", result.Status)
	}
	for _, a := range result.Actions {
		if !a.Changed {
			t.Errorf("all actions should be marked changed in dry-run for fresh install")
		}
	}
}

func TestInstallResultPrint(t *testing.T) {
	r := &InstallResult{
		Status: "applied",
		Checks: []CheckResult{
			{Name: "repo_root", Passed: true, Info: "/home/user/repo"},
			{Name: "template_skill", Passed: true, Info: "/home/user/repo/integrations/..."},
		},
		Actions: []ActionResult{
			{Kind: "copy", Source: "/src", Target: "/dst", Changed: true},
		},
	}
	out := r.Print()
	if !strings.Contains(out, "installed") {
		t.Errorf("Print should contain 'installed', got: %s", out)
	}
	if !strings.Contains(out, "✅") {
		t.Errorf("Print should contain check icons, got: %s", out)
	}
}

func TestInstallResultJSON(t *testing.T) {
	r := &InstallResult{
		Status:    "applied",
		Timestamp: "2026-05-21T10:00:00Z",
		Checks:    []CheckResult{{Name: "test", Passed: true}},
		Actions:   []ActionResult{{Kind: "copy", Source: "a", Target: "b", Changed: true}},
	}
	out := r.JSON()
	if !strings.Contains(out, `"status"`) {
		t.Errorf("JSON should contain status key, got: %s", out)
	}
}

func TestNeedsCopySourceUnreadable(t *testing.T) {
	// dst does not exist → true, no skip
	changed, skip := needsCopy("/nonexistent/src", "/tmp/dst")
	if !changed {
		t.Errorf("expected changed=true when dst missing, got false")
	}
	if skip != "" {
		t.Errorf("expected no skip reason, got: %s", skip)
	}
}

func TestNeedsCopySourceReadError(t *testing.T) {
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "existing_dst")
	os.WriteFile(dst, []byte("content"), 0644)

	changed, skip := needsCopy("/nonexistent/source", dst)
	if changed {
		t.Errorf("expected changed=false when source unreadable, got true")
	}
	if skip == "" {
		t.Errorf("expected skip reason when source unreadable, got empty")
	}
}
