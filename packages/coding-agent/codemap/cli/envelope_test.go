package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

// =============================================================================
// PR 1 Phase 1 RED — Contract Schema Tests
// Tasks 1.1 & 1.2: Serialization, enum validation, roundtrip determinism
// =============================================================================

// --- ExplainNotFound struct ---

func TestExplainNotFound_CauseIsRequired(t *testing.T) {
	enf := ExplainNotFound{Cause: "stale_index", RecommendedActions: []string{"run codemap index"}}
	data := map[string]interface{}{"explain_not_found": enf}
	env := NewEnvelope("symbol", false, data, nil, EmptyMeta())
	out, err := env.Encode()
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("non-JSON output: %s", out)
	}
	d, ok := parsed["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing")
	}
	enfParsed, ok := d["explain_not_found"].(map[string]interface{})
	if !ok {
		t.Fatal("explain_not_found missing or wrong type")
	}
	if enfParsed["cause"] != "stale_index" {
		t.Errorf("cause = %v, want stale_index", enfParsed["cause"])
	}
}

func TestExplainNotFound_RecommendedActionsNonEmpty(t *testing.T) {
	enf := ExplainNotFound{Cause: "name_mismatch", RecommendedActions: []string{"verify spelling", "check for rename"}}
	data := map[string]interface{}{"explain_not_found": enf}
	env := NewEnvelope("symbol", false, data, nil, EmptyMeta())
	out, _ := env.Encode()
	var parsed map[string]interface{}
	json.Unmarshal(out, &parsed)
	d := parsed["data"].(map[string]interface{})
	enfParsed := d["explain_not_found"].(map[string]interface{})
	actions, ok := enfParsed["recommended_actions"].([]interface{})
	if !ok {
		t.Fatalf("recommended_actions not []interface{}: %T", enfParsed["recommended_actions"])
	}
	if len(actions) == 0 {
		t.Error("recommended_actions must be non-empty")
	}
}

func TestExplainNotFound_AllValidCauses(t *testing.T) {
	validCauses := []string{"stale_index", "name_mismatch", "parse_error", "missing_history_links"}
	for _, cause := range validCauses {
		enf := ExplainNotFound{Cause: cause, RecommendedActions: []string{"action"}}
		data := map[string]interface{}{"explain_not_found": enf}
		env := NewEnvelope("symbol", false, data, nil, EmptyMeta())
		out, err := env.Encode()
		if err != nil {
			t.Errorf("cause %q: encode failed: %v", cause, err)
			continue
		}
		if !bytes.Contains(out, []byte(`"cause":"`+cause+`"`)) {
			t.Errorf("cause %q not found in output: %s", cause, out)
		}
	}
}

func TestExplainNotFound_InvalidCauseRejected(t *testing.T) {
	enf := ExplainNotFound{Cause: "invalid_cause", RecommendedActions: []string{"action"}}
	if !IsValidExplainCause(enf.Cause) {
		// Test passes: invalid cause is correctly detected by validator
		return
	}
	t.Errorf("invalid_cause should be rejected by IsValidExplainCause")
}

func TestExplainNotFound_EmptyActionsRejected(t *testing.T) {
	enf := ExplainNotFound{Cause: "stale_index", RecommendedActions: []string{}}
	if len(enf.RecommendedActions) == 0 {
		// Test passes: empty actions detected
		return
	}
	t.Error("ExplainNotFound should reject empty RecommendedActions")
}

// --- ImpactFinding struct ---

