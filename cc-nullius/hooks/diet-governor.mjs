#!/usr/bin/env node
// nullius diet governor — PreToolUse hook.
//
// The main-thread context is the bill: every absorbed token is re-paid every
// following turn. Starves the ORCHESTRATOR; subagents pass untouched except
// nullius-craftsman (tests-first + boundary gates).
//
// Main thread: Grep/Glob/Web* denied (delegate). Read: no whole reads over
// NULLIUS_MAX_READ lines, no duplicate reads (path+range+mtime ledger). Edit:
// small only (NULLIUS_MAX_EDIT changed lines; export-reshaping edits exempt —
// boundary work is the orchestrator's at any size) + tests-first ratchet
// (every NULLIUS_EDITS_PER_TEST source edits require a test-file touch).
// Write: tests/scaffolding/skeletons free, implementation bulk capped. Bash:
// heavy commands and unbounded wide searches denied, the rest tail-bounded.
//
// Craftsman: first Edit/Write must touch a test file (scaffolding exempt);
// edits dropping exported symbols denied (boundary decisions escalate up).
//
// Escapes: NULLIUS_OFF=1, .nullius-off in cwd, `#nullius:ok` in a command.
import { readFileSync, writeFileSync, statSync, existsSync, appendFileSync } from "node:fs";
import { join, basename } from "node:path";
import { tmpdir } from "node:os";

// writeFileSync(1) not console.log: process.exit right after an async pipe
// write can truncate the decision JSON.
const out = (obj) => { try { writeFileSync(1, JSON.stringify(obj) + "\n"); } catch {} process.exit(0); };
const allow = () => process.exit(0);
const deny = (reason) => out({ hookSpecificOutput: {
  hookEventName: "PreToolUse", permissionDecision: "deny",
  permissionDecisionReason: reason } });
// No permissionDecision here: pairing updatedInput with "allow" would silently
// auto-approve the command, bypassing the user's permission prompt.
const rewrite = (updatedInput) => out({ hookSpecificOutput: {
  hookEventName: "PreToolUse", updatedInput } });

let data;
try { data = JSON.parse(readFileSync(0, "utf8")); } catch { allow(); }

const cwd = data.cwd || ".";
if (process.env.NULLIUS_OFF === "1" || existsSync(join(cwd, ".nullius-off"))) allow();

const tool = data.tool_name || "";
const ti = data.tool_input || {};
const MAX_READ = parseInt(process.env.NULLIUS_MAX_READ || "250", 10);
// 25 not 12: measured (cc-nullius rep 1 on vialite-todo), a 12-line cap
// routed 7 defect-sized fixes to sonnet craftsmen — $4.58 of a $13.27 run;
// the same fixes written by the leader are what the $6.17 winner did.
const MAX_EDIT = parseInt(process.env.NULLIUS_MAX_EDIT || "25", 10);
const MAX_WRITE = parseInt(process.env.NULLIUS_MAX_WRITE || "120", 10);
const EDITS_PER_TEST = parseInt(process.env.NULLIUS_EDITS_PER_TEST || "4", 10);
const TAIL_N = parseInt(process.env.NULLIUS_TAIL_LINES || "30", 10);

const nLines = (s) => (s ? s.split("\n").length : 0);
const isTestPath = (p) =>
  /(_test\.|\.test\.|\.spec\.|(^|\/)test_[^/]*$|(^|\/)tests?\/)/.test(p || "");
const isSourcePath = (p) =>
  /\.(go|ts|tsx|js|jsx|mjs|cjs|py|rs|java|c|cc|cpp|h|hpp|cs|rb|php|kt|swift|scala|ex|exs)$/.test(p || "");

// Names declared exported in oldS and absent from newS — an edit that drops
// public surface. (Measured: a run scored zero after a "cleanup" deleted a
// public method only hidden consumers used.)
function droppedExports(oldS, newS) {
  const names = new Set();
  const RES = [
    /^func\s+(?:\([^)]*\)\s*)?([A-Z]\w*)/gm,
    /^(?:type|var|const)\s+([A-Z]\w*)/gm,
    /^export\s+(?:default\s+)?(?:async\s+)?(?:function|class|const|let|var|interface|type|enum)\s*(\w+)?/gm,
    /^\s*pub\s+(?:async\s+)?(?:fn|struct|enum|trait|const|static|type)\s+(\w+)/gm,
    /^(?:public|protected)\s+[\w<>,\s[\]]+?\s(\w+)\s*\(/gm,
  ];
  for (const re of RES) for (const m of (oldS || "").matchAll(re)) if (m[1]) names.add(m[1]);
  return [...names].filter((n) => !(newS || "").includes(n));
}

