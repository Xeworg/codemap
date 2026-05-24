package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"codrut/packages/coding-agent/codemap/store"
)

// RunAITest runs the "ai-test" command and returns an exit code.
func RunAITest(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	fs := flag.NewFlagSet("ai-test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	dbPathFlag := fs.String("db", "", "Path to SQLite database (optional)")
	if err := fs.Parse(args); err != nil {
		WriteErrorEnvelope(w, "ai-test", err.Error(), EmptyMeta())
		return 2
	}

	dbPath, err := ResolveDBPath(*dbPathFlag, repoRoot)
	if err != nil {
		WriteErrorEnvelope(w, "ai-test", "resolve db path: "+err.Error(), EmptyMeta())
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		WriteErrorEnvelope(w, "ai-test", "open db: "+err.Error(), EmptyMeta())
		return 1
	}
	defer db.Close()

	cfg, err := store.GetActiveProviderConfig(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "ai-test", "read ai config: "+err.Error(), EmptyMeta())
		return 1
	}

	result := TestConnectivityForProvider(ctx, w, AIProvider(cfg.Provider), cfg.BaseURL, cfg.Model, cfg.APIKey, cfg.TimeoutSec)

	stale := false
	var meta Meta
	snapMeta, _ := store.GetLatestSnapshotMeta(ctx, db.DB)
	if snapMeta.SnapshotID > 0 {
		meta = Meta{
			SnapshotID: snapMeta.SnapshotID,
			HeadRef:    snapMeta.HeadRef,
			IndexedAt:  snapMeta.IndexedAt,
			IsStale:    stale,
		}
	}

	var ok bool
	if result.Error == "" && result.Reachable {
		ok = true
	}
	var errs []string
	if !ok {
		errs = []string{result.Error}
	}

	envelope := NewEnvelope("ai-test", ok, result, errs, meta)
	out, _ := envelope.Encode()
	_, _ = w.Write(out)

	if !result.Reachable {
		return 1
	}
	return 0
}

// RunAISettings runs the "ai-settings" command and returns an exit code.
// When no subcommand is given, it shows current settings.
// Subcommands: "set", "get", "list-providers".
func RunAISettings(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	if len(args) == 0 {
		return showAISettings(ctx, w, repoRoot)
	}
	switch args[0] {
	case "set":
		return setAISetting(ctx, w, args[1:], repoRoot)
	case "get":
		return getAISetting(ctx, w, args[1:], repoRoot)
	case "list-providers":
		return listProviders(ctx, w)
	default:
		WriteErrorEnvelope(w, "ai-settings", "unknown subcommand: "+args[0], EmptyMeta())
		return 2
	}
}

func showAISettings(ctx context.Context, w io.Writer, repoRoot string) int {
	dbPath, err := ResolveDBPath("", repoRoot)
	if err != nil {
		WriteErrorEnvelope(w, "ai-settings", "resolve db path: "+err.Error(), EmptyMeta())
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		WriteErrorEnvelope(w, "ai-settings", "open db: "+err.Error(), EmptyMeta())
		return 1
	}
	defer db.Close()

	settings, err := store.GetAISettings(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "ai-settings", "read settings: "+err.Error(), EmptyMeta())
		return 1
	}

	envelope := NewEnvelope("ai-settings", true, settings, nil, EmptyMeta())
	out, _ := envelope.Encode()
	_, _ = w.Write(out)
	return 0
}

