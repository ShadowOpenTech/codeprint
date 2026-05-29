# codeprint — Task Breakdown

**Purpose:** ordered, buildable development tasks grouped into milestones; defines Release 0 scope.
**Phase:** 5.5 Breakdown
**Status:** confirmed
**Updated:** 2026-05-30

## Release 0 scope

**Release 0 = v0.1.0 = the full v1 feature set (M0–M5).** Includes framework hints (F-4) and complete CLI UX. Satisfies both named consumers from first tag — cradar (language/LOC/classification/manifest) and the custom API security tool (framework + manifest). Semver lock still happens later at v1.0.0 GA after consumer feedback (F-7); v0.1.0 is the first complete, usable binary, not the frozen contract.

## Build order (dependency graph)

```
types.go ─────────────┐ (schema source — built first)
walk (ignore+symlink) ─┤
   ├─ detect (enry) ───┼─► Scan (collector + determinism) ─► emit (json/pretty/summary)
   │     └─ loc, classify                                          └─► CLI (cobra, flags, progress, exit codes)
   └─ buildsys ─► framework
schemagen (from types) ─► schema drift gate
```

---

## M0 — Scaffold & schema-first CI

| Task | Acceptance |
|---|---|
| Repo layout (`cmd/`, `pkg/codeprint/`, `internal/`, `schema/`) + `go.mod` (go 1.23) | `go build ./...` succeeds on empty skeleton |
| Makefile targets (5.4) | `make build/test/lint/verify` run |
| CI pipeline skeleton (lint, test, vet, govulncheck, network-disabled build) | CI green on skeleton |
| `pkg/codeprint/types.go` — `Fingerprint` + entities (4.2) with json + jsonschema tags | structs compile; fields match data-model |
| `internal/schemagen` + `make schema` + **drift gate** | generated schema committed; CI fails on drift |
| Push `CipherRadarTestProj` to org; add as pinned submodule; add **ground-truth labels** + codeprint extensions (vendored/generated/minified/binary/oversized/symlinks/framework manifests — see test-plan) | `make test` can read the corpus; labels present |
| Performance corpus submodules (pinned SHAs) | `make bench` can read them |

**Exit:** schema-first discipline live before any feature code. Empty pipeline is green.

## M1 — Core scan (MVP, F-1 + NFR-2)

| Task | Acceptance |
|---|---|
| `internal/walk` — traversal + `.gitignore`/`.codeprintignore` (denormal/go-gitignore) + symlink containment + iterative walk + visited-set + max-file-size | walks corpus; respects ignores; skips escaping symlinks |
| `internal/detect` — enry wrapper behind `Detector` interface (language ID, vendored/generated/binary) | correct language IDs on corpus |
| `internal/loc` — gocloc wrapper + enry↔gocloc mapping table + `comment_aware` fallback | code/comment/blank counts; fallback marked |
| `Scan(ctx, root, opts...)` orchestrator — worker pool + single sorting collector | returns valid `*Fingerprint`; honors ctx cancel → `(nil, ctx.Err())` |
| `internal/emit` — compact JSON + envelope (`$schema`/`_meta`/`fingerprint`) | valid JSON; validates against schema |
| **Determinism test** — twice + shuffled concurrency, diff `fingerprint` | byte-identical |
| Options: `WithConcurrency`, `WithMaxFileSize`, `WithIncludeHidden`, `WithIgnoreFile`, `WithoutFileRecords` | each behaves per data-model/api-contracts |

**Exit:** `codeprint . ` emits a correct, deterministic language/LOC fingerprint.

## M2 — Classification + build systems (F-2, F-3)

| Task | Acceptance |
|---|---|
| `internal/classify` — `kind` (priority order binary→vendored→generated→minified→test→source) + `flags` | TP/FP/FN vs ground-truth labels within target |
| Generated-detection (enry rules + header regex) incl. documented header-less limitation | known-limitation cases behave as documented |
| Minified detection (extension + long-line heuristic) | both paths covered |
| `internal/buildsys` — full-tree marker scan → flat list with paths (10 ecosystems) | all corpus ecosystems detected with correct paths |
| `container.dockerfile_present` | true on corpus Dockerfile |

**Exit:** every file classified; build systems populated with paths.

## M3 — Framework hints (F-4)

| Task | Acceptance |
|---|---|
| `internal/framework` — per-ecosystem manifest parsers (stdlib json/xml + BurntSushi/toml) | hints emitted as list with confidence + evidence |
| Hint catalog (react/vue/angular/next/…, django/flask/fastapi, spring/…, rails, laravel/symfony, actix/rocket/axum, swiftui) | corpus manifests yield correct hints |
| Gradle Groovy regex fallback (documented partial coverage); Kotlin DSL parse | spring detected in build.gradle |
| Precision/recall on framework hints vs labels | meets target; FPs flagged |

**Exit:** `framework_hints[]` populated; always a list, never singular.

## M4 — Output modes + CLI UX (F-5, F-6)

| Task | Acceptance |
|---|---|
| `cmd/codeprint` cobra root + flags (api-contracts) | flags map to options; no business logic in CLI |
| Format auto-detection (TTY→pretty, pipe→json) + `--format`/`--pretty-json` | correct per TTY/flag |
| Pretty summary (languages/totals/build/frameworks/timing) + control-char sanitization | matches mock; malicious filenames sanitized |
| Summary one-liner mode | digest format correct |
| Progress spinner (stderr, TTY-gated, CI-silent, throttled) via internal sink | shows interactively; silent in CI/pipe |
| Exit-code mapping (NFR-6) + signal handling (128+N) + `--strict-io` | each code correct in e2e |
| `completion` subcommand; `NO_COLOR`; `--verbose`/`--debug` | all behave |

**Exit:** full human + machine UX; CLI e2e green.

## M5 — Hardening → Release 0 (NFR-1/8, F-7)

| Task | Acceptance |
|---|---|
| Perf benchmark gate (>10% regression fail) on perf corpus | gate active; baseline recorded |
| Coverage to 85% on pipeline packages | CI coverage gate green |
| `govulncheck` + gosec clean; control-char + symlink-containment tests | security tests pass |
| Schema finalize for v0.x; `$schema` URL strategy (floating pre-1.0) | output carries `$schema` + `_meta.schema_version` |
| goreleaser config (5 platforms) — *coordinated with Phase 6 deploy-plan* | cross-builds produce static binaries |
| Tag **v0.1.0**; publish artifacts | Release 0 shipped |

**Exit:** Release 0 (v0.1.0) — complete, deterministic, hardened binary + library + schema.

---

## Milestone dependency summary

```
M0 ─► M1 ─► M2 ─► M3 ─► M4 ─► M5
            └─────────────┘  (M2/M3 both feed M4's summary; M3 may parallel M2 after buildsys lands)
```

M3 (framework) depends on M2's `buildsys` but is otherwise independent of M2's `classify`, so it can overlap once build-system detection exists.

## Post-Release-0 (v0.2+ / toward v1.0.0 GA)

- Consumer feedback loop (cradar, API security tool) → refine schema.
- Schema semver **lock at v1.0.0** + immutable tag-pinned `$schema` URL + schemastore registration (F-7).
- v2 growth features (diff mode, monorepo `workspaces`, plugin system) — out of Release 0 scope.

## Change log

- 2026-05-30: initial draft; confirmed. Release 0 = full feature set (M0–M5, incl. framework hints + CLI UX). Schema-first from M0. Six milestones with per-task acceptance criteria; TP/FP/FN gates on classification + framework hints; v0.1.0 as Release 0, semver lock deferred to v1.0.0 GA.
