# codeprint — Product Specification

**Purpose:** define v1 features, acceptance criteria, and non-functional requirements.
**Phase:** 3.2 Spec
**Status:** confirmed
**Updated:** 2026-05-27

## Overview

v1 of codeprint delivers **seven features** covering the L2 scope decided in ideation. Every feature is MUST-have except where explicitly marked. Priority is the only lever the team has if the 1-quarter timeline slips — no scope creep beyond this document.

| Feature | Priority |
|---|---|
| F-1 Language + LOC analysis | MUST |
| F-2 File classification | MUST |
| F-3 Build-system detection (10 ecosystems) | MUST |
| F-4 Framework hints from manifests | SHOULD (depth is scope lever) |
| F-5 Output modes (JSON, pretty, summary) | MUST |
| F-6 Library API + CLI | MUST |
| F-7 Published JSON Schema with semver | MUST |

---

## Features

### F-1 Language + LOC analysis — MUST

**User story:** As an AppSec engineer (P1), I want a per-language breakdown of code/comment/blank LOC in a repo so that I can verify scanner coverage claims against an independent denominator.

**Acceptance criteria:**
- For every source file in the workspace, codeprint records: language ID (from `go-enry`), LOC code, LOC comment, LOC blank, byte size.
- Per-language rollup: file count, total code LOC, total comment LOC, total blank LOC, aggregate percentage of repo.
- Per-repo totals: total files, total LOC (code+comment+blank).
- Files classified as binary, vendored, or minified (see F-2) are counted but not included in LOC totals; their LOC fields are 0.
- Languages with no comment-syntax mapping in `gocloc` fall back to "count non-blank lines as code" and the fingerprint marks `comment_aware: false` for that language.
- Output respects determinism (see NFR-2).

### F-2 File classification — MUST

**User story:** As a scanner maintainer (P2), I want each file tagged as source, test, generated, vendored, binary, or minified so that my scanner can decide which files to process or skip.

**Acceptance criteria:**
- Every file carries exactly one primary `kind` field: `source` | `test` | `generated` | `vendored` | `binary` | `minified`.
- Classification rules, in priority order:
  1. `binary` — via `enry.IsBinary(content)`.
  2. `vendored` — via `enry.IsVendor(path)`.
  3. `generated` — via `enry.IsGenerated(path, content)` OR header regex matching `// Code generated` / `# DO NOT EDIT` family.
  4. `minified` — extension match (`.min.js`, `.min.css`) OR heuristic `avg_line_length > 500 && newline_ratio < 0.01`.
  5. `test` — path heuristic table per ecosystem (`_test.go`, `*.spec.ts`, `*Test.java`, `test/`, `tests/`, `spec/`, `__tests__/`).
  6. Default: `source`.
- Each file additionally carries a `flags` list for non-mutually-exclusive properties (e.g., a generated test file can be `kind: generated` with `flags: ["test"]`).
- **Known v1 limitation:** generated files that lack a recognizable header (`// Code generated`, `# DO NOT EDIT` family) and are not matched by enry's generated-file rules are classified `source`. Some tools (e.g., certain `protoc`/OpenAPI generators) emit no header and will be missed. Refinement deferred post-v1.

### F-3 Build-system detection — MUST

**User story:** As a scanner maintainer (P2), I want to know which ecosystems are present in a repo so my scanner can select appropriate rules.

**Acceptance criteria:**
- Repo-level `build_systems: []` field lists every detected build system. Each entry carries `ecosystem` + `path` (the repo-relative path to the marker file that triggered detection).
- Detection is a marker-file presence check — no parsing needed at this stage.
- v1 ecosystems (10):

| Ecosystem | Marker files |
|---|---|
| `npm` / `yarn` / `pnpm` | `package.json` (+ lockfile discriminates variant) |
| `pip` / `poetry` / `pipenv` | `requirements.txt`, `pyproject.toml`, `Pipfile` |
| `maven` | `pom.xml` |
| `gradle` | `build.gradle`, `build.gradle.kts`, `settings.gradle(.kts)` |
| `go-modules` | `go.mod` |
| `bundler` | `Gemfile` |
| `nuget` / `dotnet` | `*.csproj`, `*.sln`, `packages.config` |
| `cargo` | `Cargo.toml` |
| `composer` | `composer.json` |
| `swift-pm` | `Package.swift` |

