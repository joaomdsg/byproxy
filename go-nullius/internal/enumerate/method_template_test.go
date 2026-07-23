package enumerate

import "testing"

// The deferred-method-call template matches a method call that is DEFERRED (e.g.
// `defer mu.Unlock()`) — distinct from a plain call. This gives Recon a lens for
// defer-based lifecycle checks (the model reached for a nonexistent "defer_statement"
// template live; this is the real one).
func TestDeferredMethodCallTemplate(t *testing.T) {
	lang, name, err := LanguageFor("x.go")
	if err != nil {
		t.Fatal(err)
	}
	reg := DefaultRegistry()
	specs := []LensSpec{{
		ID:       "deferred-unlock",
		Template: "deferred-method-call",
		Params:   map[string]string{"method_regex": "^Unlock$"},
		Positive: "package p\nfunc a() { var m sync.Mutex; defer m.Unlock() }",
		Negative: "package p\nfunc a() { var m sync.Mutex; m.Unlock() }", // not deferred
	}}
	accepted, statuses := reg.AcceptDerived(specs, lang, name)
	if len(accepted) != 1 {
		t.Fatalf("deferred-method-call must match a deferred call, not a plain one: %+v", statuses)
	}
}

// The method-call template matches a method by its BARE name (the selector's field
// identifier), so the model supplies "^Lock$" — not a receiver-qualified or paren-bearing
// regex. This is the robust alternative to call-to, whose @fn is the whole selector text
// (recv.Lock) that weak models kept mis-matching.
func TestMethodCallTemplateMatchesByBareName(t *testing.T) {
	lang, name, err := LanguageFor("x.go")
	if err != nil {
		t.Fatal(err)
	}
	reg := DefaultRegistry()
	specs := []LensSpec{{
		ID:       "lock-calls",
		Template: "method-call",
		Params:   map[string]string{"method_regex": "^Lock$"},
		Positive: "package p\nfunc a() { var m sync.Mutex; m.Lock() }",
		Negative: "package p\nfunc a() { var m sync.Mutex; m.Unlock() }",
	}}
	accepted, statuses := reg.AcceptDerived(specs, lang, name)
	if len(accepted) != 1 {
		t.Fatalf("method-call lens must pass its witness (Lock matches, Unlock does not): %+v", statuses)
	}

	// It flags exactly the Lock() call in a file with both Lock and Unlock.
	f := parseGo(t, "package p\nfunc a() {\n\tvar m sync.Mutex\n\tm.Lock()\n\tm.Unlock()\n}\n")
	got := accepted[0].Enumerate(f)
	if len(got) != 1 {
		t.Fatalf("want exactly 1 Lock() hit, got %d: %+v", len(got), got)
	}
	if got[0].Line != 4 {
		t.Errorf("Lock() is on line 4, got %d", got[0].Line)
	}
}
