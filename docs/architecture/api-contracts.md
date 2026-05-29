# codeprint — API Contracts

**Purpose:** define the Go library surface (the consumer contract) and the CLI surface.
**Phase:** 4.3 API Design
**Status:** confirmed
**Updated:** 2026-05-30

## Scope

codeprint has no network API. "API" here means two surfaces:
1. **Library API** — `pkg/codeprint`, the in-process contract P2 scanner-maintainers integrate against (flows F2).
2. **CLI surface** — `cmd/codeprint` flags (flows F1, F3).

The library is the source of truth; the CLI is a thin wrapper (spec F-6).

## Library API

```go
package codeprint

// Sole entry point. ctx enables cancellation/timeout — see Decision 1.
func Scan(ctx context.Context, root string, opts ...Option) (*Fingerprint, error)

// Functional options (WithX naming). New options never change the signature.
func WithConcurrency(n int) Option         // worker pool size; default runtime.NumCPU()
func WithIgnoreFile(path string) Option     // extra ignore file, layered on .gitignore/.codeprintignore
func WithIncludeHidden(b bool) Option       // default true — hidden files NOT ignored (NFR-9)
func WithMaxFileSize(bytes int64) Option    // default 10MB; larger files flagged skipped_large (NFR-4)
func WithoutFileRecords() Option            // emit rollups only, drop files[] (4.2 opt-out)

type Option func(*config)   // config is unexported; only WithX constructors are public
```

### Decisions

1. **`ctx context.Context` is the first parameter** — deviation from spec F-6's original `Scan(root, opts...)`. A 5M-LOC scan takes ~17s and runs across 18k pipelines; callers need timeout/cancellation. Cheap to add pre-1.0, breaking to add later. Spec F-6 amended to match.
2. **`WithProgress` is NOT exported in v1.** Internal progress sink is built (drives CLI `--verbose`/`--debug` + the default spinner), but the public hook is deferred to v2. Adding it later is non-breaking; removing it would be breaking. See [system-architecture.md](./system-architecture.md) for the internal sink.
3. **Detector interface stays internal.** Feasibility's "keep detection pluggable" is met by internal swappability; a public plugin API is a v2 commitment.
4. **`*Fingerprint` mirrors the schema 1:1** ([data-model.md](./data-model.md)). Consumers read fields directly — no file round-trip, no unmarshal (flows F2).

### Errors (summary; full taxonomy in 4.4)

```go
var ErrUnreadableRoot = errors.New("codeprint: workspace root unreadable")  // CLI → exit 66
var ErrUsage          = errors.New("codeprint: invalid option")             // CLI → exit 64
```

- Fatal conditions return an error (wrapped, `errors.Is`-checkable). Per-file failures are **non-fatal** — collected in `Fingerprint.errors`, scan continues, library returns a valid `*Fingerprint` with `nil` error (NFR-5).
- The CLI maps errors → sysexits codes (NFR-6). Detailed in [error-strategy.md](./error-strategy.md) (4.4).

## CLI surface

```
codeprint [path] [flags]        # path defaults to "." (flows F3)

Output:
  --out <file>            write JSON to file instead of stdout (always JSON, never pretty)
  --format json|pretty|summary   default: auto — TTY→pretty, non-TTY→json
  --pretty-json           indented JSON (for humans reading files)
  --no-files              rollups only          → WithoutFileRecords()

Scan control:
  --no-hidden             ignore dotfiles        → WithIncludeHidden(false)
  --ignore-file <path>    extra ignore file      → WithIgnoreFile(path)
  --max-file-size <n>     override 10MB default  → WithMaxFileSize(n)
  --concurrency <n>       override NumCPU         → WithConcurrency(n)
  --strict-io             promote I/O errors to exit 74 (NFR-5/6)

Output verbosity (all → stderr):
  --verbose               progress + end summary with timing
  --debug                 per-file diagnostic lines
  --no-color              disable color (also honors NO_COLOR env)

codeprint completion [bash|zsh|fish|powershell]   # only subcommand (F3)
```

### Default CLI behavior — progress + summary

The default invocation is **not silent at an interactive terminal**, but stays **silent in CI/pipes**. Everything human goes to stderr or to pretty-stdout; the JSON artifact and pipes are never polluted.

| Surface | When | Goes to | Behavior |
|---|---|---|---|
| **Progress spinner** | stderr is a TTY | stderr | `⠋ scanning… 1,234 files` — spinner + running counter, redraw throttled ~10x/sec, cleared on completion. **Suppressed when stderr is not a TTY** (CI, redirect) to avoid log spam (P3). Honors `NO_COLOR`. |
| **Pretty summary** | stdout is a TTY (default format) | stdout | Rich human breakdown (below) including `Scanned in 0.83s`. |
| **Piped/CI output** | stdout not a TTY | stdout | Clean JSON only. No human summary injected. Timing available via `--verbose`. |

**Progress is indeterminate** (spinner + counter, no %-bar): the scan is single-pass streaming ([system-architecture.md](./system-architecture.md)), so a true percentage would require a second tree walk. The counter is enough to prove "not stuck"; the throttle keeps it off the perf budget (NFR-1).

**Pretty summary layout:**

```
codeprint  ~/projects/foo

  Languages
  Go          62.3%   38,120 LOC   284 files
  Python      28.1%   17,200 LOC    98 files
  YAML         6.2%    3,800 LOC    22 files
  Other        3.4%    2,080 LOC     8 files

  Totals      412 files · 61,200 LOC (52,940 code · 6,402 comment · 1,858 blank)
  Build       go-modules, npm
  Frameworks  3 hints (gin, react, fastapi)
  Scanned in  0.83s
```

**Timing placement:** wall-clock time appears in the pretty summary and under `--verbose` only — **never in the JSON `fingerprint` or `_meta`**. Duration is a property of the runner, not the codebase; keeping it out preserves the artifact's reproducibility ([data-model.md](./data-model.md) NFR-2). Fleet-wide "slow repo" telemetry is the CI pipeline's job (it already wraps codeprint per flows F1), not the artifact's.

## Contract stability summary

| Element | Stability promise |
|---|---|
| `Scan` signature | Stable from v1.0.0; `ctx` + variadic options absorb growth without breaking |
| `WithX` options | Additive only; new options never change the signature |
| Exported types (`Fingerprint`, …) | Mirror schema; governed by schema semver (F-7) |
| `internal/*` | No promise — free to change any release |
| CLI flags | Additive; removal/rename is a breaking (major) change |

## Change log

- 2026-05-30: initial draft; confirmed. `Scan` takes `ctx` (amends spec F-6). `WithProgress`/Detector kept internal. Defined default CLI progress spinner (TTY-gated, CI-silent) + enriched pretty summary; timing is human-output-only, never in JSON.