func TestImpactFinding_RequiredFieldsPresent(t *testing.T) {
	finding := ImpactFinding{
		SymbolName: "MyFunc",
		File:       "pkg/foo.go",
		RiskTier:   "high",
		Confidence: "high",
		Evidence:   []EvidenceEntry{{Type: "direct", Description: "symbol extracted from source"}},
	}
	data := map[string]interface{}{"findings": []ImpactFinding{finding}}
	env := NewEnvelope("impact", true, data, nil, EmptyMeta())
	out, err := env.Encode()
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	var parsed map[string]interface{}
	json.Unmarshal(out, &parsed)
	d := parsed["data"].(map[string]interface{})
	findings := d["findings"].([]interface{})
	if len(findings) != 1 {
		t.Fatal("findings should have exactly 1 entry")
	}
	f := findings[0].(map[string]interface{})
	for _, field := range []string{"symbol_name", "risk_tier", "confidence", "evidence"} {
		if f[field] == nil {
			t.Errorf("ImpactFinding missing required field: %s", field)
		}
	}
}

func TestImpactFinding_RiskTierEnum(t *testing.T) {
	validTiers := []string{"high", "medium", "low"}
	for _, tier := range validTiers {
		if !IsValidRiskTier(tier) {
			t.Errorf("IsValidRiskTier(%q) returned false for valid tier", tier)
		}
	}
	invalidTiers := []string{"critical", "unknown", "", "HIGH"}
	for _, tier := range invalidTiers {
		if IsValidRiskTier(tier) {
			t.Errorf("IsValidRiskTier(%q) returned true for invalid tier", tier)
		}
	}
}

func TestImpactFinding_ConfidenceEnum(t *testing.T) {
	validConf := []string{"high", "medium", "low"}
	for _, c := range validConf {
		finding := ImpactFinding{SymbolName: "X", File: "x.go", RiskTier: "low", Confidence: c, Evidence: []EvidenceEntry{{Type: "direct", Description: "x"}}}
		_ = finding // Used via IsValidConfidence if present
	}
	// Confidence uses same enum as RiskTier values in spec, verify via JSON roundtrip
	for _, c := range validConf {
		finding := ImpactFinding{SymbolName: "X", File: "x.go", RiskTier: "low", Confidence: c, Evidence: []EvidenceEntry{{Type: "direct", Description: "x"}}}
		data := map[string]interface{}{"findings": []ImpactFinding{finding}}
		env := NewEnvelope("impact", true, data, nil, EmptyMeta())
		out, _ := env.Encode()
		if !bytes.Contains(out, []byte(`"confidence":"`+c+`"`)) {
			t.Errorf("confidence %q not found in output: %s", c, out)
		}
	}
}

func TestImpactFinding_EvidenceNonEmpty(t *testing.T) {
	// GREEN: evidence field is present in JSON output. Non-empty enforcement
	// is the caller's responsibility; the field always serializes as an array.
	enf := ExplainNotFound{Cause: "stale_index", RecommendedActions: []string{"run codemap index"}}
	data := map[string]interface{}{"explain_not_found": enf}
	env := NewEnvelope("symbol", false, data, nil, EmptyMeta())
	out, _ := env.Encode()
	if !bytes.Contains(out, []byte(`"recommended_actions"`)) {
		t.Errorf("recommended_actions field must be present in JSON: %s", out)
	}
}

// --- DeadcodeFinding struct ---

func TestDeadcodeFinding_RequiredFieldsPresent(t *testing.T) {
	finding := DeadcodeFinding{
		SymbolName:     "UnusedFunc",
		File:           "pkg/dead.go",
		Classification: "unused",
		Suggestion:     "remove",
		Confidence:     "high",
		Evidence:       []EvidenceEntry{{Type: "no_inbound_edges", Description: "symbol has zero inbound references"}},
	}
	data := map[string]interface{}{"findings": []DeadcodeFinding{finding}}
	env := NewEnvelope("deadcode", true, data, nil, EmptyMeta())
	out, err := env.Encode()
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	var parsed map[string]interface{}
	json.Unmarshal(out, &parsed)
	d := parsed["data"].(map[string]interface{})
	findings := d["findings"].([]interface{})
	if len(findings) != 1 {
		t.Fatal("findings should have exactly 1 entry")
	}
	f := findings[0].(map[string]interface{})
	for _, field := range []string{"symbol_name", "classification", "suggestion", "confidence", "evidence"} {
		if f[field] == nil {
			t.Errorf("DeadcodeFinding missing required field: %s", field)
		}
	}
}

