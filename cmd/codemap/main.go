// cmd/codemap/main.go — Binary entry point for codemap.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"codrut/packages/coding-agent/codemap/cli"
	"codrut/packages/coding-agent/codemap/cli/installer"
)

var repoFlag string

func main() {
	ctx := context.Background()
	stdout := os.Stdout
	stderr := os.Stderr

	// Root-level flags shared by all commands.
	flag.StringVar(&repoFlag, "repo", ".", "Path to the repository to index/query")
	flag.Usage = func() { helpRoot(stderr) }
	flag.Parse()

	args := flag.Args()

	// Handle no-command or help before any routing.
	if len(args) == 0 {
		helpRoot(stderr)
		os.Exit(2)
	}

	cmd := args[0]
	subargs := args[1:]

	if cmd == "help" || cmd == "--help" || cmd == "-h" {
		if len(subargs) == 0 {
			helpRoot(stderr)
			os.Exit(0)
		}
		// Per-command help.
		helpFor(subargs[0], stderr)
		os.Exit(0)
	}

	repoRoot := repoFlag

	var exitCode int
	switch cmd {
	case "index":
		exitCode = runWithHelp(ctx, stdout, stderr, "index", subargs, repoRoot,
			func() int { return cli.RunIndex(ctx, stdout, subargs, repoRoot) })
	case "symbol":
		exitCode = runWithHelp(ctx, stdout, stderr, "symbol", subargs, repoRoot,
			func() int { return cli.RunSymbol(ctx, stdout, subargs, repoRoot) })
	case "history":
		exitCode = runWithHelp(ctx, stdout, stderr, "history", subargs, repoRoot,
			func() int { return cli.RunHistory(ctx, stdout, subargs, repoRoot) })
	case "install":
		exitCode = runInstall(ctx, stdout, stderr, subargs)
	case "doctor":
		exitCode = runDoctor(ctx, stdout, stderr, subargs)
	default:
		helpRoot(stderr)
		fmt.Fprintf(stderr, "\nError: unknown command %q.\n\n", cmd)
		writeJSONError(stderr, cmd, "unknown command: "+cmd)
		os.Exit(2)
	}
	os.Exit(exitCode)
}

// runInstall handles the install command (non-interactive + TUI modes).
func runInstall(ctx context.Context, stdout io.Writer, stderr io.Writer, subargs []string) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.Usage = func() { helpFor("install", stdout) }
	dryRun := fs.Bool("dry-run", false, "Check and report actions without applying")
	jsonOutput := fs.Bool("json", false, "Output machine-readable JSON")
	tuiMode := fs.Bool("tui", false, "Run interactive TUI installer")
	if err := fs.Parse(subargs); err != nil {
		return 2
	}

	repoRoot := resolveRepoRoot()
	inst := installer.DefaultInstaller(repoRoot)
	inst.DryRun = *dryRun
	inst.JSONOutput = *jsonOutput

	if *tuiMode {
		return installer.RunTUI(inst)
	}

	result := inst.Run()

	if *jsonOutput {
		fmt.Fprintln(stdout, result.JSON())
	} else {
		fmt.Fprintln(stdout, result.Print())
	}

	switch result.Status {
	case "applied", "up-to-date", "dry-run":
		return 0
	case "error":
		return 1
	default:
		return 1
	}
}

// detectRepoRoot finds the nearest git root or uses current directory.
func detectRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cur := cwd
	for {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return cwd, nil
		}
		cur = parent
	}
}

func resolveRepoRoot() string {
	repoRoot := repoFlag
	if repoRoot == "." {
		detected, err := detectRepoRoot()
		if err == nil && detected != "" && detected != "." {
			repoRoot = detected
		}
	}
	return repoRoot
}

func runDoctor(ctx context.Context, stdout io.Writer, stderr io.Writer, subargs []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.Usage = func() { helpFor("doctor", stdout) }
	jsonOutput := fs.Bool("json", false, "Output machine-readable JSON")
	if err := fs.Parse(subargs); err != nil {
		return 2
	}

	doctor := installer.DefaultDoctor()
	repoRoot := resolveRepoRoot()
	doctor.Installer = installer.DefaultInstaller(repoRoot)

	result := doctor.Run()

	if *jsonOutput {
		fmt.Fprintln(stdout, result.JSON())
	} else {
		fmt.Fprintln(stdout, result.Print())
	}

	switch result.Status {
	case "pass":
		return 0
	case "warn":
		return 0
	default:
		return 1
	}
}

func runWithHelp(ctx context.Context, stdout, stderr io.Writer, cmd string, subargs []string, repoRoot string, run func() int) int {
	_ = ctx
	_ = repoRoot
	for _, a := range subargs {
		if a == "-h" || a == "--help" || a == "help" {
			helpFor(cmd, stderr)
			return 0
		}
	}
	return run()
}

