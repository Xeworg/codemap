package cli

import (
	"testing"
)

func TestParseGraphQuery_Basic(t *testing.T) {
	tests := []struct {
		prompt    string
		wantSym   string
		wantDepth int
	}{
		{"what affects Add", "Add", 3},
		{"what uses Add", "Add", 3},
		{"who calls Add", "Add", 3},
		{"impact of Add", "Add", 3},
		{"blast radius of Add", "Add", 3},
		{"blast Add", "Add", 3},
		{"Add", "Add", 3},
		{"Add  ", "Add", 3},
		{"Add", "Add", 3},
	}

	for _, tt := range tests {
		t.Run(tt.prompt, func(t *testing.T) {
			intent, err := ParseGraphQuery(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if intent.Symbol != tt.wantSym {
				t.Errorf("symbol: got %q, want %q", intent.Symbol, tt.wantSym)
			}
			if intent.Depth != tt.wantDepth {
				t.Errorf("depth: got %d, want %d", intent.Depth, tt.wantDepth)
			}
			if !intent.Cache {
				t.Error("expected cache=true by default")
			}
		})
	}
}

func TestParseGraphQuery_DepthParsing(t *testing.T) {
	tests := []struct {
		prompt    string
		wantDepth int
		wantSym   string
	}{
		{"what affects Add at depth 5", 5, "Add"},
		{"Add at depth 1", 1, "Add"},
		{"Add at depth 10", 5, "Add"}, // cap
		{"Add at depth 0", 1, "Add"},  // floor
		{"Add", 3, "Add"},             // default depth
	}

	for _, tt := range tests {
		t.Run(tt.prompt, func(t *testing.T) {
			intent, err := ParseGraphQuery(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if intent.Depth != tt.wantDepth {
				t.Errorf("depth: got %d, want %d", intent.Depth, tt.wantDepth)
			}
			if intent.Symbol != tt.wantSym {
				t.Errorf("symbol: got %q, want %q", intent.Symbol, tt.wantSym)
			}
		})
	}
}

func TestParseGraphQuery_EdgeTypes(t *testing.T) {
	tests := []struct {
		prompt      string
		wantEdgeLen int
		wantEdge    string
	}{
		{"who calls Add", 1, "calls"},
		{"what uses Add", 1, "type_use"},
		{"what affects Add", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.prompt, func(t *testing.T) {
			intent, err := ParseGraphQuery(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(intent.EdgeTypes) != tt.wantEdgeLen {
				t.Errorf("edge types len: got %d, want %d", len(intent.EdgeTypes), tt.wantEdgeLen)
			}
			if tt.wantEdgeLen > 0 && intent.EdgeTypes[0] != tt.wantEdge {
				t.Errorf("edge type: got %q, want %q", intent.EdgeTypes[0], tt.wantEdge)
			}
		})
	}
}

func TestParseGraphQuery_Errors(t *testing.T) {
	tests := []struct {
		prompt string
	}{
		{""},
		{"   "},
		{"12345"},
		{"123"},
	}

	for _, tt := range tests {
		t.Run(tt.prompt, func(t *testing.T) {
			_, err := ParseGraphQuery(tt.prompt)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tt.prompt)
			}
		})
	}
}

func TestParseGraphQuery_RawSymbol(t *testing.T) {
	// Raw identifiers are title-cased (except all-caps).
	tests := []struct {
		prompt string
		want   string
	}{
		{"Add", "Add"},
		{"_private", "_private"},
		{"HTTP", "HTTP"},     // all-caps preserved
		{"ABCdef", "Abcdef"}, // not all-caps → title-case
	}
	for _, tt := range tests {
		intent, err := ParseGraphQuery(tt.prompt)
		if err != nil {
			t.Errorf("raw %q: unexpected error: %v", tt.prompt, err)
		}
		if intent.Symbol != tt.want {
			t.Errorf("raw %q: symbol got %q, want %q", tt.prompt, intent.Symbol, tt.want)
		}
	}
}

func TestIsValidIdentifier(t *testing.T) {
	valid := []string{"a", "A", "_", "_abc", "abc123", "ABC", "GetByID"}
	invalid := []string{"", "1abc", "a-b", "a b", "a.b", "a:b"}
	for _, s := range valid {
		if !isValidIdentifier(s) {
			t.Errorf("isValidIdentifier(%q): want true, got false", s)
		}
	}
	for _, s := range invalid {
		if isValidIdentifier(s) {
			t.Errorf("isValidIdentifier(%q): want false, got true", s)
		}
	}
}
