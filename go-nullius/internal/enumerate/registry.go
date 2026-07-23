package enumerate

import (
	"fmt"
	"sort"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
)

// PerLensCap bounds candidates kept per lens per file. Excess is dropped with a
// recorded Note — silent truncation is never allowed (it reads as "covered everything"
// when it didn't).
const PerLensCap = 30

// LensBuilder compiles a lens for a specific language. Query-backed baseline lenses
// need the language at compile time; walk-backed ones ignore it. A builder that errors
// for a BASELINE lens is a program bug (baseline queries are pre-verified), not a
// derivation failure.
type LensBuilder func(lang *gts.Language) (Lens, error)

type namedBuilder struct {
	id    string
	build LensBuilder
}

// Registry holds the always-on baseline lens set per language — the coverage floor
// that model-derived lenses may only ADD to, never subtract from — plus the fillable
// template library.
type Registry struct {
	baseline  map[string][]namedBuilder
	templates map[string]*Template
}

func NewRegistry() *Registry {
	return &Registry{baseline: map[string][]namedBuilder{}, templates: map[string]*Template{}}
}

func normLang(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

// RegisterBaseline adds an always-on lens builder for a language (case-insensitive).
func (r *Registry) RegisterBaseline(langName, id string, b LensBuilder) {
	k := normLang(langName)
	r.baseline[k] = append(r.baseline[k], namedBuilder{id: id, build: b})
}

// RegisterTemplate adds a fillable template.
func (r *Registry) RegisterTemplate(t *Template) { r.templates[t.ID] = t }

// Template returns a registered template by ID.
func (r *Registry) Template(id string) (*Template, bool) { t, ok := r.templates[id]; return t, ok }

// Templates returns the registered templates sorted by ID — used to build the Recon
// derivation prompt so the model knows which pre-verified shapes it may fill.
func (r *Registry) Templates() []*Template {
	out := make([]*Template, 0, len(r.templates))
	for _, t := range r.templates {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// BuildBaseline compiles the always-on lenses for a language. An error means a
// pre-verified baseline lens failed to compile — a program bug, surfaced loudly.
func (r *Registry) BuildBaseline(langName string, lang *gts.Language) ([]Lens, error) {
	var out []Lens
	for _, nb := range r.baseline[normLang(langName)] {
		l, err := nb.build(lang)
		if err != nil {
			return nil, fmt.Errorf("baseline lens %q failed to compile: %w", nb.id, err)
		}
		out = append(out, l)
	}
	return out, nil
}

// LensStatus records the disposition of one derived-lens spec at the witness gate.
type LensStatus struct {
	ID       string
	Accepted bool
	Reason   string // "" if accepted; the DERIVE_FAILED cause otherwise
}

// AcceptDerived compiles and witness-gates each spec. Accepted lenses are returned;
// every rejection is recorded in statuses with its cause and NEVER aborts the batch or
// touches the baseline. A batch where all specs fail degrades to baseline-only — the
// coverage-floor rule, enforced structurally rather than by discipline.
func (r *Registry) AcceptDerived(specs []LensSpec, lang *gts.Language, langName string) (accepted []Lens, statuses []LensStatus) {
	for _, s := range specs {
		l, err := r.buildSpec(s, lang)
		if err != nil {
			statuses = append(statuses, LensStatus{ID: s.ID, Reason: fmt.Sprintf("compile: %v", err)})
			continue
		}
		if err := Gate(l, Witness{Positive: s.Positive, Negative: s.Negative}, lang, langName); err != nil {
			statuses = append(statuses, LensStatus{ID: s.ID, Reason: err.Error()})
			continue
		}
		accepted = append(accepted, l)
		statuses = append(statuses, LensStatus{ID: s.ID, Accepted: true})
	}
	return accepted, statuses
}

func (r *Registry) buildSpec(s LensSpec, lang *gts.Language) (Lens, error) {
	switch {
	case s.Template != "":
		t, ok := r.templates[s.Template]
		if !ok {
			return nil, fmt.Errorf("unknown template %q", s.Template)
		}
		scm, err := t.Fill(s.Params)
		if err != nil {
			return nil, err
		}
		mech, anchor := firstNonEmpty(s.Mechanism, t.Mechanism), firstNonEmpty(s.Anchor, t.Anchor)
		return NewQueryLens(s.ID, scm, mech, anchor, lang)
	case s.FreeSCM != "":
		return NewQueryLens(s.ID, s.FreeSCM, s.Mechanism, s.Anchor, lang)
	default:
		return nil, fmt.Errorf("spec %q: neither a template nor free-form .scm", s.ID)
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// Result is the enumeration output over a file set.
type Result struct {
	Candidates []Candidate
	Notes      []string // truncation / over-broad / parse notes — the audit trail, never silent
}

// Run enumerates the given files. Baseline lenses always apply (capped only by
// PerLensCap); derived lenses additionally face the selectivity ceiling — an over-broad
// derived lens is dropped on a file with a recorded Note. Candidates are returned
// sorted by (file, line) for a stable ledger.
func Run(files []string, baseline, derived []Lens) (Result, error) {
	var res Result
	for _, path := range files {
		f, err := ParseFile(path)
		if err != nil {
			res.Notes = append(res.Notes, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		total := countNamed(f)
		for _, l := range baseline {
			res.Candidates, res.Notes = appendCapped(res.Candidates, res.Notes, path, l, l.Enumerate(f))
		}
		for _, l := range derived {
			hits := l.Enumerate(f)
			if total > 0 && float64(len(hits))/float64(total) > DefaultMaxHitRatio {
				res.Notes = append(res.Notes, fmt.Sprintf("%s: derived lens %q over-broad (%d/%d named nodes), dropped on this file", path, l.ID(), len(hits), total))
				continue
			}
			res.Candidates, res.Notes = appendCapped(res.Candidates, res.Notes, path, l, hits)
		}
	}
	sort.SliceStable(res.Candidates, func(i, j int) bool {
		if res.Candidates[i].File != res.Candidates[j].File {
			return res.Candidates[i].File < res.Candidates[j].File
		}
		return res.Candidates[i].Line < res.Candidates[j].Line
	})
	return res, nil
}

func appendCapped(cands []Candidate, notes []string, path string, l Lens, hits []Candidate) ([]Candidate, []string) {
	if len(hits) > PerLensCap {
		notes = append(notes, fmt.Sprintf("%s: lens %q capped %d→%d", path, l.ID(), len(hits), PerLensCap))
		hits = hits[:PerLensCap]
	}
	return append(cands, hits...), notes
}
