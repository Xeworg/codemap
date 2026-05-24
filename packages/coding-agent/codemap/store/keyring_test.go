package store

import (
	"context"
	"os"
	"testing"
)

func TestStoreMinimaxKey_SetAndGet(t *testing.T) {
	ctx := context.Background()
	key := "test-minimax-key-12345"

	// Store.
	if err := StoreMinimaxKey(ctx, key); err != nil {
		t.Skipf("keyring unavailable on this system: %v", err)
	}

	// Retrieve.
	got, err := GetMinimaxKey(ctx)
	if err != nil {
		t.Fatalf("GetMinimaxKey: %v", err)
	}
	if got != key {
		t.Errorf("got %q, want %q", got, key)
	}
}

func TestStoreMinimaxKey_Delete(t *testing.T) {
	ctx := context.Background()
	key := "delete-test-key"

	if err := StoreMinimaxKey(ctx, key); err != nil {
		t.Skipf("keyring unavailable: %v", err)
	}
	if err := DeleteMinimaxKey(ctx); err != nil {
		t.Fatalf("DeleteMinimaxKey: %v", err)
	}
	// Ensure env var is not set so we get ErrAPINotConfigured.
	os.Unsetenv("MINIMAX_API_KEY")
	_, err := GetMinimaxKey(ctx)
	if err != ErrAPINotConfigured {
		t.Errorf("expected ErrAPINotConfigured after delete, got: %v", err)
	}
}

func TestGetMinimaxKey_EnvFallback(t *testing.T) {
	ctx := context.Background()

	// Set env var.
	if err := os.Setenv("MINIMAX_API_KEY", "env-key-value"); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("MINIMAX_API_KEY")

	// When keyring is empty, falls back to env.
	got, err := GetMinimaxKey(ctx)
	if err != nil {
		t.Fatalf("GetMinimaxKey (env fallback): %v", err)
	}
	if got != "env-key-value" {
		t.Errorf("got %q, want env-key-value", got)
	}
}

func TestGetMinimaxKey_KeyringTakesPrecedence(t *testing.T) {
	ctx := context.Background()
	keyringKey := "keyring-wins-key"
	envKey := "env-fallback-key"

	// Pre-store in keyring.
	if err := StoreMinimaxKey(ctx, keyringKey); err != nil {
		t.Skipf("keyring unavailable: %v", err)
	}
	defer DeleteMinimaxKey(ctx)

	// Set env var (should be ignored).
	if err := os.Setenv("MINIMAX_API_KEY", envKey); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("MINIMAX_API_KEY")

	got, err := GetMinimaxKey(ctx)
	if err != nil {
		t.Fatalf("GetMinimaxKey: %v", err)
	}
	if got != keyringKey {
		t.Errorf("keyring should take precedence; got %q, want %q", got, keyringKey)
	}
}

func TestGetMinimaxKey_NotConfigured(t *testing.T) {
	ctx := context.Background()

	// Ensure neither keyring nor env has a key.
	_ = DeleteMinimaxKey(ctx)
	os.Unsetenv("MINIMAX_API_KEY")

	_, err := GetMinimaxKey(ctx)
	if err != ErrAPINotConfigured {
		t.Errorf("expected ErrAPINotConfigured, got: %v", err)
	}
}

func TestHasMinimaxKeyInKeyring(t *testing.T) {
	ctx := context.Background()
	key := "has-key-test"

	// Should start empty.
	has, err := HasMinimaxKeyInKeyring(ctx)
	if err != nil {
		t.Skipf("keyring unavailable: %v", err)
	}
	if has {
		t.Error("expected no key in keyring initially")
	}

	// Store and check again.
	if err := StoreMinimaxKey(ctx, key); err != nil {
		t.Skipf("keyring unavailable: %v", err)
	}
	defer DeleteMinimaxKey(ctx)

	has, err = HasMinimaxKeyInKeyring(ctx)
	if err != nil {
		t.Fatalf("HasMinimaxKeyInKeyring: %v", err)
	}
	if !has {
		t.Error("expected key to be present after store")
	}
}

func TestProviderConfig_APIKeyNeverSerialized(t *testing.T) {
	// Verify the json:"-" tag excludes APIKey from serialization.
	// Use a placeholder so the secret literal never appears in this test file.
	placeholder := "PLACEHOLDER_APIKEY_VALUE"
	cfg := ProviderConfig{
		Provider:   ProviderMinimax,
		Model:      "abab6.5s-chat",
		BaseURL:    "https://api.minimax.chat",
		APIKey:     placeholder,
		TimeoutSec: 60,
	}

	data, err := cfg.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	json := string(data)
	if contains(json, placeholder) {
		t.Errorf("APIKey placeholder should not appear in JSON serialization")
	}
}
