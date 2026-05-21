package indexer

import (
	"fmt"
	"sort"
)

// LinkStrength represents how directly a commit touches a symbol.
type LinkStrength string

const (
	LinkStrong LinkStrength = "strong"
	LinkMedium LinkStrength = "medium"
	LinkWeak   LinkStrength = "weak"
)

// IsValid reports whether s is one of the allowed enum values.
func (s LinkStrength) IsValid() bool {
	switch s {
	case LinkStrong, LinkMedium, LinkWeak:
		return true
	default:
		return false
	}
}

// CommitHunk describes the line range touched by a single commit hunk.
// Line numbers are 1-based. Ranges are half-open [StartLine, EndLine).
type CommitHunk struct {
	StartLine int // 1-based start line
	EndLine   int // exclusive end line
}

// SymbolRange is the source-code symbol's line span.
type SymbolRange struct {
	Name      string
	StartLine int
	EndLine   int
}

// HistoryEntry summarises a commit-symbol link for history output.
type HistoryEntry struct {
	SymbolName   string
	CommitHash   string
	LinkStrength LinkStrength
	CommitDate   string // ISO date string "2006-01-02"
}

// ClassifyLink computes the link strength between a symbol and a commit's hunks.
// The algorithm:
//   - strong: at least one hunk's line range intersects [sym.StartLine, sym.EndLine)
//   - medium: no intersection but at least one hunk is within ±30 lines of the symbol
//   - weak: commit exists but neither intersection nor proximity holds
func ClassifyLink(sym SymbolRange, hunks []CommitHunk) LinkStrength {
	if len(hunks) == 0 {
		return LinkWeak
	}
	symStart, symEnd := sym.StartLine, sym.EndLine
	// First pass: check for direct intersection → strong
	for _, h := range hunks {
		if max(h.StartLine, symStart) < min(h.EndLine, symEnd) {
			return LinkStrong
		}
	}
	// Second pass: check proximity window (±30 line-centre distance)
	const proximityWindow = 30
	for _, h := range hunks {
		hCenter := (h.StartLine + h.EndLine) / 2
		symCenter := (symStart + symEnd) / 2
		if abs(hCenter-symCenter) <= proximityWindow {
			return LinkMedium
		}
	}
	return LinkWeak
}

// OrderHistory sorts entries by strength descending, then recency descending.
// Within equal strength and date, original order is preserved (stable).
func OrderHistory(entries []HistoryEntry) []HistoryEntry {
	strengthRank := map[LinkStrength]int{
		LinkStrong: 3,
		LinkMedium: 2,
		LinkWeak:   1,
	}
	type sortEntry struct {
		origIdx int
		entry   HistoryEntry
		key     int
		date    int
	}
	var list []sortEntry
	for i, e := range entries {
		list = append(list, sortEntry{
			origIdx: i,
			entry:   e,
			key:     strengthRank[e.LinkStrength],
			date:    parseDate(e.CommitDate),
		})
	}
	// sort.SliceStable is guaranteed stable: equal elements maintain original relative order.
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].key != list[j].key {
			return list[i].key > list[j].key
		}
		return list[i].date > list[j].date
	})
	out := make([]HistoryEntry, len(list))
	for i, e := range list {
		out[i] = e.entry
	}
	return out
}

// parseDate converts "2006-01-02" to an int for ordering comparison.
func parseDate(s string) int {
	if len(s) < 10 {
		return 0
	}
	var y, m, d int
	n, _ := fmt.Sscanf(s[:10], "%d-%d-%d", &y, &m, &d)
	if n != 3 {
		return 0
	}
	return y*10000 + m*100 + d
}

// abs returns the absolute value of i.
func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}
