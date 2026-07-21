// Tests for ctx-sentinel.mjs — the PostToolUse context-size nudge.
// Run: node --test ctx-sentinel.test.mjs
import { test } from "node:test";
import assert from "node:assert";
import { spawnSync } from "node:child_process";
import { writeFileSync, rmSync, mkdtempSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";

const HOOK = new URL("./ctx-sentinel.mjs", import.meta.url).pathname;
let n = 0;
const sid = () => `ctxsent-test-${process.pid}-${++n}`;

// Build a fake transcript whose last usage line implies `ctx` tokens.
function fakeTranscript(dir, ctx) {
  const p = join(dir, "transcript.jsonl");
  const usage = {
    input_tokens: 2,
    cache_creation_input_tokens: 100,
    cache_read_input_tokens: ctx - 102,
    output_tokens: 500,
  };
  const lines = [
    JSON.stringify({ type: "user", message: { content: "hi" } }),
    JSON.stringify({ type: "assistant", message: { usage: { input_tokens: 1, cache_read_input_tokens: 10, cache_creation_input_tokens: 1, output_tokens: 5 } } }),
    JSON.stringify({ type: "assistant", message: { usage } }),
  ];
  writeFileSync(p, lines.join("\n") + "\n");
  return p;
}

function run(payload) {
  const res = spawnSync("node", [HOOK], {
    input: JSON.stringify({ cwd: process.cwd(), ...payload }),
    encoding: "utf8",
    env: { ...process.env, NULLIUS_OFF: "" },
  });
  const out = res.stdout.trim() ? JSON.parse(res.stdout).hookSpecificOutput : null;
  return { status: res.status, out };
}

test("below knee: silent allow", () => {
  const dir = mkdtempSync(join(tmpdir(), "ctxsent-"));
  const session_id = sid();
  const { status, out } = run({ session_id, transcript_path: fakeTranscript(dir, 60_000) });
  assert.equal(status, 0);
  assert.equal(out, null);
  rmSync(dir, { recursive: true, force: true });
});

test("above knee: nudges with PostToolUse additionalContext naming the size", () => {
  const dir = mkdtempSync(join(tmpdir(), "ctxsent-"));
  const session_id = sid();
  const { status, out } = run({ session_id, transcript_path: fakeTranscript(dir, 150_000) });
  assert.equal(status, 0);
  assert.ok(out, "expected a nudge");
  assert.equal(out.hookEventName, "PostToolUse");
  assert.match(out.additionalContext, /146k|147k|150k|14[0-9]k/, "names the ctx size in k");
  assert.match(out.additionalContext, /compact/i, "points at compaction");
  assert.ok(out.additionalContext.length < 700, "nudge stays lean");
  rmSync(dir, { recursive: true, force: true });
});

test("same band: nudges once, then silent", () => {
  const dir = mkdtempSync(join(tmpdir(), "ctxsent-"));
  const session_id = sid();
  const t = fakeTranscript(dir, 150_000);
  assert.ok(run({ session_id, transcript_path: t }).out, "first crossing nudges");
  assert.equal(run({ session_id, transcript_path: t }).out, null, "same band is silent");
  rmSync(dir, { recursive: true, force: true });
});

test("next band (+32k): nudges again", () => {
  const dir = mkdtempSync(join(tmpdir(), "ctxsent-"));
  const session_id = sid();
  assert.ok(run({ session_id, transcript_path: fakeTranscript(dir, 150_000) }).out);
  assert.equal(run({ session_id, transcript_path: fakeTranscript(dir, 155_000) }).out, null);
  assert.ok(run({ session_id, transcript_path: fakeTranscript(dir, 185_000) }).out, "crossing +32k band re-nudges");
  rmSync(dir, { recursive: true, force: true });
});

test("missing/garbled transcript: fail-open silent", () => {
  const dir = mkdtempSync(join(tmpdir(), "ctxsent-"));
  const session_id = sid();
  assert.equal(run({ session_id, transcript_path: join(dir, "nope.jsonl") }).out, null);
  const bad = join(dir, "bad.jsonl");
  writeFileSync(bad, "not json at all\n{broken\n");
  assert.equal(run({ session_id: sid(), transcript_path: bad }).out, null);
  rmSync(dir, { recursive: true, force: true });
});

test("subagent context (agent_id set): silent", () => {
  const dir = mkdtempSync(join(tmpdir(), "ctxsent-"));
  const { out } = run({ session_id: sid(), transcript_path: fakeTranscript(dir, 150_000), agent_id: "a123" });
  assert.equal(out, null);
  rmSync(dir, { recursive: true, force: true });
});

test("NULLIUS_OFF: silent even above knee", () => {
  const dir = mkdtempSync(join(tmpdir(), "ctxsent-"));
  const res = spawnSync("node", [HOOK], {
    input: JSON.stringify({ cwd: process.cwd(), session_id: sid(), transcript_path: fakeTranscript(dir, 150_000) }),
    encoding: "utf8",
    env: { ...process.env, NULLIUS_OFF: "1" },
  });
  assert.equal(res.stdout.trim(), "");
  rmSync(dir, { recursive: true, force: true });
});

test("NULLIUS_CTX_KNEE override lowers the threshold", () => {
  const dir = mkdtempSync(join(tmpdir(), "ctxsent-"));
  const res = spawnSync("node", [HOOK], {
    input: JSON.stringify({ cwd: process.cwd(), session_id: sid(), transcript_path: fakeTranscript(dir, 60_000) }),
    encoding: "utf8",
    env: { ...process.env, NULLIUS_OFF: "", NULLIUS_CTX_KNEE: "50000" },
  });
  const out = res.stdout.trim() ? JSON.parse(res.stdout).hookSpecificOutput : null;
  assert.ok(out, "60k > 50k knee should nudge");
  rmSync(dir, { recursive: true, force: true });
});
