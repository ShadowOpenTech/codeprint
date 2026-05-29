# codeprint — System Architecture

**Purpose:** define module layout, the scan pipeline, concurrency/determinism model, and the locked dependency set.
**Phase:** 4.1 Architect
**Status:** confirmed
**Updated:** 2026-05-30

## Overview

codeprint is a Go library (`pkg/codeprint`) with a thin CLI front-end (`cmd/codeprint`). The library is the source of truth: its exported Go structs drive the published JSON schema (see [../design/product-spec.md](../design/product-spec.md) F-7). Everything is a single pure-Go static binary — no CGo, no network, no runtime deps.

Stack was locked in [../discovery/feasibility-study.md](../discovery/feasibility-study.md); this doc adds the component boundaries and the two infra-level library choices that feasibility deferred to architecture.

## Module layout

```
cmd/codeprint/              thin CLI (cobra): parse flags → build Options → Scan → serialize
  main.go
  root.go                   root command, flag wiring, format/TTY resolution
  completion.go             cobra completion subcommand (only subcommand in v1)

pkg/codeprint/              PUBLIC library — stable API, source of truth for schema
  scan.go                   Scan(root string, opts ...Option) (*Fingerprint, error)  — sole entry point
  options.go                Option func type + WithConcurrency/WithIgnoreFile/WithIncludeHidden/WithMaxFileSize
  types.go                  Fingerprint, FileRecord, LanguageRollup, BuildSystem, FrameworkHint, ... (json + jsonschema tags)

internal/                   PRIVATE — no API promise, free to change between releases
  walk/                     filesystem walk + ignore-rule engine
  detect/                   enry wrapper behind a Detector interface (pluggable per risk mitigation)
  loc/                      gocloc wrapper + enry↔gocloc language-name mapping table
  classify/                 kind (source/test/generated/vendored/binary/minified) + flags
  buildsys/                 marker-file scan → flat ecosystem list with paths
  framework/                per-ecosystem manifest parsers → framework_hints
  emit/                     json / pretty / summary writers
  schemagen/                generate schema/codeprint-v1.schema.json from pkg/codeprint types

schema/
  codeprint-v1.schema.json  generated artifact; CI asserts no drift vs types.go
```

**Boundary rule:** consumers import `pkg/codeprint` only. `internal/` is invisible to them (Go enforces this). This keeps the deprecation blast radius tiny — only `Scan` + `Option` + the exported types are promised.

## Scan pipeline

Single filesystem pass, bounded worker pool, single deterministic collector.

```
walk(root) ──► [worker pool, size = NumCPU (WithConcurrency override)]
                   │
                   ▼  per file:
                   detect (enry: language ID, vendored?, generated?, binary?)
                   classify (→ kind + flags)
                   loc (gocloc count, only if scannable: not binary/minified/oversized)
                   manifest parse (only if the file is a build-system marker)
                   │
                   ▼  results streamed over a channel
              collector (single goroutine)
                   │  sorts by stable key (path, then language ID)
                   ▼
              assemble *Fingerprint  (rollups, totals, build_systems, framework_hints)
                   │
                   ▼
              emit (json | pretty | summary → stdout or --out file)
```

### Concurrency & determinism (NFR-2)

Determinism is **structural**, not best-effort:

- Workers finish in arbitrary order — that disorder never reaches the output.
- The **collector is the single assembly point**. It buffers all per-file results, then sorts by stable key before building rollups and totals.
- All output lists are sorted by a documented stable key. JSON object keys are emitted sorted.
- No timestamps or machine-specific data in the `fingerprint` payload. `_meta` is the only place volatile data (producer version, timestamps) may appear, and consumers are told to ignore `_meta` when diffing.
- **CI gate:** run codeprint twice on a fixture corpus, assert byte-identical `fingerprint`.

### The enry ↔ gocloc seam

The two libraries have separate language tables (flagged in feasibility). Resolution:

