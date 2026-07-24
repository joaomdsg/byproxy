package enumerate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func parseSrc(t *testing.T, src string) *ParsedFile {
	t.Helper()
	p := filepath.Join(t.TempDir(), "a.go")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := ParseFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func linesOf(cs []Candidate) map[int]string {
	m := map[int]string{}
	for _, c := range cs {
		m[c.Line] = c.Facts["tautology"]
	}
	return m
}

func TestBoolTautologyFlagsConstantComparisons(t *testing.T) {
	src := `package p

func subscribed(lastReads []string) bool {
	return len(lastReads) >= 0
}

func f(x int) bool {
	return x == x
}

func g(n int) bool {
	return len(nil) < 0
}

func ok(a, b int) bool {
	return a >= b
}

func alsook(xs []int) bool {
	return len(xs) == 0
}
`
	f := parseSrc(t, src)
	got := linesOf(BoolTautology(f))

	// line 4: len(lastReads) >= 0  → flagged (the over-wake shape)
	if _, ok := got[4]; !ok {
		t.Errorf("len(x) >= 0 must be flagged; got %v", got)
	}
	// line 8: x == x → flagged
	if _, ok := got[8]; !ok {
		t.Errorf("x == x must be flagged; got %v", got)
	}
	// line 12: len(nil) < 0 → flagged (always false)
	if _, ok := got[12]; !ok {
		t.Errorf("len(x) < 0 must be flagged; got %v", got)
	}
	// line 16: a >= b → NOT a tautology
	if _, ok := got[16]; ok {
		t.Errorf("a >= b (distinct operands) must NOT be flagged")
	}
	// line 20: len(xs) == 0 → legitimate emptiness check, NOT a tautology
	if _, ok := got[20]; ok {
		t.Errorf("len(xs) == 0 must NOT be flagged")
	}
}

func TestBoolTautologyRegisteredInBaseline(t *testing.T) {
	f := parseSrc(t, "package p\n\nfunc s(r []int) bool { return len(r) >= 0 }\n")
	base, err := DefaultRegistry().BuildBaseline("go", f.Lang)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, l := range base {
		if l.ID() == "bool-tautology" {
			found = true
			if got := l.Enumerate(f); len(got) != 1 {
				t.Errorf("baseline bool-tautology should flag the always-true guard, got %d", len(got))
			}
		}
	}
	if !found {
		t.Fatalf("bool-tautology not registered in the go baseline floor")
	}
}

func TestBoolTautologyIdenticalOperandStrings(t *testing.T) {
	// guard against a false negative on multi-token identical operands.
	f := parseSrc(t, "package p\nfunc f(m map[string]int) bool { return m[\"k\"] != m[\"k\"] }\n")
	got := BoolTautology(f)
	if len(got) != 1 || !strings.Contains(got[0].Facts["tautology"], "identical operands") {
		t.Fatalf("identical map-index operands must flag; got %+v", got)
	}
}
