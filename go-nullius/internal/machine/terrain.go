package machine

import (
	"fmt"
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
	Files     []string
	Lang      string
	NodeKinds map[string]int
	Funcs     []string

	lang *gts.Language // resolved grammar, reused by the machine for lens compilation
}

// BuildTerrain parses every file and accumulates the digest. All files are expected to
// share one language (the first file's language wins); a parse error is fatal, since a
// terrain we cannot read is not a terrain we can hunt honestly.
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
