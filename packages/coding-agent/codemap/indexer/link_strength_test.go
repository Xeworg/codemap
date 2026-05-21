package indexer

import (
	"fmt"
	"testing"
)

// RED-phase tests for link_strength classifier and ordering.
// Types (LinkStrength, CommitHunk, SymbolRange, HistoryEntry) and
// functions (ClassifyLink, OrderHistory, parseDate, abs) are defined
// in history_linker.go.

func TestLinkStrengthIsValid(t *testing.T) {
	tests := []struct {
		ls     LinkStrength
		expect bool
	}{
		{LinkStrong, true},
		{LinkMedium, true},
		{LinkWeak, true},
		{LinkStrength(""), false},
		{LinkStrength("stronger"), false},
		{LinkStrength("STRONG"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.ls), func(t *testing.T) {
			if got := tt.ls.IsValid(); got != tt.expect {
				t.Errorf("LinkStrength(%q).IsValid() = %v, want %v", tt.ls, got, tt.expect)
			}
		})
	}
}

func TestClassifyLinkStrong(t *testing.T) {
	sym := SymbolRange{Name: "Foo", StartLine: 10, EndLine: 20}
	hunks := []CommitHunk{{StartLine: 5, EndLine: 15}}
	got := ClassifyLink(sym, hunks)
	if got != LinkStrong {
		t.Errorf("strong: got %v, want %v", got, LinkStrong)
	}
}

func TestClassifyLinkStrongMultipleHunks(t *testing.T) {
	sym := SymbolRange{Name: "Foo", StartLine: 50, EndLine: 60}
	hunks := []CommitHunk{
		{StartLine: 1, EndLine: 10},
		{StartLine: 55, EndLine: 65}, // intersects symbol
	}
	got := ClassifyLink(sym, hunks)
	if got != LinkStrong {
		t.Errorf("strong (multi-hunk): got %v, want %v", got, LinkStrong)
	}
}

func TestClassifyLinkMediumProximity(t *testing.T) {
	// Symbol at 100-110; hunk at 40-50 → center distance exceeds window → weak
	sym := SymbolRange{Name: "Foo", StartLine: 100, EndLine: 110}
	hunks := []CommitHunk{{StartLine: 40, EndLine: 50}}
	got := ClassifyLink(sym, hunks)
	if got != LinkWeak {
		t.Errorf("weak (far proximity): got %v, want %v", got, LinkWeak)
	}

	// Within proximity window: hunk center ~88 vs sym center ~105, diff=17 ≤ 30 → medium
	sym2 := SymbolRange{Name: "Foo", StartLine: 100, EndLine: 110}
	hunks2 := []CommitHunk{{StartLine: 81, EndLine: 95}}
	got2 := ClassifyLink(sym2, hunks2)
	if got2 != LinkMedium {
		t.Errorf("medium: got %v, want %v", got2, LinkMedium)
	}
}

func TestClassifyLinkWeakNoOverlap(t *testing.T) {
	sym := SymbolRange{Name: "Foo", StartLine: 100, EndLine: 110}
	hunks := []CommitHunk{{StartLine: 1, EndLine: 10}}
	got := ClassifyLink(sym, hunks)
	if got != LinkWeak {
		t.Errorf("weak: got %v, want %v", got, LinkWeak)
	}
}

func TestClassifyLinkWeakEmptyHunks(t *testing.T) {
	sym := SymbolRange{Name: "Foo", StartLine: 10, EndLine: 20}
	got := ClassifyLink(sym, nil)
	if got != LinkWeak {
		t.Errorf("weak (nil hunks): got %v, want %v", got, LinkWeak)
	}
	got2 := ClassifyLink(sym, []CommitHunk{})
	if got2 != LinkWeak {
		t.Errorf("weak (empty hunks): got %v, want %v", got2, LinkWeak)
	}
}

func TestClassifyLinkEdgeAtBoundary(t *testing.T) {
	// Symbol 10-20; hunk ends at 10 (exclusive) → no intersection
	sym := SymbolRange{Name: "Foo", StartLine: 10, EndLine: 20}
	hunks := []CommitHunk{{StartLine: 1, EndLine: 10}}
	got := ClassifyLink(sym, hunks)
	// No intersection; proximity: hunk center=5, sym center=15, diff=10 ≤ 30 → medium
	if got != LinkMedium {
		t.Errorf("boundary (end==start): got %v, want %v", got, LinkMedium)
	}

	// Symbol 10-20; hunk starts at 20 → no intersection (exclusive)
	hunks2 := []CommitHunk{{StartLine: 20, EndLine: 30}}
	got2 := ClassifyLink(sym, hunks2)
	// hunk center=25, sym center=15, diff=10 ≤ 30 → medium
	if got2 != LinkMedium {
		t.Errorf("boundary (start==end): got %v, want %v", got2, LinkMedium)
	}
}

