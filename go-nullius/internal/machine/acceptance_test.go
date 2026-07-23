package machine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-nullius/internal/caller"
)

// TestLiveAcceptance_D2vsFP pins the design's core claim against the REAL local models:
// given two same-shape lock sites — one correct (Lock + defer Unlock), one a real defect
// (Lock, no Unlock) — the pipeline must CONFIRM the defect and CLEAR the twin. It is gated
// by NULLIUS_LIVE_ACCEPT=1 (needs the llama.cpp endpoints) so the hermetic suite stays
// offline-green; the fake-caller tests above pin the discrimination LOGIC deterministically.
//
// The invariant asserted is tolerant of Recon lens-derivation variance: the FALSE POSITIVE
// (Get) must NEVER be confirmed (hard), and the real defect (Set) must be confirmed when the
// run produced lock candidates at all (else it skips — an infra/derivation miss, not a
// discrimination failure).
func TestLiveAcceptance_D2vsFP(t *testing.T) {
	if os.Getenv("NULLIUS_LIVE_ACCEPT") == "" {
		t.Skip("set NULLIUS_LIVE_ACCEPT=1 to run against the live local models")
	}
	env := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	c := caller.New(os.Getenv("OPENAI_API_KEY"), map[caller.Tier]caller.Endpoint{
		caller.Smart: {BaseURL: env("NULLIUS_SMART_URL", "http://192.168.11.41:8080/v1"), Model: env("NULLIUS_SMART_MODEL", "minimax-m2.7")},
		caller.Fast:  {BaseURL: env("NULLIUS_FAST_URL", "http://192.168.11.41:8081/v1"), Model: env("NULLIUS_FAST_MODEL", "qwen3.6")},
	})

	dir := t.TempDir()
	file := filepath.Join(dir, "store.go")
	if err := os.WriteFile(file, []byte(lockPairSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	m := New(c)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	res, err := m.Run(ctx, Mandate{
		Task:  "Review this Go file for mutex/locking bugs: a Lock() that is never released, a missing Unlock, or deadlock risk.",
		Files: []string{file},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Judged) == 0 {
		t.Skip("Recon derived no lock lens this run (model/infra); discrimination not exercised")
	}

	var setConfirmed, getConfirmed bool
	for _, j := range res.Confirmed() {
		switch j.Candidate.Fn {
		case "Set":
			setConfirmed = true
		case "Get":
			getConfirmed = true
		}
	}
	if getConfirmed {
		t.Errorf("FALSE POSITIVE: the correct Get lock (defer Unlock) was confirmed as a defect")
	}
	if !setConfirmed {
		t.Errorf("MISS: the real missing-Unlock defect in Set was not confirmed; judged=%+v", res.Judged)
	}
}
