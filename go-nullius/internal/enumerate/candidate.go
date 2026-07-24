package enumerate

import "go-nullius/internal/ledger"

// Candidate is a pre-judgment enumeration hit — a site a lens matched. Recall is
// mechanical: a Candidate is only ever built from a node that exists in source read
// from disk, so the enumerator cannot fabricate one. Precision (is this actually a
// defect?) is a downstream forced choice; the Candidate carries the mechanical facts
// that choice is made against.
type Candidate struct {
	Lens      string // ID of the lens that produced this hit
	Mechanism string // token the corroborate decisive-line check expects (witness (c))
	File      string
	Line      int               // 1-indexed line of the anchor node
	Fn        string            // enclosing function name, best-effort ("" if unknown)
	Snippet   string            // anchor node text, verbatim from disk
	Facts     map[string]string // capture name -> captured text (mechanical facts for the judge)
	Evidence  []int             // 1-indexed lines this lens implicates — the decisive-line
	// coherence gate (Corroborate filter 1) requires the judge's decisive_line to fall in
	// this set, so a DEFECT ruling must be ABOUT the flagged site, not an unrelated line in
	// the same function (guards the promotion library against off-lens confirms).
}

// OnLens reports whether line is one this candidate implicates — the decisive-line coherence
// check. Falls back to the anchor line when Evidence is unset, so a candidate that predates
// evidence population is still gated to its own site rather than the whole function.
func (c Candidate) OnLens(line int) bool {
	if len(c.Evidence) == 0 {
		return line == c.Line
	}
	for _, l := range c.Evidence {
		if l == line {
			return true
		}
	}
	return false
}

// ToFinding projects a Candidate into the ledger's Finding shape. Verdict is left
// empty — judgment fills it.
func (c Candidate) ToFinding() ledger.Finding {
	head := c.Snippet
	if len(head) > 120 {
		head = head[:120]
	}
	return ledger.Finding{
		File:        c.File,
		Line:        c.Line,
		Lens:        c.Lens,
		Fn:          c.Fn,
		SnippetHead: head,
	}
}
