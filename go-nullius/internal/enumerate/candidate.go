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
