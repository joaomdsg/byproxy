package machine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-nullius/internal/enumerate"
)

// affirmAll drives Judge=DEFECT / Refute=stands for every candidate — everything confirms.
func affirmAll(p string) (string, bool) {
	switch {
	case strings.Contains(p, "JUDGE phase"):
		return `{"answer":"DEFECT","decisive_line":4,"because":"x"}`, true
	case strings.Contains(p, "REFUTE phase"):
		return `{"stands":true,"refuting_line":null}`, true
	}
	return "", false
}

// writeGoIn writes src into dir/name and returns the path — for tests that need the file
// under a known Mandate.Dir so the lens library lands in dir/.nullius/.
func writeGoIn(t *testing.T, dir, name, src string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPromotionPersistsConfirmedDerivedLens(t *testing.T) {
	dir := t.TempDir()
	f := writeGoIn(t, dir, "a.go", brownfieldSrc)
	m := New(fixFake(affirmAll))
	res, err := m.Run(context.Background(), Mandate{Task: "review f", Files: []string{f}, Dir: dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if countConfirmed(res) == 0 {
		t.Fatal("expected confirmed defects to drive promotion")
	}

	store, err := LoadLensStore(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	ids := map[string]bool{}
	for _, s := range store.Seeds("go") {
		ids[s.ID] = true
	}
	if !ids["calls-helper"] {
		t.Fatalf("derived lens calls-helper was not promoted: %v", ids)
	}
	// The baseline lens is NOT model-derived, so it must never enter the library.
	if ids["stmt-after-return"] {
		t.Fatal("baseline lens stmt-after-return must not be promoted")
	}
}

func TestSeededLensRunsWithoutModelDerivation(t *testing.T) {
	dir := t.TempDir()
	f := writeGoIn(t, dir, "a.go", brownfieldSrc)

	// Pre-seed the library, then run with a model that derives NOTHING.
	store := &LensStore{}
	store.Promote("go", enumerate.LensSpec{
		ID: "calls-helper", Template: "call-to", Params: map[string]string{"fn_regex": "helper"},
		Mechanism: "call", Anchor: "call",
		Positive: "package p\nfunc a(){ helper() }", Negative: "package p\nfunc a(){ other() }",
	})
	if err := store.Save(dir); err != nil {
		t.Fatalf("seed save: %v", err)
	}

	fc := fakeCaller{
		orient: `{"intent_summary":"x","focus_pkgs":[],"risk_note":"y"}`,
		gate:   `{"mode":"FIX","has_inscope_code":true,"justification":"code"}`,
		recon:  `{"lenses":[]}`, // model derives nothing — the seed must carry the hunt
		custom: affirmAll,
	}
	m := New(fc)
	res, err := m.Run(context.Background(), Mandate{Task: "review f", Files: []string{f}, Dir: dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, c := range res.Judged {
		if c.Candidate.Lens == "calls-helper" {
			found = true
		}
	}
	if !found {
		t.Fatal("seeded lens calls-helper did not enumerate any candidate")
	}
}

func spec(id string) enumerate.LensSpec {
	return enumerate.LensSpec{
		ID:        id,
		Template:  "method-call",
		Params:    map[string]string{"method_regex": "^Lock$"},
		Mechanism: "call",
		Anchor:    "call",
		Positive:  "package p\nfunc a(){ var m sync.Mutex; m.Lock() }",
		Negative:  "package p\nfunc a(){ x := map[string]int{}; _ = x }",
	}
}

func TestLoadLensStoreMissingIsEmpty(t *testing.T) {
	dir := t.TempDir()
	s, err := LoadLensStore(dir)
	if err != nil {
		t.Fatalf("missing store must not error: %v", err)
	}
	if len(s.Seeds("go")) != 0 {
		t.Fatalf("missing store must be empty, got %d seeds", len(s.Seeds("go")))
	}
}

func TestPromoteSaveReloadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	s, _ := LoadLensStore(dir)
	if !s.Promote("go", spec("missing-unlock")) {
		t.Fatal("first promote must report newly-added")
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	reloaded, err := LoadLensStore(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got := reloaded.Seeds("go")
	if len(got) != 1 || got[0].ID != "missing-unlock" {
		t.Fatalf("roundtrip lost the spec: %+v", got)
	}
	// Witnesses must survive the roundtrip — AcceptDerived re-gates on them.
	if got[0].Positive == "" || got[0].Params["method_regex"] != "^Lock$" {
		t.Fatalf("roundtrip dropped witness/params: %+v", got[0])
	}
}

func TestPromoteDedupsByID(t *testing.T) {
	s := &LensStore{}
	if !s.Promote("go", spec("dup")) {
		t.Fatal("first promote should add")
	}
	if s.Promote("go", spec("dup")) {
		t.Fatal("second promote of same ID must be a no-op")
	}
	if len(s.Seeds("go")) != 1 {
		t.Fatalf("dedup failed: %d specs", len(s.Seeds("go")))
	}
}

func TestSeedsAreLanguageScoped(t *testing.T) {
	s := &LensStore{}
	s.Promote("go", spec("go-lens"))
	s.Promote("python", spec("py-lens"))
	if len(s.Seeds("go")) != 1 || s.Seeds("go")[0].ID != "go-lens" {
		t.Fatalf("go scope wrong: %+v", s.Seeds("go"))
	}
	if len(s.Seeds("rust")) != 0 {
		t.Fatalf("unknown lang must be empty, got %d", len(s.Seeds("rust")))
	}
}

func TestMergeSeedsSeedWinsAndTracksIDs(t *testing.T) {
	seeds := []enumerate.LensSpec{spec("shared"), spec("seed-only")}
	model := []enumerate.LensSpec{spec("shared"), spec("model-only")}
	merged, seededIDs := mergeSeeds(seeds, model)

	if len(merged) != 3 {
		t.Fatalf("expected 3 unique specs, got %d: %+v", len(merged), merged)
	}
	// Seed-first: the "shared" ID must be counted as seeded (already persisted).
	if !seededIDs["shared"] || !seededIDs["seed-only"] {
		t.Fatalf("seeded IDs wrong: %v", seededIDs)
	}
	if seededIDs["model-only"] {
		t.Fatal("model-only must NOT be marked seeded — it's promotable")
	}
}