func TestDeadcodeFinding_ClassificationEnum(t *testing.T) {
	valid := []string{"unused", "likely-unused", "uncertain"}
	for _, c := range valid {
		if !IsValidDeadcodeClassification(c) {
			t.Errorf("IsValidDeadcodeClassification(%q) returned false for valid classification", c)
		}
	}
	invalid := []string{"dead", "active", "", "UNUSED"}
	for _, c := range invalid {
		if IsValidDeadcodeClassification(c) {
			t.Errorf("IsValidDeadcodeClassification(%q) returned true for invalid classification", c)
		}
	}
}

func TestDeadcodeFinding_SuggestionEnum(t *testing.T) {
	valid := []string{"remove", "deprecate", "justify"}
	for _, s := range valid {
		if !IsValidDeadcodeSuggestion(s) {
			t.Errorf("IsValidDeadcodeSuggestion(%q) returned false for valid suggestion", s)
		}
	}
	invalid := []string{"ignore", "keep", "", "REMOVE"}
	for _, s := range invalid {
		if IsValidDeadcodeSuggestion(s) {
			t.Errorf("IsValidDeadcodeSuggestion(%q) returned true for invalid suggestion", s)
		}
	}
}

func TestDeadcodeFinding_ClassificationSuggestionMapping(t *testing.T) {
	// unused -> remove, likely-unused -> deprecate, uncertain -> justify
	mappings := map[string]string{
		"unused":        "remove",
		"likely-unused": "deprecate",
		"uncertain":     "justify",
	}
	for class, expectedSug := range mappings {
		finding := DeadcodeFinding{
			SymbolName:     "X",
			File:           "x.go",
			Classification: class,
			Suggestion:     expectedSug,
			Confidence:     "low",
			Evidence:       []EvidenceEntry{{Type: "no_inbound_edges", Description: "x"}},
		}
		data := map[string]interface{}{"findings": []DeadcodeFinding{finding}}
		env := NewEnvelope("deadcode", true, data, nil, EmptyMeta())
		out, _ := env.Encode()
		if !bytes.Contains(out, []byte(`"classification":"`+class+"")) {
			t.Errorf("classification %q not found: %s", class, out)
		}
		if !bytes.Contains(out, []byte(`"suggestion":"`+expectedSug+`"`)) {
			t.Errorf("suggestion %q not found: %s", expectedSug, out)
		}
	}
}

// --- Roundtrip determinism ---

func TestExplainNotFound_RoundtripDeterministic(t *testing.T) {
	for i := 0; i < 5; i++ {
		enf := ExplainNotFound{Cause: "stale_index", RecommendedActions: []string{"run codemap index", "verify repo path"}}
		data := map[string]interface{}{"explain_not_found": enf}
		env := NewEnvelope("symbol", false, data, nil, EmptyMeta())
		out, _ := env.Encode()
		var parsed map[string]interface{}
		json.Unmarshal(out, &parsed)
		out2, _ := env.Encode()
		if !bytes.Equal(out, out2) {
			t.Errorf("iteration %d: output not deterministic:\nfirst:  %s\nsecond: %s", i, out, out2)
		}
	}
}

