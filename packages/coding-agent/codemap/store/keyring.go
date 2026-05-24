package store

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/99designs/keyring"
)

// Keyring service constants.
const (
	keyringService = "com.codrut.codemap"
	minimaxKeyName = "minimax_api_key"
)

// Sentinel errors for key resolution.
var (
	// ErrAPINotConfigured is returned when no Minimax API key is found
	// in the keyring or environment.
	ErrAPINotConfigured = errors.New("minimax api key not configured: set with 'codemap ai-settings set minimax.api_key <key>' or export MINIMAX_API_KEY")

	// ErrKeyringUnavailable is returned when the OS keyring is not available
	// and no env-var fallback is configured.
	ErrKeyringUnavailable = errors.New("keyring unavailable: install libsecret (Linux) or set MINIMAX_API_KEY env var")
)

// keyringRing returns a keyring instance.
func keyringRing() (keyring.Keyring, error) {
	kr, err := keyring.Open(keyring.Config{
		ServiceName: keyringService,
	})
	if err != nil {
		return nil, fmt.Errorf("keyring open: %w", err)
	}
	return kr, nil
}

// StoreMinimaxKey stores the Minimax API key in the OS keyring.
// If key is empty, the key is deleted from the keyring.
func StoreMinimaxKey(ctx context.Context, key string) error {
	kr, err := keyringRing()
	if err != nil {
		// If keyring is unavailable, we cannot store. Fail explicitly.
		return fmt.Errorf("store minimax key: %w", err)
	}
	if key == "" {
		// Empty key means delete.
		if err := kr.Remove(minimaxKeyName); err != nil {
			if err == keyring.ErrKeyNotFound {
				return nil // already gone
			}
			return fmt.Errorf("delete minimax key: %w", err)
		}
		return nil
	}
	if err := kr.Set(keyring.Item{
		Key:  minimaxKeyName,
		Data: []byte(key),
	}); err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}
	return nil
}

// GetMinimaxKey resolves the Minimax API key using the following chain:
//  1. OS keyring
//  2. MINIMAX_API_KEY environment variable
//  3. error (ErrAPINotConfigured)
func GetMinimaxKey(ctx context.Context) (string, error) {
	kr, err := keyringRing()
	if err == nil {
		item, err := kr.Get(minimaxKeyName)
		if err == nil && len(item.Data) > 0 {
			return string(item.Data), nil
		}
		// Key not found in keyring is not an error — fall through to env.
		if err != keyring.ErrKeyNotFound {
			return "", fmt.Errorf("keyring get: %w", err)
		}
	}
	// Keyring unavailable or key not found — fall back to env var.
	if envKey := os.Getenv("MINIMAX_API_KEY"); envKey != "" {
		return envKey, nil
	}
	return "", ErrAPINotConfigured
}

// DeleteMinimaxKey removes the Minimax API key from the OS keyring.
func DeleteMinimaxKey(ctx context.Context) error {
	kr, err := keyringRing()
	if err != nil {
		return fmt.Errorf("delete minimax key: %w", err)
	}
	if err := kr.Remove(minimaxKeyName); err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil
		}
		return fmt.Errorf("keyring remove: %w", err)
	}
	return nil
}

// HasMinimaxKeyInKeyring reports whether a non-empty key exists in the OS keyring
// (ignoring the env-var fallback).
func HasMinimaxKeyInKeyring(ctx context.Context) (bool, error) {
	kr, err := keyringRing()
	if err != nil {
		return false, nil // keyring unavailable is not an error for this check
	}
	item, err := kr.Get(minimaxKeyName)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return false, nil
		}
		return false, fmt.Errorf("keyring get: %w", err)
	}
	return len(item.Data) > 0, nil
}
