package enumerate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- QueryLens ----

func TestQueryLensEnumeratesLiteralArg(t *testing.T) {
	lang, name, err := LanguageFor("x.go")
	if err != nil {
		t.Fatal(err)
	}
	l, err := NewQueryLens("nil-arg",
		`(call_expression
			function: (_) @fn
			arguments: (argument_list (_) @arg)
			(#match? @fn "broadcast")
			(#match? @arg "^nil$")) @call`,
		"call", "call", lang)
	if err != nil {
		t.Fatalf("NewQueryLens: %v", err)
	}
	src := []byte("package p\nfunc g(ctx, key any) { broadcast(ctx, nil, key) }\n")
	f, err := ParseSourceLang(src, lang, name)
	if err != nil {
		t.Fatal(err)
	}
	f.Path = "g.go"
	cs := l.Enumerate(f)
	if len(cs) != 1 {
		t.Fatalf("got %d candidates, want 1: %+v", len(cs), cs)
	}
	if !strings.Contains(cs[0].Snippet, "broadcast(ctx, nil, key)") {
		t.Fatalf("snippet = %q", cs[0].Snippet)
	}
	if cs[0].Facts["arg"] != "nil" {
		t.Fatalf("fact arg = %q, want nil", cs[0].Facts["arg"])
	}
	if cs[0].Fn != "g" {
		t.Fatalf("enclosing fn = %q, want g", cs[0].Fn)
	}
}

func TestNewQueryLensMalformedErrors(t *testing.T) {
	lang, _, _ := LanguageFor("x.go")
	if _, err := NewQueryLens("bad", `(nonexistent_node_kind) @x`, "", "", lang); err == nil {
		t.Fatal("expected compile error for unknown node kind")
	}
	if _, err := NewQueryLens("bad2", `(call_expression`, "", "", lang); err == nil {
		t.Fatal("expected compile error for unbalanced parens")
	}
}

// ---- WalkLens (order backend) ----

func TestWalkLensStmtAfterReturn(t *testing.T) {
	lang, name, _ := LanguageFor("x.go")
	src := []byte("package p\nfunc g() int {\n\treturn 1\n\tx := 2\n\t_ = x\n}\n")
	f, err := ParseSourceLang(src, lang, name)
	if err != nil {
		t.Fatal(err)
	}
	f.Path = "g.go"
	l := NewWalkLens("stmt-after-return", "unreachable", StmtAfterReturn)
	cs := l.Enumerate(f)
	if len(cs) != 2 {
		t.Fatalf("got %d unreachable stmts, want 2: %+v", len(cs), cs)
	}
	if cs[0].Lens != "stmt-after-return" || cs[0].Mechanism != "unreachable" {
		t.Fatalf("walk lens did not stamp id/mechanism: %+v", cs[0])
	}
}

// ---- Template fill + injection safety ----

func TestTemplateFill(t *testing.T) {
	tp := &Template{ID: "t", SCM: `(#match? @fn "{{fn_regex}}")`, Holes: []string{"fn_regex"}}
	out, err := tp.Fill(map[string]string{"fn_regex": "broadcast"})
	if err != nil {
		t.Fatal(err)
	}
	if out != `(#match? @fn "broadcast")` {
		t.Fatalf("fill = %q", out)
	}
}

func TestTemplateFillMissingAndUnknown(t *testing.T) {
	tp := &Template{ID: "t", SCM: `{{a}}`, Holes: []string{"a"}}
	if _, err := tp.Fill(map[string]string{}); err == nil {
		t.Fatal("expected missing-hole error")
	}
	if _, err := tp.Fill(map[string]string{"a": "x", "b": "y"}); err == nil {
		t.Fatal("expected unknown-hole error")
	}
}

func TestTemplateFillInjectionInert(t *testing.T) {
	// A value carrying a quote must be escaped so it stays INSIDE the "..." string and
	// cannot become query structure. Any injected @capture token must appear only as
	// escaped regex text, never as a real capture.
	lang, _, _ := LanguageFor("x.go")
	tp := &Template{
		ID: "call-to", SCM: `(call_expression function: (_) @fn (#match? @fn "{{fn_regex}}")) @call`,
		Holes: []string{"fn_regex"}, Mechanism: "call", Anchor: "call",
	}
	// payload tries to close the string and add an @evil capture; a quote, no regex-
	// metachar imbalance, so the escaped result is a valid (if silly) regex.
	scm, err := tp.Fill(map[string]string{"fn_regex": `foo" @evil bar`})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(scm, `foo" @evil`) {
		t.Fatalf("quote broke out of the string (unescaped): %q", scm)
	}
	if !strings.Contains(scm, `foo\" @evil bar`) {
		t.Fatalf("payload not escaped as expected: %q", scm)
	}
	// Compiles as ONE pattern; @evil is regex text, so the query still has exactly the
	// two intended captures (@fn, @call) and enumerates only real calls.
	l, err := NewQueryLens("inj", scm, "call", "call", lang)
	if err != nil {
		t.Fatalf("escaped query should still compile: %v", err)
	}
	f, _ := ParseSourceLang([]byte("package p\nfunc g(){ h() }\n"), lang, "go")
	if got := l.Enumerate(f); len(got) != 0 {
		t.Fatalf("injected capture leaked into results: %+v", got)
	}
}

