package enumerate

import (
	"fmt"

	gts "github.com/odvcencio/gotreesitter"
)

// DefaultMaxHitRatio is the selectivity ceiling applied to DERIVED lenses at Run time:
// a derived lens matching more than this fraction of a file's named nodes is treated as
// a non-discriminating flood and dropped on that file (with a recorded Note). Baseline
// lenses are exempt — the coverage floor is never subtracted.
const DefaultMaxHitRatio = 0.10

// Witness holds the gate inputs for a derived lens.
type Witness struct {
	Positive string // snippet the lens MUST match (>=1 candidate) — proves it is not vacuous/over-narrow
	Negative string // snippet the lens MUST NOT match (0 candidates) — proves it discriminates (over-broad guard)
}

// Gate runs the witness gate for a compiled lens. It returns nil if the lens is
// admissible, or an error naming the failure. This is rule-5 applied to lenses: a lens
// that cannot match its own positive snippet is vacuous; one that also matches its
// negative (benign) snippet does not discriminate. Both un-gated-by-compile failure
// modes become caught here.
func Gate(l Lens, w Witness, lang *gts.Language, langName string) error {
	if w.Positive == "" || w.Negative == "" {
		return fmt.Errorf("witness: lens %q needs both a positive and a negative snippet", l.ID())
	}
	pf, err := ParseSourceLang([]byte(w.Positive), lang, langName)
	if err != nil {
		return fmt.Errorf("witness: parse positive: %w", err)
	}
	if len(l.Enumerate(pf)) == 0 {
		return fmt.Errorf("witness: lens %q did not match its own positive snippet (vacuous or over-narrow)", l.ID())
	}
	nf, err := ParseSourceLang([]byte(w.Negative), lang, langName)
	if err != nil {
		return fmt.Errorf("witness: parse negative: %w", err)
	}
	if n := len(l.Enumerate(nf)); n > 0 {
		return fmt.Errorf("witness: lens %q matched its negative snippet (%d hits) — not discriminating", l.ID(), n)
	}
	return nil
}
