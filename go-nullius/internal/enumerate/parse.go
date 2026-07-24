package enumerate

import (
	"fmt"
	"os"
	"sort"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// sortedKeys returns the keys of a line-set as a sorted slice — a stable Evidence list.
func sortedKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// ParsedFile is a source file parsed into a tree-sitter CST, carrying everything a
// lens needs: the raw bytes (node text is sliced from here), the language (node kinds
// and field names are language-relative), and the tree.
type ParsedFile struct {
	Path     string
	Src      []byte
	Lang     *gts.Language
	LangName string
	Tree     *gts.Tree
	Profile  *LangProfile // role map for walk lenses; nil for unprofiled languages
}

// LanguageFor resolves a file path to a tree-sitter language by extension. This is the
// language-agnostic entry point — no Go-specific assumption lives here.
func LanguageFor(path string) (*gts.Language, string, error) {
	e := grammars.DetectLanguage(path)
	if e == nil {
		return nil, "", fmt.Errorf("enumerate: no grammar for %q", path)
	}
	return e.Language(), e.Name, nil
}

// ParseFile reads and parses a file from disk.
func ParseFile(path string) (*ParsedFile, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSource(path, src)
}

// ParseSource parses in-memory source; path is used only for language detection and
// for the record in resulting Candidates.
func ParseSource(path string, src []byte) (*ParsedFile, error) {
	lang, name, err := LanguageFor(path)
	if err != nil {
		return nil, err
	}
	f, err := parse(src, lang, name)
	if err != nil {
		return nil, fmt.Errorf("enumerate: parse %q: %w", path, err)
	}
	f.Path = path
	return f, nil
}

// ParseSourceLang parses in-memory source with an already-resolved language, skipping
// extension detection. Used by the witness gate for snippets that have no real path.
func ParseSourceLang(src []byte, lang *gts.Language, langName string) (*ParsedFile, error) {
	return parse(src, lang, langName)
}

func parse(src []byte, lang *gts.Language, langName string) (*ParsedFile, error) {
	tree, err := gts.NewParser(lang).Parse(src)
	if err != nil {
		return nil, err
	}
	return &ParsedFile{Path: "<snippet>", Src: src, Lang: lang, LangName: langName, Tree: tree, Profile: profileFor(langName)}, nil
}

// nodeText returns the verbatim source text a node spans. Empty for a nil node or an
// out-of-range span (defensive — a lens must never fabricate text).
func nodeText(src []byte, n *gts.Node) string {
	if n == nil {
		return ""
	}
	s, e := int(n.StartByte()), int(n.EndByte())
	if s < 0 || e > len(src) || s > e {
		return ""
	}
	return string(src[s:e])
}

// nodeLine returns the 1-indexed start line of a node.
func nodeLine(n *gts.Node) int {
	if n == nil {
		return 0
	}
	return int(n.StartPoint().Row) + 1
}

// spanLines returns every 1-indexed line the node covers, inclusive — the implicated region
// for a candidate's evidence set (the decisive-line coherence gate checks membership here).
func spanLines(n *gts.Node) []int {
	if n == nil {
		return nil
	}
	s, e := int(n.StartPoint().Row)+1, int(n.EndPoint().Row)+1
	if e < s {
		e = s
	}
	out := make([]int, 0, e-s+1)
	for l := s; l <= e; l++ {
		out = append(out, l)
	}
	return out
}
