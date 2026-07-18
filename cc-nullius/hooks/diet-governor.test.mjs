// Behavioral tests for the diet governor. Run: node --test hooks/
// Each test pipes a synthetic PreToolUse payload into the hook and asserts
// on the decision JSON (or silent allow). Unique session ids keep the
// tmpdir ledger/ratchet state isolated per test.
import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { join, dirname } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";

const HOOK = join(dirname(fileURLToPath(import.meta.url)), "diet-governor.mjs");
const CWD = mkdtempSync(join(tmpdir(), "nullius-test-cwd-"));
let n = 0;
const sid = () => `nullius-test-${process.pid}-${n++}`;

function run(payload) {
  const res = spawnSync("node", [HOOK], {
    input: JSON.stringify({ cwd: CWD, session_id: sid(), ...payload }),
    encoding: "utf8", env: { ...process.env, NULLIUS_OFF: "" },
  });
  const decision = res.stdout.trim() ? JSON.parse(res.stdout).hookSpecificOutput : null;
  return { status: res.status, decision };
}
const bash = (cmd) => spawnSync("bash", ["-c", cmd], { encoding: "utf8" });

// ---- Bash rewrite ----------------------------------------------------------

test("rewrite survives a trailing semicolon (was: `{ x; ; }` syntax error)", () => {
  const { decision } = run({ tool_name: "Bash", tool_input: { command: "echo hi;" } });
  assert.ok(decision.updatedInput, "expected a rewrite");
  const r = bash(decision.updatedInput.command);
  assert.equal(r.status, 0, r.stderr);
  assert.match(r.stdout, /hi/);
});

test("commands with # comments are not rewritten (comment ate the closing brace)", () => {
  const { decision } = run({ tool_name: "Bash", tool_input: { command: "echo hi # done" } });
  assert.equal(decision, null, "must pass through unrewritten");
});

test("rewrite preserves the command's exit code (was: tail masked failures)", () => {
  const { decision } = run({ tool_name: "Bash", tool_input: { command: "false" } });
  assert.ok(decision.updatedInput);
  assert.equal(bash(decision.updatedInput.command).status, 1);
  const ok = run({ tool_name: "Bash", tool_input: { command: "true" } });
  assert.equal(bash(ok.decision.updatedInput.command).status, 0);
});

test("rewrite does not auto-approve: no permissionDecision alongside updatedInput", () => {
  const { decision } = run({ tool_name: "Bash", tool_input: { command: "echo hi" } });
  assert.ok(decision.updatedInput);
  assert.equal(decision.permissionDecision, undefined,
    "permissionDecision:allow would bypass the user's permission prompt");
});

test("grep -R (uppercase) and --recursive are denied as unbounded wide searches", () => {
  for (const cmd of ["grep -R foo /etc", "grep --recursive foo /etc"]) {
    const { decision } = run({ tool_name: "Bash", tool_input: { command: cmd } });
    assert.equal(decision?.permissionDecision, "deny", cmd);
  }
});

test("modern heavy runners are denied (vitest/jest/bun/deno/pip install)", () => {
  for (const cmd of ["vitest run", "jest --ci", "bun test", "deno test", "pip install requests"]) {
    const { decision } = run({ tool_name: "Bash", tool_input: { command: cmd } });
    assert.equal(decision?.permissionDecision, "deny", cmd);
  }
});

test("#nullius:ok escape still bypasses everything", () => {
  const { decision } = run({ tool_name: "Bash", tool_input: { command: "go test ./... #nullius:ok" } });
  assert.equal(decision, null);
});

// ---- Read gate ---------------------------------------------------------------

test("binary files are exempt from the whole-read line cap", () => {
  const p = join(CWD, "shot.png");
  const buf = Buffer.alloc(200_000);
  for (let i = 0; i < buf.length; i += 7) buf[i] = 10; // plenty of newline bytes
  buf[3] = 0;
  writeFileSync(p, buf);
  const { decision } = run({ tool_name: "Read", tool_input: { file_path: p } });
  assert.equal(decision, null, "a PNG is not a 28000-line file");
});

test("NUL-sniff exempts extensionless binaries too", () => {
  const p = join(CWD, "blob");
  const buf = Buffer.alloc(100_000, 10);
  buf[0] = 0;
  writeFileSync(p, buf);
  const { decision } = run({ tool_name: "Read", tool_input: { file_path: p } });
  assert.equal(decision, null);
});

