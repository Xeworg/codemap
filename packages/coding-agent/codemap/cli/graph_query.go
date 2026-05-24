package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	"codrut/packages/coding-agent/codemap/store"
)

// defaultGraphQueryDepth is the default max traversal depth.
const defaultGraphQueryDepth = 3

// maxGraphQueryDepth is the hard upper bound for depth traversal.
const maxGraphQueryDepth = 5

// GraphQueryIntent captures the parsed intent from a natural-language prompt.
type GraphQueryIntent struct {
	Symbol    string
	Depth     int
	Cache     bool     // true = use cache if available
	EdgeTypes []string // empty = all edge types
}

// ParseGraphQuery parses a natural-language prompt into a GraphQueryIntent.
// It is fully deterministic and requires no LLM.
// Supported patterns:
//   - "what affects <symbol>"
//   - "what uses <symbol>"
//   - "what depends on <symbol>"
//   - "impact of <symbol>"
//   - "blast radius of <symbol>" / "blast <symbol>"
//   - "who calls <symbol>"
//   - "symbol" alone (defaults depth=3, cache=true)
//
// Depth may be embedded: "3 hops from <symbol>", "<symbol> at depth 3"
var parseRe = regexp.MustCompile(`(?i)(?:(?:what|who)(?:\s+(?:affects|uses|calls|depends\s+on))?|impact\s+of|blast\s+radius\s+of|blast\s+)`)

// depthRe matches "N hops from X", "X at depth N".
var depthRe = regexp.MustCompile(`(?i)(?:(\d+)\s+hops?\s+from|at\s+depth\s+(\d+))`)

// ParseGraphQuery parses a natural-language prompt into a GraphQueryIntent.
// It returns nil intent with an error message on parse failure.
func ParseGraphQuery(prompt string) (*GraphQueryIntent, error) {
	if prompt = strings.TrimSpace(prompt); prompt == "" {
		return nil, fmt.Errorf("prompt required")
	}

	intent := &GraphQueryIntent{
		Depth: defaultGraphQueryDepth,
		Cache: true,
	}

	// Extract depth first (may appear anywhere in prompt).
	depthMatches := depthRe.FindAllStringSubmatch(prompt, -1)
	for _, dm := range depthMatches {
		for _, g := range dm[1:] {
			if g != "" {
				var d int
				if _, err := fmt.Sscanf(g, "%d", &d); err == nil {
					if d < 1 {
						d = 1
					}
					if d > maxGraphQueryDepth {
						d = maxGraphQueryDepth
					}
					intent.Depth = d
					break
				}
			}
		}
	}

	// Extract symbol name: check if prompt starts with a known keyword pattern.
	matched := parseRe.MatchString(prompt)
	if matched {
		// Regex matched: extract the last identifier from the prompt.
		// The keyword prefix already matched, so find the identifier that follows.
		symbol := extractLastIdentifier(prompt)
		if symbol == "" {
			return nil, fmt.Errorf("no symbol found after keyword in prompt %q", prompt)
		}
		intent.Symbol = normalizeSymbol(symbol)
	} else {
		// No keyword match: try to extract the last identifier from the prompt.
		symbol := extractLastIdentifier(prompt)
		if symbol == "" {
			return nil, fmt.Errorf("could not parse symbol from prompt %q", prompt)
		}
		intent.Symbol = normalizeSymbol(symbol)
	}

	// Extract edge type hints.
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "calls") {
		intent.EdgeTypes = append(intent.EdgeTypes, "calls")
	}
	if strings.Contains(lower, "type") || strings.Contains(lower, "uses") {
		intent.EdgeTypes = append(intent.EdgeTypes, "type_use")
	}
	if strings.Contains(lower, "imports") || strings.Contains(lower, "import") {
		intent.EdgeTypes = append(intent.EdgeTypes, "imports")
	}

	return intent, nil
}

// isNumeric reports whether s consists entirely of decimal digits.
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