- Detection walks the full tree and emits a **flat, deduped list** of every ecosystem found, each with its marker-file path. Nested markers (e.g., `frontend/package.json`, `services/api/pom.xml`) ARE detected and listed — but they are NOT grouped into per-workspace rollups. v1 stays one flat fingerprint; per-workspace grouping is deferred to v2 (`workspaces: []`).
- Dockerfile detected separately in `container.dockerfile_present: bool` (not a build system, but useful).

### F-4 Framework hints — SHOULD

**User story:** As a scanner maintainer (P2), I want framework hints per ecosystem so my scanner can pick the right rule profile (e.g., Django vs Flask for Python).

**Acceptance criteria:**
- Repo-level `framework_hints: []` field — always a list, never a singular `framework`.
- Each hint has: `name`, `ecosystem`, `confidence` (`high` | `medium` | `low`), `evidence` (path to the manifest file + keyword that triggered the hint).
- v1 hint catalog (initial; extensible):

| Ecosystem | Hints |
|---|---|
| npm / yarn / pnpm | react, vue, angular, next, nuxt, svelte, express, fastify, nest |
| pip / poetry | django, flask, fastapi |
| maven / gradle | spring, spring-boot, micronaut, quarkus |
| gradle (android) | android |
| bundler | rails |
| dotnet | aspnet-core |
| composer | laravel, symfony |
| cargo | actix, rocket, axum |
| swift-pm | swiftui (detected via target deps) |

- **Scope-lever rule:** if schedule slips, F-4 degrades gracefully — we ship with fewer ecosystems populated. The field is always present (even if empty); consumers code against `framework_hints: []` either way.
- Gradle Groovy DSL uses regex pattern-match (documented partial coverage). Kotlin DSL parses cleaner.

### F-5 Output modes — MUST

**User story:** As a DevOps engineer (P3), I want codeprint to produce JSON by default in CI, and as a local dev (P1) I want readable pretty output in my terminal.

**Acceptance criteria:**
- Three emit modes: `--format=json`, `--format=pretty`, `--format=summary`.
- Default = auto: TTY → pretty; non-TTY → json.
- `summary` = one-line digest: `Go 62% • Python 28% • 48,212 LOC • 412 files • 3 frameworks`.
- `--out <path>` writes to file instead of stdout. File always gets JSON regardless of TTY (pretty-print is TTY-only).
- Logs (progress, errors, warnings) always go to stderr. stdout is reserved for the requested output.
- `NO_COLOR` env var disables colors even on TTY.
- Compact JSON by default; `--pretty-json` produces indented JSON (for humans reading files).
- **Default progress indicator (added in architecture 4.3):** when stderr is a TTY, the default invocation shows an indeterminate spinner + running file counter on stderr (`⠋ scanning… 1,234 files`), cleared on completion. **Suppressed when stderr is not a TTY** (CI, pipe, redirect) to keep pipeline logs clean. Honors `NO_COLOR`. See [../architecture/api-contracts.md](../architecture/api-contracts.md).
- **Enriched pretty summary:** the `pretty` output ends with a human summary — per-language split (%, LOC, files), totals (code/comment/blank), build systems, framework count, and wall-clock scan time. Timing appears in the pretty/`--verbose` human output only — **never in the JSON `fingerprint` or `_meta`** (duration is a runner property, not a codebase property; keeps the artifact reproducible per NFR-2).

### F-6 Library API + CLI — MUST

**User story:** As a scanner maintainer (P2), I want to consume codeprint in-process as a Go library with a stable functional-options API.

