# codeprint — Security Audit

**Purpose:** threat model and security findings for a tool that processes untrusted code from ~18k repos.
**Phase:** 4.5 Security Check
**Status:** confirmed
**Updated:** 2026-05-30

## Posture summary

codeprint's attack surface is inherently small: **produce-only, offline (no network, build-time verified), pure Go (no CGo), reads file content but never emits it** (spec NFR-8). The real risk surface is **processing adversarial repository input** — a malicious repo trying to make codeprint read host files, corrupt a terminal, exhaust resources, or leak data.

Findings below are practical, not theater. Most confirm or strengthen existing NFRs; two change behavior (symlink containment, control-char sanitization).

## Trust boundary

```
UNTRUSTED: the workspace tree (arbitrary code from any of ~18k repos)
TRUSTED:   the codeprint binary, its flags, the --out destination
```

Everything inside the workspace is attacker-controlled. codeprint must treat file names, file contents, manifest contents, and symlink targets as hostile.

## Findings

| # | Risk | Severity | Status | Mitigation |
|---|---|---|---|---|
| 1 | Symlink escapes workspace (→ `/etc/passwd`, `~/.ssh`, sibling repos) | Medium | **Mitigated (amends NFR-8)** | Follow symlinks only while resolved path stays inside the workspace root; skip escapes with a `--debug` line |
| 2 | Terminal injection via malicious filename (ANSI escapes in pretty/progress/`--verbose`) | Medium | **Mitigated** | Sanitize/escape control characters in all human (TTY) output. JSON path already safe via stdlib encoder |
| 3 | Traversal DoS — symlink DAG re-walk; deep nesting stack overflow | Medium | Mitigated | Visited-set keyed on **resolved** path; **iterative** walk (no recursion → no stack blowup). Strengthens NFR-8 depth-40 |
| 4 | Giant-file / newline-less megaline OOM | Low | Mitigated | `WithMaxFileSize` 10MB cap (NFR-4) bounds read before processing |
| 5 | ReDoS on adversarial manifests (Gradle Groovy regex, minified heuristic) | None | By construction | Go `regexp` is **RE2** — linear time, no catastrophic backtracking. No third-party regex engines permitted |
| 6 | Content / secret leakage into output | Low | Mitigated | NFR-8: contents read, never emitted. `errors[].reason` = OS error only, never file bytes. Logs never echo content |
| 7 | Supply-chain (enry, gocloc, cobra, …) | Low | Managed | Pinned versions + `go.sum`; `govulncheck` in CI; minimal deps; licenses cleared (feasibility — Apache-2.0/MIT) |
| 8 | Memory-safety (native overflow) | None | By construction | Pure Go, `CGO_ENABLED=0` (NFR-3) — no native code surface |
| 9 | Privilege / write surface | Low | Mitigated | Runs as CI user, read-only; sole write is `--out` (NFR-8); no setuid; no network |

## Detailed decisions

### Finding 1 — Symlink containment (amends NFR-8)

**Decision:** codeprint follows a symlink **only while its resolved target stays within the workspace root**. A symlink whose target escapes the root is **not followed** — it is skipped and noted under `--debug`. In-tree symlinks are still followed (loop detection preserved).

**Rationale:** prevents a crafted repo from steering codeprint to stat/read sensitive host files (`/etc/passwd`, SSH keys, other repos on the runner). Content is never emitted regardless (NFR-8), but eliminating the out-of-tree read is the conservative, low-regret choice.

**Tradeoff accepted:** a repo that legitimately symlinks to shared code *outside* its tree will have that code skipped (under-count). Judged acceptable — such layouts are rare and a flat per-repo fingerprint shouldn't reach outside its own root anyway.

**Amends spec NFR-8**, which previously said "symlinks followed (loop detection, max depth 40)" without a containment rule.

### Finding 2 — Control-character sanitization

All human-facing output (pretty summary, progress spinner, `--verbose`/`--debug` lines, error messages) escapes non-printable / control characters in attacker-controlled strings (filenames, manifest-derived keywords) before writing to a terminal. The JSON encoder (stdlib) already escapes these for the machine path; this closes the human path. Baked into `emit/` ([system-architecture.md](./system-architecture.md)).

## CI security gates

- `govulncheck` on every build — fail on known vulns in the dep tree.
- Network-disabled build/test (NFR-8) — proves zero network calls.
- `CGO_ENABLED=0` enforced in the release build.
- Dependency license check stays green (Apache-2.0 / MIT only).

These wire into the Phase 7 `CLAUDE.md` enforcement (`/sec-audit`).

## Out of scope (by product definition)

codeprint does **not** do CVE lookup, secret scanning, SAST, or any security *findings* — it is a fingerprint producer (ideation scope boundary). Those are downstream consumers' jobs. This audit covers codeprint's own safety as a tool, not security analysis of the repos it scans.

## Change log

- 2026-05-30: initial draft; confirmed. 9-finding threat model. Symlink containment decided (amends NFR-8); control-char sanitization for TTY output; iterative walk + resolved-path visited-set; RE2/no-CGo noted as by-construction safety; govulncheck + network-disabled build as CI gates.
