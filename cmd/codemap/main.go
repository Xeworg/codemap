// cmd/codemap/main.go — Binary entry point for codemap.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"codrut/packages/coding-agent/codemap/cli"
)

var repoFlag string

func main() {
	// Root-level flags shared by all commands.
	flag.StringVar(&repoFlag, "repo", ".", "Path to the repository to index/query")
	flag.Usage = usage
	flag.Parse()

	ctx := context.Background()
	stdout := os.Stdout

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	cmd := args[0]
	subargs := args[1:]
	repoRoot := repoFlag

	var exitCode int
	switch cmd {
	case "index":
		exitCode = cli.RunIndex(ctx, stdout, subargs, repoRoot)
	case "symbol":
		exitCode = cli.RunSymbol(ctx, stdout, subargs, repoRoot)
	case "history":
		exitCode = cli.RunHistory(ctx, stdout, subargs, repoRoot)
	case "help", "--help", "-h":
		usage()
		os.Exit(0)
	default:
		writeRootError(stdout, cmd)
		os.Exit(2)
	}
	os.Exit(exitCode)
}

func usage() {
	w := os.Stderr
	fmt.Fprintf(w, "Usage: codemap [flags] <command> [command flags] [args]\n\n")
	fmt.Fprintf(w, "Flags:\n")
	fmt.Fprintf(w, "  -repo path   Repository root (default: current directory)\n")
	fmt.Fprintf(w, "\nCommands:\n")
	fmt.Fprintf(w, "  codemap index              Scan and index a Go repository\n")
	fmt.Fprintf(w, "  codemap symbol --db DB <name>  Query a symbol by name or ID\n")
	fmt.Fprintf(w, "  codemap history --db DB <name> Query commit history for a symbol\n\n")
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  codemap index --db myrepo.db\n")
	fmt.Fprintf(w, "  codemap symbol --db myrepo.db MyFunction\n")
	fmt.Fprintf(w, "  codemap history --db myrepo.db MyFunction\n")
}

func writeRootError(w io.Writer, cmd string) {
	env := cli.NewEnvelope(cmd, false, struct{}{}, []string{"unknown command: " + cmd}, cli.EmptyMeta())
	b, _ := json.Marshal(env)
	fmt.Fprintln(w, string(b))
}