func TestClassifyLinkAllEnumValues(t *testing.T) {
	sym := SymbolRange{Name: "X", StartLine: 50, EndLine: 60}
	tests := []struct {
		hunks []CommitHunk
		want  LinkStrength
	}{
		// Direct intersection → strong
		{[]CommitHunk{{StartLine: 55, EndLine: 65}}, LinkStrong},
		{[]CommitHunk{{StartLine: 55, EndLine: 56}}, LinkStrong}, // 1-line hunk intersects
		// Near but no intersection; center diff = |34.5-55| = 20.5 ≤ 30 → medium
		{[]CommitHunk{{StartLine: 20, EndLine: 49}}, LinkMedium},
		// Far away → weak
		{[]CommitHunk{{StartLine: 1, EndLine: 10}}, LinkWeak},  // center diff ≈ 49.5 > 30
		{[]CommitHunk{{StartLine: 91, EndLine: 99}}, LinkWeak}, // center diff = 40 > 30
		// Exactly at boundary (exclusive) → medium (center diff = 10 ≤ 30)
		{[]CommitHunk{{StartLine: 60, EndLine: 70}}, LinkMedium},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("case%d", i), func(t *testing.T) {
			got := ClassifyLink(sym, tt.hunks)
			if got != tt.want {
				t.Errorf("case %d: got %v, want %v", i, got, tt.want)
			}
		})
	}
}

func TestOrderHistoryStrengthDesc(t *testing.T) {
	entries := []HistoryEntry{
		{SymbolName: "Foo", CommitHash: "c1", LinkStrength: LinkWeak, CommitDate: "2024-01-01"},
		{SymbolName: "Foo", CommitHash: "c2", LinkStrength: LinkStrong, CommitDate: "2024-01-01"},
		{SymbolName: "Foo", CommitHash: "c3", LinkStrength: LinkMedium, CommitDate: "2024-01-01"},
	}
	got := OrderHistory(entries)
	want := []LinkStrength{LinkStrong, LinkMedium, LinkWeak}
	for i, e := range got {
		if e.LinkStrength != want[i] {
			t.Errorf("position %d: got %v, want %v", i, e.LinkStrength, want[i])
		}
	}
}

func TestOrderHistoryRecencyDesc(t *testing.T) {
	entries := []HistoryEntry{
		{SymbolName: "Foo", CommitHash: "c1", LinkStrength: LinkStrong, CommitDate: "2024-01-01"},
		{SymbolName: "Foo", CommitHash: "c2", LinkStrength: LinkStrong, CommitDate: "2024-03-01"},
	}
	got := OrderHistory(entries)
	if got[0].CommitHash != "c2" || got[1].CommitHash != "c1" {
		t.Errorf("recency ordering wrong: got %v, want [c2, c1]", commitHashes(got))
	}
}

func TestOrderHistoryStable(t *testing.T) {
	entries := []HistoryEntry{
		{SymbolName: "Foo", CommitHash: "c1", LinkStrength: LinkStrong, CommitDate: "2024-01-01"},
		{SymbolName: "Foo", CommitHash: "c2", LinkStrength: LinkStrong, CommitDate: "2024-01-01"},
	}
	got := OrderHistory(entries)
	if got[0].CommitHash != "c1" || got[1].CommitHash != "c2" {
		t.Errorf("stability broken: got %v", commitHashes(got))
	}
}

// TestOrderHistoryStableThreePlus catches the O(n²) insertion-sort instability bug
// where equal-strength/date elements can be reordered when a higher-priority element
// is swapped into their prefix. Counter-example: [A,B,C] all same strength/date,
// plus a later element D with same strength but newer date. After sort, A and B
// must keep their original relative order (stable), and D comes first.
func TestOrderHistoryStableThreePlus(t *testing.T) {
	// Input order: A(strong,2024-01-01), B(strong,2024-01-01), C(strong,2024-01-01)
	// D(strong,2024-02-01) gets inserted between B and C by recency.
	// Stable sort must preserve A before B before C.
	entries := []HistoryEntry{
		{SymbolName: "Foo", CommitHash: "a1", LinkStrength: LinkStrong, CommitDate: "2024-01-01"},
		{SymbolName: "Foo", CommitHash: "b2", LinkStrength: LinkStrong, CommitDate: "2024-01-01"},
		{SymbolName: "Foo", CommitHash: "c3", LinkStrength: LinkStrong, CommitDate: "2024-01-01"},
		{SymbolName: "Foo", CommitHash: "d4", LinkStrength: LinkStrong, CommitDate: "2024-02-01"}, // newest first
	}
	got := OrderHistory(entries)
	// d4 must be first (newest date among strong), rest in original relative order.
	if got[0].CommitHash != "d4" {
		t.Errorf("d4 should be first (newest date), got %v", got[0].CommitHash)
	}
	// Remaining entries must maintain original relative order (a1 < b2 < c3).
	remaining := commitHashes(got[1:])
	wantRemaining := []string{"a1", "b2", "c3"}
	for i, w := range wantRemaining {
		if remaining[i] != w {
			t.Errorf("position %d: got %v, want %v (stable order broken)", i, remaining[i], w)
		}
	}
}

func commitHashes(entries []HistoryEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.CommitHash
	}
	return out
}