// extractLastIdentifier extracts the last valid identifier from a space-separated
// string. It skips Go keywords, common NLP noise words, and punctuation.
func extractLastIdentifier(s string) string {
	parts := strings.Fields(s)
	// Filter out common NLP/stop words and Go keywords that are not symbol names.
	skip := map[string]bool{
		"of": true, "from": true, "to": true, "the": true, "a": true, "an": true,
		"is": true, "are": true, "was": true, "be": true, "been": true,
		"at": true, "in": true, "on": true, "by": true, "for": true,
		"depth": true, "hops": true, "hop": true,
		"what": true, "who": true, "calls": true, "uses": true,
		"affects": true, "depends": true, "impact": true,
	}
	for i := len(parts) - 1; i >= 0; i-- {
		p := parts[i]
		// Strip trailing punctuation.
		p = strings.TrimRight(p, ",.!?;:)")
		pLower := strings.ToLower(p)
		if skip[pLower] || isGoKeyword(pLower) {
			continue
		}
		if isValidIdentifier(p) {
			return p
		}
	}
	return ""
}

// isGoKeyword reports whether lowercased s is a Go keyword.
func isGoKeyword(s string) bool {
	keywords := []string{
		"break", "case", "chan", "const", "continue", "default", "defer",
		"else", "fallthrough", "for", "func", "go", "goto", "if", "import",
		"interface", "map", "package", "range", "return", "select", "struct",
		"switch", "type", "var",
	}
	for _, k := range keywords {
		if s == k {
			return true
		}
	}
	return false
}

// isValidIdentifier reports whether s is a valid Go-style identifier.
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	first := s[0]
	if (first < 'A' || first > 'Z') && (first < 'a' || first > 'z') && first != '_' {
		return false
	}
	for _, c := range s[1:] {
		if (c < '0' || c > '9') && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && c != '_' {
			return false
		}
	}
	return true
}

// normalizeSymbol returns a canonical symbol name: first char upper, rest lower,
// unless the symbol is all-caps (preserve it as-is for constants like "HTTP").
func normalizeSymbol(s string) string {
	// Preserve all-caps identifiers.
	if isAllCaps(s) {
		return s
	}
	if len(s) == 0 {
		return s
	}
	// Title-case: uppercase first rune, lowercase the rest.
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	if len(runes) > 1 {
		for i := 1; i < len(runes); i++ {
			runes[i] = unicode.ToLower(runes[i])
		}
	}
	return string(runes)
}

// isAllCaps reports whether s is an all-uppercase identifier (e.g. "HTTP", "API").
func isAllCaps(s string) bool {
	hasLetter := false
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			hasLetter = true
		} else if c >= 'a' && c <= 'z' {
			return false
		}
	}
	return hasLetter
}

