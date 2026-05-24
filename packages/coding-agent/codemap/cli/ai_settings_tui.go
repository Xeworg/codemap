package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"codrut/packages/coding-agent/codemap/store"
)

// TUI state machine for AI settings.
type aiState int

const (
	aiStateProvider aiState = iota
	aiStateModel
	aiStateBaseURL
	aiStateAPIKey
	aiStateTimeout
	aiStateConfirm
	aiStateDone
)

// AITUIResult holds the final settings collected by the TUI.
type AITUIResult struct {
	ActiveProvider string
	Model          string
	BaseURL        string
	APIKey         string
	TimeoutSec     int
	Saved          bool
	Error          string
}

// AITUIState mirrors the Bubble Tea model state for AI settings TUI.
type AITUIState struct {
	provider   string // "ollama" or "minimax"
	model      string
	baseURL    string
	apiKey     string
	timeoutSec int
	state      aiState
	saved      bool
	errorMsg   string
	quit       bool
}

// NewAITSModel creates an AI settings TUI model seeded with current config.
func NewAITSModel(provider, model, baseURL, apiKey string, timeoutSec int) *AITUIState {
	return &AITUIState{
		provider:   provider,
		model:      model,
		baseURL:    baseURL,
		apiKey:     apiKey,
		timeoutSec: timeoutSec,
		state:      aiStateProvider,
	}
}

// Init is required by tea.Model.
func (m *AITUIState) Init() tea.Cmd { return nil }

