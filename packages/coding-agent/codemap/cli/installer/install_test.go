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

func TestRunChecks(t *testing.T) {
	tmp := t.TempDir()
	home := t.TempDir()
	piBase := filepath.Join(home, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	i := &Installer{RepoRoot: "/nonexistent", PiRuntimeBase: piBase, SkillTargetDir: "", ToolTargetDir: ""}
	checks := i.runChecks()
	if checks[0].Passed {
		t.Error("repo_root should fail for nonexistent path")
	}

	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	toolDir := filepath.Join(tmp, "integrations", "pi", "tools")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(toolDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill"), 0644)
	os.WriteFile(filepath.Join(toolDir, "codemap-tool.json"), []byte("tool"), 0644)

	i = &Installer{
		RepoRoot:       tmp,
		PiRuntimeBase:  piBase,
		SkillTargetDir: filepath.Join(piBase, "skills"),
		ToolTargetDir:  filepath.Join(piBase, "tools"),
	}
	checks = i.runChecks()
	if len(checks) != 5 {
		t.Errorf("expected 5 checks, got %d: %v", len(checks), checks)
	}
	for _, c := range checks {
		if !c.Passed {
			t.Errorf("check %q should pass but did not: %v", c.Name, c)
		}
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
