package enumerate

import gts "github.com/odvcencio/gotreesitter"

// goTemplates are pre-verified .scm SKELETONS for Go — parameterized shapes, not
// defect-specific queries. Recon supplies which names/literals to match; the shape is
// fixed and compile-verified. Each hole sits inside a "..." predicate string, so a hole
// value can never alter query structure (see Template).
var goTemplates = []*Template{
	{
		ID:   "call-with-literal-arg",
		Lang: "go",
		SCM: `(call_expression
			function: (_) @fn
			arguments: (argument_list (_) @arg)
			(#match? @fn "{{fn_regex}}")
			(#match? @arg "{{lit_regex}}")) @call`,
		Holes:     []string{"fn_regex", "lit_regex"},
		Mechanism: "call",
		Anchor:    "call",
	},
	{
		ID:        "call-to",
		Lang:      "go",
		SCM:       `(call_expression function: (_) @fn (#match? @fn "{{fn_regex}}")) @call`,
		Holes:     []string{"fn_regex"},
		Mechanism: "call",
		Anchor:    "call",
	},
	{
		// method-call matches a METHOD invocation by its bare name: @method is the
		// selector's field identifier ("Lock" in `s.mu.Lock()`), NOT the receiver-qualified
		// text. So method_regex is a plain "^Lock$" — the robust way to match a method
		// regardless of receiver, avoiding the call-to @fn confusion weak models fall into.
		ID:        "method-call",
		Lang:      "go",
		SCM:       `(call_expression function: (selector_expression field: (field_identifier) @method) (#match? @method "{{method_regex}}")) @call`,
		Holes:     []string{"method_regex"},
		Mechanism: "call",
		Anchor:    "call",
	},
	{
		// deferred-method-call matches a DEFERRED method call (`defer x.Unlock()`) — for
		// defer-based lifecycle lenses. @method is the bare method name, as in method-call.
		ID:        "deferred-method-call",
		Lang:      "go",
		SCM:       `(defer_statement (call_expression function: (selector_expression field: (field_identifier) @method) (#match? @method "{{method_regex}}"))) @defer`,
		Holes:     []string{"method_regex"},
		Mechanism: "defer",
		Anchor:    "defer",
	},
}

// DefaultRegistry returns a registry seeded with the Go template library and the Go
// baseline coverage floor. The baseline is deliberately small — one walk lens. Confirmed
// derived lenses grow the coverage floor NOT by mutating this baseline but via the on-disk
// lens library (machine.LensStore): they are re-seeded each run as always-on derived
// lenses, kept under witness-gating + the selectivity ceiling because they are
// model-authored (baseline lenses are pre-verified and exempt from both).
func DefaultRegistry() *Registry {
	r := NewRegistry()
	for _, t := range goTemplates {
		r.RegisterTemplate(t)
	}
	r.RegisterBaseline("go", "stmt-after-return", func(*gts.Language) (Lens, error) {
		return NewWalkLens("stmt-after-return", "unreachable", StmtAfterReturn), nil
	})
	// bool-tautology: constant-by-construction comparisons (len(x)>=0 always-true guards,
	// identical-operand comparisons). Task-agnostic class coverage; catches always-true
	// subscription predicates like the vialite over-wake defect.
	r.RegisterBaseline("go", "bool-tautology", func(*gts.Language) (Lens, error) {
		return NewWalkLens("bool-tautology", "tautology", BoolTautology), nil
	})
	return r
}
