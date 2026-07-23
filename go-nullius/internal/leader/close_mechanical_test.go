package leader

import (
	"context"
	"strings"
	"testing"
)

// runCloseCmd must capture the REAL exit code and the command output — the
// whole point of the mechanical close is a truthful verbatim record, so a
// nonzero exit can never read as success.
func TestRunCloseCmdCapturesExitAndOutput(t *testing.T) {
	out, code := runCloseCmd(context.Background(), t.TempDir(), "echo hello")
	if code != 0 || !strings.Contains(out, "hello") {
		t.Fatalf("echo: code=%d out=%q, want 0 + hello", code, out)
	}
	if _, code := runCloseCmd(context.Background(), t.TempDir(), "exit 3"); code != 3 {
		t.Fatalf("exit 3: code=%d, want 3", code)
	}
}

// MechanicalClose always produces a record containing the language-agnostic
// git/0-byte records, regardless of project type — it must never return
// silence, which is the broken-build-undetected failure mode.
func TestMechanicalCloseAlwaysRecords(t *testing.T) {
	rec := MechanicalClose(context.Background(), t.TempDir())
	for _, must := range []string{"MECHANICAL CLOSE-OUT RECORD", "$ git status --porcelain", "$ git diff --stat"} {
		if !strings.Contains(rec, must) {
			t.Errorf("close record missing %q:\n%s", must, rec)
		}
	}
}
