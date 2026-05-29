# codeprint — Test Plan

**Purpose:** define testing strategy, critical paths, fixtures, and coverage gates.
**Phase:** 5.1 Test Plan
**Status:** confirmed
**Updated:** 2026-05-30

## Two corpora, two purposes

A recurring confusion worth nailing down: **correctness** and **performance** need different test inputs.

| Layer | Measures | Input requirement | Corpus |
|---|---|---|---|
| **Correctness** (TP/FP/FN) | right languages / classification / ecosystems / framework hints — precision & recall | **known ground truth** | `CipherRadarTestProj` (extended), version-pinned |
| **Performance** (NFR-1) | sustained throughput ≥300k LOC/sec | **volume** (big real repos); no ground truth | Kubernetes / Django / Rails / React / Linux-subset, pinned |

## Critical paths (highest-value tests)

1. **Determinism (NFR-2)** — run `Scan` twice on the correctness corpus, assert byte-identical `fingerprint`. The signature test.
2. **Schema drift (F-7)** — regenerate schema from structs, diff vs committed `schema/codeprint-v1.schema.json`, fail on mismatch.
3. **Language + LOC accuracy (F-1)** — golden output vs hand-verified ground truth.
4. **File classification (F-2)** — fixtures for each `kind` + priority ordering (binary→vendored→generated→minified→test→source).
5. **Performance regression (NFR-1)** — `go test -bench`; >10% regression fails.
6. **Exit codes & error tolerance (NFR-5/6)** — CLI e2e per code; non-fatal `errors[]`; symlink containment (NFR-8).

## Test pyramid

```
Unit (most)        per internal/* pkg: detect, loc, classify, walk, buildsys, framework, emit
Golden/fixture     correctness corpus → expected output; the determinism + accuracy core
Integration        full Scan(ctx, root, ...) on the corpus; library API contract
Benchmark          go test -bench on the performance corpus (NFR-1)
CLI e2e (fewest)   invoke the binary: stdout/stderr split, --out, --format, exit codes, NO_COLOR
Build-property     network-disabled build/test (zero network calls, NFR-8); CGO_ENABLED=0
```

## Correctness corpus — CipherRadarTestProj (extended)

**Decision:** push `CipherRadarTestProj` to `ShadowOpenTech`, consume it in codeprint as a **git submodule pinned to a SHA** (reproducible golden tests; shared source of truth with cradar). It already spans 13+ languages + `pom.xml`/`pyproject.toml`/Dockerfile/k8s.

**Extensions needed for codeprint** (it was built crypto-focused for cradar; codeprint needs classification + ecosystem breadth). Tracked as an early breakdown task:

| Add | Tests |
|---|---|
| `vendored/` dir (node_modules-style) | `kind: vendored` |
| generated file with `// Code generated` header | `kind: generated` |
| `.min.js` + a long-line file | `kind: minified` (both detection paths) |
| binary blob | `kind: binary`, excluded from LOC |
| >10MB file | `flags: ["skipped_large"]` (NFR-4) |
| `package.json` (react+express), `requirements.txt` (django+flask), `build.gradle` (spring) | build-system + framework hints (F-3/F-4) |
| in-tree symlink + escaping symlink | NFR-8 containment (follow in-tree, skip escape) |
| **ground-truth label file** (`expected.json` or per-dir labels) | the TP/FP/FN oracle |

**TP/FP/FN method:** for detection/classification/framework hints, compare codeprint output against the ground-truth labels and report precision/recall. A regression in detection accuracy fails the suite.

## Performance corpus

**Decision:** benchmark repos as **git submodules pinned to fixed SHAs** — same code every run, so benchmark numbers are comparable over time. Speed-only; no correctness assertions. CI fetches submodules before the bench step.

## Coverage gate

- **Target: 85% on the analysis pipeline** — `pkg/codeprint` + `internal/{detect,loc,classify,walk,buildsys,framework,emit}`. Enforced in CI on these packages.
- **No hard target on `cmd/`** — cobra glue is covered by CLI e2e, not unit coverage.
- Rationale: weight rigor where the logic lives; don't spend effort unit-testing flag wiring.

## Determinism test detail

```
out1 = Scan(ctx, corpus); out2 = Scan(ctx, corpus)
assert out1.fingerprint == out2.fingerprint   // byte-identical; _meta excluded
```

Run with shuffled concurrency (`WithConcurrency(1)` vs `WithConcurrency(N)`) to prove worker count never affects output ordering.

## CI gates summary (feed Phase 7 /guard, /perf-audit)

| Gate | Fails build when |
|---|---|
| `go test ./...` | any test fails |
| coverage | pipeline packages < 85% |
| determinism | two runs differ in `fingerprint` |
| schema drift | generated schema ≠ committed |
| benchmark | >10% throughput regression (NFR-1) |
| `govulncheck` | known vuln in deps (NFR-8 / security-audit) |
| network-disabled build | any network call attempted |

## Change log

- 2026-05-30: initial draft; confirmed. Separated correctness (TP/FP/FN, CipherRadarTestProj as pinned submodule + extensions) from performance (big-repo submodules, speed-only). Coverage 85% on pipeline, none on cmd/. Determinism + schema-drift + benchmark + govulncheck as CI gates.
