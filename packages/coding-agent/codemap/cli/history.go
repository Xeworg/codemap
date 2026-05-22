package cli

import (
	"context"
	"flag"
	"io"

	"codrut/packages/coding-agent/codemap/store"
)

// RunHistory runs the "history" command and returns an exit code.
func RunHistory(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	dbPathFlag := fs.String("db", "", "Path to SQLite database (optional)")
	_ = fs.Bool("json", false, "Output JSON envelope (default)")
	if err := fs.Parse(args); err != nil {
		WriteErrorEnvelope(w, "history", err.Error(), EmptyMeta())
		return 2
	}

	symbolArg := ""
	if fs.NArg() > 0 {
		symbolArg = fs.Arg(0)
	}

	// Validation.
	if symbolArg == "" {
		WriteErrorEnvelope(w, "history", "symbol name required", EmptyMeta())
		return 2
	}

	dbPath, err := ResolveDBPath(*dbPathFlag, repoRoot)
	if err != nil {
		WriteErrorEnvelope(w, "history", "resolve db path: "+err.Error(), EmptyMeta())
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		WriteErrorEnvelope(w, "history", "open db: "+err.Error(), EmptyMeta())
		return 1
	}
	defer db.Close()

	// Load meta.
	meta, err := store.GetLatestSnapshotMeta(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "history", "read meta: "+err.Error(), EmptyMeta())
		return 1
	}
	if meta.SnapshotID == 0 {
		WriteErrorEnvelope(w, "history", "no index found (run 'codemap index' first)", EmptyMeta())
		return 3
	}

	stale := StaleNow(meta.IndexedAt)

	// Lookup symbol ID.
	sym, err := store.GetSymbolByName(ctx, db.DB, symbolArg)
	if err != nil {
		WriteErrorEnvelope(w, "history", "query: "+err.Error(), EmptyMeta())
		return 1
	}
	if sym == nil {
		// Symbol not found: derive cause and return structured explain_not_found.
		cause, actions := DeriveHistoryNotFoundCause(ctx, db.DB, symbolArg, false, false)
		enf := ExplainNotFound{
			Cause:              cause,
			RecommendedActions: actions,
		}
		envelope := NewEnvelope("history", false, map[string]interface{}{
			"explain_not_found": enf,
		}, []string{"symbol \"" + symbolArg + "\" not found"}, Meta{
			SnapshotID: meta.SnapshotID,
			HeadRef:    meta.HeadRef,
			IndexedAt:  meta.IndexedAt,
			IsStale:    stale,
		})
		out, _ := envelope.Encode()
		_, _ = w.Write(out)
		return 3
	}

	// Get history entries.
	entries, err := store.GetSymbolHistory(ctx, db.DB, sym.ID)
	if err != nil {
		WriteErrorEnvelope(w, "history", "history query: "+err.Error(), EmptyMeta())
		return 1
	}

	// Build evidence list from history entries.
	var evidence []EvidenceEntry
	confidence := "low"
	if len(entries) > 0 {
		confidence = "medium"
		for _, e := range entries {
			evidence = append(evidence, EvidenceEntry{
				Type:        "commit_link",
				Description: e.ChangeType + " on " + e.CommitDate + " (" + e.CommitHash[:8] + ")",
				Source:      e.CommitHash,
			})
		}
	}
	if len(evidence) == 0 {
		// Symbol exists but has no history links.
		cause, actions := DeriveHistoryNotFoundCause(ctx, db.DB, symbolArg, true, false)
		enf := ExplainNotFound{
			Cause:              cause,
			RecommendedActions: actions,
		}
		envelope := NewEnvelope("history", true, map[string]interface{}{
			"symbol_name":       symbolArg,
			"confidence":        "low",
			"evidence":          []EvidenceEntry{{Type: "no_history", Description: "no commit history found for this symbol"}},
			"explain_not_found": enf,
		}, nil, Meta{
			SnapshotID: meta.SnapshotID,
			HeadRef:    meta.HeadRef,
			IndexedAt:  meta.IndexedAt,
			IsStale:    stale,
		})
		out, _ := envelope.Encode()
		_, _ = w.Write(out)
		return 0
	}

	data := HistoryData{
		SymbolName: symbolArg,
		Confidence: confidence,
		Evidence:   evidence,
	}

	envelope := NewEnvelope("history", true, data, nil, Meta{
		SnapshotID: meta.SnapshotID,
		HeadRef:    meta.HeadRef,
		IndexedAt:  meta.IndexedAt,
		IsStale:    stale,
	})

	out, _ := envelope.Encode()
	_, _ = w.Write(out)
	return 0
}
