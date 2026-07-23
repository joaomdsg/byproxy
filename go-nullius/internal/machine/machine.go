// Package machine is the deterministic-orchestrator state machine. Go finds; the model
// only discriminates between concrete alternatives Go extracted. Wired live over the caller
// substrate: the dynamic-lens front half — Orient → Gate → Recon → Enumerate (smart) — the
// discrimination half — Judge → Corroborate (fast, per candidate) — Plan (smart, one fix
// plan per confirmed defect, NO writes) — and a report Close. Drain (craftsman writes,
// snapshot/build/-race/revert-retry) and Audit (frozen-lens re-hunt over changed files) —
// the code-MUTATION step — are the remaining stubs.
//
// Every phase fails CLOSED toward doing LESS, safely: Orient failure → empty orientation;
// Gate failure or doubt → FIX (run the full hunt); Recon failure → baseline only (the
// coverage floor); Judge/refuter failure → CANT_TELL / unverified (never an auto-confirm).
// Corroborate treats a refutation as testimony: it clears a defect only when it cites a
// valid, distinct safety line. No model failure — nor a whole-tier outage — can silently
// skip the hunt or falsely confirm/clear a defect.
package machine

import (
	"context"
	"fmt"
	"strings"

	"go-nullius/internal/caller"
	"go-nullius/internal/enumerate"
)

// Phase names each step for the trace/ledger.
type Phase string

const (
	PhaseOrient      Phase = "orient"
	PhaseGate        Phase = "gate"
	PhaseRecon       Phase = "recon"
	PhaseEnumerate   Phase = "enumerate"
	PhaseJudge       Phase = "judge"
	PhaseCorroborate Phase = "corroborate"
	PhasePlan        Phase = "plan"
	PhaseDrain       Phase = "drain"
	PhaseAudit       Phase = "audit"
	PhaseClose       Phase = "close"
)

// backHalf is the not-yet-wired tail of the machine (step 6/7).
var backHalf = []Phase{PhaseJudge, PhaseCorroborate, PhasePlan, PhaseDrain, PhaseAudit, PhaseClose}

// Mode is the three-way gate result.
type Mode string

const (
	ModeAnswer Mode = "ANSWER"
	ModeBuild  Mode = "BUILD"
	ModeFix    Mode = "FIX"
)

// Mandate is one unit of work: a task, the in-scope files (the terrain), and optionally the
// git workdir. Drain runs only when Dir is set AND the Machine has a Craftsman.
type Mandate struct {
	Task  string
	Files []string
	Dir   string // git worktree root; "" disables Drain (plans are reported, not written)
}

// Trace is one recorded phase event, in order.
type Trace struct {
	Phase Phase
	Msg   string
}

// Result is the machine's output: the gate ruling, orientation, the derived-lens
// dispositions, the enumerated candidates, their Judge/Corroborate dispositions, and the
// full trace.
type Result struct {
	Mode         Mode
	Orient       OrientOut
	LensStatuses []enumerate.LensStatus
	Candidates   []enumerate.Candidate
	Judged       []Confirmation
	Plans        []FixPlan
	Drained      []DrainResult
	Notes        []string
	Trace        []Trace
}

// Confirmed returns only the candidates that survived Judge + Corroborate.
func (r *Result) Confirmed() []Confirmation {
	var out []Confirmation
	for _, c := range r.Judged {
		if c.Confirmed {
			out = append(out, c)
		}
	}
	return out
}

// logf records a phase event to the trace and the optional live sink.
type logf func(Phase, string, ...any)

// Per-phase output caps (design: verdict 400 / plan 1500 / report 3000). Orient/Gate are
// small; Recon may emit several lenses with witness snippets.
// These are REASONING models on the local endpoints: the grammar-constrained JSON is
// emitted in reasoning_content, so the cap must cover the chain-of-thought AND the JSON.
// Sized with generous headroom — a truncated reply (unexpected EOF) is a fallback, not a
// result, so under-budgeting silently costs coverage.
const (
	orientMaxTokens = 2500
	gateMaxTokens   = 1500
	reconMaxTokens  = 8000
)

// Machine runs the state machine over a Caller and a lens Registry.
type Machine struct {
	Caller    caller.Caller
	Reg       *enumerate.Registry
	Smart     caller.Tier         // Orient/Gate/Recon/Plan — judgment
	Fast      caller.Tier         // Judge/Corroborate — bulk discrimination
	Craftsman Writer              // code writer for Drain; nil disables Drain
	Log       func(Phase, string) // optional live sink; nil = silent
}

// New builds a Machine with the default Go lens registry, the smart tier for judgment, and
// the fast tier for per-candidate discrimination.
func New(c caller.Caller) *Machine {
	return &Machine{Caller: c, Reg: enumerate.DefaultRegistry(), Smart: caller.Smart, Fast: caller.Fast}
}

