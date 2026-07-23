package enumerate

// Role is an abstract structural concept a walk lens reasons about, decoupled from any
// grammar's concrete node names. A LangProfile maps each Role to the concrete node kinds
// that realize it in one language, so a SINGLE walk body runs across languages by
// swapping the profile — the walk-backend analogue of what the query engine already does
// for .scm shape lenses.
//
// Bootstrap note: tree-sitter's community normalization layer (tags.scm) would populate
// the definition/reference roles (Function, Call) for free, but gotreesitter v0.47.0
// ships EMPTY tags queries for every language (highlight queries are populated but too
// coarse to distinguish control-flow roles). So profiles are hand-written for now; when
// more languages are added, revisit vendoring upstream tags.scm vs deriving from
// highlights. Go-only today.
type Role int

const (
	RoleFunction Role = iota // any function/method/closure boundary (a scope owner)
	RoleClosure              // an anonymous function literal specifically (subset of Function)
	RoleBody                 // a brace/indented body block
	RoleStmtList             // the ordered statement sequence inside a body
	RoleReturn               // a return statement
	RoleLoop                 // a loop construct
	RoleCall                 // a call expression
	RoleDefer                // deferred / finally-scheduled execution
)

// LangProfile is a per-language role map. It carries no grammar objects — just node-kind
// strings and field names — so it is cheap, immutable, and testable in isolation.
type LangProfile struct {
	Name      string
	nameField string // field holding a function/method's name (for enclosingFn); "" if none
	kinds     map[Role]map[string]bool
}

func newProfile(name, nameField string, roles map[Role][]string) *LangProfile {
	p := &LangProfile{Name: name, nameField: nameField, kinds: map[Role]map[string]bool{}}
	for r, ks := range roles {
		m := make(map[string]bool, len(ks))
		for _, k := range ks {
			m[k] = true
		}
		p.kinds[r] = m
	}
	return p
}

// Is reports whether a node kind realizes a role in this language.
func (p *LangProfile) Is(r Role, kind string) bool {
	m := p.kinds[r]
	return m != nil && m[kind]
}

// IsFunctionBoundary reports whether a kind opens a new function scope — a named
// declaration OR a closure. Walks that reason about "same function scope" must stop
// here, so a return/defer inside a closure is never attributed to the enclosing function.
func (p *LangProfile) IsFunctionBoundary(kind string) bool {
	return p.Is(RoleFunction, kind) || p.Is(RoleClosure, kind)
}

// goProfile — hand-written Go role map. func_literal is BOTH a Function boundary and a
// Closure (Go's anonymous functions); it has no name field, so enclosingFn skips it when
// naming but still treats it as a scope boundary.
var goProfile = newProfile("go", "name", map[Role][]string{
	RoleFunction: {"function_declaration", "method_declaration", "func_literal"},
	RoleClosure:  {"func_literal"},
	RoleBody:     {"block"},
	RoleStmtList: {"statement_list"},
	RoleReturn:   {"return_statement"},
	RoleLoop:     {"for_statement"},
	RoleCall:     {"call_expression"},
	RoleDefer:    {"defer_statement"},
})

// profileFor returns the role map for a language, or nil if none is defined. Walk lenses
// require a profile; query lenses do not (they are grammar-specific by construction).
func profileFor(langName string) *LangProfile {
	switch normLang(langName) {
	case "go":
		return goProfile
	default:
		return nil
	}
}
