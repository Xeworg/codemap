package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"codrut/packages/coding-agent/codemap/store"
)

// defaultImpactLimit is the maximum number of findings returned per impact query.
const defaultImpactLimit = 50

// defaultImpactDepth is the default max traversal depth for blast-radius queries.
const defaultImpactDepth = 3

// maxImpactDepth is the hard upper bound for depth traversal.
const maxImpactDepth = 5

// RunImpact runs the "impact" command and returns an exit code.
func RunImpact(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	fs := flag.NewFlagSet("impact", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	dbPathFlag := fs.String("db", "", "Path to SQLite database (optional)")
	depthFlag := fs.Int("depth", defaultImpactDepth, fmt.Sprintf("Maximum traversal depth (1-%d)", maxImpactDepth))
	noCacheFlag := fs.Bool("no-cache", false, "Force live CTE query, bypassing the impact cache")
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
	depth := *depthFlag
	if depth < 1 {
		depth = 1
	}
	if depth > maxImpactDepth {
		depth = maxImpactDepth
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

	var hits []store.SymbolHit
	cacheHit := false

	if !*noCacheFlag {
		// Try cache first.
		var ok bool
		hits, ok, err = store.GetCachedImpact(ctx, db.DB, sym.ID, depth, store.DefaultCacheTTL)
		if err != nil {
			// Log and fall through to live query.
			_, _ = fmt.Fprintf(io.Discard, "cache lookup failed: %v\n", err)
		}
		cacheHit = ok
	}

	if !cacheHit {
		// Fall back to live CTE traversal.
		hits, err = store.BlastRadiusQuery(ctx, db.DB, sym.ID, depth, nil)
		if err != nil {
			WriteErrorEnvelope(w, "impact", "blast radius query: "+err.Error(), EmptyMeta())
			return 1
		}
		// Write to cache asynchronously if we skipped it.
		if !*noCacheFlag && len(hits) > 0 {
			go func() {
				_ = store.WriteCachedImpact(context.Background(), db.DB, sym.ID, hits)
			}()
		}
	}

	// Build symbol ID -> hit map.
	hitMap := make(map[int64]store.SymbolHit, len(hits))
	for _, h := range hits {
		hitMap[h.SymbolID] = h
	}

	// Resolve target symbol details and build findings.
	// For depth-1 (direct edges), use the existing incident-edge approach
	// for richer evidence. For depth > 1, use cached hit metadata only.
	var findings []ImpactFinding
	if depth == 1 || len(hitMap) == 0 {
		// Direct edges with full evidence.
		edges, err := store.GetSymbolEdges(ctx, db.DB, sym.ID)
		if err != nil {
			WriteErrorEnvelope(w, "impact", "edges query: "+err.Error(), EmptyMeta())
			return 1
		}
		seen := make(map[int64]bool)
		for _, e := range edges {
			otherID := e.FromSymbolID
			if otherID == sym.ID {
				otherID = e.ToSymbolID
			}
			if otherID == sym.ID || seen[otherID] {
				continue
			}
			seen[otherID] = true

			other, err := store.GetSymbolByID(ctx, db.DB, otherID)
			if err != nil || other == nil {
				continue
			}
			riskTier := deriveRiskTier(e.EdgeType)
			confidence := deriveImpactConfidence(e.EdgeType, other.Kind)
			findings = append(findings, ImpactFinding{
				SymbolName: other.Name,
				File:       other.File,
				Kind:       other.Kind,
				StartLine:  other.StartLine,
				EndLine:    other.EndLine,
				RiskTier:   riskTier,
				Confidence: confidence,
				Depth:      1,
				EdgePath:   []string{e.EdgeType},
				Evidence: []EvidenceEntry{
					{Type: "symbol_link", Description: "linked via " + e.EdgeType, Source: other.File},
				},
			})
		}
	} else {
		// Multi-hop: build findings from hit map with cached metadata.
		for targetID, h := range hitMap {
			if targetID == sym.ID {
				continue
			}
			other, err := store.GetSymbolByID(ctx, db.DB, targetID)
			if err != nil || other == nil {
				continue
			}
			edgeTypes := parseEdgePath(h.EdgePath)
			riskTier := riskTierFromEdgePath(edgeTypes)
			confidence := confidenceFromDepthAndKind(h.Depth, other.Kind, edgeTypes)
			findings = append(findings, ImpactFinding{
				SymbolName: other.Name,
				File:       other.File,
				Kind:       other.Kind,
				StartLine:  other.StartLine,
				EndLine:    other.EndLine,
				RiskTier:   riskTier,
				Confidence: confidence,
				Depth:      h.Depth,
				EdgePath:   edgeTypes,
				Evidence: []EvidenceEntry{
					{Type: "blast_radius", Description: fmt.Sprintf("reached via %d hop(s)", h.Depth), Source: other.File},
				},
			})
		}
	}

	// Ensure non-nil slice for deterministic JSON.
	if findings == nil {
		findings = []ImpactFinding{}
	}

	// Sort: depth first (shallow = higher priority), then risk, then name.
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Depth != findings[j].Depth {
			return findings[i].Depth < findings[j].Depth
		}
		ri := RiskTierPriority(findings[i].RiskTier)
		rj := RiskTierPriority(findings[j].RiskTier)
		if ri != rj {
			return ri < rj
		}
		if findings[i].SymbolName != findings[j].SymbolName {
			return findings[i].SymbolName < findings[j].SymbolName
		}
		return findings[i].File < findings[j].File
	})

	// Apply cap.
	if len(findings) > defaultImpactLimit {
		findings = findings[:defaultImpactLimit]
	}

	stale := StaleNow(meta.IndexedAt)
	data := ImpactData{
		TargetSymbol: sym.Name,
		Findings:     findings,
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

// parseEdgePath splits a comma-separated edge path string into individual edge types.
func parseEdgePath(path string) []string {
	if path == "" {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(path, ","), ",")
	// Trim the leading empty element from the split.
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// riskTierFromEdgePath returns the highest risk tier along an edge path.
func riskTierFromEdgePath(edgeTypes []string) string {
	highest := "low"
	for _, et := range edgeTypes {
		tier := deriveRiskTier(et)
		if tier == "high" {
			return "high"
		}
		if tier == "medium" {
			highest = "medium"
		}
	}
	return highest
}

// confidenceFromDepthAndKind returns a confidence level based on traversal depth
// and the target symbol's kind.
func confidenceFromDepthAndKind(depth int, kind string, edgeTypes []string) string {
	if depth == 1 && len(edgeTypes) > 0 {
		return deriveImpactConfidence(edgeTypes[0], kind)
	}
	if depth == 2 {
		return "medium"
	}
	return "low"
}

// deriveRiskTier maps an edge type to a risk tier heuristic.
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
	if edgeType == "calls" || edgeType == "type_use" {
		switch symbolKind {
		case "func", "type", "interface":
			return "high"
		case "var", "const":
			return "medium"
		}
	}
	switch edgeType {
	case "imports", "casts":
		return "medium"
	default:
		return "low"
	}
}