**Acceptance criteria:**
- Import path: `github.com/ShadowOpenTech/codeprint/pkg/codeprint`.
- Exactly one public entry point: `Scan(ctx context.Context, root string, opts ...Option) (*Fingerprint, error)`. (`ctx` added in architecture 4.3 for cancellation/timeout at fleet scale — see [../architecture/api-contracts.md](../architecture/api-contracts.md).)
- `Option` is an exported function type; options follow `WithX` naming (`WithConcurrency`, `WithIgnoreFile`, `WithIncludeHidden`, `WithMaxFileSize`).
- `Fingerprint` struct mirrors the JSON schema exactly; all exported, all with `json:` struct tags.
- CLI lives at `cmd/codeprint`. CLI parses flags, constructs `Options`, calls `Scan()`, serializes. No business logic in CLI.
- CLI uses `spf13/cobra`. Root command only; one built-in subcommand (`completion`).
- `go 1.23` module floor.

### F-7 Published JSON Schema with semver — MUST

**User story:** As a scanner maintainer (P2), I want a stable, versioned JSON schema published at a permanent URL so my integration survives codeprint releases.

**Acceptance criteria:**
- Canonical schema file: `schema/codeprint-v1.schema.json` in the repo.
- Generated from Go structs using `invopop/jsonschema` (JSON Schema 2020-12).
- CI step generates schema and diffs against the committed file — build fails on drift.
- Every output JSON embeds a `$schema` field + `_meta.schema_version` (string, e.g., `"1.0"`). The `$schema` URL strategy during v0.x vs v1.0.0 is resolved in architecture (open question #4 below). The **immutable tag-pinned URL is a v1.0.0 GA requirement, not a v0.x one** — pre-1.0 may use a floating URL or omit it.
- Output envelope shape fixed forever: `{ "$schema": ..., "_meta": {...}, "fingerprint": {...} }`. Only `fingerprint` is subject to determinism rules.
- Semver discipline locks at v1.0.0. Until then (v0.x), schema may change freely; warnings in docs + changelog entries.
- Migration between majors documented in `docs/schema/CHANGELOG.md`.

---

## Non-functional requirements

### NFR-1 — Performance

- **Target:** ≥300k LOC/sec throughput on typical CI runner hardware (Moderate tier from feasibility).
- **Envelope:** 50k LOC ≤ 170ms; 500k LOC ≤ 1.7s; 5M LOC ≤ 17s.
- **Benchmark gate:** `go test -bench` suite runs on a curated set of open-source repos (Kubernetes, Django, Rails, React, Linux kernel subset). Regression of > 10% fails CI.

### NFR-2 — Determinism

- Byte-identical output for byte-identical input. Enforced in CI by running codeprint twice on a fixture corpus and diffing.
- No timestamps in the `fingerprint` data. Timestamps permitted in `_meta` envelope only (and consumers are told to ignore `_meta` when diffing).
- All lists sorted by stable key. JSON keys sorted.

### NFR-3 — Distribution

- Single static Go binary. `CGO_ENABLED=0`.
- No runtime dependencies. No config files required.
- Binary size target: < 30MB stripped.
- Release artifacts: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64. Published to JFrog (primary) and GitHub Releases (mirror).

### NFR-4 — Resource limits

- **Max file size:** 10 MB default; configurable via `WithMaxFileSize`. Files above are counted but not scanned for LOC — marked `flags: ["skipped_large"]`.
- **Max total memory:** soft target 512 MB on 5M-LOC repo. No hard cap — but docs note this expectation.
- **Concurrency:** defaults to `runtime.NumCPU()` worker pool. Configurable via `WithConcurrency`.

### NFR-5 — Error tolerance

- Unreadable individual files do **not** abort the scan — they are recorded under `fingerprint.errors: []` with path + reason. Final exit code remains 0.
- Unreadable workspace root (the path passed to `Scan`) aborts with exit 66 (`EX_NOINPUT`).
- Usage errors exit 64 (`EX_USAGE`). I/O errors after partial work exit 74 (`EX_IOERR`) only when `--strict-io` is passed; otherwise they go into `fingerprint.errors`.

### NFR-6 — Exit codes (sysexits.h-aligned)

| Code | Meaning | Condition |
|---|---|---|
| 0 | Success | Scan completed (incl. empty workspace) |
| 64 | Usage | Bad flag, bad argument |
| 66 | NoInput | Workspace path does not exist or is unreadable |
| 70 | Software | Internal panic or unexpected bug (recovered) |
| 74 | IOError | Only emitted with `--strict-io`; I/O error during scan |

Codes chosen per [sysexits.h BSD convention](https://man7.org/linux/man-pages/man3/sysexits.h.3head.html).

### NFR-7 — Observability

- `--verbose` adds human-readable progress lines to stderr (e.g., `scanned 412 files in 0.8s`).
- `--debug` adds per-file diagnostic lines to stderr (path, language, LOC — for triaging why a scan looks wrong).
- No logging framework. Plain stderr writes. No file logging.

### NFR-8 — Security

- No network calls. Ever. Verified by build-time test with network disabled.
- No file writes outside the `--out` path.
- Symlinks followed but with loop detection (max follow-depth 40) **and workspace containment**: a symlink is followed only while its resolved target stays inside the workspace root; targets that escape the root are skipped (not followed). Added in architecture 4.5 — see [../architecture/security-audit.md](../architecture/security-audit.md) finding #1.
- File paths in output are repo-relative (never absolute) — avoids leaking CI paths/usernames.
- File contents are read but never emitted in the output. Only metadata.

### NFR-9 — Ignore rules

- By default: respects `.gitignore` at workspace root and nested directories (using `github.com/sabhiram/go-gitignore` or similar).
- Additional ignore files: `.codeprintignore` (same format as .gitignore) merged on top.
- `--ignore-file <path>` adds a specific file.
- Hidden files (leading `.`) are **not ignored by default** — configurable via `--no-hidden`.

### NFR-10 — Internationalization

- English-only output in v1. No localization infrastructure. Path/content handling is UTF-8 only — non-UTF-8 files are classified `binary`.

---

## Scope boundary — what v1 explicitly does NOT do

- No SBOM generation (delegated to syft / cyclonedx-cli).
- No CVE / vulnerability lookup.
- No AST parsing or symbol extraction.
- No code-quality metrics (complexity, duplication).
- No coverage-gap computation — that's a consumer-side responsibility.
- No diff mode (deferred to v2).
- No per-workspace fingerprints in monorepos (deferred to v2).
- No network I/O of any kind.
- No dashboards, UIs, reporting, or storage.
- No git history / blame / churn analysis.
- No configuration file format. Flags only.
- No domain subcommands (only `completion`).

---

## Open questions to resolve in architecture phase

- Ignore-rules library choice — `sabhiram/go-gitignore` vs `denormal/go-gitignore` vs roll-our-own subset. Matters because gitignore semantics are subtle (nested ignores, negation).
- Whether to pre-parse Gradle Kotlin DSL vs. use a regex-only approach (Groovy DSL is regex-only — Kotlin DSL could go either way).
- Whether `invopop/jsonschema` or Google's new 2026 JSON Schema package is the better choice for schema generation.
- Where the schema file's `$schema` URL points during pre-1.0 — a floating `main` branch URL, or skip `$schema` until v1.0.0?

## Change log

- 2026-04-22: initial draft.
- 2026-05-27: confirmed. Resolved before lock: F-3 detection changed to flat full-tree list with marker paths (was "root + one level down"); F-7 `$schema` immutable-URL requirement scoped to v1.0.0 GA (was unconditional, conflicted with v0.x freedom); F-2 added known limitation note for header-less generated files. Exit-code conflict with `user-flows.md` resolved in favor of NFR-6 sysexits codes (flows doc updated to match).
- 2026-05-30: architecture-phase amendments (4.3 API Design). F-6: `Scan` now takes `ctx context.Context` as first param (cancellation/timeout). F-5: added default TTY-gated progress spinner (CI-silent) + enriched pretty summary with timing; timing is human-output-only, never in JSON.
- 2026-05-30: architecture-phase amendment (4.5 Security Check). NFR-8: symlink following now requires workspace containment (out-of-tree targets skipped).