func getAISetting(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	if len(args) < 1 {
		WriteErrorEnvelope(w, "ai-settings get", "usage: ai-settings get <key>", EmptyMeta())
		return 2
	}
	key := args[0]

	dbPath, err := ResolveDBPath("", repoRoot)
	if err != nil {
		WriteErrorEnvelope(w, "ai-settings get", "resolve db path: "+err.Error(), EmptyMeta())
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		WriteErrorEnvelope(w, "ai-settings get", "open db: "+err.Error(), EmptyMeta())
		return 1
	}
	defer db.Close()

	settings, err := store.GetAISettings(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "ai-settings get", "read settings: "+err.Error(), EmptyMeta())
		return 1
	}

	var val string
	switch key {
	case "active_provider":
		val = string(settings.ActiveProvider)
	case "ollama.model":
		val = settings.Ollama.Model
	case "ollama.base_url":
		val = settings.Ollama.BaseURL
	case "ollama.timeout_sec":
		val = fmt.Sprintf("%d", settings.Ollama.TimeoutSec)
	case "minimax.model":
		val = settings.Minimax.Model
	case "minimax.base_url":
		val = settings.Minimax.BaseURL
	case "minimax.timeout_sec":
		val = fmt.Sprintf("%d", settings.Minimax.TimeoutSec)
	case "minimax.api_key":
		// Never expose the key value. Report only presence/absence.
		hasKey, err := store.HasMinimaxKeyInKeyring(ctx)
		if err != nil {
			val = "error checking keyring"
		} else if hasKey {
			val = "[configured in keyring]"
		} else if os.Getenv("MINIMAX_API_KEY") != "" {
			val = "[configured via MINIMAX_API_KEY env var]"
		} else {
			val = "[not configured]"
		}
	default:
		WriteErrorEnvelope(w, "ai-settings get", "unknown key: "+key, EmptyMeta())
		return 2
	}

	envelope := NewEnvelope("ai-settings get", true, map[string]string{"key": key, "value": val}, nil, EmptyMeta())
	out, _ := envelope.Encode()
	_, _ = w.Write(out)
	return 0
}

func setAISetting(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	if len(args) < 2 {
		WriteErrorEnvelope(w, "ai-settings set", "usage: ai-settings set <key> <value>", EmptyMeta())
		return 2
	}
	key := args[0]
	value := args[1]

	dbPath, err := ResolveDBPath("", repoRoot)
	if err != nil {
		WriteErrorEnvelope(w, "ai-settings set", "resolve db path: "+err.Error(), EmptyMeta())
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		WriteErrorEnvelope(w, "ai-settings set", "open db: "+err.Error(), EmptyMeta())
		return 1
	}
	defer db.Close()

	settings, err := store.GetAISettings(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "ai-settings set", "read settings: "+err.Error(), EmptyMeta())
		return 1
	}

	switch key {
	case "active_provider":
		provider := store.Provider(value)
		if !provider.IsValid() {
			WriteErrorEnvelope(w, "ai-settings set", "invalid provider (use: ollama, minimax)", EmptyMeta())
			return 2
		}
		settings.ActiveProvider = provider
	case "ollama.model":
		settings.Ollama.Model = value
	case "ollama.base_url":
		settings.Ollama.BaseURL = value
	case "ollama.timeout_sec":
		fmt.Sscanf(value, "%d", &settings.Ollama.TimeoutSec)
	case "minimax.model":
		settings.Minimax.Model = value
	case "minimax.base_url":
		settings.Minimax.BaseURL = value
	case "minimax.timeout_sec":
		fmt.Sscanf(value, "%d", &settings.Minimax.TimeoutSec)
	case "minimax.api_key":
		if err := store.StoreMinimaxKey(ctx, value); err != nil {
			WriteErrorEnvelope(w, "ai-settings set", "failed to store api key: "+err.Error(), EmptyMeta())
			return 1
		}
		// Confirm secure storage, never echo the value.
		envelope := NewEnvelope("ai-settings set", true, map[string]string{
			"key": key, "status": "stored in OS keyring", "provider": "minimax"}, nil, EmptyMeta())
		out, _ := envelope.Encode()
		_, _ = w.Write(out)
		return 0
	default:
		WriteErrorEnvelope(w, "ai-settings set", "unknown key: "+key, EmptyMeta())
		return 2
	}

	if err := store.SaveAISettings(ctx, db.DB, settings); err != nil {
		WriteErrorEnvelope(w, "ai-settings set", "save settings: "+err.Error(), EmptyMeta())
		return 1
	}

	envelope := NewEnvelope("ai-settings set", true, map[string]string{"key": key, "value": value}, nil, EmptyMeta())
	out, _ := envelope.Encode()
	_, _ = w.Write(out)
	return 0
}

func listProviders(ctx context.Context, w io.Writer) int {
	providers := []struct {
		Name  string `json:"name"`
		Valid bool   `json:"is_valid"`
	}{
		{"ollama", true},
		{"minimax", true},
	}
	envelope := NewEnvelope("ai-settings list-providers", true, providers, nil, EmptyMeta())
	out, _ := envelope.Encode()
	_, _ = w.Write(out)
	return 0
}

// TUILaunch runs the AI settings Bubble Tea TUI.
func TUILaunch(repoRoot string) int {
	return RunAITUI(repoRoot)
}
