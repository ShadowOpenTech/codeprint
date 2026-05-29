---
description: Verify byte-identical output and pinpoint any nondeterminism source
---

Guard codeprint's #1 invariant: byte-identical `fingerprint` for identical input (NFR-2, `docs/architecture/data-model.md`).

1. Run `Scan` on the correctness corpus **twice**, and again with `WithConcurrency(1)` vs `WithConcurrency(N)`. Diff the `fingerprint` payloads (exclude `_meta` — it is intentionally volatile).

2. If all identical, report PASS and stop.

3. If any differ, find the source. The usual culprits, in order of likelihood:
   - **Map ranged directly into output** — the cardinal sin. Search for `for ... := range <map>` feeding any field that reaches the fingerprint. Fix: collect into a slice, sort by the documented stable key.
   - **Missing/incorrect sort** before emit (check the sort-key table in `docs/architecture/data-model.md`).
   - **Concurrency leaking ordering** — results must funnel through the single collector, sorted there, not appended in completion order.
   - **Float formatting** — `percent` must derive from sorted int counts and round consistently.
   - **Timestamps/volatile data** leaking into `fingerprint` (they belong only in `_meta`).

4. Report the offending field + file:line and the fix. Re-run to confirm PASS.
