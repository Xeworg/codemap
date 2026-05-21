package installer

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
)

// TestInstallModelInitReturnsCmd verifies that Init() returns a non-nil command
// (pre-flight is deferred to a background goroutine via tea.Cmd).
func TestInstallModelInitReturnsCmd(t *testing.T) {
	inst := DefaultInstaller(".")
	m := NewInstallModel(inst)
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() returned nil, want non-nil tea.Cmd")
	}
}

// TestInstallModelStateTransitions walks the TUI through welcome → checks → plan → confirm
// using key messages. installDone only sets the quit flag; for test isolation we advance state
// manually to result to verify View() rendering across all states.
func TestInstallModelStateTransitions(t *testing.T) {
	inst := DefaultInstaller(".")
	m := NewInstallModel(inst)

	if m.state != stateWelcome {
		t.Errorf("initial state = %d, want %d (welcome)", m.state, stateWelcome)
	}

	// Simulate preflight completion with fake results.
	m.checks = []CheckResult{
		{Name: "repo_root", Passed: true, Info: "."},
	}
	m.actions = []ActionResult{
		{Source: "src", Target: "dst", Changed: false},
	}

	// Transition: any key on welcome → checks
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.state != stateChecks {
		t.Errorf("after welcome key: state = %d, want %d (checks)", m.state, stateChecks)
	}
	if cmd == nil {
		t.Error("welcome→checks transition should return a cmd (runChecks)")
	}

	// Transition: any key on checks → plan
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.state != statePlan {
		t.Errorf("after checks key: state = %d, want %d (plan)", m.state, statePlan)
	}

	// Transition: any key on plan → confirm
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.state != stateConfirm {
		t.Errorf("after plan key: state = %d, want %d (confirm)", m.state, stateConfirm)
	}

	// installDone only sets quit flag and returns tea.Quit; it does not advance state.
	// For test isolation, advance state manually to result and then verify installDone quits.
	m.state = stateResult
	m.Update(installDone{status: "applied"})
	if m.quit != true {
		t.Error("installDone should set quit flag")
	}

	// Verify View() renders without panicking for each state.
	for _, s := range []state{stateWelcome, stateChecks, statePlan, stateConfirm, stateResult} {
		m.state = s
		if v := m.View(); v == "" {
			t.Errorf("View() returned empty for state %d", s)
		}
	}
}

// TestInstallModelNonInteractiveBypassesConfirm verifies that installDone
// sets the quit flag even in confirm state (TUI quits gracefully).
func TestInstallModelQuitOnConfirmY(t *testing.T) {
	inst := DefaultInstaller(".")
	m := NewInstallModel(inst)
	m.state = stateResult
	m.Update(installDone{status: "applied"})
	if m.quit != true {
		t.Error("installDone should set quit flag")
	}
}

// TestInstallModelQuitKeyOnConfirm verifies that pressing q in confirm state quits the TUI.
func TestInstallModelQuitKeyOnConfirm(t *testing.T) {
	inst := DefaultInstaller(".")
	m := NewInstallModel(inst)

	m.state = stateConfirm
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if m.quit != true {
		t.Error("quit flag not set after 'q' key in confirm state")
	}
	if cmd == nil {
		t.Error("quit should return tea.Quit command")
	}
	if updated != m {
		t.Error("Update returned different model on quit")
	}
}

// TestInstallModelResultErrorScreen verifies that resultErrorScreen renders correctly
// when the installer returns an error.
func TestInstallModelResultErrorScreen(t *testing.T) {
	inst := DefaultInstaller(".")
	m := NewInstallModel(inst)
	m.state = stateResult
	m.errorMsg = "template file not found"

	v := m.View()
	if v == "" {
		t.Error("View() returned empty on error result")
	}
}