// RunGraphQuery runs the "graph-query" command and returns an exit code.
func RunGraphQuery(ctx context.Context, w io.Writer, args []string, repoRoot string) int {
	fs := flag.NewFlagSet("graph-query", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	dbPathFlag := fs.String("db", "", "Path to SQLite database (optional)")
	noCacheFlag := fs.Bool("no-cache", false, "Force live CTE query, bypassing the impact cache")
	symbolFlag := fs.String("symbol", "", "Explicit symbol name (overrides parsed symbol)")
	explicitDepth := fs.Int("depth", defaultGraphQueryDepth, fmt.Sprintf("Override parsed depth (1-%d)", maxGraphQueryDepth))
	if err := fs.Parse(args); err != nil {
		WriteErrorEnvelope(w, "graph-query", err.Error(), EmptyMeta())
		return 2
	}

	promptArg := ""
	if fs.NArg() > 0 {
		promptArg = fs.Arg(0)
	}
	if promptArg == "" && *symbolFlag == "" {
		WriteErrorEnvelope(w, "graph-query", "prompt or --symbol required", EmptyMeta())
		return 2
	}

	// Parse intent unless symbol was forced.
	var intent *GraphQueryIntent
	var err error

	if *symbolFlag != "" {
		intent = &GraphQueryIntent{
			Symbol: *symbolFlag,
			Depth:  *explicitDepth,
			Cache:  !*noCacheFlag,
		}
	} else {
		intent, err = ParseGraphQuery(promptArg)
		if err != nil {
			WriteErrorEnvelope(w, "graph-query", err.Error(), EmptyMeta())
			return 2
		}
		// CLI flags override parsed values.
		if *explicitDepth != defaultGraphQueryDepth {
			intent.Depth = *explicitDepth
		}
		if *noCacheFlag {
			intent.Cache = false
		}
	}

	// Bounds check depth.
	if intent.Depth < 1 {
		intent.Depth = 1
	}
	if intent.Depth > maxGraphQueryDepth {
		intent.Depth = maxGraphQueryDepth
	}

	dbPath, err := ResolveDBPath(*dbPathFlag, repoRoot)
	if err != nil {
		WriteErrorEnvelope(w, "graph-query", "resolve db path: "+err.Error(), EmptyMeta())
		return 1
	}
	db, err := store.Open(dbPath)
	if err != nil {
		WriteErrorEnvelope(w, "graph-query", "open db: "+err.Error(), EmptyMeta())
		return 1
	}
	defer db.Close()

	meta, err := store.GetLatestSnapshotMeta(ctx, db.DB)
	if err != nil {
		WriteErrorEnvelope(w, "graph-query", "read meta: "+err.Error(), EmptyMeta())
		return 1
	}
	if meta.SnapshotID == 0 {
		WriteErrorEnvelope(w, "graph-query", "no index found (run 'codemap index' first)", EmptyMeta())
		return 3
	}

	sym, err := store.GetSymbolByName(ctx, db.DB, intent.Symbol)
	if err != nil {
		WriteErrorEnvelope(w, "graph-query", "symbol query: "+err.Error(), EmptyMeta())
		return 1
	}
	if sym == nil {
		WriteErrorEnvelope(w, "graph-query", "symbol \""+intent.Symbol+"\" not found", EmptyMeta())
		return 3
	}

	var hits []store.SymbolHit
	cacheHit := false

	if intent.Cache {
		hits, cacheHit, err = store.GetCachedImpact(ctx, db.DB, sym.ID, intent.Depth, store.DefaultCacheTTL)
		if err != nil {
			_, _ = fmt.Fprintf(io.Discard, "cache lookup failed: %v\n", err)
		}
	}

	if !cacheHit {
		hits, err = store.BlastRadiusQuery(ctx, db.DB, sym.ID, intent.Depth, intent.EdgeTypes)
		if err != nil {
			WriteErrorEnvelope(w, "graph-query", "blast radius query: "+err.Error(), EmptyMeta())
			return 1
		}
		if intent.Cache && len(hits) > 0 {
			go func() {
				_ = store.WriteCachedImpact(context.Background(), db.DB, sym.ID, hits)
			}()
		}
	}

	// Build findings from hits.
	hitMap := make(map[int64]store.SymbolHit, len(hits))
	for _, h := range hits {
		hitMap[h.SymbolID] = h
	}

	var findings []ImpactFinding
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

	if findings == nil {
		findings = []ImpactFinding{}
	}

	stale := StaleNow(meta.IndexedAt)
	data := ImpactData{
		TargetSymbol: sym.Name,
		Findings:     findings,
		Evidence: []EvidenceEntry{
			{Type: "parser_hint", Description: fmt.Sprintf("depth=%d cache=%v", intent.Depth, intent.Cache)},
		},
	}

	envelope := NewEnvelope("graph-query", true, data, nil, Meta{
		SnapshotID: meta.SnapshotID,
		HeadRef:    meta.HeadRef,
		IndexedAt:  meta.IndexedAt,
		IsStale:    stale,
	})
	out, _ := envelope.Encode()
	_, _ = w.Write(out)
	return 0
}