- `detect/` owns language identity — **enry is authoritative** (matches consumer expectation that language IDs equal go-enry/linguist IDs).
- `loc/` maps the enry language ID → gocloc language name to get comment-aware counts.
- Unmapped languages fall back to "non-blank lines = code, 0 comments" and the language rollup is marked `comment_aware: false`.
- The mapping table lives in `loc/` and is unit-tested against both libraries' language universes so drift is caught when either dep is bumped.

## Locked dependencies

| Concern | Library | License | Decision origin |
|---|---|---|---|
| Language detection | `github.com/go-enry/go-enry/v2` | Apache-2.0 | feasibility |
| LOC counting | `github.com/hhatto/gocloc` | MIT | feasibility |
| CLI framework | `github.com/spf13/cobra` | Apache-2.0 | spec F-6 |
| Schema generation | `github.com/invopop/jsonschema` | MIT | **4.1 (this doc)** |
| Ignore rules | `github.com/denormal/go-gitignore` | MIT | **4.1 (this doc)** |
| TOML parsing (manifests) | `github.com/BurntSushi/toml` | MIT | feasibility (F-5) |
| TTY/color | `github.com/fatih/color` + `mattn/go-isatty` | MIT | flows F3 |

**No CGo.** `CGO_ENABLED=0`. Forgoes enry's Oniguruma speedup to preserve the single-static-binary invariant (feasibility decision).

### 4.1 library-choice rationale

- **Ignore rules → `denormal/go-gitignore`.** Chosen over `sabhiram/go-gitignore` (the spec's default candidate) because it handles **nested `.gitignore` files and negation semantics** properly. Across an 18k-repo fleet, odd ignore patterns are guaranteed; the simpler lib would force us to hand-roll per-directory merging, which is exactly the gitignore footgun we want to avoid. `.codeprintignore` is layered on top of `.gitignore` using the same engine. Supersedes the open question in spec ("ignore-rules library choice").
- **Schema generation → `invopop/jsonschema`.** Chosen over Google's Jan-2026 JSON Schema package because the schema is a **v1 contract artifact** — battle-tested and reflection-from-struct-tags matters more than novelty here. invopop is mature, actively maintained, emits JSON Schema 2020-12. Supersedes the open question in spec ("invopop vs Google pkg").

## Component responsibilities (boundaries)

| Package | Owns | Does NOT do |
|---|---|---|
| `walk` | tree traversal, ignore-rule application, symlink-loop detection, max-file-size gate | language/LOC logic |
| `detect` | language ID, vendored/generated/binary flags (enry) | counting, classification policy |
| `loc` | code/comment/blank counts, enry↔gocloc mapping | language identity |
| `classify` | final `kind` + `flags` decision (priority order from spec F-2) | reading file content twice (reuses detect's read) |
| `buildsys` | marker-file → flat ecosystem list with paths | parsing manifest contents |
| `framework` | manifest parse → `framework_hints` (list, with confidence + evidence) | asserting a single framework |
| `emit` | json/pretty/summary, stdout vs file, NO_COLOR, TTY detection | computing the fingerprint |
| `schemagen` | generate schema from types | runtime validation (that's a consumer concern) |

## Open questions carried into later 4.x steps

- **Gradle DSL parsing** (Groovy regex vs Kotlin parse) → resolved in 4.x framework detail / 5.x as needed; spec already documents partial Groovy coverage.
- **`$schema` URL during v0.x** → resolved per spec F-7 (floating/omit pre-1.0; immutable tag-pinned at v1.0.0 GA). No further architecture decision needed.
- **Exact `Fingerprint` field set** → 4.2 Data Model.
- **`Scan`/`Option` surface details** → 4.3 API Design.

## Change log

- 2026-05-30: initial draft; confirmed. Resolved two spec open questions — ignore lib (`denormal/go-gitignore`) and schema-gen lib (`invopop/jsonschema`). Defined module layout, scan pipeline, structural determinism model, and enry↔gocloc seam.