// helpRoot prints the top-level usage summary.
func helpRoot(w io.Writer) {
	fmt.Fprintf(w, "CodeMap — Go symbol indexer and history tracker\n\n")
	fmt.Fprintf(w, "Usage:\n  codemap [flags] <command> [command flags]\n\n")
	fmt.Fprintf(w, "Flags:\n")
	fmt.Fprintf(w, "  -repo path   Repository root (default: current directory)\n")
	fmt.Fprintf(w, "\nCommands:\n")
	fmt.Fprintf(w, "  codemap index       Scan and index a Go repository\n")
	fmt.Fprintf(w, "  codemap symbol      Query a symbol by name\n")
	fmt.Fprintf(w, "  codemap history     Query commit history for a symbol\n")
	fmt.Fprintf(w, "  codemap install     Install codemap skill and tool into Pi runtime\n")
	fmt.Fprintf(w, "  codemap doctor      Diagnose codemap environment and integration\n")
	fmt.Fprintf(w, "\nUse 'codemap help <command>' for per-command usage.\n")
	fmt.Fprintf(w, "\nExamples:\n")
	fmt.Fprintf(w, "  codemap index --db myrepo.db\n")
	fmt.Fprintf(w, "  codemap symbol --db myrepo.db MyFunction\n")
	fmt.Fprintf(w, "  codemap history --db myrepo.db MyFunction\n")
}

// helpFor prints per-command help.
func helpFor(cmd string, w io.Writer) {
	switch cmd {
	case "index":
		fmt.Fprintf(w, "Usage: codemap index [flags]\n\n")
		fmt.Fprintf(w, "Scan and index all Go source files in a repository.\n\n")
		fmt.Fprintf(w, "Flags:\n")
		fmt.Fprintf(w, "  -db path    Path to SQLite database (optional; default: ~/.cache/codemap/<hash>.db)\n")
		fmt.Fprintf(w, "\nExample:\n  codemap index -db myrepo.db  # custom path\n  codemap index                # uses default cache path\n")
	case "symbol":
		fmt.Fprintf(w, "Usage: codemap symbol [flags] <name>\n\n")
		fmt.Fprintf(w, "Look up a symbol by name and return its definition.\n\n")
		fmt.Fprintf(w, "Flags:\n")
		fmt.Fprintf(w, "  -db path    Path to SQLite database (optional; default: ~/.cache/codemap/<hash>.db)\n")
		fmt.Fprintf(w, "\nExample:\n  codemap symbol -db myrepo.db MyFunction  # custom path\n  codemap symbol MyFunction                  # uses default cache path\n")
	case "history":
		fmt.Fprintf(w, "Usage: codemap history [flags] <name>\n\n")
		fmt.Fprintf(w, "Return commit history for a symbol, ordered by link strength.\n\n")
		fmt.Fprintf(w, "Flags:\n")
		fmt.Fprintf(w, "  -db path    Path to SQLite database (optional; default: ~/.cache/codemap/<hash>.db)\n")
		fmt.Fprintf(w, "\nExample:\n  codemap history MyFunction\n")
	case "install":
		fmt.Fprintf(w, "Usage: codemap install [flags]\n\n")
		fmt.Fprintf(w, "Install or update the codemap skill and tool into Pi runtime.\n\n")
		fmt.Fprintf(w, "Flags:\n")
		fmt.Fprintf(w, "  --dry-run     Check and report actions without applying\n")
		fmt.Fprintf(w, "  --json        Output machine-readable JSON\n")
		fmt.Fprintf(w, "  --tui         Run interactive TUI installer\n")
		fmt.Fprintf(w, "\nExample:\n  codemap install          # apply\n  codemap install --dry-run # preview\n  codemap install --tui     # TUI installer\n")
	case "doctor":
		fmt.Fprintf(w, "Usage: codemap doctor [flags]\n\n")
		fmt.Fprintf(w, "Diagnose the codemap environment and Pi integration status.\n\n")
		fmt.Fprintf(w, "Flags:\n")
		fmt.Fprintf(w, "  --json        Output machine-readable JSON\n")
		fmt.Fprintf(w, "\nExample:\n")
		fmt.Fprintf(w, "  codemap doctor            # human-readable report\n")
		fmt.Fprintf(w, "  codemap doctor --json     # JSON for automation\n")
	default:
		fmt.Fprintf(w, "No help available for %q.\n", cmd)
	}
}

// writeJSONError writes a deterministic JSON error envelope.
func writeJSONError(w io.Writer, cmd, msg string) {
	env := cli.NewEnvelope(cmd, false, struct{}{}, []string{msg}, cli.EmptyMeta())
	b, _ := json.Marshal(env)
	fmt.Fprintln(w, string(b))
}
