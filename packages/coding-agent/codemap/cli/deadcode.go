package cli

import (
	"context"
	"flag"
	"io"
	"sort"
	"strings"

	"codrut/packages/coding-agent/codemap/store"
)

// defaultDeadcodeLimit is the maximum number of findings returned per deadcode query.
const defaultDeadcodeLimit = 100

// RunDeadcode runs the "deadcode" command and returns an exit code.
// It analyzes symbols with zero inbound edges and classifies them as dead code.
func RunDeadcode(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	fs := flag.NewFlagSet("deadcode", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	dbPathFlag := fs.String("db", "", "Path to SQLite database (optional)")
	limitFlag := fs.Int("limit", defaultDeadcodeLimit, "Maximum number of findings to return")
	_ = fs.Bool("json", false, "Output JSON envelope (default)")
	if err := fs.Parse(args); err != nil {
		WriteErrorEnvelope(w, "deadcode", err.Error(), EmptyMeta())
		return 2
	}

	dbPath, err := ResolveDBPath(*dbPathFlag, repoRoot)
	if err != nil {
		WriteErrorEnvelope(w, "deadcode", "resolve db path: "+err.Error(), EmptyMeta())
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		WriteErrorEnvelope(w, "deadcode", "open db: "+err.Error(), EmptyMeta())
		return 1
	}
	defer db.Close()

	// Require indexed state.
	meta, err := store.GetLatestSnapshotMeta(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "deadcode", "read meta: "+err.Error(), EmptyMeta())
		return 1
	}
	if meta.SnapshotID == 0 {
		WriteErrorEnvelope(w, "deadcode", "no index found (run 'codemap index' first)", EmptyMeta())
		return 3
	}

	// Get symbols with zero inbound edges.
	symbols, err := store.GetSymbolsWithZeroInboundEdges(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "deadcode", "query symbols: "+err.Error(), EmptyMeta())
		return 1
	}

	// Filter out generated files and test fixtures.
	var filtered []store.SymbolRow
	for _, sym := range symbols {
		if isGeneratedOrTestFixture(sym.File) {
			continue
		}
		filtered = append(filtered, sym)
	}
	symbols = filtered

	// Build findings.
	var findings []DeadcodeFinding
	for _, sym := range symbols {
		// Check edge count for classification.
		edges, err := store.GetSymbolEdges(ctx, db.DB, sym.ID)
		if err != nil {
			continue
		}
		inboundCount := 0
		for _, e := range edges {
			if e.ToSymbolID == sym.ID {
				inboundCount++
			}
		}
		classification, suggestion, confidence := classifyDeadcode(inboundCount, sym.Kind)
		finding := DeadcodeFinding{
			SymbolName:     sym.Name,
			File:           sym.File,
			Kind:           sym.Kind,
			StartLine:      sym.StartLine,
			EndLine:        sym.EndLine,
			Classification: classification,
			Suggestion:     suggestion,
			Confidence:     confidence,
			Evidence: []EvidenceEntry{
				{
					Type:        "no_inbound_edges",
					Description: "symbol has no inbound references in the code graph",
				},
			},
		}
		findings = append(findings, finding)
	}

	// Sort: classification rank → confidence → symbol name → file.
	sortDeadcodeFindings(findings)

	// Apply limit.
	if len(findings) > *limitFlag {
		findings = findings[:*limitFlag]
	}

	// Ensure non-nil slice for deterministic JSON.
	if findings == nil {
		findings = []DeadcodeFinding{}
	}

	stale := StaleNow(meta.IndexedAt)
	data := DeadcodeData{
		Findings: findings,
	}

	envelope := NewEnvelope("deadcode", true, data, nil, Meta{
		SnapshotID: meta.SnapshotID,
		HeadRef:    meta.HeadRef,
		IndexedAt:  meta.IndexedAt,
		IsStale:    stale,
	})
	out, _ := envelope.Encode()
	_, _ = w.Write(out)
	return 0
}

// deadcodeClassificationRank returns sort priority (lower = more severe).
func deadcodeClassificationRank(class string) int {
	switch class {
	case "unused":
		return 0
	case "likely-unused":
		return 1
	case "uncertain":
		return 2
	default:
		return 3
	}
}

// isGeneratedOrTestFixture returns true if the file path indicates a generated
// file or test fixture that should be excluded from deadcode analysis.
func isGeneratedOrTestFixture(filePath string) bool {
	lowercase := strings.ToLower(filePath)
	patterns := []string{
		"_generated",
		".gen.go",
		"_test.go",
		"_mock.go",
		"_fake.go",
		"testdata/",
		"vendor/",
		"third_party/",
		"_pb.go",   // protobuf
		".pb.go",   // protobuf
		"_grpc.go", // grpc
	}
	for _, p := range patterns {
		if strings.Contains(lowercase, p) {
			return true
		}
	}
	return false
}

// classifyDeadcode determines the classification, suggestion, and confidence
// for a deadcode finding based on edge count and symbol kind.
func sortDeadcodeFindings(findings []DeadcodeFinding) {
	sort.SliceStable(findings, func(i, j int) bool {
		ci := deadcodeClassificationRank(findings[i].Classification)
		cj := deadcodeClassificationRank(findings[j].Classification)
		if ci != cj {
			return ci < cj
		}
		ri := ConfidenceRank(findings[i].Confidence)
		rj := ConfidenceRank(findings[j].Confidence)
		if ri != rj {
			return ri < rj
		}
		if findings[i].SymbolName != findings[j].SymbolName {
			return findings[i].SymbolName < findings[j].SymbolName
		}
		return findings[i].File < findings[j].File
	})
}

func classifyDeadcode(inboundCount int, symbolKind string) (classification, suggestion, confidence string) {
	if inboundCount == 0 {
		// Zero inbound edges: could be truly unused.
		// Higher confidence for well-known stable kinds.
		switch symbolKind {
		case "func", "type":
			return "unused", "remove", "high"
		case "var", "const":
			return "unused", "remove", "medium"
		default:
			return "unused", "remove", "low"
		}
	}
	// Some edges exist: fall through to uncertain.
	return "uncertain", "justify", "low"
}
