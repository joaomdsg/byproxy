package machine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// drainStepTimeout bounds one craftsman write (the fast model can hang).
const drainStepTimeout = 4 * time.Minute

// Writer performs ONE code change described by an objective, inside dir, and returns its
// output. The production implementation is SubprocessCraftsman (a --craftsman subprocess);
// tests supply scripted writers. A nil Writer on the Machine disables Drain entirely.
type Writer interface {
	Write(ctx context.Context, dir, objective string) (string, error)
}

// SubprocessCraftsman shells out to a go-nullius --craftsman subprocess (fast tier,
// Read/Edit/Bash, one pinned change). This is the design's write-absorber: the ONE place a
// model writes, kept in a throwaway process so its bulk never enters the orchestrator.
type SubprocessCraftsman struct {
	Bin   string   // path to the go-nullius binary
	Model string   // fast-tier model alias/id
	Dir   string   // workspace dir passed as --dir (defaults to the drain dir)
	Env   []string // extra env; NULLIUS_TRACE=1 is always added
}

func (c SubprocessCraftsman) Write(ctx context.Context, dir, objective string) (string, error) {
	d := c.Dir
	if d == "" {
		d = dir
	}
	cmd := exec.CommandContext(ctx, c.Bin, "-p", objective, "--model", c.Model, "--craftsman", "--dir", d)
	cmd.Env = append(append(os.Environ(), c.Env...), "NULLIUS_TRACE=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("craftsman: %v", err)
	}
	return string(out), nil
}

// DrainStatus is the terminal disposition of one drained plan.
type DrainStatus string

const (
	DrainDone   DrainStatus = "DONE"
	DrainFailed DrainStatus = "FAILED"
)

// DrainResult is the mechanically-verified outcome of draining one plan. Every field is a
// machine record, never a craftsman claim.
type DrainResult struct {
	Plan     FixPlan
	Status   DrainStatus
	Diffstat string
	Detail   string
	Attempts int
}

// drain runs each plan through drainOne sequentially (a shared checkout — concurrent
// same-tree writers would lose updates). Returns one result per plan.
func (m *Machine) drain(ctx context.Context, dir string, plans []FixPlan, log logf) []DrainResult {
	out := make([]DrainResult, 0, len(plans))
	for _, p := range plans {
		out = append(out, m.drainOne(ctx, dir, p, log))
	}
	return out
}

// drainOne is the safety net (nullius in verba — the craftsman's DONE is never taken on
// faith): snapshot → write → the diff must be non-empty → go build → touched-package -race
// → any failure reverts and retries once with the failure fed back → a second failure marks
// FAILED and leaves the tree reverted. The worktree is always restored on failure.
func (m *Machine) drainOne(ctx context.Context, dir string, p FixPlan, log logf) DrainResult {
	dr := DrainResult{Plan: p}
	base := craftObjective(p)
	feedback := ""
	for attempt := 1; attempt <= 2; attempt++ {
		dr.Attempts = attempt
		snap, err := gitSnapshot(ctx, dir)
		if err != nil {
			dr.Status, dr.Detail = DrainFailed, "snapshot: "+err.Error()
			return dr
		}

		stepCtx, cancel := context.WithTimeout(ctx, drainStepTimeout)
		_, werr := m.Craftsman.Write(stepCtx, dir, withFeedback(base, feedback))
		cancel()
		if werr != nil {
			feedback = "the previous attempt errored: " + firstLine(werr.Error())
			_ = snap.revert(ctx)
			dr.Detail = feedback
			continue
		}

		changed, stat, _ := snap.changed(ctx)
		if !changed {
			feedback = "your previous attempt wrote NOTHING; make the actual edit"
			dr.Detail = "empty diff (wrote nothing)"
			continue // nothing to revert
		}
		dr.Diffstat = stat

		if bout, err := goRun(ctx, dir, "build", "./..."); err != nil {
			feedback = "the build failed: " + firstLine(bout)
			_ = snap.revert(ctx)
			dr.Detail = "build failed: " + firstLine(bout)
			log(PhaseDrain, "%s attempt %d: build failed, reverted", short(p), attempt)
			continue
		}
		pkgs := snap.changedPkgs(ctx)
		if tout, err := goRun(ctx, dir, append([]string{"test", "-race", "-count=1"}, pkgs...)...); err != nil {
			feedback = "the tests failed: " + firstLine(tout)
			_ = snap.revert(ctx)
			dr.Detail = "tests failed: " + firstLine(tout)
			log(PhaseDrain, "%s attempt %d: tests failed, reverted", short(p), attempt)
			continue
		}

		dr.Status, dr.Detail = DrainDone, "verified: build + "+strings.Join(pkgs, " ")+" -race"
		log(PhaseDrain, "%s DONE (%s)", short(p), stat)
		return dr
	}
	dr.Status = DrainFailed
	log(PhaseDrain, "%s FAILED after %d attempts: %s (reverted)", short(p), dr.Attempts, dr.Detail)
	return dr
}