// Export-decl density recognizes API-skeleton files (boundary work).
function exportDeclCount(s) {
  let n = 0;
  const RES = [
    /^func\s+(?:\([^)]*\)\s*)?[A-Z]\w*/gm, /^(?:type|var|const)\s+[A-Z]\w*/gm,
    /^export\s+/gm, /^\s*pub\s+/gm, /^(?:public|protected)\s+/gm,
  ];
  for (const re of RES) n += [...(s || "").matchAll(re)].length;
  return n;
}

// ---- craftsman gates --------------------------------------------------------
// agent_id, not agent_type, discriminates subagent calls: agent_type can be
// set on the MAIN thread in `claude --agent` sessions (per CLI docs).
if (data.agent_id && data.agent_type === "nullius-craftsman") {
  if (tool === "Edit" || tool === "Write") {
    const path = ti.file_path || "";
    if (tool === "Edit") {
      const dropped = droppedExports(ti.old_string, ti.new_string);
      if (dropped.length) deny(
        `nullius boundary gate: removing/renaming exported ${dropped.join(", ")} is the ` +
        "orchestrator's decision. Implement within the pinned mechanism, or report " +
        "needs-orchestrator-ruling naming the symbol.");
    }
    const marker = join(tmpdir(),
      `nullius-craft-${data.session_id || "nosession"}-${data.agent_id}`);
    if (isTestPath(path)) { try { appendFileSync(marker, "t\n"); } catch {} allow(); }
    if (!isSourcePath(path)) allow(); // scaffolding/config/docs: nothing to test
    if (!existsSync(marker)) deny(
      "nullius tests-first gate: touch a test file first — write/extend the failing " +
      "test for the pinned defect, quote its RED verbatim, then edit source.");
  }
  allow();
}

// Other subagents are throwaway read/verify contexts — never starved.
if (data.agent_id) allow();

// ---- main thread ------------------------------------------------------------
if (tool === "Grep" || tool === "Glob") deny(
  "nullius: sweeps are bulk. Dispatch nullius-scout or nullius-lens-hunter " +
  "(Agent tool), batched in parallel when independent.");

if (tool === "WebFetch" || tool === "WebSearch") deny(
  "nullius: dispatch nullius-scout with the URL/question; it returns a capped, " +
  "anchored report.");

// Tests-first ratchet state: [source edits since last test-touch].
const ratchet = join(tmpdir(), `nullius-ratchet-${data.session_id || "nosession"}`);
const rGet = () => { try { return parseInt(readFileSync(ratchet, "utf8"), 10) || 0; } catch { return 0; } };
const rSet = (n) => { try { writeFileSync(ratchet, String(n)); } catch {} };

if (tool === "Edit" || tool === "Write") {
  const path = ti.file_path || "";
  if (isTestPath(path)) { rSet(0); allow(); }               // tests: any size, resets ratchet
  if (tool === "Write" && !isSourcePath(path)) allow();     // scaffolding/config/docs

  if (isSourcePath(path) && rGet() >= EDITS_PER_TEST) deny(
    `nullius tests-first ratchet: ${rGet()} source edits since the last test touch. ` +
    "Write/extend the test that pins the behavior you just changed (a 3-line fix " +
    "that skips a lifecycle path is how regressions ship), then continue.");

  if (tool === "Edit") {
    // Non-source (docs/config) is exempt like Write; export-reshaping edits
    // are boundary work — the orchestrator's at any size.
    if (isSourcePath(path) && !droppedExports(ti.old_string, ti.new_string).length) {
      const changed = Math.max(nLines(ti.old_string), nLines(ti.new_string));
      if (changed > MAX_EDIT) deny(
        `nullius: ~${changed}-line edit (> ${MAX_EDIT}). Small surgical fixes are yours; ` +
        "dispatch nullius-craftsman for this one. (Export-reshaping edits are exempt.)");
    }
  } else {
    const len = nLines(ti.content);
    const skeleton = exportDeclCount(ti.content) >= len / 30;
    if (isSourcePath(path) && !skeleton && len > MAX_WRITE) deny(
      `nullius: ${len} lines of implementation (> ${MAX_WRITE}). Sketch the exported ` +
      "skeleton yourself (exempt) and dispatch nullius-craftsman for the bodies.");
  }
  if (isSourcePath(path)) rSet(rGet() + 1);
  allow();
}

