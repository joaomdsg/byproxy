package machine

import (
	"strings"
	"testing"
)

// coherenceFake wires FIX + calls-helper, with a Judge responder the test supplies.
func coherenceFake(judge func(string) (string, bool)) fakeCaller {
	o, g, r := reconCallsHelper()
	return fakeCaller{orient: o, gate: g, recon: r, custom: judge}
}

func TestOffLensRulingIsNotConfirmed(t *testing.T) {
	// The calls-helper candidate for fa() sits at line 6 (its only Evidence line). The judge
	// rules DEFECT but cites line 5 (`func fa() {`) — in the shown window but NOT implicated by
	// the lens. Coherence filter 1b must downgrade it to CANT_TELL (the crypto.go FP shape).
	f := writeGo(t, "a.go", pairSrc)
	fc := coherenceFake(func(p string) (string, bool) {
		switch {
		case strings.Contains(p, "JUDGE phase"):
			return `{"answer":"DEFECT","decisive_line":5,"because":"off-lens claim"}`, true
		case strings.Contains(p, "REFUTE phase"):
			return `{"stands":true,"refuting_line":null}`, true
		}
		return "", false
	})
	res := run(t, fc, []string{f}, "audit")
	if countConfirmed(res) != 0 {
		t.Fatalf("an off-lens decisive line must NOT confirm; got %d confirmed", countConfirmed(res))
	}
}

func TestOnLensRulingConfirms(t *testing.T) {
	// Control: same setup, decisive line 6 (the flagged call itself) → confirmed.
	f := writeGo(t, "a.go", pairSrc)
	fc := coherenceFake(func(p string) (string, bool) {
		switch {
		case strings.Contains(p, "JUDGE phase"):
			return `{"answer":"DEFECT","decisive_line":6,"because":"the flagged call"}`, true
		case strings.Contains(p, "REFUTE phase"):
			return `{"stands":true,"refuting_line":null}`, true
		}
		return "", false
	})
	res := run(t, fc, []string{f}, "audit")
	if countConfirmed(res) < 1 {
		t.Fatalf("an on-lens decisive line should confirm; got %d", countConfirmed(res))
	}
}

func TestOffLensNoteRecordedAsLead(t *testing.T) {
	f := writeGo(t, "a.go", pairSrc)
	fc := coherenceFake(func(p string) (string, bool) {
		switch {
		case strings.Contains(p, "JUDGE phase"):
			return `{"answer":"CORRECT","decisive_line":6,"because":"fine","off_lens_note":"nearby nil deref maybe"}`, true
		}
		return "", false
	})
	res := run(t, fc, []string{f}, "audit")
	var sawLead bool
	for _, c := range res.Judged {
		if strings.Contains(c.Note, "off-lens lead") && strings.Contains(c.Note, "nil deref") {
			sawLead = true
		}
	}
	if !sawLead {
		t.Fatalf("off_lens_note should be recorded as a lead; judged: %+v", res.Judged)
	}
}