// Run executes the front half of the machine and walks the back-half stubs. It never
// returns a model error to the caller: model failures degrade to the fail-closed default
// and are recorded in the trace. It returns an error only for mechanical impossibilities
// (unparseable terrain, no language, a baseline that fails to compile).
func (m *Machine) Run(ctx context.Context, md Mandate) (*Result, error) {
	res := &Result{}
	var log logf = func(p Phase, format string, a ...any) {
		msg := fmt.Sprintf(format, a...)
		res.Trace = append(res.Trace, Trace{p, msg})
		if m.Log != nil {
			m.Log(p, msg)
		}
	}

	ter, err := BuildTerrain(md.Files)
	if err != nil {
		return nil, err
	}
	if ter.lang == nil {
		return nil, fmt.Errorf("machine: no language resolved for the given files")
	}
	digest := ter.Digest()
	log(PhaseOrient, "terrain: %d file(s), lang=%s, %d node kinds, %d funcs", len(ter.Files), ter.Lang, len(ter.NodeKinds), len(ter.Funcs))

	// ORIENT — intent + risk read (fail → empty orientation, flow continues).
	var o OrientOut
	if err := m.Caller.Ask(ctx, m.Smart, orientPrompt(md.Task, digest), caller.GBNF(jsonGrammar), &o, caller.WithMaxTokens(orientMaxTokens)); err != nil {
		log(PhaseOrient, "FALLBACK (%v): empty orientation", err)
		o = OrientOut{RiskNote: "orient failed; conservative default"}
	} else {
		log(PhaseOrient, "intent=%q focus=%v risk=%q", o.IntentSummary, o.FocusPkgs, o.RiskNote)
	}
	res.Orient = o

	// GATE — three-way, fail-closed to FIX.
	var g GateOut
	mode, reason := ModeFix, "gate not reached"
	if err := m.Caller.Ask(ctx, m.Smart, gatePrompt(md.Task, digest, o), caller.GBNF(jsonGrammar), &g, caller.WithMaxTokens(gateMaxTokens)); err != nil {
		log(PhaseGate, "FALLBACK (%v): fail-closed to FIX", err)
	} else {
		mode, reason = normalizeMode(g, ter)
		log(PhaseGate, "model mode=%q inscope=%v → ruled %s (%s)", g.Mode, g.HasInscopeCode, mode, reasonOr(reason, g.Justification))
	}
	res.Mode = mode

	switch mode {
	case ModeAnswer:
		log(PhaseEnumerate, "ANSWER: read-only, Enumerate skipped")
		stubBackHalf(log)
		return res, nil
	case ModeBuild:
		log(PhaseEnumerate, "BUILD: greenfield — machine would enter at Plan with an empty candidate set; new code is hunted post-build at Audit")
		stubBackHalf(log)
		return res, nil
	}

	// FIX → RECON: derive lenses (fail → baseline-only coverage floor).
	var rc ReconOut
	if err := m.Caller.Ask(ctx, m.Smart, reconPrompt(md.Task, digest, templateDoc(m.Reg, ter.Lang), o), caller.GBNF(jsonGrammar), &rc, caller.WithMaxTokens(reconMaxTokens)); err != nil {
		rc = ReconOut{} // discard any partial decode from the failed parse; baseline-only
		log(PhaseRecon, "FALLBACK (%v): baseline-only (coverage floor holds)", err)
	} else {
		log(PhaseRecon, "model derived %d candidate lens(es)", len(rc.Lenses))
		for _, l := range rc.Lenses {
			log(PhaseRecon, "  spec %q template=%q params=%v mech=%q pos=%q", l.ID, l.Template, l.Params, l.Mechanism, truncate(l.Positive, 80))
		}
	}

	// ENUMERATE — baseline always runs; derived lenses are witness-gated then added.
	baseline, err := m.Reg.BuildBaseline(ter.Lang, ter.lang)
	if err != nil {
		return nil, fmt.Errorf("machine: baseline compile: %w", err)
	}
	accepted, statuses := m.Reg.AcceptDerived(rc.specs(), ter.lang, ter.Lang)
	res.LensStatuses = statuses
	for _, s := range statuses {
		if s.Accepted {
			log(PhaseRecon, "lens %q ACCEPTED (witness passed)", s.ID)
		} else {
			log(PhaseRecon, "lens %q DERIVE_FAILED: %s", s.ID, s.Reason)
		}
	}
	er, err := enumerate.Run(ter.Files, baseline, accepted)
	if err != nil {
		return nil, fmt.Errorf("machine: enumerate: %w", err)
	}
	res.Candidates = er.Candidates
	res.Notes = er.Notes
	log(PhaseEnumerate, "%d candidate(s) from %d baseline + %d derived lens(es); %d note(s)", len(er.Candidates), len(baseline), len(accepted), len(er.Notes))
	for _, n := range er.Notes {
		log(PhaseEnumerate, "note: %s", n)
	}
	if len(er.Candidates) == 0 {
		log(PhaseJudge, "no candidates to judge")
		stubBackHalf(log)
		return res, nil
	}

	// JUDGE + CORROBORATE — turn candidate SITES into confirmed RULINGS on the fast tier.
	// Each candidate is independent; a per-candidate model failure degrades to CANT_TELL and
	// never blocks the others.
	log(PhaseJudge, "judging %d candidate(s) on the fast tier", len(er.Candidates))
	for _, c := range er.Candidates {
		conf := m.judgeAndCorroborate(ctx, md.Task, c, log)
		res.Judged = append(res.Judged, conf)
	}
	confirmed := res.Confirmed()
	log(PhaseCorroborate, "%d/%d candidate(s) CONFIRMED after Judge + Corroborate", len(confirmed), len(res.Judged))

	// PLAN — one fix plan per DISTINCT defect target (smart tier, NO writes). Same-lens
	// defects in the same function collapse to one plan: a single craftsman fixes the
	// function, and separate overlapping plans would have the second find the work done
	// (measured over-decomposition on the dead-code acceptance).
	targets := dedupeConfirmed(confirmed)
	if len(targets) < len(confirmed) {
		log(PhasePlan, "collapsed %d confirmed defect(s) into %d distinct plan target(s)", len(confirmed), len(targets))
	}
	for _, c := range targets {
		res.Plans = append(res.Plans, m.planFix(ctx, md.Task, c, log))
	}
	if len(targets) > 0 {
		log(PhasePlan, "%d fix plan(s) produced", len(res.Plans))
	}

	// DRAIN — the craftsman writes each plan; every DONE is mechanically verified (non-empty
	// diff + build + touched-package -race) and failures are reverted. Runs only with a
	// craftsman AND a git workdir; otherwise the plans are reported, not written.
	if md.Dir != "" && m.Craftsman != nil && len(res.Plans) > 0 {
		log(PhaseDrain, "draining %d plan(s) via craftsman in %s", len(res.Plans), md.Dir)
		res.Drained = m.drain(ctx, md.Dir, res.Plans, log)
		log(PhaseDrain, "%d/%d plan(s) DONE", countDone(res.Drained), len(res.Drained))
		// AUDIT — re-run the FROZEN lens set over the (now-modified) files; residual
		// candidates are reported. The bounded audit→judge re-entry loop is a step-7 add.
		m.audit(ctx, md, baseline, accepted, log)
	} else {
		log(PhaseDrain, "SKELETON: no craftsman/dir — %d plan(s) reported, not written", len(res.Plans))
		log(PhaseAudit, "skipped (no drain)")
	}

	// CLOSE — deterministic report tally (surface diff + suite rerun join here in step 7).
	log(PhaseClose, "mode=%s: %d confirmed, %d planned, %d drained-DONE, %d judged",
		mode, len(confirmed), len(res.Plans), countDone(res.Drained), len(res.Judged))
	return res, nil
}