if (tool === "Read") {
  const path = ti.file_path || "";
  let mtime = null, lines = null;
  const BIN_RE = /\.(png|jpe?g|gif|webp|bmp|ico|pdf|zip|gz|tgz|tar|7z|woff2?|ttf|otf|eot|mp[34]|wav|ogg|webm|avif|heic|bin|dat|db|sqlite3?|exe|dll|so|dylib|wasm|pyc|class|jar)$/i;
  try {
    const st = statSync(path);
    mtime = st.mtimeMs;
    if (st.isFile() && st.size < 10_000_000 && !BIN_RE.test(path)) {
      const buf = readFileSync(path);
      let nul = false;
      lines = 0;
      for (let i = 0; i < buf.length; i++) {
        if (buf[i] === 10) lines++;
        else if (buf[i] === 0) { nul = true; break; }
      }
      if (nul) lines = null; // binary: newline bytes are not lines
      else if (buf.length && buf[buf.length - 1] !== 10) lines++;
    }
  } catch { allow(); } // nonexistent/unreadable: let the tool error itself

  const bounded = ti.pages != null || (ti.limit != null && ti.limit <= MAX_READ);
  if (!bounded && lines != null && lines > MAX_READ) deny(
    `nullius: whole read of ${basename(path)} (${lines} lines > ${MAX_READ}). Read the ` +
    "decisive region (offset+limit) or dispatch nullius-scout to distill it.");

  const ledger = join(tmpdir(), `nullius-ledger-${data.session_id || "nosession"}`);
  const key = `${path}|${ti.offset ?? ""}|${ti.limit ?? ""}|${ti.pages ?? ""}|${mtime}`;
  try {
    const seen = existsSync(ledger) ? readFileSync(ledger, "utf8").split("\n") : [];
    if (seen.includes(key)) deny(
      "nullius: you already read exactly this range and it is unchanged. Use what " +
      "your context holds; if compaction dropped it, re-read a narrower decisive " +
      "range with offset+limit (a different range is a new ledger key).");
    appendFileSync(ledger, key + "\n");
  } catch { /* ledger is best-effort */ }
  allow();
}

if (tool === "Bash") {
  const cmd = ti.command || "";
  if (cmd.includes("#nullius:ok")) allow();

  const HEAVY_RE = /\b(go\s+(test|build|vet)|npm\s+(test|run|ci|install)|pnpm|yarn|pytest|vitest|jest|bun\s+(test|install|run)|deno\s+(test|task|check)|pip3?\s+install|uv\s+(sync|run|pip)|cargo\s+(test|build|check|clippy)|make\b|tsc\b|eslint|ruff|mypy|mvn\b|gradle|dotnet\s+(test|build)|ctest)\b/;
  if (HEAVY_RE.test(cmd)) deny(
    "nullius: builds/tests flood the orchestrator. Dispatch nullius-scout " +
    "(quick check or close-out record). #nullius:ok only if it truly must run here.");

  const WIDE_RE = /\b(grep\s+(-[a-zA-Z]*[rR]\b|--recursive)|rg\b|find\s+\.|find\s+\/|ag\b)/;
  const BOUND_RE = /\|\s*(tail|head)\b|\bwc\b|-l\b|--count|--files-with-matches|-m\s*\d/;
  if (WIDE_RE.test(cmd) && !BOUND_RE.test(cmd)) deny(
    "nullius: unbounded wide search. Delegate to nullius-scout or bound it " +
    "(| head -n 20 / -l / --count).");

  // Trailing `;` would make `{ x; ; }` a syntax error; a `#` comment would
  // swallow the closing `; }`; `exit "${PIPESTATUS[0]}"` keeps the command's
  // real exit code (tail alone reports success for failed commands).
  const trimmed = cmd.replace(/[;\s]+$/, "");
  if (trimmed && !trimmed.includes("\n") && !trimmed.includes("#") &&
      !BOUND_RE.test(trimmed) && trimmed.length < 500 &&
      !/<<|\bfor\b|\bwhile\b|\bif\b|\bcd\b|\bexport\b|&$/.test(trimmed)) {
    rewrite({ command:
      `{ ${trimmed} ; } 2>&1 | tail -n ${TAIL_N}; exit "\${PIPESTATUS[0]}"` });
  }
}

allow();
