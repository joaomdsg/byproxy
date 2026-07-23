package enumerate

import "testing"

func parseGo(t *testing.T, src string) *ParsedFile {
	t.Helper()
	lang, name, err := LanguageFor("x.go")
	if err != nil {
		t.Fatal(err)
	}
	f, err := ParseSourceLang([]byte(src), lang, name)
	if err != nil {
		t.Fatal(err)
	}
	f.Path = "x.go"
	return f
}

func TestGoProfileResolves(t *testing.T) {
	f := parseGo(t, "package p\nfunc g(){}\n")
	if f.Profile == nil {
		t.Fatal("Go file has no LangProfile")
	}
	if !f.Profile.Is(RoleReturn, "return_statement") {
		t.Error("return_statement not mapped to RoleReturn")
	}
	if !f.Profile.IsFunctionBoundary("func_literal") {
		t.Error("func_literal must be a function boundary (closure)")
	}
	if !f.Profile.Is(RoleFunction, "func_literal") {
		t.Error("func_literal must be in RoleFunction")
	}
}

func TestProfileForUnknownLangNil(t *testing.T) {
	if profileFor("brainfuck") != nil {
		t.Fatal("unprofiled language should return nil profile")
	}
}

// A return INSIDE a closure must NOT mark statements after the closure (in the outer
// scope) as unreachable — the walk reasons per statement_list, which is the closure
// boundary. This is the trap the profile's closure-awareness exists to avoid.
func TestStmtAfterReturnClosureSafe(t *testing.T) {
	f := parseGo(t, "package p\nfunc outer() {\n\tf := func() { return }\n\tf()\n\tg()\n}\n")
	cs := StmtAfterReturn(f)
	if len(cs) != 0 {
		t.Fatalf("closure return leaked into outer scope: %d flagged: %+v", len(cs), cs)
	}
}

// A return followed by code INSIDE the same closure IS flagged, and attributed to the
// nearest NAMED enclosing function (closures are anonymous, skipped for naming).
func TestStmtAfterReturnInsideClosure(t *testing.T) {
	f := parseGo(t, "package p\nfunc outer() {\n\tf := func() int { return 1\n\t\tx := 2\n\t\t_ = x }\n\t_ = f\n}\n")
	cs := StmtAfterReturn(f)
	if len(cs) != 2 {
		t.Fatalf("want 2 unreachable stmts inside closure, got %d: %+v", len(cs), cs)
	}
	for _, c := range cs {
		if c.Fn != "outer" {
			t.Errorf("unreachable stmt attributed to %q, want nearest named fn outer", c.Fn)
		}
	}
}

func TestWalkLensNilProfileSafe(t *testing.T) {
	// a file with no profile yields no walk candidates, never a panic
	f := &ParsedFile{Path: "x", Src: []byte("x"), Profile: nil}
	if got := StmtAfterReturn(f); got != nil {
		t.Fatalf("nil-profile walk should return nil, got %+v", got)
	}
}
