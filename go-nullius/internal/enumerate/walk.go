package enumerate

import gts "github.com/odvcencio/gotreesitter"

// walkNamed calls fn for every named node reachable from root, depth-first. Named
// nodes skip punctuation/anonymous tokens, which is what structural lenses care about.
func walkNamed(root *gts.Node, fn func(n *gts.Node)) {
	if root == nil {
		return
	}
	fn(root)
	for i := 0; i < root.NamedChildCount(); i++ {
		walkNamed(root.NamedChild(i), fn)
	}
}

// countNamed returns the number of named nodes in a parsed file (denominator for the
// selectivity ceiling).
func countNamed(f *ParsedFile) int {
	n := 0
	walkNamed(f.Tree.RootNode(), func(*gts.Node) { n++ })
	return n
}

// enclosingFn returns the name of the smallest NAMED function/method that contains node,
// best-effort. Closures (RoleClosure) are scope boundaries but anonymous, so they are
// skipped for naming — a statement inside a closure reports the nearest named function
// that encloses the closure. Node kinds and the name field come from the LangProfile;
// no profile (or no name) yields "".
func enclosingFn(f *ParsedFile, node *gts.Node) string {
	if node == nil || f.Profile == nil {
		return ""
	}
	target := node.StartByte()
	best := ""
	bestSpan := ^uint32(0)
	walkNamed(f.Tree.RootNode(), func(n *gts.Node) {
		if !f.Profile.Is(RoleFunction, n.Type(f.Lang)) {
			return
		}
		if n.StartByte() <= target && target < n.EndByte() {
			if span := n.EndByte() - n.StartByte(); span < bestSpan {
				if name := nodeText(f.Src, n.ChildByFieldName(f.Profile.nameField, f.Lang)); name != "" {
					bestSpan = span
					best = name
				}
			}
		}
	})
	return best
}

// StmtAfterReturn is a WalkLens body: it flags any statement that is a later sibling of a
// return within the same statement sequence — unreachable code. This is a pure
// statement-ORDER property (sibling index ordering) that tree-sitter queries cannot
// express — the demonstrator for the walk backend. It is closure-safe by construction:
// each RoleStmtList is processed independently, so a return inside a closure cannot mark
// statements in an enclosing scope. Role-driven, so language-agnostic given a profile.
func StmtAfterReturn(f *ParsedFile) []Candidate {
	if f.Profile == nil {
		return nil
	}
	var out []Candidate
	walkNamed(f.Tree.RootNode(), func(n *gts.Node) {
		if !f.Profile.Is(RoleStmtList, n.Type(f.Lang)) {
			return
		}
		seenReturn := false
		for i := 0; i < n.NamedChildCount(); i++ {
			c := n.NamedChild(i)
			kind := c.Type(f.Lang)
			if seenReturn && kind != "comment" {
				out = append(out, Candidate{
					File:    f.Path,
					Line:    nodeLine(c),
					Fn:      enclosingFn(f, c),
					Snippet: nodeText(f.Src, c),
					Facts:   map[string]string{"unreachable": nodeText(f.Src, c)},
				})
			}
			if f.Profile.Is(RoleReturn, kind) {
				seenReturn = true
			}
		}
	})
	return out
}
