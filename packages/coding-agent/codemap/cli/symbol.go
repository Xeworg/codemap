package cli

import (
	"context"
	"flag"
	"io"

	"codrut/packages/coding-agent/codemap/store"
)

// RunSymbol runs the "symbol" command and returns an exit code.
func RunSymbol(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	fs := flag.NewFlagSet("symbol", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	dbPathFlag := fs.String("db", "", "Path to SQLite database (optional)")
	_ = fs.Bool("json", false, "Output JSON envelope (default)")
	if err := fs.Parse(args); err != nil {
		WriteErrorEnvelope(w, "symbol", err.Error(), EmptyMeta())
		return 2
	}

	symbolArg := ""
	if fs.NArg() > 0 {
		symbolArg = fs.Arg(0)
	}

	// Validation.
	if symbolArg == "" {
		WriteErrorEnvelope(w, "symbol", "symbol name or ID required", EmptyMeta())
		return 2
	}

	dbPath, err := ResolveDBPath(*dbPathFlag, repoRoot)
	if err != nil {
		WriteErrorEnvelope(w, "symbol", "resolve db path: "+err.Error(), EmptyMeta())
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		WriteErrorEnvelope(w, "symbol", "open db: "+err.Error(), EmptyMeta())
		return 1
	}
	defer db.Close()

	// Load meta (for stale detection).
	metaOut, err := store.GetLatestSnapshotMeta(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "symbol", "read meta: "+err.Error(), EmptyMeta())
		return 1
	}
	if metaOut.SnapshotID == 0 {
		WriteErrorEnvelope(w, "symbol", "no index found (run 'codemap index' first)", EmptyMeta())
		return 3
	}

	stale := StaleNow(metaOut.IndexedAt)

	// Lookup symbol.
	sym, err := store.GetSymbolByName(ctx, db.DB, symbolArg)
	if err != nil {
		WriteErrorEnvelope(w, "symbol", "query: "+err.Error(), EmptyMeta())
		return 1
	}
	if sym == nil {
		// Not found: derive cause and return structured explain_not_found.
		cause, actions := DeriveSymbolNotFoundCause(ctx, db.DB, symbolArg, false)
		enf := ExplainNotFound{
			Cause:              cause,
			RecommendedActions: actions,
		}
		envelope := NewEnvelope("symbol", false, map[string]interface{}{
			"explain_not_found": enf,
		}, []string{"symbol \"" + symbolArg + "\" not found"}, Meta{
			SnapshotID: metaOut.SnapshotID,
			HeadRef:    metaOut.HeadRef,
			IndexedAt:  metaOut.IndexedAt,
			IsStale:    stale,
		})
		out, _ := envelope.Encode()
		_, _ = w.Write(out)
		return 3
	}

	evidence := DefaultEvidence()
	evidence = append(evidence, EvidenceEntry{
		Type:        "file_location",
		Description: "found in " + sym.File,
	})

	data := SymbolData{
		Name:       sym.Name,
		Kind:       sym.Kind,
		Signature:  sym.Signature,
		StartLine:  sym.StartLine,
		EndLine:    sym.EndLine,
		File:       sym.File,
		Confidence: ConfidenceForSymbol(sym.Kind),
		Evidence:   evidence,
	}

	envelope := NewEnvelope("symbol", true, data, nil, Meta{
		SnapshotID: metaOut.SnapshotID,
		HeadRef:    metaOut.HeadRef,
		IndexedAt:  metaOut.IndexedAt,
		IsStale:    stale,
	})

	out, _ := envelope.Encode()
	_, _ = w.Write(out)
	return 0
}
