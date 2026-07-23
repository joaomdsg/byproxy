package machine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go-nullius/internal/enumerate"
)

// lensStoreFile is the on-disk lens library, under the already-gitignored .nullius/ dir.
const lensStoreDir = ".nullius"
const lensStoreFile = "lenses.json"

// LensStore is the compounding lens library: witness-gated derived lens specs that once
// produced a CONFIRMED defect, keyed by language. Future runs SEED these as always-on
// derived lenses so the model need not re-derive them — the coverage floor grows with
// every confirmed find. A seeded spec is re-witness-gated on every run (see AcceptDerived),
// so a spec that later goes stale (template removed, grammar drift) is silently dropped,
// never trusted blindly. This is nullius in verba applied to the pipeline's own memory:
// a stored lens re-earns its trust each run.
type LensStore struct {
	// ByLang maps a language name to its promoted specs. Exported for JSON round-trip.
	ByLang map[string][]enumerate.LensSpec `json:"by_lang"`
}

// LoadLensStore reads the lens library from <dir>/.nullius/lenses.json. A missing file is
// an EMPTY store, not an error — the library simply hasn't been seeded yet. A malformed
// file returns an error so a corrupt library is surfaced, not silently ignored.
func LoadLensStore(dir string) (*LensStore, error) {
	s := &LensStore{ByLang: map[string][]enumerate.LensSpec{}}
	path := filepath.Join(dir, lensStoreDir, lensStoreFile)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("lens store %s: %w", path, err)
	}
	if err := json.Unmarshal(b, s); err != nil {
		return nil, fmt.Errorf("lens store %s: %w", path, err)
	}
	if s.ByLang == nil {
		s.ByLang = map[string][]enumerate.LensSpec{}
	}
	return s, nil
}

// Seeds returns the stored specs for a language (nil if none), to be merged into the
// derived-lens set before witness-gating.
func (s *LensStore) Seeds(lang string) []enumerate.LensSpec { return s.ByLang[lang] }

// Promote adds spec to the language's library if no spec with the same ID is already
// stored. Returns true iff it was newly added. Dedup is by ID: model-derived IDs are
// semantic kebab names ("missing-unlock"), so a repeat ID is the same lens re-confirmed.
func (s *LensStore) Promote(lang string, spec enumerate.LensSpec) bool {
	if s.ByLang == nil {
		s.ByLang = map[string][]enumerate.LensSpec{}
	}
	for _, existing := range s.ByLang[lang] {
		if existing.ID == spec.ID {
			return false
		}
	}
	s.ByLang[lang] = append(s.ByLang[lang], spec)
	return true
}

// Save writes the library to <dir>/.nullius/lenses.json, creating .nullius/ if needed.
// Specs are sorted by ID within each language for a stable, diff-friendly file.
func (s *LensStore) Save(dir string) error {
	for lang := range s.ByLang {
		specs := s.ByLang[lang]
		sort.Slice(specs, func(i, j int) bool { return specs[i].ID < specs[j].ID })
	}
	dst := filepath.Join(dir, lensStoreDir)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("lens store: mkdir %s: %w", dst, err)
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("lens store: marshal: %w", err)
	}
	path := filepath.Join(dst, lensStoreFile)
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("lens store: write %s: %w", path, err)
	}
	return nil
}

// mergeSeeds prepends stored seeds to model-derived specs, deduped by ID (seed wins). It
// returns the merged spec list and the set of IDs that came from the store — those are
// already persisted and must not be re-promoted. Seed-first keeps candidate.Lens → spec
// mapping unambiguous: a confirmed candidate's lens ID resolves to exactly one spec.
func mergeSeeds(seeds, model []enumerate.LensSpec) (merged []enumerate.LensSpec, seededIDs map[string]bool) {
	seededIDs = map[string]bool{}
	seen := map[string]bool{}
	for _, s := range seeds {
		if seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		seededIDs[s.ID] = true
		merged = append(merged, s)
	}
	for _, s := range model {
		if seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		merged = append(merged, s)
	}
	return merged, seededIDs
}
