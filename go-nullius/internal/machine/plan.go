package machine

import (
	"context"

	"go-nullius/internal/caller"
)

const planMaxTokens = 3000 // design: plan 1500 + reasoning-model headroom

// PlanOut is the smart tier's fix plan for one confirmed defect. It describes a change; it
// never performs one (writes belong to Drain, the next increment).
type PlanOut struct {
	Target      string `json:"target"`       // file (and symbol) to change
	Intent      string `json:"intent"`       // the fix, carrying the ruling's mechanism (<=300)
	TestName    string `json:"test_name"`    // the pinning test to add
	TestSketch  string `json:"test_sketch"`  // how the test proves the fix (<=600)
	BlastRadius string `json:"blast_radius"` // what else the change could touch
}

// FixPlan pairs a confirmed defect with its plan. Fallback marks a plan the model could not
// produce (a mechanical template stood in) — surfaced, never silent.
type FixPlan struct {
	Confirmation Confirmation
	Plan         PlanOut
	Fallback     bool
}

// planFix asks the smart tier for a fix plan for one confirmed defect. On failure it
// degrades to a minimal mechanical plan (fail-closed: a defect never silently loses its
// plan) flagged Fallback, so a downstream Drain still has a target and the report is honest.
func (m *Machine) planFix(ctx context.Context, task string, c Confirmation, log logf) FixPlan {
	cand := c.Candidate
	lines, err := readLines(cand.File)
	window := ""
	if err == nil {
		window, _, _ = enclosingWindow(cand.File, lines, cand.Line)
	}
	var p PlanOut
	if err := m.Caller.Ask(ctx, m.Smart, planPrompt(task, c, window), caller.GBNF(jsonGrammar), &p, caller.WithMaxTokens(planMaxTokens)); err != nil {
		log(PhasePlan, "%s:%d FALLBACK (%v): templated plan", cand.File, cand.Line, err)
		return FixPlan{
			Confirmation: c,
			Fallback:     true,
			Plan: PlanOut{
				Target:      cand.File,
				Intent:      "address the confirmed defect: " + c.Judge.Because,
				TestName:    "TestFix_" + sanitizeIdent(cand.Fn),
				TestSketch:  "add a test that fails on the current behavior and passes after the fix",
				BlastRadius: "unknown (plan fallback)",
			},
		}
	}
	log(PhasePlan, "%s:%d plan: target=%q intent=%q test=%q radius=%q", cand.File, cand.Line, p.Target, truncate(p.Intent, 60), p.TestName, truncate(p.BlastRadius, 40))
	return FixPlan{Confirmation: c, Plan: p}
}

func planPrompt(task string, c Confirmation, window string) string {
	cand := c.Candidate
	return `You are the PLAN phase. A defect has been CONFIRMED. Produce a plan to fix it — describe the
change, do NOT write code. Reply with ONLY a JSON object of exactly this shape:
{"target": "file and symbol", "intent": "<=300 chars: the fix and WHY", "test_name": "TestXxx",
 "test_sketch": "<=600 chars: how the test pins the fixed behavior", "blast_radius": "what else this could affect"}

TASK:
` + task + `

CONFIRMED DEFECT at ` + cand.File + `:` + itoa(cand.Line) + ` in function ` + cand.Fn + `
LENS: ` + cand.Lens + `  —  ` + c.Judge.Because + `

CODE (line: text):
` + window
}

// sanitizeIdent makes a fn name safe as a Go test-name suffix; empty → "Defect".
func sanitizeIdent(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '_' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "Defect"
	}
	return string(out)
}
