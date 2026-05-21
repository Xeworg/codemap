package indexer

import "testing"

// --- Edge cases for history linking ---

// Test: commit touches same file but hunk does NOT intersect symbol range.
func TestClassifyLinkSameFileOutsideRange(t *testing.T) {
	// Symbol at lines 10-20; commit changes lines 50-60 in the same file.
	sym := SymbolRange{Name: "Foo", StartLine: 10, EndLine: 20}
	hunks := []CommitHunk{{StartLine: 50, EndLine: 60}}
	got := ClassifyLink(sym, hunks)
	// No intersection; center diff = |55-15| = 40 > 30 → weak
	if got != LinkWeak {
		t.Errorf("commit touching same file outside range: got %v, want weak", got)
	}
}

// Test: rename-like edit in same file (two hunks: old location removed, new location added).
func TestClassifyLinkRenameLikeEdit(t *testing.T) {
	// Symbol "Foo" moved from lines 10-20 to lines 30-40.
	// Commit has two hunks: deletion at 10-20 and addition at 30-40.
	symNew := SymbolRange{Name: "Foo", StartLine: 30, EndLine: 40}
	hunks := []CommitHunk{
		{StartLine: 10, EndLine: 20}, // old location (no longer exists in new sym)
		{StartLine: 30, EndLine: 40}, // new location — intersects new symbol
	}
	got := ClassifyLink(symNew, hunks)
	// Second hunk intersects → strong
	if got != LinkStrong {
		t.Errorf("rename-like edit: got %v, want strong (addition hunk intersects)", got)
	}
}

// Test: commit with multiple hunks, only one touches the symbol.
func TestClassifyLinkMultipleHunksOneIntersects(t *testing.T) {
	sym := SymbolRange{Name: "Bar", StartLine: 100, EndLine: 110}
	hunks := []CommitHunk{
		{StartLine: 1, EndLine: 10},    // far away
		{StartLine: 5, EndLine: 15},    // no intersection; center diff 10 ≤ 30 → medium
		{StartLine: 105, EndLine: 115}, // intersects → strong
	}
	got := ClassifyLink(sym, hunks)
	if got != LinkStrong {
		t.Errorf("multi-hunk one intersects: got %v, want strong", got)
	}
}

// Test: single-line symbol with single-line hunk at same position.
func TestClassifyLinkSingleLineExactMatch(t *testing.T) {
	sym := SymbolRange{Name: "X", StartLine: 7, EndLine: 8}
	hunks := []CommitHunk{{StartLine: 7, EndLine: 8}}
	got := ClassifyLink(sym, hunks)
	if got != LinkStrong {
		t.Errorf("exact single-line match: got %v, want strong", got)
	}
}

// Test: symbol at very start of file, hunk at very end.
func TestClassifyLinkExtremePositions(t *testing.T) {
	sym := SymbolRange{Name: "Top", StartLine: 1, EndLine: 5}
	hunks := []CommitHunk{{StartLine: 100, EndLine: 110}}
	got := ClassifyLink(sym, hunks)
	// No intersection; center diff = |105-3| = 102 > 30 → weak
	if got != LinkWeak {
		t.Errorf("extreme positions: got %v, want weak", got)
	}
}

// Test: multiple symbols in same file, each gets correct classification.
func TestClassifyLinkMultipleSymbolsSameCommit(t *testing.T) {
	hunks := []CommitHunk{{StartLine: 50, EndLine: 70}}
	syms := []struct {
		name  string
		start int
		end   int
		want  LinkStrength
	}{
		{"in_range", 55, 65, LinkStrong}, // intersects hunk [50,70)
		{"nearby", 20, 25, LinkWeak},     // center 22.5 vs hunk center 60, diff=37.5>30 → weak
		{"far", 1, 10, LinkWeak},         // center 5.5, diff 54.5 > 30
	}
	for _, tc := range syms {
		t.Run(tc.name, func(t *testing.T) {
			sym := SymbolRange{Name: tc.name, StartLine: tc.start, EndLine: tc.end}
			got := ClassifyLink(sym, hunks)
			if got != tc.want {
				t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// Test: OrderHistory with mixed strengths and dates (full integration of both functions).
func TestOrderHistoryFullMixed(t *testing.T) {
	entries := []HistoryEntry{
		{SymbolName: "S1", CommitHash: "w1", LinkStrength: LinkWeak, CommitDate: "2024-03-01"},
		{SymbolName: "S1", CommitHash: "s1", LinkStrength: LinkStrong, CommitDate: "2024-01-15"},
		{SymbolName: "S1", CommitHash: "m1", LinkStrength: LinkMedium, CommitDate: "2024-02-01"},
		{SymbolName: "S1", CommitHash: "s2", LinkStrength: LinkStrong, CommitDate: "2024-03-01"},
		{SymbolName: "S1", CommitHash: "m2", LinkStrength: LinkMedium, CommitDate: "2024-03-15"},
		{SymbolName: "S1", CommitHash: "w2", LinkStrength: LinkWeak, CommitDate: "2024-04-01"},
	}
	got := OrderHistory(entries)
	expectedHashes := []string{"s2", "s1", "m2", "m1", "w2", "w1"}
	for i, want := range expectedHashes {
		if got[i].CommitHash != want {
			t.Errorf("position %d: got %s, want %s", i, got[i].CommitHash, want)
		}
	}
}
