package machine

import (
	"fmt"
	"strings"

	"go-nullius/internal/enumerate"
)

// jsonGrammar is the canonical llama.cpp GBNF for well-formed JSON. It forces the weak
// model to emit parseable JSON of ANY shape; strict-decode (DisallowUnknownFields) then
// enforces the exact per-phase schema. Tightening the grammar to each phase's key set is
// a step-7 refinement — well-formed-JSON + strict-decode + retries is the v1 guarantee.
const jsonGrammar = `root   ::= object
value  ::= object | array | string | number | ("true" | "false" | "null") ws
object ::= "{" ws ( string ":" ws value ("," ws string ":" ws value)* )? "}" ws
array  ::= "[" ws ( value ("," ws value)* )? "]" ws
string ::= "\"" ( [^"\\] | "\\" (["\\/bfnrt] | "u" [0-9a-fA-F] [0-9a-fA-F] [0-9a-fA-F] [0-9a-fA-F]) )* "\"" ws
number ::= ("-"? ([0-9] | [1-9] [0-9]*)) ("." [0-9]+)? ([eE] [-+]? [0-9]+)? ws
ws ::= ([ \t\n] ws)?`

// Measured (2026-07-23): a STRICT per-phase GBNF for Recon backfires on the local reasoning
// models. Grammar constrains the reasoning_content channel too, so a rigid structure gags
// the model's deliberation — it burns the whole token budget without completing the JSON
// (270s → unexpected EOF). The permissive jsonGrammar above (thinking room) + a SIMPLE
// requested schema is what works; that is the lever, not a tighter grammar.

// OrientOut — Orient analyzes intent + terrain. It does NOT pick lenses (Recon derives
// them) and it does NOT rule the gate; it produces an intent read + risk note that sharpen
// the two phases after it.
type OrientOut struct {
	IntentSummary string   `json:"intent_summary"`
	FocusPkgs     []string `json:"focus_pkgs"`
	RiskNote      string   `json:"risk_note"`
}

// GateOut — the three-way ANSWER/BUILD/FIX classification. Fail-closed: any answer that is
// not a confident ANSWER/BUILD with no in-scope code collapses to FIX (normalizeMode).
type GateOut struct {
	Mode           string `json:"mode"`
	HasInscopeCode bool   `json:"has_inscope_code"`
	Justification  string `json:"justification"`
}

// ReconLens is one derived lens as the model emits it; it maps 1:1 onto enumerate.LensSpec.
type ReconLens struct {
	ID        string            `json:"id"`
	Template  string            `json:"template"`
	Params    map[string]string `json:"params"`
	FreeSCM   string            `json:"free_scm"`
	Mechanism string            `json:"mechanism"`
	Anchor    string            `json:"anchor"`
	Positive  string            `json:"positive"`
	Negative  string            `json:"negative"`
}

// ReconOut is Recon's full emission: the derived lens set (may be empty → baseline-only).
type ReconOut struct {
	Lenses []ReconLens `json:"lenses"`
}

func (r ReconOut) specs() []enumerate.LensSpec {
	out := make([]enumerate.LensSpec, 0, len(r.Lenses))
	for _, l := range r.Lenses {
		out = append(out, enumerate.LensSpec{
			ID:        l.ID,
			Template:  l.Template,
			Params:    l.Params,
			FreeSCM:   l.FreeSCM,
			Mechanism: l.Mechanism,
			Anchor:    l.Anchor,
			Positive:  l.Positive,
			Negative:  l.Negative,
		})
	}
	return out
}

func orientPrompt(task, digest string) string {
	return `You are the ORIENT phase of a code-analysis pipeline. Read the developer's task and the
mechanical terrain digest. Summarize the intent and name the single riskiest property to
check. Do NOT propose fixes or lenses. Reply with ONLY a JSON object of exactly this shape:
{"intent_summary": "<=200 chars", "focus_pkgs": ["pkg"], "risk_note": "<=200 chars, the riskiest property"}

TASK:
` + task + `

TERRAIN:
` + digest
}

