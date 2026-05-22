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
// It analyzes symbols and classifies them as dead code using inbound edge
// counts and heuristic entrypoint/public-API detection.
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

	// Get all symbols with pre-computed inbound edge counts in one query.
	symbols, err := store.GetAllSymbolsWithInboundCounts(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "deadcode", "query symbols: "+err.Error(), EmptyMeta())
		return 1
	}

	// Filter out generated files, test fixtures, and symbols with inbound edges.
	var findings []DeadcodeFinding
	for _, sym := range symbols {
		if isGeneratedOrTestFixture(sym.File) {
			continue
		}
		classification, suggestion, confidence := classifyDeadcode(sym.InboundCount, sym.Kind, sym.Name, sym.File)
		evidence := deadcodeEvidence(sym.InboundCount, sym.Name, sym.File)
		finding := DeadcodeFinding{
			SymbolName:     sym.Name,
			File:           sym.File,
			Kind:           sym.Kind,
			StartLine:      sym.StartLine,
			EndLine:        sym.EndLine,
			Classification: classification,
			Suggestion:     suggestion,
			Confidence:     confidence,
			Evidence:       evidence,
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

// isRuntimeEntrypoint returns true for main/init entrypoints.
func isRuntimeEntrypoint(name, file string) bool {
	if name == "main" || name == "init" {
		return true
	}
	return false
}

// isPublicAPI returns true for exported symbols that may be used externally.
// Heuristic: name starts with uppercase letter (v1 boundary).
func isPublicAPI(name string) bool {
	if name == "" {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}

// isEntrypointFile returns true for files matching cmd/ entrypoint patterns.
func isEntrypointFile(file string) bool {
	lowercase := strings.ToLower(file)
	return strings.Contains(lowercase, "/cmd/") ||
		strings.Contains(lowercase, "/main.go") ||
		strings.Contains(lowercase, "\\cmd\\")
}

// deadcodeEvidence builds a composable evidence slice for a symbol.
func deadcodeEvidence(inbound int, name, file string) []EvidenceEntry {
	var ev []EvidenceEntry
	if inbound == 0 {
		ev = append(ev, EvidenceEntry{Type: EvidenceNoInboundEdges, Description: "symbol has no inbound references in the code graph"})
	} else {
		ev = append(ev, EvidenceEntry{Type: EvidenceInboundEdges, Description: "symbol has inbound references"})
	}
	if isRuntimeEntrypoint(name, file) {
		ev = append(ev, EvidenceEntry{Type: EvidenceImplicitRuntime, Description: "symbol is a runtime entrypoint (main or init)"})
	}
	if isPublicAPI(name) {
		ev = append(ev, EvidenceEntry{Type: EvidencePublicAPISurface, Description: "symbol is part of the public API surface"})
	}
	return ev
}

// classifyDeadcode determines the classification, suggestion, confidence, and
// evidence for a deadcode finding based on inbound count, symbol kind, name, and file.
func classifyDeadcode(inbound int, kind, name, file string) (classification, suggestion, confidence string) {
	if inbound > 0 {
		return "uncertain", "review", "low"
	}
	// inbound == 0
	if isRuntimeEntrypoint(name, file) {
		return "uncertain", "review", "low"
	}
	if isPublicAPI(name) {
		return "uncertain", "justify", "low"
	}
	// No heuristics apply: use kind-based confidence.
	switch kind {
	case "func", "type":
		return "unused", "remove", "high"
	case "var", "const":
		return "unused", "remove", "medium"
	case "method":
		return "unused", "remove", "medium"
	default:
		return "unused", "remove", "low"
	}
}

// sortDeadcodeFindings sorts findings deterministically.
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
