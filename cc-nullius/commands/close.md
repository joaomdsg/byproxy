---
description: Run the nullius close-out protocol — scout verification record, public-surface regression check, qualified report.
---

Close out the current nullius task:

1. Confirm every checklist item has an explicit disposition (FIXED /
   REFUTED / out-of-mandate). A confirmed defect left as a disclosure is
   a failed run — fix it before closing. A silently dropped suspect is a
   failed close.
2. Dispatch ONE `nullius-scout` to produce the close record: **verify** —
   full test suite + build + vet + the project's linters,
   `-race`/equivalent where concurrency was touched, decisive output
   VERBATIM; **surface** — `git diff` against the base revision, every
   removed or signature-changed exported/public symbol listed verbatim.
3. You rule on the record: any failure, and any surface change you did
   not decide by name, goes back into the loop — never rationalized into
   RISKS. A failed close gets a fresh scout after fixes.
4. Report to the user: STATUS / FACTS (each with its quoted evidence or
   the scout's verbatim record) / RISKS (with reasons) / UNKNOWN /
   ASSUMED (every self-answered question). Never unqualified success.
