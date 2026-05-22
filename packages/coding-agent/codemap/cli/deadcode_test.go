package cli

import "testing"

func TestClassifyDeadcode_Deterministic(t *testing.T) {
	const runs = 10
	firstClass, firstSuggestion, firstConfidence := classifyDeadcode(0, "func", "privateFunc", "pkg/file.go")
	firstEvidence := deadcodeEvidence(0, "privateFunc", "pkg/file.go")

	for i := 0; i < runs; i++ {
		class, suggestion, confidence := classifyDeadcode(0, "func", "privateFunc", "pkg/file.go")
		evidence := deadcodeEvidence(0, "privateFunc", "pkg/file.go")
		if class != firstClass || suggestion != firstSuggestion || confidence != firstConfidence {
			t.Fatalf("non-deterministic classification at run %d: got (%s,%s,%s), want (%s,%s,%s)", i, class, suggestion, confidence, firstClass, firstSuggestion, firstConfidence)
		}
		if len(evidence) != len(firstEvidence) {
			t.Fatalf("non-deterministic evidence length at run %d: got %d want %d", i, len(evidence), len(firstEvidence))
		}
		for j := range evidence {
			if evidence[j].Type != firstEvidence[j].Type {
				t.Fatalf("non-deterministic evidence[%d] at run %d: got %q want %q", j, i, evidence[j].Type, firstEvidence[j].Type)
			}
		}
	}
}