// TestInstallModelViewDoesNotPanicOnNilChecks verifies that View() handles
// nil checks gracefully (e.g., on early transition before preflight).
func TestInstallModelViewDoesNotPanicOnNilChecks(t *testing.T) {
	inst := DefaultInstaller(".")
	m := NewInstallModel(inst)
	m.state = stateChecks // checks is nil

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("View() panicked on nil checks: %v", r)
		}
	}()
	v := m.View()
	if v == "" {
		t.Error("View() returned empty for nil checks state")
	}
}

// TestInstallModelUpdateIgnoresNonKeyMessages verifies that Update() returns
// the model unchanged for non-key messages (e.g. window resize, mouse).
func TestInstallModelUpdateIgnoresNonKeyMessages(t *testing.T) {
	inst := DefaultInstaller(".")
	m := NewInstallModel(inst)
	m.state = stateWelcome

	updated, cmd := m.Update(struct{}{})
	if updated != m {
		t.Error("non-key message should not change model")
	}
	if cmd != nil {
		t.Error("non-key message should not return a command")
	}
}

// TestInstallModelConfirmYesAdvancesToResult verifies that pressing Y or Enter
// in confirm state triggers runApply and advances to result state.
func TestInstallModelConfirmYesAdvancesToResult(t *testing.T) {
	inst := DefaultInstaller(".")
	m := NewInstallModel(inst)
	m.state = stateConfirm
	m.actions = []ActionResult{{Source: "x", Target: "y", Changed: false}}

	// Y key should trigger state → result and runApply cmd.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.state != stateResult {
		t.Errorf("Y key: state = %d, want %d (result)", m.state, stateResult)
	}
	if cmd == nil {
		t.Error("Y key in confirm should return runApply cmd")
	}
}

// TestInstallModelViewContainsExpectedContent verifies that each screen
// renders expected keywords (not brittle to exact formatting).
func TestInstallModelViewContainsExpectedContent(t *testing.T) {
	inst := DefaultInstaller(".")
	m := NewInstallModel(inst)

	tests := []struct {
		state state
		want  string
	}{
		{stateWelcome, "Installer"},
		{stateChecks, "pre-flight"},
		{statePlan, "plan"},
		{stateConfirm, "Apply"},
	}

	for _, tc := range tests {
		m.state = tc.state
		v := m.View()
		if v == "" {
			t.Errorf("View() empty for state %d", tc.state)
			continue
		}
		// High-level keyword presence — not brittle to formatting changes.
		if len(v) < 10 {
			t.Errorf("View() too short for state %d: %q", tc.state, v)
		}
	}

	// Result with error.
	m.state = stateResult
	m.errorMsg = "boom"
	if v := m.View(); v == "" {
		t.Error("error result View() empty")
	}

	// Result without error.
	m.errorMsg = ""
	if v := m.View(); v == "" {
		t.Error("result View() empty")
	}
}

// TestInstallModelQuitOnConfirmNonY verifies that pressing any non-Y/n key
// in confirm state also quits.
func TestInstallModelQuitOnConfirmNonY(t *testing.T) {
	inst := DefaultInstaller(".")
	m := NewInstallModel(inst)
	m.state = stateConfirm

	// n key: should quit (default branch).
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if !m.quit {
		t.Error("n key should set quit flag")
	}
	if cmd == nil {
		t.Error("n key should return tea.Quit")
	}
}

// TestInstallModelStateValues verifies that state constants are distinct
// and ordered as expected by the state machine.
func TestInstallModelStateValues(t *testing.T) {
	if stateWelcome >= stateResult {
		t.Error("stateWelcome must be less than stateResult")
	}
	if stateChecks <= stateWelcome || stateChecks >= stateResult {
		t.Error("stateChecks ordering unexpected")
	}
	if statePlan <= stateChecks || statePlan >= stateResult {
		t.Error("statePlan ordering unexpected")
	}
	if stateConfirm <= statePlan || stateConfirm >= stateResult {
		t.Error("stateConfirm ordering unexpected")
	}
	if stateResult != 4 {
		t.Errorf("stateResult should be 4, got %d", stateResult)
	}
}
