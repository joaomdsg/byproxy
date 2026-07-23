// Command nullius-det runs the deterministic orchestrator (step-5 skeleton) over a set of
// files against the live local llama.cpp endpoints and streams each phase to stdout, so a
// run can be observed live. Go finds; the model only discriminates.
//
//	nullius-det -task "review internal/foo for concurrency bugs" internal/foo/*.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-nullius/internal/caller"
	"go-nullius/internal/machine"
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	task := flag.String("task", "", "the mandate: what to analyze (required)")
	smartURL := flag.String("smart-url", envOr("NULLIUS_SMART_URL", "http://192.168.11.41:8080/v1"), "smart-tier endpoint")
	fastURL := flag.String("fast-url", envOr("NULLIUS_FAST_URL", "http://192.168.11.41:8081/v1"), "fast-tier endpoint")
	smartModel := flag.String("smart-model", envOr("NULLIUS_SMART_MODEL", "minimax-m2.7"), "smart-tier model id")
	fastModel := flag.String("fast-model", envOr("NULLIUS_FAST_MODEL", "qwen3.6"), "fast-tier model id")
	dir := flag.String("dir", "", "git worktree root; with -craftsman-bin, enables Drain (writes fixes)")
	craftBin := flag.String("craftsman-bin", "", "path to a go-nullius binary used as the Drain craftsman; enables Drain")
	timeout := flag.Duration("timeout", 15*time.Minute, "overall run timeout")
	flag.Parse()

	if *task == "" || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: nullius-det -task \"<mandate>\" <file.go> [file.go ...]")
		os.Exit(2)
	}

	files, err := expandFiles(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "file error:", err)
		os.Exit(2)
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no files matched")
		os.Exit(2)
	}

	c := caller.New(os.Getenv("OPENAI_API_KEY"), map[caller.Tier]caller.Endpoint{
		caller.Smart: {BaseURL: *smartURL, Model: *smartModel},
		caller.Fast:  {BaseURL: *fastURL, Model: *fastModel},
	})

	m := machine.New(c)
	if *craftBin != "" && *dir != "" {
		m.Craftsman = machine.SubprocessCraftsman{Bin: *craftBin, Model: *fastModel, Dir: *dir}
		fmt.Printf("drain ENABLED: craftsman=%s model=%s dir=%s\n", *craftBin, *fastModel, *dir)
	}
	start := time.Now()
	m.Log = func(p machine.Phase, msg string) {
		fmt.Printf("[%7s] %s\n", p, msg)
	}

	fmt.Printf("nullius-det: smart=%s (%s) fast=%s (%s)\n", *smartURL, *smartModel, *fastURL, *fastModel)
	fmt.Printf("task: %s\nfiles: %s\n\n", *task, strings.Join(files, ", "))

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	res, err := m.Run(ctx, machine.Mandate{Task: *task, Files: files, Dir: *dir})
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nrun error:", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== RESULT (%.1fs) ===\n", time.Since(start).Seconds())
	fmt.Printf("mode: %s\n", res.Mode)
	if len(res.LensStatuses) > 0 {
		fmt.Println("derived lenses:")
		for _, s := range res.LensStatuses {
			verdict := "ACCEPTED"
			if !s.Accepted {
				verdict = "DERIVE_FAILED: " + s.Reason
			}
			fmt.Printf("  %-24s %s\n", s.ID, verdict)
		}
	}
	fmt.Printf("candidates: %d\n", len(res.Candidates))
	if len(res.Judged) > 0 {
		confirmed := res.Confirmed()
		fmt.Printf("rulings (%d judged, %d CONFIRMED):\n", len(res.Judged), len(confirmed))
		for _, j := range res.Judged {
			tag := strings.ToUpper(j.Judge.Answer)
			switch {
			case j.Confirmed:
				tag = "CONFIRMED"
			case j.Refuted:
				tag = "REFUTED"
			}
			fmt.Printf("  %-10s %s:%d [%s] fn=%s — %s\n", tag, j.Candidate.File, j.Candidate.Line, j.Candidate.Lens, j.Candidate.Fn, j.Judge.Because)
		}
	} else {
		for _, cand := range res.Candidates {
			fmt.Printf("  %s:%d  [%s] fn=%s  %s\n", cand.File, cand.Line, cand.Lens, cand.Fn, oneLine(cand.Snippet))
		}
	}
	if len(res.Plans) > 0 {
		fmt.Printf("fix plans (%d):\n", len(res.Plans))
		for _, p := range res.Plans {
			fb := ""
			if p.Fallback {
				fb = " (FALLBACK)"
			}
			fmt.Printf("  %s:%d%s\n    intent: %s\n    test:   %s\n    blast:  %s\n",
				p.Confirmation.Candidate.File, p.Confirmation.Candidate.Line, fb, p.Plan.Intent, p.Plan.TestName, p.Plan.BlastRadius)
		}
	}
	if len(res.Drained) > 0 {
		fmt.Printf("drained (%d):\n", len(res.Drained))
		for _, d := range res.Drained {
			fmt.Printf("  %-6s %s:%d — %s\n", d.Status, d.Plan.Confirmation.Candidate.File, d.Plan.Confirmation.Candidate.Line, d.Detail)
			if d.Diffstat != "" {
				fmt.Printf("           %s\n", oneLine(d.Diffstat))
			}
		}
	}
	if len(res.Notes) > 0 {
		fmt.Println("notes:")
		for _, n := range res.Notes {
			fmt.Printf("  %s\n", n)
		}
	}
}

// expandFiles resolves each argument as a glob (falling back to the literal path) and
// dedupes, preserving order.
func expandFiles(args []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, a := range args {
		matches, err := filepath.Glob(a)
		if err != nil {
			return nil, err
		}
		if matches == nil {
			matches = []string{a}
		}
		for _, mth := range matches {
			if !seen[mth] {
				seen[mth] = true
				out = append(out, mth)
			}
		}
	}
	return out, nil
}

func oneLine(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) > 70 {
		return s[:70] + "…"
	}
	return s
}