func craftObjective(p FixPlan) string {
	c := p.Confirmation.Candidate
	return "Fix ONE defect and add its pinning test, then stop. The judgment is already made — execute it, do not re-decide scope.\n" +
		"TARGET: " + p.Plan.Target + " (" + c.File + ":" + itoa(c.Line) + ", function " + c.Fn + ")\n" +
		"FIX: " + p.Plan.Intent + "\n" +
		"ADD TEST: " + p.Plan.TestName + " — " + p.Plan.TestSketch + "\n" +
		"Make the MINIMAL change. Do not touch anything unrelated."
}

func withFeedback(base, feedback string) string {
	if feedback == "" {
		return base
	}
	return base + "\n\nRETRY — " + feedback
}

func short(p FixPlan) string {
	c := p.Confirmation.Candidate
	return filepath.Base(c.File) + ":" + itoa(c.Line)
}

func goRun(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// --- git snapshot / revert (mirrors internal/leader/gitstep; the machine needs its own,
// as those are unexported there). Tracked state is captured via `git stash create` (a
// dangling commit, worktree untouched); untracked files by set difference. ---

type gitSnap struct {
	dir       string
	commit    string
	untracked map[string]bool
}

func gitCmd(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, firstLine(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func gitSnapshot(ctx context.Context, dir string) (*gitSnap, error) {
	commit, err := gitCmd(ctx, dir, "stash", "create")
	if err != nil {
		return nil, err
	}
	if commit == "" {
		if commit, err = gitCmd(ctx, dir, "rev-parse", "HEAD"); err != nil {
			return nil, err
		}
	}
	unt, err := gitUntracked(ctx, dir)
	if err != nil {
		return nil, err
	}
	return &gitSnap{dir: dir, commit: commit, untracked: unt}, nil
}

func gitUntracked(ctx context.Context, dir string) (map[string]bool, error) {
	out, err := gitCmd(ctx, dir, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, f := range strings.Split(out, "\n") {
		if f = strings.TrimSpace(f); f != "" {
			set[f] = true
		}
	}
	return set, nil
}

func (s *gitSnap) changed(ctx context.Context) (bool, string, error) {
	stat, err := gitCmd(ctx, s.dir, "diff", "--stat", s.commit)
	if err != nil {
		return false, "", err
	}
	fresh := s.freshFiles(ctx)
	if len(fresh) > 0 {
		stat = strings.TrimSpace(stat + "\n new: " + strings.Join(fresh, ", "))
	}
	return stat != "", stat, nil
}

// changedPkgs returns the "./dir/..." package patterns touched since the snapshot, so the
// verification -race run targets only what changed. Empty → the whole module.
func (s *gitSnap) changedPkgs(ctx context.Context) []string {
	files := []string{}
	if out, err := gitCmd(ctx, s.dir, "diff", "--name-only", s.commit); err == nil {
		for _, f := range strings.Split(out, "\n") {
			if f = strings.TrimSpace(f); f != "" {
				files = append(files, f)
			}
		}
	}
	files = append(files, s.freshFiles(ctx)...)
	seen := map[string]bool{}
	var pkgs []string
	for _, f := range files {
		if !strings.HasSuffix(f, ".go") {
			continue
		}
		d := filepath.Dir(f)
		pat := "./..."
		if d != "." && d != "" {
			pat = "./" + d + "/..."
		}
		if !seen[pat] {
			seen[pat] = true
			pkgs = append(pkgs, pat)
		}
	}
	if len(pkgs) == 0 {
		return []string{"./..."}
	}
	return pkgs
}

func (s *gitSnap) freshFiles(ctx context.Context) []string {
	now, err := gitUntracked(ctx, s.dir)
	if err != nil {
		return nil
	}
	var fresh []string
	for f := range now {
		if !s.untracked[f] {
			fresh = append(fresh, f)
		}
	}
	return fresh
}

func (s *gitSnap) revert(ctx context.Context) error {
	if _, err := gitCmd(ctx, s.dir, "checkout", s.commit, "--", "."); err != nil {
		return err
	}
	for _, f := range s.freshFiles(ctx) {
		if err := os.Remove(filepath.Join(s.dir, f)); err != nil {
			return err
		}
	}
	return nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
