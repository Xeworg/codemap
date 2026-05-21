package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"sort"

	"codrut/packages/coding-agent/codemap/store"
)

// RunImpact runs the "impact" command and returns an exit code.
func RunImpact(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	fs := flag.NewFlagSet("impact", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	dbPathFlag := fs.String("db", "", "Path to SQLite database (optional)")
	_ = fs.Bool("json", false, "Output JSON envelope (default)")
	if err := fs.Parse(args); err != nil {
		WriteErrorEnvelope(w, "impact", err.Error(), EmptyMeta())
		return 2
	}

	symbolArg := ""
	if fs.NArg() > 0 {
		symbolArg = fs.Arg(0)
	}

	// Validation.
	if symbolArg == "" {
		WriteErrorEnvelope(w, "impact", "symbol name required", EmptyMeta())
		return 2
	}

	dbPath, err := ResolveDBPath(*dbPathFlag, repoRoot)
	if err != nil {
		WriteErrorEnvelope(w, "impact", "resolve db path: "+err.Error(), EmptyMeta())
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		WriteErrorEnvelope(w, "impact", "open db: "+err.Error(), EmptyMeta())
		return 1
	}
	defer db.Close()

	// Require indexed state.
	meta, err := store.GetLatestSnapshotMeta(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "impact", "read meta: "+err.Error(), EmptyMeta())
		return 1
	}
	if meta.SnapshotID == 0 {
		WriteErrorEnvelope(w, "impact", "no index found (run 'codemap index' first)", EmptyMeta())
		return 3
	}

	// Resolve target symbol.
	sym, err := store.GetSymbolByName(ctx, db.DB, symbolArg)
	if err != nil {
		WriteErrorEnvelope(w, "impact", "query: "+err.Error(), EmptyMeta())
		return 1
	}
	if sym == nil {
		WriteErrorEnvelope(w, "impact", "symbol \""+symbolArg+"\" not found", EmptyMeta())
		return 3
	}

	// Get incident edges.
	edges, err := store.GetSymbolEdges(ctx, db.DB, sym.ID)
	if err != nil {
		WriteErrorEnvelope(w, "impact", "edges query: "+err.Error(), EmptyMeta())
		return 1
	}

	// Build affected symbols set (other end of each edge).
	affectedSet := make(map[string]bool)
	for _, e := range edges {
		otherID := e.FromSymbolID
		if otherID == sym.ID {
			otherID = e.ToSymbolID
		}
		other, err := store.GetSymbolByID(ctx, db.DB, otherID)
		if err != nil || other == nil {
			continue
		}
		affectedSet[other.Name] = true
	}
	var affected []string
	for name := range affectedSet {
		affected = append(affected, name)
	}
	sort.Strings(affected)
	if affected == nil {
		affected = []string{}
	}

	// Build evidence entries.
	var evidence []EvidenceEntry
	for _, e := range edges {
		otherID := e.FromSymbolID
		if otherID == sym.ID {
			otherID = e.ToSymbolID
		}
		other, err := store.GetSymbolByID(ctx, db.DB, otherID)
		if err != nil || other == nil {
			continue
		}
		desc := fmt.Sprintf("linked via %s", e.EdgeType)
		evidence = append(evidence, EvidenceEntry{
			Type:        "symbol_link",
			Description: desc,
			Source:      other.File,
		})
	}
	if evidence == nil {
		evidence = []EvidenceEntry{}
	}
	// Sort evidence by description for determinism.
	sort.SliceStable(evidence, func(i, j int) bool {
		return evidence[i].Description < evidence[j].Description
	})

	stale := StaleNow(meta.IndexedAt)
	data := ImpactData{
		TargetSymbol:    sym.Name,
		AffectedSymbols: affected,
		Evidence:        evidence,
	}

	envelope := NewEnvelope("impact", true, data, nil, Meta{
		SnapshotID: meta.SnapshotID,
		HeadRef:    meta.HeadRef,
		IndexedAt:  meta.IndexedAt,
		IsStale:    stale,
	})
	out, _ := envelope.Encode()
	_, _ = w.Write(out)
	return 0
}