// dedupeConfirmed collapses confirmed defects that share (file, function, lens) — the same
// fix — keeping the first (lowest line, since candidates are sorted). Distinct lenses or
// functions stay separate.
func dedupeConfirmed(cs []Confirmation) []Confirmation {
	seen := map[string]bool{}
	var out []Confirmation
	for _, c := range cs {
		k := c.Candidate.File + "\x00" + c.Candidate.Fn + "\x00" + c.Candidate.Lens
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, c)
	}
	return out
}

func countDone(ds []DrainResult) int {
	n := 0
	for _, d := range ds {
		if d.Status == DrainDone {
			n++
		}
	}
	return n
}

// audit re-runs the frozen lens set over the (possibly modified) terrain files and reports
// what remains — a fix that removed the flagged shape leaves nothing; residuals are surfaced.
func (m *Machine) audit(ctx context.Context, md Mandate, baseline, accepted []enumerate.Lens, log logf) {
	er, err := enumerate.Run(md.Files, baseline, accepted)
	if err != nil {
		log(PhaseAudit, "re-hunt failed: %v", err)
		return
	}
	log(PhaseAudit, "frozen-lens re-hunt over %d file(s): %d candidate(s) remain", len(md.Files), len(er.Candidates))
	for _, c := range er.Candidates {
		log(PhaseAudit, "residual: %s:%d [%s] fn=%s", c.File, c.Line, c.Lens, c.Fn)
	}
}

// normalizeMode enforces the fail-closed three-way gate. The mechanical backstop is
// load-bearing: if the terrain shows ANY declared function/method (in-scope code), the mode
// is FIX regardless of what the model claimed — misclassifying a brownfield fix as ANSWER
// would skip the whole hunt with no downstream check. Only a mechanically-empty terrain can
// be ANSWER or BUILD.
func normalizeMode(g GateOut, ter *Terrain) (Mode, string) {
	if g.HasInscopeCode || len(ter.Funcs) > 0 {
		return ModeFix, "in-scope code present → fail-closed FIX"
	}
	switch strings.ToUpper(strings.TrimSpace(g.Mode)) {
	case "ANSWER":
		return ModeAnswer, ""
	case "BUILD":
		return ModeBuild, ""
	default:
		return ModeFix, "unrecognized mode → fail-closed FIX"
	}
}

func reasonOr(reason, justification string) string {
	if reason != "" {
		return reason
	}
	return justification
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func stubBackHalf(log func(Phase, string, ...any)) {
	for _, p := range backHalf {
		log(p, "SKELETON: not yet wired (step 5 wires the front half only)")
	}
}
