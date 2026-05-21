package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplates(t *testing.T) {
	tmp := t.TempDir()
	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	extDir := filepath.Join(tmp, "integrations", "pi", "extensions")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(extDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill content"), 0644)
	os.WriteFile(filepath.Join(extDir, "codemap-extension.ts"), []byte("extension content"), 0644)

	home := t.TempDir()
	piBase := filepath.Join(home, ".pi", "agent")
	i := &Installer{
		RepoRoot:           tmp,
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
	}

	templates := i.Templates()
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
	if templates[0].Kind != TemplateSkill {
		t.Errorf("first template should be skill, got %v", templates[0].Kind)
	}
	if templates[1].Kind != TemplateExtension {
		t.Errorf("second template should be extension, got %v", templates[1].Kind)
	}
}

func TestRunChecksRepoNotFound(t *testing.T) {
	home := t.TempDir()
	piBase := filepath.Join(home, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	i := &Installer{RepoRoot: "/nonexistent", PiRuntimeBase: piBase, SkillTargetDir: "", ExtensionTargetDir: ""}
	checks := i.runChecks()
	if checks[0].Passed {
		t.Error("repo_root should fail for nonexistent path")
	}
}

func TestRunChecksRuntimeMissing(t *testing.T) {
	// piBase intentionally not created — simulating first-time install

	integrations := filepath.Join(t.TempDir(), "integrations", "pi", "skills", "codemap-usage")
	extDir := filepath.Join(t.TempDir(), "integrations", "pi", "extensions")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(extDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill"), 0644)
	os.WriteFile(filepath.Join(extDir, "codemap-extension.ts"), []byte("tool"), 0644)

	piBase := filepath.Join(t.TempDir(), ".pi", "agent")
	i := &Installer{
		RepoRoot:           t.TempDir(),
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
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
	extDir := filepath.Join(tmp, "integrations", "pi", "extensions")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(extDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill"), 0644)
	os.WriteFile(filepath.Join(extDir, "codemap-extension.ts"), []byte("tool"), 0644)

	i := &Installer{
		RepoRoot:           tmp,
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
		BinaryTargetDir:    filepath.Join(t.TempDir(), "bin"),
		BinaryName:         "codemap",
	}
	checks := i.runChecks()
	// 5 template checks + binary_source check + pi_runtime = 6
	if len(checks) != 6 {
		t.Errorf("expected 6 checks (5 templates + pi_runtime + binary_source), got %d: %v", len(checks), checks)
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
	extDir := filepath.Join(tmp, "integrations", "pi", "extensions")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(extDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill content"), 0644)
	os.WriteFile(filepath.Join(extDir, "codemap-extension.ts"), []byte("extension content"), 0644)

	skillDst := filepath.Join(piBase, "skills", "codemap-usage", "SKILL.md")
	extDst := filepath.Join(piBase, "extensions", "codemap-extension.ts")
	os.MkdirAll(filepath.Dir(skillDst), 0755)
	os.MkdirAll(filepath.Dir(extDst), 0755)
	os.WriteFile(skillDst, []byte("skill content"), 0644)
	os.WriteFile(extDst, []byte("extension content"), 0644)

	i := &Installer{
		RepoRoot:           tmp,
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
		BinaryTargetDir:    filepath.Join(t.TempDir(), "bin"),
		BinaryName:         "codemap",
		DryRun:             true,
	}
	result := i.Run()
	if result.Status != "dry-run" {
		t.Errorf("expected dry-run, got %s", result.Status)
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
	for _, a := range result.Actions {
		// copy actions should not be marked changed if dest already matches src
		if a.Kind == "copy" && a.Changed {
			t.Errorf("copy action should not be marked changed when dest matches src")
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
	extDir := filepath.Join(tmp, "integrations", "pi", "extensions")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(extDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill explicit"), 0644)
	os.WriteFile(filepath.Join(extDir, "codemap-extension.ts"), []byte("extension explicit"), 0644)

	i := &Installer{
		RepoRoot:           tmp, // explicit path, not "."
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
		BinaryTargetDir:    filepath.Join(t.TempDir(), "bin"),
		BinaryName:         "codemap",
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
	extDir := filepath.Join(tmp, "integrations", "pi", "extensions")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(extDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill v1"), 0644)
	os.WriteFile(filepath.Join(extDir, "codemap-extension.ts"), []byte("extension v1"), 0644)

	i := &Installer{
		RepoRoot:           tmp,
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
		BinaryTargetDir:    filepath.Join(t.TempDir(), "bin"),
		BinaryName:         "codemap",
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
	extDir := filepath.Join(tmp, "integrations", "pi", "extensions")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(extDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill fresh"), 0644)
	os.WriteFile(filepath.Join(extDir, "codemap-extension.ts"), []byte("extension fresh"), 0644)

	i := &Installer{
		RepoRoot:           tmp,
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
		BinaryTargetDir:    filepath.Join(t.TempDir(), "bin"),
		BinaryName:         "codemap",
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
	extDst := filepath.Join(piBase, "extensions", "codemap-extension.ts")
	if data, err := os.ReadFile(skillDst); err != nil || string(data) != "skill fresh" {
		t.Errorf("skill should be installed, got data=%q err=%v", data, err)
	}
	if data, err := os.ReadFile(extDst); err != nil || string(data) != "extension fresh" {
		t.Errorf("extension should be installed, got data=%q err=%v", data, err)
	}
}

func TestDryRunShowsChanges(t *testing.T) {
	tmp := t.TempDir()
	home := t.TempDir()
	piBase := filepath.Join(home, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	extDir := filepath.Join(tmp, "integrations", "pi", "extensions")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(extDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill fresh"), 0644)
	os.WriteFile(filepath.Join(extDir, "codemap-extension.ts"), []byte("extension fresh"), 0644)

	i := &Installer{
		RepoRoot:           tmp,
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
		BinaryTargetDir:    filepath.Join(t.TempDir(), "bin"),
		BinaryName:         "codemap",
		DryRun:             true,
	}
	result := i.Run()
	if result.Status != "dry-run" {
		t.Errorf("expected dry-run, got %s (error: %s)", result.Status, result.Error)
	}
	// Verify copy actions are marked changed (binary install may skip due to missing source in temp dir)
	var copyActions int
	for _, a := range result.Actions {
		if a.Kind == "copy" {
			copyActions++
			if !a.Changed {
				t.Errorf("copy action should be marked changed in dry-run for fresh install")
			}
		}
	}
	if copyActions == 0 {
		t.Errorf("expected at least one copy action, got none")
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

func detectRepoRoot() (string, error) {
	// Heuristic: look for go.mod upward from current dir
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			return cwd, nil
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}
	return "", fmt.Errorf("go.mod not found")
}

func TestValidateBinaryName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "codemap", false},
		{"valid with dash", "my-binary", false},
		{"valid with underscore", "my_binary_1", false},
		{"valid with dots", "v1.2.3", false},
		{"valid with version", "codemap-v0", false},
		{"empty", "", true},
		{"path separator slash", "foo/bar", true},
		{"path separator backslash", "foo\\bar", true},
		{"starts with dot", ".codemap", true},
		{"too long", strings.Repeat("a", 65), true},
		{"exactly 64", strings.Repeat("a", 64), false},
		{"space", "my binary", true},
		{"special char", "my@binary", true},
		{"newline", "my\nbinary", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBinaryName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBinaryName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestInstallerValidate(t *testing.T) {
	i := &Installer{BinaryName: ""}
	if err := i.Validate(); err == nil {
		t.Error("expected error for empty BinaryName")
	}

	i = &Installer{BinaryName: "codemap"}
	if err := i.Validate(); err != nil {
		t.Errorf("expected no error for valid BinaryName, got: %v", err)
	}
}

func TestPlanBinaryInstallSourceNotFound(t *testing.T) {
	tmp := t.TempDir()
	piBase := filepath.Join(tmp, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	// No cmd/codemap/main.go in tmp repo
	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	extDir := filepath.Join(tmp, "integrations", "pi", "extensions")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(extDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill"), 0644)
	os.WriteFile(filepath.Join(extDir, "codemap-extension.ts"), []byte("ext"), 0644)

	i := &Installer{
		RepoRoot:           tmp,
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
		BinaryTargetDir:    filepath.Join(tmp, "bin"),
		BinaryName:         "codemap",
	}
	changed, skip, _ := i.planBinaryInstall(
		filepath.Join(tmp, "cmd", "codemap"),
		filepath.Join(tmp, "bin", "codemap"),
	)
	if changed {
		t.Error("planBinaryInstall should return changed=false when source unavailable")
	}
	if skip == "" {
		t.Error("planBinaryInstall should set skip reason when source unavailable")
	}
}

func TestPlanBinaryInstallDryRunSkipsBuild(t *testing.T) {
	tmp := t.TempDir()
	piBase := filepath.Join(tmp, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	// Create minimal binary source
	cmdDir := filepath.Join(tmp, "cmd", "codemap")
	os.MkdirAll(cmdDir, 0755)
	os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)

	// Create an existing binary destination
	binDir := filepath.Join(tmp, "bin")
	os.MkdirAll(binDir, 0755)
	os.WriteFile(filepath.Join(binDir, "codemap"), []byte("fake binary"), 0755)

	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	extDir := filepath.Join(tmp, "integrations", "pi", "extensions")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(extDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill"), 0644)
	os.WriteFile(filepath.Join(extDir, "codemap-extension.ts"), []byte("ext"), 0644)

	i := &Installer{
		RepoRoot:           tmp,
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
		BinaryTargetDir:    binDir,
		BinaryName:         "codemap",
		DryRun:             true,
	}
	changed, skip, _ := i.planBinaryInstall(
		filepath.Join(tmp, "cmd", "codemap"),
		filepath.Join(binDir, "codemap"),
	)
	if changed {
		t.Error("planBinaryInstall should return changed=false in dry-run when binary exists")
	}
	if skip == "" {
		t.Error("planBinaryInstall should set skip reason in dry-run")
	}
	// Verify NO go build temp file was created (dry-run writes nothing)
	entries, _ := os.ReadDir(os.TempDir())
	for _, e := range entries {
		if strings.Contains(e.Name(), "codemap.install-check") {
			t.Errorf("dry-run created temp file %s — dry-run must not write filesystem", e.Name())
		}
	}
}

func TestDryRunDoesNotBuildOrWrite(t *testing.T) {
	// Use current repo as a real build context
	repoRoot, err := detectRepoRoot()
	if err != nil {
		t.Skip("could not detect repo root for binary test")
	}

	tmp := t.TempDir()
	piBase := filepath.Join(tmp, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	// Create minimal integrations to allow runChecks to pass
	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	extDir := filepath.Join(tmp, "integrations", "pi", "extensions")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(extDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill"), 0644)
	os.WriteFile(filepath.Join(extDir, "codemap-extension.ts"), []byte("ext"), 0644)

	i := &Installer{
		RepoRoot:           repoRoot, // real repo with cmd/codemap
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
		BinaryTargetDir:    filepath.Join(tmp, "bin"),
		BinaryName:         "codemap",
		DryRun:             true,
	}

	result := i.Run()
	if result.Status != "dry-run" {
		t.Errorf("expected dry-run, got %s (error: %s)", result.Status, result.Error)
	}
	// The binary action should NOT be marked as changed in dry-run
	for _, a := range result.Actions {
		if a.Kind == "install_binary" && a.Changed {
			t.Errorf("install_binary action should not be marked Changed in dry-run")
		}
	}
	// Ensure no binary was written to the target dir
	if _, err := os.Stat(filepath.Join(tmp, "bin", "codemap")); err == nil {
		t.Errorf("dry-run should not write binary to %s", filepath.Join(tmp, "bin"))
	}
}

func TestApplyBinaryInstallHappyPath(t *testing.T) {
	repoRoot, err := detectRepoRoot()
	if err != nil {
		t.Skip("could not detect repo root for binary test")
	}

	tmp := t.TempDir()
	piBase := filepath.Join(tmp, ".pi", "agent")
	os.MkdirAll(piBase, 0755)

	integrations := filepath.Join(tmp, "integrations", "pi", "skills", "codemap-usage")
	extDir := filepath.Join(tmp, "integrations", "pi", "extensions")
	os.MkdirAll(integrations, 0755)
	os.MkdirAll(extDir, 0755)
	os.WriteFile(filepath.Join(integrations, "SKILL.md"), []byte("skill"), 0644)
	os.WriteFile(filepath.Join(extDir, "codemap-extension.ts"), []byte("ext"), 0644)

	i := &Installer{
		RepoRoot:           repoRoot,
		PiRuntimeBase:      piBase,
		SkillTargetDir:     filepath.Join(piBase, "skills"),
		ExtensionTargetDir: filepath.Join(piBase, "extensions"),
		BinaryTargetDir:    filepath.Join(tmp, "bin"),
		BinaryName:         "codemap",
	}

	result := i.Run()
	if result.Error != "" {
		t.Fatalf("install failed: %s", result.Error)
	}
	if result.Status != "applied" {
		t.Fatalf("expected applied, got %s", result.Status)
	}

	// Verify binary was actually written
	binPath := filepath.Join(tmp, "bin", "codemap")
	info, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("binary not written to %s: %v", binPath, err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Errorf("binary %s is not executable", binPath)
	}

	// Second run should be up-to-date
	r2 := i.Run()
	if r2.Status != "up-to-date" {
		t.Errorf("second run should be up-to-date, got %s", r2.Status)
	}
}

func TestInvalidBinaryNameRejectsPathTraversal(t *testing.T) {
	i := &Installer{
		RepoRoot:           ".",
		PiRuntimeBase:      "/tmp/.pi",
		SkillTargetDir:     "/tmp/.pi/skills",
		ExtensionTargetDir: "/tmp/.pi/extensions",
		BinaryTargetDir:    "/tmp/bin",
		BinaryName:         "../../../etc/passwd",
	}
	if err := i.Validate(); err == nil {
		t.Error("expected error for path traversal BinaryName")
	}
}
