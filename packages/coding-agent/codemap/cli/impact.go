package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"sort"

	"codrut/packages/coding-agent/codemap/store"
)

// defaultImpactLimit is the maximum number of findings returned per impact query.
const defaultImpactLimit = 50

// riskTierPriority maps risk tier to sort priority (lower = higher priority).
var riskTierPriority = map[string]int{
	"high":   0,
	"medium": 1,
	"low":    2,
}

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

	// Build findings per affected symbol.
	findings := make(map[int64]*ImpactFinding) // dedup by symbol ID
	for _, e := range edges {
		otherID := e.FromSymbolID
		if otherID == sym.ID {
			otherID = e.ToSymbolID
		}
		if otherID == sym.ID {
			continue // skip self-edge
		}
		if _, seen := findings[otherID]; seen {
			continue
		}
		other, err := store.GetSymbolByID(ctx, db.DB, otherID)
		if err != nil || other == nil {
			continue
		}
		riskTier := deriveRiskTier(e.EdgeType)
		confidence := deriveImpactConfidence(e.EdgeType, other.Kind)
		evidence := []EvidenceEntry{
			{
				Type:        "symbol_link",
				Description: fmt.Sprintf("linked via %s", e.EdgeType),
				Source:      other.File,
			},
		}
		findings[otherID] = &ImpactFinding{
			SymbolName: other.Name,
			File:       other.File,
			Kind:       other.Kind,
			StartLine:  other.StartLine,
			EndLine:    other.EndLine,
			RiskTier:   riskTier,
			Confidence: confidence,
			Evidence:   evidence,
		}
	}

	// Convert to slice and sort.
	var sorted []ImpactFinding
	for _, f := range findings {
		sorted = append(sorted, *f)
	}
	// Ensure non-nil slice for deterministic JSON serialization.
	if sorted == nil {
		sorted = []ImpactFinding{}
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		ri := RiskTierPriority(sorted[i].RiskTier)
		rj := RiskTierPriority(sorted[j].RiskTier)
		if ri != rj {
			return ri < rj
		}
		ci := ConfidenceRank(sorted[i].Confidence)
		cj := ConfidenceRank(sorted[j].Confidence)
		if ci != cj {
			return ci < cj
		}
		if sorted[i].SymbolName != sorted[j].SymbolName {
			return sorted[i].SymbolName < sorted[j].SymbolName
		}
		return sorted[i].File < sorted[j].File
	})

	// Apply cap.
	if len(sorted) > defaultImpactLimit {
		sorted = sorted[:defaultImpactLimit]
	}

	stale := StaleNow(meta.IndexedAt)
	data := ImpactData{
		TargetSymbol: sym.Name,
		Findings:     sorted,
		Evidence:     []EvidenceEntry{},
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

// deriveRiskTier maps an edge type to a risk tier heuristic.
// Order: calls > type_use > imports > casts > default=medium.
func deriveRiskTier(edgeType string) string {
	switch edgeType {
	case "calls":
		return "high"
	case "type_use", "references":
		return "medium"
	case "imports", "casts", "subtype", "exports":
		return "medium"
	default:
		return "low"
	}
}

// deriveImpactConfidence returns a confidence level based on edge type and symbol kind.
func deriveImpactConfidence(edgeType, symbolKind string) string {
	// Strong edges (calls, type_use) with important kinds get high confidence.
	if edgeType == "calls" || edgeType == "type_use" {
		switch symbolKind {
		case "func", "type", "interface":
			return "high"
		case "var", "const":
			return "medium"
		}
	}
	// Edge quality degrades for weaker link types.
	switch edgeType {
	case "imports", "casts":
		return "medium"
	default:
		return "low"
	}
}