// ---- Witness gate ----

func specGood() LensSpec {
	return LensSpec{
		ID: "nil-broadcast", Template: "call-with-literal-arg",
		Params:   map[string]string{"fn_regex": "broadcast", "lit_regex": "^nil$"},
		Positive: "package p\nfunc g(ctx, key any){ broadcast(ctx, nil, key) }\n",
		Negative: "package p\nfunc g(ctx, s, key any){ broadcast(ctx, s, key) }\n",
	}
}

func TestGateAcceptsGood(t *testing.T) {
	lang, name, _ := LanguageFor("x.go")
	r := DefaultRegistry()
	accepted, statuses := r.AcceptDerived([]LensSpec{specGood()}, lang, name)
	if len(accepted) != 1 {
		t.Fatalf("accepted %d, want 1; statuses=%+v", len(accepted), statuses)
	}
	if !statuses[0].Accepted {
		t.Fatalf("status not accepted: %+v", statuses[0])
	}
}

func TestGateRejectsVacuousOverbroadMalformed(t *testing.T) {
	lang, name, _ := LanguageFor("x.go")
	r := DefaultRegistry()

	vacuous := specGood()
	vacuous.ID = "vacuous"
	vacuous.Positive = "package p\nfunc g(ctx, s, key any){ broadcast(ctx, s, key) }\n" // no nil → lens can't match its own positive

	overbroad := specGood()
	overbroad.ID = "overbroad"
	overbroad.Negative = "package p\nfunc g(ctx, key any){ broadcast(ctx, nil, key) }\n" // negative DOES contain the pattern

	malformed := LensSpec{
		ID: "malformed", FreeSCM: `(call_expression`, // unbalanced
		Positive: "package p\n", Negative: "package p\n",
	}

	accepted, statuses := r.AcceptDerived([]LensSpec{vacuous, overbroad, malformed}, lang, name)
	if len(accepted) != 0 {
		t.Fatalf("accepted %d, want 0", len(accepted))
	}
	for _, s := range statuses {
		if s.Accepted || s.Reason == "" {
			t.Fatalf("spec %q should be rejected with a reason: %+v", s.ID, s)
		}
	}
}

// ---- Coverage floor: a failed derivation degrades to baseline-only ----

func TestCoverageFloorHolds(t *testing.T) {
	lang, name, _ := LanguageFor("x.go")
	r := DefaultRegistry()
	base, err := r.BuildBaseline(name, lang)
	if err != nil {
		t.Fatalf("BuildBaseline: %v", err)
	}
	if len(base) == 0 {
		t.Fatal("baseline empty — no coverage floor")
	}
	// every derived spec fails the gate
	bad := LensSpec{ID: "bad", FreeSCM: `(call_expression`, Positive: "package p\n", Negative: "package p\n"}
	derived, statuses := r.AcceptDerived([]LensSpec{bad}, lang, name)
	if len(derived) != 0 {
		t.Fatal("bad spec should not be accepted")
	}
	if statuses[0].Accepted {
		t.Fatal("bad spec marked accepted")
	}
	// baseline is untouched regardless of derived outcome
	base2, _ := r.BuildBaseline(name, lang)
	if len(base2) != len(base) {
		t.Fatalf("baseline changed after failed derivation: %d != %d", len(base2), len(base))
	}
}

// ---- Run: baseline always applies; derived subject to selectivity ceiling ----

func TestRunBaselineAndDerived(t *testing.T) {
	lang, name, _ := LanguageFor("x.go")
	r := DefaultRegistry()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	src := "package p\n\nfunc g(ctx, key any) int {\n\tbroadcast(ctx, nil, key)\n\treturn 1\n\tx := 2\n\t_ = x\n}\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	base, err := r.BuildBaseline(name, lang)
	if err != nil {
		t.Fatal(err)
	}
	derived, _ := r.AcceptDerived([]LensSpec{specGood()}, lang, name)

	res, err := Run([]string{path}, base, derived)
	if err != nil {
		t.Fatal(err)
	}
	var sawUnreachable, sawNil bool
	for _, c := range res.Candidates {
		if c.Lens == "stmt-after-return" {
			sawUnreachable = true
		}
		if c.Lens == "nil-broadcast" {
			sawNil = true
		}
	}
	if !sawUnreachable {
		t.Error("baseline stmt-after-return produced no candidate")
	}
	if !sawNil {
		t.Error("derived nil-broadcast produced no candidate")
	}
}

func TestRunDropsOverbroadDerived(t *testing.T) {
	lang, name, _ := LanguageFor("x.go")
	// a free-form lens matching every identifier — over-broad flood
	flood, err := NewQueryLens("flood", `(identifier) @id`, "id", "id", lang)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "s.go")
	if err := os.WriteFile(path, []byte("package p\nfunc g(){ a := b; _ = a }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run([]string{path}, nil, []Lens{flood})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("over-broad derived lens should be dropped, got %d candidates", len(res.Candidates))
	}
	if len(res.Notes) == 0 {
		t.Fatal("over-broad drop must be recorded in Notes, never silent")
	}
	_ = name
}
