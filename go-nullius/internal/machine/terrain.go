package machine

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"go-nullius/internal/enumerate"

	gts "github.com/odvcencio/gotreesitter"
)

// Terrain is the mechanical digest fed to the smart phases: the file inventory, the
// node-kind histogram of the ACTUAL parsed trees (design Q4: grounds the weak model in the
// real grammar vocabulary instead of its priors), and the declared function/method names.
// Nothing here is model-authored — it is pure Go over the CST.
type Terrain struct {
	Files     []string // ALL files, kept for the digest (recon sees test scaffolding as context)
	EnumFiles []string // PRODUCT files only — the set Enumerate/Audit hunt (test support excluded)
	Excluded  []string // "path (reason)" per excluded file — surfaced in Notes, never silent
	Lang      string
	NodeKinds map[string]int
	Funcs     []string

	lang *gts.Language // resolved grammar, reused by the machine for lens compilation
}

// BuildTerrain parses every file and accumulates the digest. All files are expected to
// share one language (the first file's language wins); a parse error is fatal, since a
// terrain we cannot read is not a terrain we can hunt honestly. Test and test-support files
// are classified out of EnumFiles (we don't hunt defects in scaffolding) but kept in Files
// so the digest still shows recon the whole tree.
func BuildTerrain(files []string) (*Terrain, error) {
	t := &Terrain{NodeKinds: map[string]int{}}
	for _, path := range files {
		f, err := enumerate.ParseFile(path)
		if err != nil {
			return nil, fmt.Errorf("terrain: %w", err)
		}
		if t.lang == nil {
			t.lang = f.Lang
			t.Lang = f.LangName
		}
		t.Files = append(t.Files, path)
		if class, reason := classifyFile(path, f); class == fileProduct {
			t.EnumFiles = append(t.EnumFiles, path)
		} else {
			t.Excluded = append(t.Excluded, fmt.Sprintf("%s (%s: %s)", path, class, reason))
		}
		walkAll(f.Tree.RootNode(), func(n *gts.Node) {
			kind := n.Type(f.Lang)
			t.NodeKinds[kind]++
			if kind == "function_declaration" || kind == "method_declaration" {
				if name := fieldText(f.Src, n.ChildByFieldName("name", f.Lang)); name != "" {
					t.Funcs = append(t.Funcs, name)
				}
			}
		})
	}
	return t, nil
}

const (
	fileProduct     = "product"
	fileTest        = "test"
	fileTestSupport = "testsupport"
)

// testDirs are exact terminal directory-segment names that mark test scaffolding. Only EXACT
// segment matches are used (never substring) — "latest/" and "contest.go" are real-word traps
// that a `*test*` match would wrongly catch (fable counsel 2026-07-24).
var testDirs = map[string]bool{"testdata": true, "mocks": true, "fakes": true, "stubs": true}

// classifyFile labels a Go file product / test / testsupport by MECHANICAL signals only:
//  1. `*_test.go`                    → test
//  2. imports "testing" (or testing/…) → testsupport (the strong, unambiguous signal: a
//     package importing testing is test scaffolding by Go convention — this is what caught
//     backplanetest/conformance.go, the source of 2/3 FPs in the vialite bench)
//  3. terminal dir ∈ {testdata,mocks,fakes,stubs} → testsupport
//
// The fragile "dir/pkg ends in test" heuristic is deliberately NOT used ("latest" ends in
// "test"); the import-of-testing signal already covers the real cases without the trap.
func classifyFile(path string, f *enumerate.ParsedFile) (class, reason string) {
	if strings.HasSuffix(filepath.Base(path), "_test.go") {
		return fileTest, "_test.go"
	}
	for _, imp := range importPaths(f) {
		if imp == "testing" || strings.HasPrefix(imp, "testing/") {
			return fileTestSupport, "imports testing"
		}
	}
	if dir := filepath.Base(filepath.Dir(path)); testDirs[dir] {
		return fileTestSupport, "dir " + dir
	}
	return fileProduct, ""
}

// importPaths returns the unquoted import paths declared in a parsed Go file, via the CST
// (import_spec → path field), so it is robust to formatting.
func importPaths(f *enumerate.ParsedFile) []string {
	var out []string
	walkAll(f.Tree.RootNode(), func(n *gts.Node) {
		if n.Type(f.Lang) != "import_spec" {
			return
		}
		raw := fieldText(f.Src, n.ChildByFieldName("path", f.Lang))
		out = append(out, strings.Trim(raw, "\"`"))
	})
	return out
}

func walkAll(n *gts.Node, fn func(*gts.Node)) {
	if n == nil {
		return
	}
	fn(n)
	for i := 0; i < n.NamedChildCount(); i++ {
		walkAll(n.NamedChild(i), fn)
	}
}

func fieldText(src []byte, n *gts.Node) string {
	if n == nil {
		return ""
	}
	s, e := int(n.StartByte()), int(n.EndByte())
	if s < 0 || e > len(src) || s > e {
		return ""
	}
	return string(src[s:e])
}

// Digest renders the terrain as compact text for a prompt: bounded node-kind histogram and
// a capped function list, so a large module cannot blow the prompt-size wall.
func (t *Terrain) Digest() string {
	var b strings.Builder
	fmt.Fprintf(&b, "language: %s\nfiles (%d):\n", t.Lang, len(t.Files))
	for _, f := range t.Files {
		fmt.Fprintf(&b, "  %s\n", f)
	}

	type kv struct {
		k string
		n int
	}
	kinds := make([]kv, 0, len(t.NodeKinds))
	for k, n := range t.NodeKinds {
		kinds = append(kinds, kv{k, n})
	}
	sort.Slice(kinds, func(i, j int) bool {
		if kinds[i].n != kinds[j].n {
			return kinds[i].n > kinds[j].n
		}
		return kinds[i].k < kinds[j].k
	})
	b.WriteString("node kinds present (kind:count):\n")
	for i, e := range kinds {
		if i >= 30 {
			fmt.Fprintf(&b, "  … (+%d more kinds)\n", len(kinds)-30)
			break
		}
		fmt.Fprintf(&b, "  %s:%d\n", e.k, e.n)
	}

	if len(t.Funcs) > 0 {
		b.WriteString("functions/methods:\n")
		for i, fn := range t.Funcs {
			if i >= 40 {
				fmt.Fprintf(&b, "  … (+%d more)\n", len(t.Funcs)-40)
				break
			}
			fmt.Fprintf(&b, "  %s\n", fn)
		}
	}
	return b.String()
}
