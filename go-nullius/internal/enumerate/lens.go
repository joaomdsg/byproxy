package enumerate

import (
	"fmt"

	gts "github.com/odvcencio/gotreesitter"
)

// Lens enumerates candidate sites in a parsed file. A lens NEVER judges; it only
// finds. Two backends implement it: QueryLens (.scm shape queries) and WalkLens
// (Go-side CST walks for the order/lifecycle properties queries cannot express).
type Lens interface {
	ID() string
	Mechanism() string
	Enumerate(f *ParsedFile) []Candidate
}

// ---- QueryLens: tree-sitter .scm SHAPE backend ----

// QueryLens wraps a compiled .scm query. Predicates (#match?, #eq?, #any-of?, ...) are
// evaluated by the query engine, so Execute yields only matches that satisfy them.
type QueryLens struct {
	id        string
	mechanism string
	anchor    string // capture name used as the candidate anchor; "" => first capture
	query     *gts.Query
}

// NewQueryLens compiles a .scm query for lang. A malformed query — bad syntax, an
// unknown node kind, an unknown field name — errors HERE (the compile-time gate); the
// caller drops the lens or retries with the error appended.
func NewQueryLens(id, scm, mechanism, anchor string, lang *gts.Language) (*QueryLens, error) {
	q, err := gts.NewQuery(scm, lang)
	if err != nil {
		return nil, fmt.Errorf("lens %q: %w", id, err)
	}
	return &QueryLens{id: id, mechanism: mechanism, anchor: anchor, query: q}, nil
}

func (l *QueryLens) ID() string        { return l.id }
func (l *QueryLens) Mechanism() string { return l.mechanism }

func (l *QueryLens) Enumerate(f *ParsedFile) []Candidate {
	matches := l.query.Execute(f.Tree)
	out := make([]Candidate, 0, len(matches))
	for _, m := range matches {
		anchor := anchorNode(m, l.anchor)
		if anchor == nil {
			continue
		}
		facts := make(map[string]string, len(m.Captures))
		ev := map[int]bool{}
		for _, c := range m.Captures {
			facts[c.Name] = nodeText(f.Src, c.Node)
			for _, l := range spanLines(c.Node) {
				ev[l] = true
			}
		}
		for _, l := range spanLines(anchor) {
			ev[l] = true
		}
		out = append(out, Candidate{
			Lens:      l.id,
			Mechanism: l.mechanism,
			File:      f.Path,
			Line:      nodeLine(anchor),
			Fn:        enclosingFn(f, anchor),
			Snippet:   nodeText(f.Src, anchor),
			Facts:     facts,
			Evidence:  sortedKeys(ev),
		})
	}
	return out
}

func anchorNode(m gts.QueryMatch, anchor string) *gts.Node {
	if anchor != "" {
		for _, c := range m.Captures {
			if c.Name == anchor {
				return c.Node
			}
		}
		return nil
	}
	if len(m.Captures) > 0 {
		return m.Captures[0].Node
	}
	return nil
}

// ---- WalkLens: Go-side CST walk backend (ORDER / LIFECYCLE) ----

// WalkLens wraps a Go function that walks the CST. Use it for properties .scm cannot
// express: statement order, before/after, dominance, lifecycle. The walk body sets the
// mechanical facts; WalkLens stamps the ID and mechanism if the body left them blank.
type WalkLens struct {
	id        string
	mechanism string
	fn        func(f *ParsedFile) []Candidate
}

func NewWalkLens(id, mechanism string, fn func(f *ParsedFile) []Candidate) *WalkLens {
	return &WalkLens{id: id, mechanism: mechanism, fn: fn}
}

func (l *WalkLens) ID() string        { return l.id }
func (l *WalkLens) Mechanism() string { return l.mechanism }

func (l *WalkLens) Enumerate(f *ParsedFile) []Candidate {
	cs := l.fn(f)
	for i := range cs {
		if cs[i].Lens == "" {
			cs[i].Lens = l.id
		}
		if cs[i].Mechanism == "" {
			cs[i].Mechanism = l.mechanism
		}
	}
	return cs
}
