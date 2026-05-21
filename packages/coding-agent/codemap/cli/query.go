package cli

import (
	"context"
	"flag"
	"io"
	"sort"
	"strings"

	"codrut/packages/coding-agent/codemap/store"
)

// RunQuery runs the "query" command and returns an exit code.
func RunQuery(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	dbPathFlag := fs.String("db", "", "Path to SQLite database (optional)")
	_ = fs.Bool("json", false, "Output JSON envelope (default)")
	if err := fs.Parse(args); err != nil {
		WriteErrorEnvelope(w, "query", err.Error(), EmptyMeta())
		return 2
	}

	queryTerm := ""
	if fs.NArg() > 0 {
		queryTerm = fs.Arg(0)
	}

	// Validation.
	if queryTerm == "" {
		WriteErrorEnvelope(w, "query", "query term required", EmptyMeta())
		return 2
	}

	dbPath, err := ResolveDBPath(*dbPathFlag, repoRoot)
	if err != nil {
		WriteErrorEnvelope(w, "query", "resolve db path: "+err.Error(), EmptyMeta())
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		WriteErrorEnvelope(w, "query", "open db: "+err.Error(), EmptyMeta())
		return 1
	}
	defer db.Close()

	// Require indexed state.
	meta, err := store.GetLatestSnapshotMeta(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "query", "read meta: "+err.Error(), EmptyMeta())
		return 1
	}
	if meta.SnapshotID == 0 {
		WriteErrorEnvelope(w, "query", "no index found (run 'codemap index' first)", EmptyMeta())
		return 3
	}

	var matches []QueryMatch

	// Exact match first.
	exact, err := store.GetSymbolByName(ctx, db.DB, queryTerm)
	if err != nil {
		WriteErrorEnvelope(w, "query", "query: "+err.Error(), EmptyMeta())
		return 1
	}
	if exact != nil {
		matches = append(matches, QueryMatch{
			Name:      exact.Name,
			Kind:      exact.Kind,
			File:      exact.File,
			Signature: exact.Signature,
		})
	}

	// Prefix fallback: all symbols whose name starts with the query term.
	allSymbols, err := store.GetAllSymbols(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "query", "symbols query: "+err.Error(), EmptyMeta())
		return 1
	}
	for _, sym := range allSymbols {
		if sym.Name == queryTerm {
			continue // already added as exact match
		}
		if strings.HasPrefix(sym.Name, queryTerm) {
			matches = append(matches, QueryMatch{
				Name:      sym.Name,
				Kind:      sym.Kind,
				File:      sym.File,
				Signature: sym.Signature,
			})
		}
	}

	// Deterministic sort: by name, then file.
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Name != matches[j].Name {
			return matches[i].Name < matches[j].Name
		}
		return matches[i].File < matches[j].File
	})

	// Ensure non-nil slices for JSON serialization.
	if matches == nil {
		matches = []QueryMatch{}
	}

	stale := StaleNow(meta.IndexedAt)
	data := QueryData{
		Query:   queryTerm,
		Matches: matches,
		Count:   len(matches),
	}

	envelope := NewEnvelope("query", true, data, nil, Meta{
		SnapshotID: meta.SnapshotID,
		HeadRef:    meta.HeadRef,
		IndexedAt:  meta.IndexedAt,
		IsStale:    stale,
	})
	out, _ := envelope.Encode()
	_, _ = w.Write(out)
	return 0
}