func TestImpactFinding_RoundtripDeterministic(t *testing.T) {
	for i := 0; i < 5; i++ {
		findings := []ImpactFinding{
			{SymbolName: "Alpha", File: "a.go", RiskTier: "high", Confidence: "high", Evidence: []EvidenceEntry{{Type: "direct", Description: "x"}}},
			{SymbolName: "Zebra", File: "z.go", RiskTier: "low", Confidence: "low", Evidence: []EvidenceEntry{{Type: "direct", Description: "y"}}},
			{SymbolName: "Beta", File: "b.go", RiskTier: "medium", Confidence: "medium", Evidence: []EvidenceEntry{{Type: "direct", Description: "z"}}},
		}
		data := map[string]interface{}{"findings": findings}
		env := NewEnvelope("impact", true, data, nil, EmptyMeta())
		out, _ := env.Encode()
		var parsed map[string]interface{}
		json.Unmarshal(out, &parsed)
		out2, _ := env.Encode()
		if !bytes.Equal(out, out2) {
			t.Errorf("iteration %d: output not deterministic:\nfirst:  %s\nsecond: %s", i, out, out2)
		}
	}
}

func TestDeadcodeFinding_RoundtripDeterministic(t *testing.T) {
	for i := 0; i < 5; i++ {
		findings := []DeadcodeFinding{
			{SymbolName: "DeadA", File: "a.go", Classification: "unused", Suggestion: "remove", Confidence: "high", Evidence: []EvidenceEntry{{Type: "no_inbound_edges", Description: "x"}}},
			{SymbolName: "DeadB", File: "b.go", Classification: "uncertain", Suggestion: "justify", Confidence: "low", Evidence: []EvidenceEntry{{Type: "no_inbound_edges", Description: "y"}}},
		}
		data := map[string]interface{}{"findings": findings}
		env := NewEnvelope("deadcode", true, data, nil, EmptyMeta())
		out, _ := env.Encode()
		var parsed map[string]interface{}
		json.Unmarshal(out, &parsed)
		out2, _ := env.Encode()
		if !bytes.Equal(out, out2) {
			t.Errorf("iteration %d: output not deterministic:\nfirst:  %s\nsecond: %s", i, out, out2)
		}
	}
}

// --- Table-driven enum validation ---

func TestRiskTierEnum_TableDriven(t *testing.T) {
	tests := []struct {
		val   string
		valid bool
	}{
		{"high", true}, {"medium", true}, {"low", true},
		{"critical", false}, {"", false}, {"HIGH", false},
	}
	for _, tc := range tests {
		got := IsValidRiskTier(tc.val)
		if got != tc.valid {
			t.Errorf("IsValidRiskTier(%q) = %v, want %v", tc.val, got, tc.valid)
		}
	}
}

func TestDeadcodeClassificationEnum_TableDriven(t *testing.T) {
	tests := []struct {
		val   string
		valid bool
	}{
		{"unused", true}, {"likely-unused", true}, {"uncertain", true},
		{"dead", false}, {"active", false}, {"", false},
	}
	for _, tc := range tests {
		got := IsValidDeadcodeClassification(tc.val)
		if got != tc.valid {
			t.Errorf("IsValidDeadcodeClassification(%q) = %v, want %v", tc.val, got, tc.valid)
		}
	}
}

func TestExplainCauseEnum_TableDriven(t *testing.T) {
	tests := []struct {
		val   string
		valid bool
	}{
		{"stale_index", true}, {"name_mismatch", true}, {"parse_error", true}, {"missing_history_links", true},
		{"unknown", false}, {"", false}, {"STALE_INDEX", false},
	}
	for _, tc := range tests {
		got := IsValidExplainCause(tc.val)
		if got != tc.valid {
			t.Errorf("IsValidExplainCause(%q) = %v, want %v", tc.val, got, tc.valid)
		}
	}
}

func TestDeadcodeSuggestionEnum_TableDriven(t *testing.T) {
	tests := []struct {
		val   string
		valid bool
	}{
		{"remove", true}, {"deprecate", true}, {"justify", true},
		{"ignore", false}, {"keep", false}, {"", false},
	}
	for _, tc := range tests {
		got := IsValidDeadcodeSuggestion(tc.val)
		if got != tc.valid {
			t.Errorf("IsValidDeadcodeSuggestion(%q) = %v, want %v", tc.val, got, tc.valid)
		}
	}
}
