package enumerate

import (
	"strings"

	gts "github.com/odvcencio/gotreesitter"
)

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

// BoolTautology is a WalkLens body: it flags comparison expressions whose truth value is
// constant by construction — a defect CLASS (dead/always-true guards) that is task-agnostic
// baseline coverage, not fixture-specific. The catch that motivated it: `len(x) >= 0` (always
// true) as a subscription predicate wakes every client on every change (the vialite over-wake
// miss). Purely syntactic, no type info: len(...) compared to 0 in the always-true / always-
// false directions, and comparisons of textually-identical operands (x==x, i<i). The verdict
// is effectively pre-seeded DEFECT (a constant guard is almost never intended), but Judge still
// rules — the lens only FINDS.
func BoolTautology(f *ParsedFile) []Candidate {
	var out []Candidate
	walkNamed(f.Tree.RootNode(), func(n *gts.Node) {
		if n.Type(f.Lang) != "binary_expression" {
			return
		}
		left := n.ChildByFieldName("left", f.Lang)
		right := n.ChildByFieldName("right", f.Lang)
		op := nodeText(f.Src, n.ChildByFieldName("operator", f.Lang))
		if left == nil || right == nil || op == "" {
			return
		}
		reason := tautologyReason(f, left, op, right)
		if reason == "" {
			return
		}
		out = append(out, Candidate{
			File:     f.Path,
			Line:     nodeLine(n),
			Fn:       enclosingFn(f, n),
			Snippet:  nodeText(f.Src, n),
			Facts:    map[string]string{"tautology": reason},
			Evidence: spanLines(n),
		})
	})
	return out
}

// tautologyReason returns a short reason if (left op right) is a constant comparison, else "".
func tautologyReason(f *ParsedFile, left *gts.Node, op string, right *gts.Node) string {
	// len(...) vs 0 — len is always >= 0, so `>= 0` is always true and `< 0` is always false.
	// Both operand orders (len(x) >= 0 and 0 <= len(x)).
	lenLeft, zeroRight := isLenCall(f, left), isZeroLit(f, right)
	zeroLeft, lenRight := isZeroLit(f, left), isLenCall(f, right)
	if lenLeft && zeroRight {
		switch op {
		case ">=":
			return "len(...) >= 0 is always true"
		case "<":
			return "len(...) < 0 is always false"
		}
	}
	if zeroLeft && lenRight {
		switch op {
		case "<=":
			return "0 <= len(...) is always true"
		case ">":
			return "0 > len(...) is always false"
		}
	}
	// Identical operands: x==x / x<=x / x>=x are always true; x!=x / x<x / x>x always false.
	lt, rt := strings.TrimSpace(nodeText(f.Src, left)), strings.TrimSpace(nodeText(f.Src, right))
	if lt != "" && lt == rt {
		switch op {
		case "==", "<=", ">=", "!=", "<", ">":
			return "comparison of identical operands (" + lt + " " + op + " " + rt + ")"
		}
	}
	return ""
}

func isLenCall(f *ParsedFile, n *gts.Node) bool {
	if n == nil || n.Type(f.Lang) != "call_expression" {
		return false
	}
	return nodeText(f.Src, n.ChildByFieldName("function", f.Lang)) == "len"
}

func isZeroLit(f *ParsedFile, n *gts.Node) bool {
	return n != nil && n.Type(f.Lang) == "int_literal" && strings.TrimSpace(nodeText(f.Src, n)) == "0"
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
					File:     f.Path,
					Line:     nodeLine(c),
					Fn:       enclosingFn(f, c),
					Snippet:  nodeText(f.Src, c),
					Facts:    map[string]string{"unreachable": nodeText(f.Src, c)},
					Evidence: spanLines(c),
				})
			}
			if f.Profile.Is(RoleReturn, kind) {
				seenReturn = true
			}
		}
	})
	return out
}
