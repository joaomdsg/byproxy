package machine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildTerrainClassifiesTestSupport(t *testing.T) {
	dir := t.TempDir()
	must := func(rel, body string) string {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	prod := must("prod.go", "package p\n\nimport \"fmt\"\n\nfunc F() { fmt.Println() }\n")
	support := must("conformance.go", "package p\n\nimport \"testing\"\n\nfunc Run(t *testing.T) {}\n")
	testf := must("prod_test.go", "package p\n\nfunc TestF() {}\n")
	data := must("testdata/gen.go", "package d\n\nfunc G() {}\n")

	ter, err := BuildTerrain([]string{prod, support, testf, data})
	if err != nil {
		t.Fatal(err)
	}
	if len(ter.EnumFiles) != 1 || ter.EnumFiles[0] != prod {
		t.Fatalf("EnumFiles should be product-only [prod.go], got %v", ter.EnumFiles)
	}
	if len(ter.Files) != 4 {
		t.Fatalf("all 4 files must stay in the digest, got %d", len(ter.Files))
	}
	joined := strings.Join(ter.Excluded, "\n")
	for _, want := range []string{"conformance.go (testsupport: imports testing)", "prod_test.go (test: _test.go)", "testdata/gen.go (testsupport: dir testdata)"} {
		if !strings.Contains(joined, want) {
			t.Errorf("Excluded missing %q; got:\n%s", want, joined)
		}
	}
}

func TestClassifyFileDoesNotTrapRealWords(t *testing.T) {
	// "latest" ends in "test" but is not test scaffolding — the import-of-testing signal,
	// not a fragile suffix match, is what classifies. A latest/ file with no testing import
	// must stay product.
	dir := t.TempDir()
	p := filepath.Join(dir, "latest", "client.go")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("package latest\n\nimport \"fmt\"\n\nfunc C() { fmt.Println() }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ter, err := BuildTerrain([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if len(ter.EnumFiles) != 1 {
		t.Fatalf("latest/ file must be product, got excluded: %v", ter.Excluded)
	}
}
