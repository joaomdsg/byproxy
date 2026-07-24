package enumerate

import "testing"

func TestCandidateOnLensFallbackToAnchor(t *testing.T) {
	c := Candidate{Line: 10} // no Evidence → gated to the anchor line only
	if !c.OnLens(10) {
		t.Error("anchor line must be on-lens")
	}
	if c.OnLens(9) || c.OnLens(11) {
		t.Error("without Evidence, only the anchor line is on-lens")
	}
}

func TestCandidateOnLensEvidenceSet(t *testing.T) {
	c := Candidate{Line: 10, Evidence: []int{10, 11, 25}} // e.g. an order lens: clear@10-11, use@25
	for _, l := range []int{10, 11, 25} {
		if !c.OnLens(l) {
			t.Errorf("line %d in Evidence must be on-lens", l)
		}
	}
	if c.OnLens(12) {
		t.Error("a line NOT in Evidence must be off-lens (the crypto.go FP shape)")
	}
}

func TestWalkLensesPopulateEvidence(t *testing.T) {
	f := parseSrc(t, "package p\n\nfunc s(r []int) bool {\n\treturn len(r) >= 0\n}\n")
	cs := BoolTautology(f)
	if len(cs) != 1 {
		t.Fatalf("want 1 tautology candidate, got %d", len(cs))
	}
	if !cs[0].OnLens(cs[0].Line) {
		t.Errorf("candidate must implicate its own anchor line via Evidence: %+v", cs[0])
	}
	if len(cs[0].Evidence) == 0 {
		t.Errorf("walk candidate must carry Evidence")
	}
}