test("whole read of a big text file is still denied", () => {
  const p = join(CWD, "big.txt");
  writeFileSync(p, "line\n".repeat(400));
  const { decision } = run({ tool_name: "Read", tool_input: { file_path: p } });
  assert.equal(decision?.permissionDecision, "deny");
});

test("duplicate read denied; deny reason names the narrower-range escape", () => {
  const p = join(CWD, "small.txt");
  writeFileSync(p, "a\n".repeat(10));
  const s = sid();
  const payload = { session_id: s, tool_name: "Read", tool_input: { file_path: p } };
  assert.equal(run(payload).decision, null);
  const second = run(payload).decision;
  assert.equal(second?.permissionDecision, "deny");
  assert.match(second.permissionDecisionReason, /offset|narrower|range/i,
    "after compaction the orchestrator needs a documented way out");
});

// ---- Edit/Write caps ---------------------------------------------------------

test("non-source Edit is exempt from the small-edit cap (docs edits stay on main thread)", () => {
  const { decision } = run({ tool_name: "Edit", tool_input: {
    file_path: join(CWD, "README.md"), old_string: "a\n".repeat(20), new_string: "b" } });
  assert.equal(decision, null);
});

test("defect-sized source Edit (20 lines) stays on the main thread", () => {
  // Measured (cc-nullius rep 1): a 12-line cap pushed 7 seeded-defect fixes
  // into sonnet craftsman dispatches — $4.58 of the $13.27 total.
  const { decision } = run({ tool_name: "Edit", tool_input: {
    file_path: join(CWD, "main.go"), old_string: "a\n".repeat(20), new_string: "b" } });
  assert.equal(decision, null);
});

test("big source Edit (>25 lines) still denied", () => {
  const { decision } = run({ tool_name: "Edit", tool_input: {
    file_path: join(CWD, "main.go"), old_string: "a\n".repeat(30), new_string: "b" } });
  assert.equal(decision?.permissionDecision, "deny");
});

test("tests-first ratchet: 4 source edits then deny until a test touch", () => {
  const s = sid();
  const edit = (f) => run({ session_id: s, tool_name: "Edit",
    tool_input: { file_path: join(CWD, f), old_string: "a", new_string: "b" } });
  for (let i = 0; i < 4; i++) assert.equal(edit("main.go").decision, null, `edit ${i}`);
  assert.equal(edit("main.go").decision?.permissionDecision, "deny");
  assert.equal(run({ session_id: s, tool_name: "Edit", tool_input: {
    file_path: join(CWD, "main_test.go"), old_string: "a", new_string: "b" } }).decision, null);
  assert.equal(edit("main.go").decision, null, "ratchet reset by test touch");
});

// ---- Subagents -----------------------------------------------------------------

test("non-craftsman subagents pass untouched", () => {
  const { decision } = run({ agent_type: "nullius-scout", agent_id: "a1",
    tool_name: "Grep", tool_input: { pattern: "x" } });
  assert.equal(decision, null);
});

test("craftsman: source edit denied before a test touch, allowed after; marker is session-scoped", () => {
  const s = sid();
  const src = { session_id: s, agent_type: "nullius-craftsman", agent_id: "craft1",
    tool_name: "Edit", tool_input: { file_path: join(CWD, "impl.go"), old_string: "a", new_string: "b" } };
  assert.equal(run(src).decision?.permissionDecision, "deny");
  assert.equal(run({ ...src, tool_input: { file_path: join(CWD, "impl_test.go"),
    old_string: "a", new_string: "b" } }).decision, null);
  assert.equal(run(src).decision, null, "test touched → source edit allowed");
  // a different craftsman in a different session must NOT inherit the marker
  assert.equal(run({ ...src, session_id: sid(), agent_id: "craft2" })
    .decision?.permissionDecision, "deny");
});

test("craftsman boundary gate: dropping an exported symbol is denied", () => {
  const { decision } = run({ agent_type: "nullius-craftsman", agent_id: "c9",
    tool_name: "Edit", tool_input: { file_path: join(CWD, "api.go"),
      old_string: "func AppendToHead(x int) {}", new_string: "// gone" } });
  assert.equal(decision?.permissionDecision, "deny");
  assert.match(decision.permissionDecisionReason, /AppendToHead/);
});

test("cleanup", () => { rmSync(CWD, { recursive: true, force: true }); });
