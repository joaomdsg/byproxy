package enumerate

import (
	"fmt"
	"regexp"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
)

// Template is a pre-verified .scm skeleton with named {{holes}}. Recon (the model)
// fills PARAMETERS, never syntax — this makes derivation a forced choice over concrete
// values (the machine's native move) instead of open-ended query authoring, and the
// substituted result is still NewQuery-gated. Holes are predicate-string values only
// (regexes/literals that land inside "..."): a hole can never alter query STRUCTURE.
type Template struct {
	ID        string
	Lang      string   // language name this template targets
	SCM       string   // .scm text containing {{hole}} markers, each inside a "..." literal
	Holes     []string // required hole names
	Mechanism string
	Anchor    string
}

// LensSpec is what Recon emits per derived lens: a template fill (tier 1) or free-form
// .scm (tier 2), plus the witness snippets that gate it before it is trusted.
type LensSpec struct {
	ID        string
	Template  string            // template ID; "" => free-form
	Params    map[string]string // hole -> value (template mode)
	FreeSCM   string            // free-form .scm (tier 2)
	Mechanism string
	Anchor    string
	Positive  string // snippet the lens MUST match
	Negative  string // snippet the lens MUST NOT match
}

var holeRE = regexp.MustCompile(`\{\{(\w+)\}\}`)

// Fill substitutes params into the template. Each value is escaped so it is inert
// inside the surrounding "..." predicate string — a stray quote, backslash, or newline
// cannot break out and change the query. Unknown or missing holes are an error.
func (t *Template) Fill(params map[string]string) (string, error) {
	need := make(map[string]bool, len(t.Holes))
	for _, h := range t.Holes {
		need[h] = true
	}
	for k := range params {
		if !need[k] {
			return "", fmt.Errorf("template %q: unknown hole %q", t.ID, k)
		}
	}
	var missing []string
	for _, h := range t.Holes {
		if _, ok := params[h]; !ok {
			missing = append(missing, h)
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("template %q: missing holes %v", t.ID, missing)
	}
	return holeRE.ReplaceAllStringFunc(t.SCM, func(m string) string {
		name := holeRE.FindStringSubmatch(m)[1]
		return escapeSCMString(params[name])
	}), nil
}

// escapeSCMString escapes a value for safe inclusion inside a .scm double-quoted string.
func escapeSCMString(v string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`).Replace(v)
}

// Build compiles the template-filled query into a QueryLens.
func (t *Template) Build(id string, params map[string]string, lang *gts.Language) (*QueryLens, error) {
	scm, err := t.Fill(params)
	if err != nil {
		return nil, err
	}
	mech, anchor := t.Mechanism, t.Anchor
	return NewQueryLens(id, scm, mech, anchor, lang)
}
