package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// Provider is the set of allowed AI providers.
type Provider string

const (
	ProviderOllama  Provider = "ollama"
	ProviderMinimax Provider = "minimax"
)

// IsValid reports whether p is a known provider.
func (p Provider) IsValid() bool {
	switch p {
	case ProviderOllama, ProviderMinimax:
		return true
	}
	return false
}

// ProviderConfig holds connection parameters for a single provider.
// APIKey is intentionally excluded from JSON serialization (json:"-").
// Always resolve it via GetMinimaxKey at call time.
type ProviderConfig struct {
	Provider   Provider `json:"provider"`
	Model      string   `json:"model"`
	BaseURL    string   `json:"base_url"`
	APIKey     string   `json:"-"` // never serialized; use GetMinimaxKey instead
	TimeoutSec int      `json:"timeout_sec"`
	Extra      *string  `json:"extra,omitempty"` // optional JSON blob
}

// MarshalJSON implements json.Marshaler for ProviderConfig,
// intentionally excluding APIKey from serialization for security.
func (c ProviderConfig) MarshalJSON() ([]byte, error) {
	withoutKey := struct {
		Provider   Provider `json:"provider"`
		Model      string   `json:"model"`
		BaseURL    string   `json:"base_url"`
		APIKey     string   `json:"-"` // excluded
		TimeoutSec int      `json:"timeout_sec"`
		Extra      *string  `json:"extra,omitempty"`
	}{
		Provider:   c.Provider,
		Model:      c.Model,
		BaseURL:    c.BaseURL,
		APIKey:     "",
		TimeoutSec: c.TimeoutSec,
		Extra:      c.Extra,
	}
	return json.Marshal(withoutKey)
}

// ExtraSettings holds additional per-provider tuning options.
type ExtraSettings struct {
	Temperature float64  `json:"temperature,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	StopWords   []string `json:"stop,omitempty"`
}

// DefaultExtraSettings returns safe defaults for new configs.
func DefaultExtraSettings() ExtraSettings {
	return ExtraSettings{
		Temperature: 0.7,
		MaxTokens:   2048,
	}
}

// AISettings is the top-level AI configuration stored in the DB.
type AISettings struct {
	ActiveProvider Provider       `json:"active_provider"`
	Ollama         ProviderConfig `json:"ollama"`
	Minimax        ProviderConfig `json:"minimax"`
	Version        int            `json:"version"` // schema version for migrations
}

const aiSettingsKey = "ai_settings"

var defaultAISettings = AISettings{
	Version:        1,
	ActiveProvider: ProviderOllama,
	Ollama: ProviderConfig{
		Provider:   ProviderOllama,
		Model:      "llama3",
		BaseURL:    "http://localhost:11434",
		TimeoutSec: 30,
	},
	Minimax: ProviderConfig{
		Provider:   ProviderMinimax,
		Model:      "abab6.5s-chat",
		TimeoutSec: 60,
	},
}

// GetAISettings returns the persisted AI settings, or defaults if none saved.
// It is safe to call on a fresh/unmigrated DB — returns default settings.
func GetAISettings(ctx context.Context, db *sql.DB) (AISettings, error) {
	var raw string
	query := `SELECT value FROM settings WHERE key = ?`
	err := db.QueryRowContext(ctx, query, aiSettingsKey).Scan(&raw)
	if err == sql.ErrNoRows {
		return defaultAISettings, nil
	}
	if err != nil {
		// Table may not exist yet on a fresh DB.
		if containsNoSuchTable(err) {
			return defaultAISettings, nil
		}
		return AISettings{}, fmt.Errorf("GetAISettings: %w", err)
	}
	var settings AISettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return AISettings{}, fmt.Errorf("GetAISettings unmarshal: %w", err)
	}
	return settings, nil
}

// SaveAISettings persists the given AI settings, overwriting any previous value.
func SaveAISettings(ctx context.Context, db *sql.DB, s AISettings) error {
	raw, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("SaveAISettings marshal: %w", err)
	}
	query := `INSERT OR REPLACE INTO settings(key, value) VALUES (?, ?)`
	_, err = db.ExecContext(ctx, query, aiSettingsKey, string(raw))
	if err != nil {
		// Table may not exist yet — try to create it and retry.
		if containsNoSuchTable(err) {
			if err := ensureSettingsTable(ctx, db); err != nil {
				return err
			}
			_, err = db.ExecContext(ctx, query, aiSettingsKey, string(raw))
		}
		if err != nil {
			return fmt.Errorf("SaveAISettings exec: %w", err)
		}
	}
	return nil
}

// ActiveConfig returns the ProviderConfig for the currently active provider.
func (s AISettings) ActiveConfig() *ProviderConfig {
	switch s.ActiveProvider {
	case ProviderOllama:
		return &s.Ollama
	case ProviderMinimax:
		return &s.Minimax
	}
	// Fallback to Ollama as the default active provider.
	dp := defaultAISettings
	return &dp.Ollama
}

// GetActiveProviderConfig returns the active provider config from the DB,
// or a default if no settings are persisted.
// For Minimax provider, the API key is resolved at call time via keyring/env-var
// and is never persisted to SQLite.
func GetActiveProviderConfig(ctx context.Context, db *sql.DB) (ProviderConfig, error) {
	settings, err := GetAISettings(ctx, db)
	if err != nil {
		return ProviderConfig{}, err
	}
	cfg := settings.ActiveConfig()
	if cfg == nil {
		return ProviderConfig{}, fmt.Errorf("no active provider configured")
	}
	result := *cfg
	// Resolve the API key at call time rather than storing it in the config.
	// This ensures APIKey is never persisted to SQLite.
	if result.Provider == ProviderMinimax {
		key, err := GetMinimaxKey(ctx)
		if err != nil {
			return result, err // includes ErrAPINotConfigured
		}
		result.APIKey = key
	}
	return result, nil
}

// containsNoSuchTable reports whether err indicates a missing table.
func containsNoSuchTable(err error) bool {
	return err != nil && (contains(err.Error(), "no such table") ||
		contains(err.Error(), "SQL logic error"))
}

func contains(s, substr string) bool {
	return len(substr) <= len(s) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}

// ensureSettingsTable creates the settings table if it doesn't exist.
func ensureSettingsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY, value TEXT)`)
	return err
}