func gatePrompt(task, digest string, o OrientOut) string {
	return `You are the GATE phase. Classify the task as exactly one mode:
- "ANSWER": a pure question; NO code will be changed AND there is no in-scope pre-existing code to analyze.
- "BUILD": greenfield; brand-new code only, no inherited in-scope code.
- "FIX": there is in-scope pre-existing code (any functions/types already present). This is the SAFE DEFAULT.
HARD RULE: if ANY in-scope code exists, you MUST answer "FIX". When unsure, answer "FIX".
Reply with ONLY a JSON object of exactly this shape:
{"mode": "ANSWER"|"BUILD"|"FIX", "has_inscope_code": true|false, "justification": "<=160 chars"}

TASK:
` + task + `

INTENT (from Orient): ` + o.IntentSummary + `

TERRAIN:
` + digest
}

func reconPrompt(task, digest, templateDoc string, o OrientOut) string {
	return `You are the RECON phase. Derive tree-sitter lenses that FIND the code sites relevant to the
task's risk. A lens is either a template-fill (set "template" to a listed id and fill "params")
or a free-form query (set "template":"" and provide "free_scm"). Lenses only ADD to an
always-on baseline, so deriving nothing is safe — derive only lenses you are confident in.
Each lens MUST carry witness snippets: "positive" (a tiny Go snippet the lens SHOULD match) and
"negative" (a snippet it must NOT match). A lens whose witnesses fail is dropped automatically.

` + templateDoc + `

Reply with ONLY a JSON object of exactly this shape (5 keys per lens, nothing else):
{"lenses": [{"id": "kebab-name", "template": "template-id", "params": {"hole": "value"},
 "positive": "package p\nfunc a(){ ... }", "negative": "package p\nfunc a(){ ... }"}]}
"params" is an OBJECT mapping each hole name to its string value (never an array). Use only
the hole names listed for the template you chose. Emit no other keys.

TASK:
` + task + `

RISK (from Orient): ` + o.RiskNote + `

TERRAIN (use ONLY node kinds that appear here):
` + digest
}

// templateDoc renders the registry's templates for the given language into the Recon prompt.
func templateDoc(r *enumerate.Registry, lang string) string {
	var b strings.Builder
	b.WriteString("Available templates:\n")
	for _, t := range r.Templates() {
		if lang != "" && t.Lang != lang {
			continue
		}
		fmt.Fprintf(&b, "  id=%q holes=%v mechanism=%q\n", t.ID, t.Holes, t.Mechanism)
	}
	b.WriteString("Each hole value is a regex matched inside a \"...\" predicate string (it cannot change query structure).\n")
	b.WriteString("To match a METHOD by name regardless of receiver (e.g. every `x.Lock()`), PREFER template\n")
	b.WriteString("\"method-call\" with method_regex=\"^Lock$\" — @method is the bare method name, so NO receiver prefix\n")
	b.WriteString("and NO parentheses go in the regex. Use \"call-to\" only for package-level or free functions.\n")
	b.WriteString("IMPORTANT — what fn_regex matches: for template \"call-to\"/\"call-with-literal-arg\", @fn is the ENTIRE called\n")
	b.WriteString("expression text. For a method call `recv.Method(...)` that text is \"recv.Method\" (WITH the receiver), and for\n")
	b.WriteString("a package call `pkg.Func(...)` it is \"pkg.Func\". So to match a method by name regardless of receiver use\n")
	b.WriteString("fn_regex=\"\\\\.Method$\" or \"Method\", NOT \"^Method$\" (which never matches because of the receiver prefix).\n")
	b.WriteString("Your positive witness MUST contain a real call_expression that @fn matches (e.g. `recv.Method()`), not a bare\n")
	b.WriteString("map index or assignment — those are not calls and will fail the witness.\n")
	b.WriteString("If no template fits, set \"template\":\"\" and write a full \"free_scm\" tree-sitter query.\n")
	return b.String()
}