// Update handles user input and transitions.
func (m *AITUIState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch tmsg := msg.(type) {
	case aiSaveDone:
		m.saved = true
		m.quit = true
		return m, tea.Quit
	case aiSaveError:
		m.errorMsg = tmsg.msg
		m.state = aiStateConfirm
		return m, nil
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	k := key.String()

	switch m.state {
	case aiStateProvider:
		if k == "up" || k == "down" {
			if m.provider == "ollama" {
				m.provider = "minimax"
			} else {
				m.provider = "ollama"
			}
		}
		if k == "enter" {
			m.state = aiStateModel
		}
	case aiStateModel:
		if k == "enter" {
			m.state = aiStateBaseURL
		}
		if len(key.Runes) > 0 {
			m.model += string(key.Runes[0])
		}
		if k == "backspace" && len(m.model) > 0 {
			m.model = m.model[:len(m.model)-1]
		}
	case aiStateBaseURL:
		if k == "enter" {
			m.state = aiStateAPIKey
		}
		if len(key.Runes) > 0 {
			m.baseURL += string(key.Runes[0])
		}
		if k == "backspace" && len(m.baseURL) > 0 {
			m.baseURL = m.baseURL[:len(m.baseURL)-1]
		}
	case aiStateAPIKey:
		if k == "enter" {
			m.state = aiStateTimeout
		}
		if len(key.Runes) > 0 {
			m.apiKey += string(key.Runes[0])
		}
		if k == "backspace" && len(m.apiKey) > 0 {
			m.apiKey = m.apiKey[:len(m.apiKey)-1]
		}
	case aiStateTimeout:
		if k == "enter" {
			m.state = aiStateConfirm
		}
		if len(key.Runes) > 0 {
			m.timeoutSec = m.timeoutSec*10 + int(key.Runes[0]-'0')
		}
		if k == "backspace" && m.timeoutSec > 0 {
			m.timeoutSec /= 10
		}
	case aiStateConfirm:
		switch k {
		case "y", "Y", "enter":
			m.state = aiStateDone
			return m, m.saveSettings()
		default:
			m.quit = true
			return m, tea.Quit
		}
	case aiStateDone:
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

var (
	bold    = lipgloss.NewStyle().Bold(true)
	success = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	errorS  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4949"))
	dimmed  = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
)

// View renders the current TUI screen.
func (m *AITUIState) View() string {
	var b strings.Builder
	b.WriteString(bold.Render("CodeMap AI Settings\n"))
	b.WriteString(dimmed.Render("────────────────────────────────────────") + "\n\n")

	switch m.state {
	case aiStateProvider:
		b.WriteString("Select active provider:\n\n")
		b.WriteString(providerRow("ollama", m.provider == "ollama"))
		b.WriteString(providerRow("minimax", m.provider == "minimax"))
		b.WriteString("\n" + dimmed.Render("↑↓ select   enter confirm"))
	case aiStateModel:
		b.WriteString(fmt.Sprintf("Provider: %s\n\n", bold.Render(m.provider)))
		b.WriteString("Model name:\n\n")
		b.WriteString(inputLine(m.model))
		b.WriteString("\n" + dimmed.Render("type model   enter next"))
	case aiStateBaseURL:
		b.WriteString(fmt.Sprintf("Provider: %s   Model: %s\n\n", m.provider, bold.Render(m.model)))
		b.WriteString("Base URL:\n\n")
		b.WriteString(inputLine(m.baseURL))
		b.WriteString("\n" + dimmed.Render("type URL   enter next"))
	case aiStateTimeout:
		b.WriteString(fmt.Sprintf("Provider: %s   Model: %s   URL: %s\n\n",
			m.provider, m.model, m.baseURL))
		b.WriteString("Timeout (seconds):\n\n")
		b.WriteString(inputLine(fmt.Sprintf("%d", m.timeoutSec)))
		b.WriteString("\n" + dimmed.Render("type number   enter next"))
	case aiStateAPIKey:
		b.WriteString(fmt.Sprintf("Provider: %s   Model: %s   URL: %s   Timeout: %ds\n\n",
			m.provider, m.model, m.baseURL, m.timeoutSec))
		b.WriteString("API Key (optional, minimax only):\n\n")
		b.WriteString(inputLine(m.apiKey))
		b.WriteString("\n" + dimmed.Render("type key   enter confirm"))
	case aiStateConfirm:
		b.WriteString(bold.Render("Confirm settings\n"))
		b.WriteString(dimmed.Render("────────────────────────────────────────") + "\n\n")
		b.WriteString(fmt.Sprintf("  Provider:    %s\n", m.provider))
		b.WriteString(fmt.Sprintf("  Model:       %s\n", m.model))
		b.WriteString(fmt.Sprintf("  Base URL:     %s\n", m.baseURL))
		b.WriteString(fmt.Sprintf("  Timeout:     %ds\n", m.timeoutSec))
		if m.apiKey != "" {
			b.WriteString(fmt.Sprintf("  API Key:     %s\n", dimmed.Render("••••••••")))
		}
		b.WriteString("\n" + bold.Render("Apply these settings?") + " [Y/n]")
	case aiStateDone:
		if m.errorMsg != "" {
			b.WriteString(errorS.Render("❌ Error: "+m.errorMsg) + "\n\n")
		} else {
			b.WriteString(success.Render("✅ Settings saved.") + "\n\n")
		}
		b.WriteString(dimmed.Render("Press any key to exit"))
	}

	if !m.quit {
		b.WriteString("\n" + dimmed.Render("q quit"))
	}
	return b.String()
}

func providerRow(name string, selected bool) string {
	icon := "  "
	if selected {
		icon = bold.Render("→ ")
	}
	marker := " "
	if selected {
		marker = bold.Render("◉")
	}
	return fmt.Sprintf("%s%s %s%s\n", icon, marker, name, dimmed.Render("  (active)"))
}

func inputLine(value string) string {
	return "  " + bold.Render("› ") + value + dimmed.Render("▌") + "\n"
}

type aiSaveDone struct{}
type aiSaveError struct{ msg string }

func (m *AITUIState) saveSettings() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		dbPath, err := ResolveDBPath("", ".")
		if err != nil {
			return aiSaveError{msg: "resolve db: " + err.Error()}
		}
		db, err := store.Open(dbPath)
		if err != nil {
			return aiSaveError{msg: "open db: " + err.Error()}
		}
		defer db.Close()

		settings, err := store.GetAISettings(ctx, db.DB)
		if err != nil {
			return aiSaveError{msg: "read settings: " + err.Error()}
		}

		provider := store.Provider(m.provider)
		settings.ActiveProvider = provider
		if provider == "ollama" {
			settings.Ollama.Model = m.model
			settings.Ollama.BaseURL = m.baseURL
			settings.Ollama.TimeoutSec = m.timeoutSec
		} else {
			settings.Minimax.Model = m.model
			settings.Minimax.BaseURL = m.baseURL
			settings.Minimax.TimeoutSec = m.timeoutSec
			// Store the API key in the OS keyring, not in SQLite.
			if m.apiKey != "" {
				if err := store.StoreMinimaxKey(ctx, m.apiKey); err != nil {
					return aiSaveError{msg: "store api key: " + err.Error()}
				}
			}
		}

		if err := store.SaveAISettings(ctx, db.DB, settings); err != nil {
			return aiSaveError{msg: "save settings: " + err.Error()}
		}
		return aiSaveDone{}
	}
}

// RunAITUI runs the AI settings Bubble Tea TUI and returns an exit code.
func RunAITUI(repoRoot string) int {
	dbPath, err := ResolveDBPath("", repoRoot)
	if err != nil {
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return 1
	}
	defer db.Close()

	settings, err := store.GetAISettings(context.Background(), db.DB)
	if err != nil {
		return 1
	}

	cfg := settings.ActiveConfig()
	model := NewAITSModel(
		string(settings.ActiveProvider),
		cfg.Model,
		cfg.BaseURL,
		cfg.APIKey,
		cfg.TimeoutSec,
	)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return 1
	}
	return 0
}
