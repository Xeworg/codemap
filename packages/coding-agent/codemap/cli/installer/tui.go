package installer

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TUI state machine for the install wizard.
type state int

const (
	stateWelcome state = iota
	stateChecks
	statePlan
	stateConfirm
	stateResult
)

var (
	bold    = lipgloss.NewStyle().Bold(true)
	success = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	warning = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))
	errorS  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4949"))
	dimmed  = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	info    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B9EFF"))
)

// InstallModel is the Bubble Tea model for the install TUI.
type InstallModel struct {
	installer *Installer
	state     state
	checks    []CheckResult
	actions   []ActionResult
	errorMsg  string
	quit      bool
}

// NewInstallModel creates a TUI install model backed by an Installer.
func NewInstallModel(installer *Installer) *InstallModel {
	return &InstallModel{installer: installer, state: stateWelcome}
}

// Init runs pre-flight checks as soon as the TUI starts.
func (m *InstallModel) Init() tea.Cmd {
	return m.runChecks()
}

// Update handles messages and state transitions.
func (m *InstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch tmsg := msg.(type) {
	case preflightDone:
		m.checks = tmsg.checks
		m.actions = tmsg.actions
		return m, nil
	case installDone:
		m.quit = true
		return m, tea.Quit
	case installError:
		m.errorMsg = tmsg.msg
		return m, nil
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch m.state {
	case stateWelcome:
		m.state = stateChecks
		return m, m.runChecks()
	case stateChecks:
		m.state = statePlan
		return m, nil
	case statePlan:
		m.state = stateConfirm
		return m, nil
	case stateConfirm:
		switch key.String() {
		case "y", "Y", "enter":
			m.state = stateResult
			return m, m.runApply()
		default:
			m.quit = true
			return m, tea.Quit
		}
	case stateResult:
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

// View renders the current screen.
func (m *InstallModel) View() string {
	var b strings.Builder
	switch m.state {
	case stateWelcome:
		b.WriteString(welcomeScreen())
	case stateChecks:
		b.WriteString(checksScreen(m.checks))
	case statePlan:
		b.WriteString(planScreen(m.actions))
		b.WriteString(confirmPrompt())
	case stateConfirm:
		b.WriteString(planScreen(m.actions))
		b.WriteString(confirmPrompt())
	case stateResult:
		if m.errorMsg != "" {
			b.WriteString(resultErrorScreen(m.errorMsg))
		} else {
			b.WriteString(resultScreen(m.installer.Run().Status))
		}
	}
	if !m.quit {
		b.WriteString(bottomNav())
	}
	return b.String()
}

// runChecks performs pre-flight synchronously to populate checks/actions.
func (m *InstallModel) runChecks() tea.Cmd {
	return func() tea.Msg {
		res := m.installer.Run()
		return preflightDone{checks: res.Checks, actions: res.Actions}
	}
}

// runApply applies the installation.
func (m *InstallModel) runApply() tea.Cmd {
	return func() tea.Msg {
		res := m.installer.Run()
		if res.Status == "error" {
			return installError{msg: res.Error}
		}
		return installDone{status: res.Status}
	}
}

type preflightDone struct {
	checks  []CheckResult
	actions []ActionResult
}
type installDone struct{ status string }
type installError struct{ msg string }

// ---- Screen builders ----

func welcomeScreen() string {
	return bold.Render("CodeMap Pi Installer\n") +
		dimmed.Render("────────────────────────────────────────") + "\n\n" +
		"Welcome! This wizard will install the CodeMap skill and tool\n" +
		"into your Pi runtime so agents can query Go symbols, history\n" +
		"and evidence before editing code.\n\n" +
		info.Render("Press any key to start →")
}

func checksScreen(checks []CheckResult) string {
	var b strings.Builder
	b.WriteString(bold.Render("Running pre-flight checks\n"))
	b.WriteString(dimmed.Render("────────────────────────────────────────") + "\n\n")
	for _, c := range checks {
		icon := success.Render("✅")
		if !c.Passed {
			icon = errorS.Render("❌")
		}
		extra := ""
		if c.Exists != nil {
			if *c.Exists {
				extra = dimmed.Render(" (exists)")
			} else {
				extra = dimmed.Render(" (will be created)")
			}
		}
		b.WriteString(fmt.Sprintf("%s %-18s%s%s\n", icon, c.Name, extra, dimmed.Render("  "+c.Info)))
	}
	b.WriteString("\n" + info.Render("Press any key to see the plan →"))
	return b.String()
}

func planScreen(actions []ActionResult) string {
	var b strings.Builder
	b.WriteString(bold.Render("Install plan\n"))
	b.WriteString(dimmed.Render("────────────────────────────────────────") + "\n\n")
	for _, a := range actions {
		if a.Changed {
			b.WriteString(fmt.Sprintf("%s %s → %s\n",
				warning.Render("→"), bold.Render(a.Source), dimmed.Render(a.Target)))
		} else {
			b.WriteString(fmt.Sprintf("%s %s (up-to-date)\n",
				success.Render("✓"), dimmed.Render(a.Source)))
		}
	}
	return b.String()
}

func confirmPrompt() string {
	return "\n" + bold.Render("Apply these changes?") + " [Y/n]\n\n"
}

func resultScreen(status string) string {
	var icon string
	var text string
	switch status {
	case "applied":
		icon = success.Render("✅")
		text = "Installed successfully!"
	case "up-to-date":
		icon = info.Render("ℹ")
		text = "Already up-to-date."
	default:
		icon = errorS.Render("❌")
		text = "Completed with status: " + status
	}
	return "\n" + icon + " " + bold.Render(text) + "\n\n" +
		dimmed.Render("Press any key to exit\n")
}

func resultErrorScreen(err string) string {
	return "\n" + errorS.Render("❌ Error: "+err) + "\n\n" +
		dimmed.Render("Press any key to exit\n")
}

func bottomNav() string {
	return "\n" + dimmed.Render("Enter confirm   q quit")
}

// RunTUI runs the Bubble Tea installer. Returns exit code 0 on success.
func RunTUI(installer *Installer) int {
	model := NewInstallModel(installer)
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		return 1
	}
	return 0
}
