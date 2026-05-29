# codeprint — Error Strategy

**Purpose:** define error categories, behavior, user-facing messages, and exit-code mapping.
**Phase:** 4.4 Error Strategy
**Status:** confirmed
**Updated:** 2026-05-30

## Principle

Errors are categorized by **blast radius**, not by source. The spine is set by spec NFR-5 (error tolerance) and NFR-6 (exit codes); this doc makes the behavior concrete. Audience is engineers and scanners — messages are terse and actionable, never decorative.

## Three tiers

| Tier | Examples | Behavior | Library returns | CLI exit |
|---|---|---|---|---|
| **Fatal (pre-scan)** | bad flag/arg; workspace root missing or unreadable | abort, nothing produced | `(nil, err)` | 64 usage / 66 no-input |
| **Non-fatal (per-file)** | a file unreadable mid-scan; permission denied on one file | record in `fingerprint.errors[]`, continue | `(*Fingerprint, nil)` | 0 (74 with `--strict-io`) |
| **Soft / degradation** | malformed manifest; non-UTF-8 file; oversized file | degrade gracefully, no error | `(*Fingerprint, nil)` | 0 |

### `errors[]` has a strict meaning

`fingerprint.errors[]` means exactly: **"I attempted to read this file and the OS refused"** (`{path, reason}`, data-model 4.2). Nothing softer goes in it. This keeps the field meaningful so consumers don't learn to ignore it.

### Soft-degradation rules (what does NOT go in errors[])

| Condition | Handling |
|---|---|
| Malformed manifest (F-4 framework parse fails) | that ecosystem yields no framework hint; a line under `--debug`; **not** in `errors[]`. F-4 is best-effort — junk `package.json` is not a scan failure |
| Non-UTF-8 file (NFR-10) | classified `binary`; not an error |
| Oversized file (NFR-4) | counted, `flags: ["skipped_large"]`; not an error |
| Symlink loop hit (NFR-8, depth 40) | traversal stops following; `--debug` line; not an error |

## Fatal errors (typed, exported)

```go
var ErrUsage          = errors.New("codeprint: invalid option")
var ErrUnreadableRoot = errors.New("codeprint: workspace root unreadable")
```

- Wrapped with `fmt.Errorf("%w: %s", ...)` so consumers use `errors.Is`.
- The CLI maps these to sysexits codes. Library callers inspect the error themselves.

## Context cancellation (4.3 ctx)

- On `ctx` cancellation/timeout mid-scan, `Scan` returns **`(nil, ctx.Err())`** — a clean abort.
- Rationale: a cancelled scan is incomplete. A partial `*Fingerprint` would be indistinguishable from a complete one if the caller ignored the error, and it would violate determinism (missing files look like they don't exist). Returning `nil` forces the caller to handle "no valid result."
- **CLI:** v1 has no `--timeout` flag, so cancellation reaches the CLI only via signal. The CLI installs a signal handler that cancels the scan `ctx`, then exits **`128 + signum`** (130 for SIGINT/Ctrl-C, 143 for SIGTERM) — standard shell convention, distinct from codeprint's sysexits app codes, so NFR-6 is unaffected.

## Panic isolation

- **Per-file recover:** each worker recovers a panic from processing a single file, records it in `errors[]` as an internal error (`reason: "internal: <msg>"`), and the scan continues. One pathological file cannot abort a scan — critical at 18k-repo fleet scale.
- **Top-level recover:** any panic that escapes the worker layer is recovered at the `Scan` boundary and surfaces as an internal error → CLI exit **70** (NFR-6 Software). This should be unreachable in practice; it's the backstop.

## Exit-code mapping (CLI)

```
0    success (incl. empty workspace, incl. non-fatal errors[] present)
64   usage           ErrUsage
66   no-input        ErrUnreadableRoot
70   software        recovered top-level panic
74   I/O error       only with --strict-io (per-file I/O promoted to fatal)
128+N signal         scan cancelled by signal N (130 SIGINT, 143 SIGTERM)
```

Mirrors spec NFR-6; signal codes added here as standard convention (not a new app code).

## Message style

- All human messages → **stderr** (stdout reserved for output). Terse, actionable.
- Format: `codeprint: <what failed>: <path>: <os reason>`.
- No stack traces unless `--debug`.
- `errors[]` entries are data (`{path, reason}`), not prose — machine-consumable.

Example:
```
codeprint: cannot read file: src/legacy/blob.dat: permission denied   # → stderr, also errors[]
codeprint: workspace root unreadable: /ws/missing: no such file       # → stderr, exit 66
```

## Change log

- 2026-05-30: initial draft; confirmed. ctx cancel → (nil, ctx.Err()) clean abort; per-file panic recover + continue; errors[] reserved for read failures only; soft-degradation rules defined; signal exit codes (128+N) documented alongside NFR-6 sysexits codes.
