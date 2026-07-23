package api

import (
	"context"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// The prompt-size wall refuses an oversized request BEFORE sending it —
// turning the unbounded prompt-assembly blowup (measured: a 2.86M-token
// send) into a clean, logged abort instead of a runaway request.
func TestStreamRefusesOversizedPrompt(t *testing.T) {
	c := New("http://127.0.0.1:0", "x", "m")       // never dialed — the wall fires first
	huge := strings.Repeat("x", maxPromptTokens*5) // ~5*max bytes → est well over the wall
	params := anthropic.MessageNewParams{
		System: []anthropic.TextBlockParam{{Text: huge}},
	}
	_, err := c.Stream(context.Background(), params, nil)
	if err == nil || !strings.Contains(err.Error(), "prompt-size wall") {
		t.Fatalf("oversized prompt: err=%v, want a prompt-size wall refusal", err)
	}
}

// A request carrying no explicit MaxTokens gets a sane default cap, so a
// server that ignores an absent limit cannot generate unbounded.
func TestToRequestDefaultsMaxTokens(t *testing.T) {
	c := New("http://x", "x", "m")
	req := c.toRequest(anthropic.MessageNewParams{}) // no MaxTokens set
	if req.MaxTokens != defaultMaxTokens {
		t.Fatalf("MaxTokens=%d, want default %d", req.MaxTokens, defaultMaxTokens)
	}
}
