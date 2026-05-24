package store

import (
	"context"
	"database/sql"
	"testing"
)

func TestGetAISettingsDefaults(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file::memory:?mode=memory")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Fresh DB with no settings table → returns defaults.
	settings, err := GetAISettings(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if settings.ActiveProvider != ProviderOllama {
		t.Errorf("default active provider: want ollama, got %s", settings.ActiveProvider)
	}
	if settings.Ollama.Model != "llama3" {
		t.Errorf("default ollama model: want llama3, got %s", settings.Ollama.Model)
	}
	if settings.Minimax.Model != "abab6.5s-chat" {
		t.Errorf("default minimax model: want abab6.5s-chat, got %s", settings.Minimax.Model)
	}
}

func TestSaveAndGetAISettings(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file::memory:?mode=memory")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	settings := AISettings{
		ActiveProvider: ProviderMinimax,
		Ollama: ProviderConfig{
			Provider:   ProviderOllama,
			Model:      "mistral",
			BaseURL:    "http://localhost:11434",
			TimeoutSec: 60,
		},
		Minimax: ProviderConfig{
			Provider:   ProviderMinimax,
			Model:      "abab6.5s-chat",
			BaseURL:    "https://api.minimax.chat",
			APIKey:     "TESTKEY",
			TimeoutSec: 120,
		},
	}

	if err := SaveAISettings(ctx, db, settings); err != nil {
		t.Fatal(err)
	}

	loaded, err := GetAISettings(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ActiveProvider != ProviderMinimax {
		t.Errorf("active provider: want minimax, got %s", loaded.ActiveProvider)
	}
	if loaded.Ollama.Model != "mistral" {
		t.Errorf("ollama model: want mistral, got %s", loaded.Ollama.Model)
	}
	// APIKey must never be serialized to DB — it is excluded via json:"-"
	// and resolved at call time via GetMinimaxKey (keyring/env-var).
	if loaded.Minimax.APIKey != "" {
		t.Errorf("minimax api key: should not be persisted in DB, got %q", loaded.Minimax.APIKey)
	}
}

func TestProviderIsValid(t *testing.T) {
	tests := []struct {
		provider Provider
		want     bool
	}{
		{"ollama", true},
		{"minimax", true},
		{"openai", false},
		{"anthropic", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.provider.IsValid(); got != tt.want {
			t.Errorf("IsValid(%q): want %v, got %v", tt.provider, tt.want, got)
		}
	}
}

func TestActiveConfig(t *testing.T) {
	settings := AISettings{
		ActiveProvider: ProviderMinimax,
		Ollama:         ProviderConfig{Model: "llama3"},
		Minimax:        ProviderConfig{Model: "abab6.5s-chat"},
	}
	cfg := settings.ActiveConfig()
	if cfg == nil {
		t.Fatal("ActiveConfig returned nil")
	}
	if cfg.Model != "abab6.5s-chat" {
		t.Errorf("ActiveConfig minimax model: want abab6.5s-chat, got %s", cfg.Model)
	}

	settings.ActiveProvider = ProviderOllama
	cfg = settings.ActiveConfig()
	if cfg.Model != "llama3" {
		t.Errorf("ActiveConfig ollama model: want llama3, got %s", cfg.Model)
	}
}

func TestGetActiveProviderConfig(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file::memory:?mode=memory")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	settings := AISettings{
		ActiveProvider: ProviderOllama,
		Ollama: ProviderConfig{
			Provider:   ProviderOllama,
			Model:      "llama3",
			BaseURL:    "http://localhost:11434",
			TimeoutSec: 30,
		},
	}
	if err := SaveAISettings(ctx, db, settings); err != nil {
		t.Fatal(err)
	}

	cfg, err := GetActiveProviderConfig(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "llama3" {
		t.Errorf("model: want llama3, got %s", cfg.Model)
	}
	if cfg.Provider != ProviderOllama {
		t.Errorf("provider: want ollama, got %s", cfg.Provider)
	}
}
