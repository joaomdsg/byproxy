package enumerate

import "testing"

func TestParseSourceGo(t *testing.T) {
	src := []byte("package p\n\nfunc f() int { return 0 }\n")
	f, err := ParseSource("x.go", src)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if f.LangName == "" {
		t.Fatal("empty LangName from extension detection")
	}
	root := f.Tree.RootNode()
	if root == nil {
		t.Fatal("nil root node")
	}
	if got := nodeText(src, root); got == "" {
		t.Fatal("nodeText(root) empty")
	}
	if got := nodeLine(root); got != 1 {
		t.Fatalf("nodeLine(root) = %d, want 1", got)
	}
}

func TestLanguageForUnknownExt(t *testing.T) {
	if _, _, err := LanguageFor("x.unknownext"); err == nil {
		t.Fatal("expected error for unknown extension")
	}
}

func TestNodeTextOutOfRange(t *testing.T) {
	// defensive: nil node yields empty, never panics
	if nodeText([]byte("abc"), nil) != "" {
		t.Fatal("nodeText(nil) should be empty")
	}
	if nodeLine(nil) != 0 {
		t.Fatal("nodeLine(nil) should be 0")
	}
}
