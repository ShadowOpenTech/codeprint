# codeprint — Product Vision

**Purpose:** fingerprint a codebase for downstream scan/coverage tooling.
**Phase:** 1.1 Ideate
**Status:** draft
**Updated:** 2026-04-21

## One-liner

`codeprint` is a fast Go CLI + library that produces a stable, versioned JSON inventory of a codebase, designed to be consumed by security scanners, coverage trackers, and platform dashboards.

## Problem statement

Security and platform teams running scanners (SonarQube, cradar, Snyk, Checkmarx, etc.) across many apps have no independent view of *what each app actually contains*. Today, nothing answers "what's in this app?" — every team cobbles together `cloc` + `find` + manual Bitbucket checks to fill the gap.

**Two concrete pain points today:**

1. **Scanner coverage is unverifiable.** SonarQube (and others) report "X LOC scanned" — but app teams freely exclude directories from scans. The reported LOC has no reliable denominator. We can't tell whether the scan covered 100% or 30% of the code.

2. **Deployment change-type attestation is manual.** App teams declare deployments as "DB only," "config only," or "no code change" to bypass security review. Verifying the claim requires a human to open Bitbucket and diff by hand. At scale, this doesn't work — and false attestations go undetected.

**Why this matters:**

- **Coverage gaps go undetected** — a scanner runs but had no rules (or exclusions suppressed) for a language present in the repo.
- **Portfolio metrics are inconsistent** — each team counts files and languages differently.
- **Integration work is repeated per scanner** — every new scanner adds its own ad-hoc inventory logic.
- **Attestation is unenforceable** — "config only" claims can't be spot-checked without human effort.

`codeprint` fills that gap: **one scanner-agnostic, independently-produced fingerprint, stable schema, consumed once by many tools.** It becomes the trusted ground-truth denominator for coverage verification and (in v2) the automated change-type detector for deployment attestation.

## Competitive landscape

| Tool | Type | Gap codeprint fills |
|---|---|---|
| `scc` | LOC counter CLI (Go) | No manifest/framework/test-split; no stable versioned schema |
| `tokei` | LOC counter CLI (Rust) | No manifest/framework/test-split; no stable versioned schema |
| `cloc` | LOC counter CLI (Perl) | Slow; no manifest/framework detection |
| `github-linguist` | Language detector (Ruby) | No LOC; UI-driven, not pipeline-driven |
| `go-enry` | Language detector (Go library) | Detection only — **used as codeprint dependency** |
| `onefetch` | Terminal summary (Rust) | Not data-producer |
| `syft` / `cyclonedx-cli` | SBOM generators | Dependency-centric, not file/code inventory |

**Positioning:** codeprint is *not* a faster `scc`. It's the **stable JSON contract** between a repo and N scanners.

## Core features (v1 — L2 scope)

1. **Language distribution + LOC** — per-language and per-file code/comment/blank split. Powered by `go-enry` detection + LOC counter.
2. **File classification** — source vs. test vs. generated vs. vendored vs. binary vs. minified.
3. **Build system detection** — 10 ecosystems hardcoded: JS/TS, Python, Java, Go, Ruby, C#, Rust, PHP, Kotlin, Swift.
4. **Framework hints** — extracted from manifest contents only (e.g., React from `package.json` deps, Django from `pyproject.toml`). Emitted as a list — never claim a single "framework."
5. **Output modes** — canonical JSON (schema-versioned), pretty-print for humans, single-line summary digest.
6. **Library API + thin CLI** — Go package consumable in-process; CLI is a reference producer.
7. **Published schema** — `codeprint.schema.json v1.0` with semver guarantees; breaking changes require major version bump.

## Growth features (v2+)

- **Diff mode** — fingerprint delta between two git refs (enables automated deployment change-type attestation: "was this really config-only?").
- Plugin system for custom ecosystems.
- `--format=cyclonedx` export for SBOM-adjacent consumers.
- Git metadata (contributor counts, file age, churn hints).
- Repo-size analytics (vendored bloat detection).
- Monorepo workspace awareness (npm/Cargo/Gradle multi-module).
- Incremental mode (delta fingerprint since last run) for CI speed.
- Live language-rule updates (track upstream linguist changes).

## Explicitly out of scope (forever)

- **Network / transport / storage** — codeprint is **produce-only**. Existing pipeline infra pushes output to JFrog / ELK and syncs to release tickets. codeprint never calls out.
- **Dashboards / reporting UI** — consumed downstream by existing tools.
- **SBOM generation with version resolution** — delegated to `syft` / `cyclonedx-cli`.
- **Security findings / CVE lookup** — delegated to cradar, Snyk, Trivy.
- **AST parsing / symbol extraction** — delegated to language-specific tools.
- **Code quality metrics (complexity, duplication)** — `scc` does this; we don't.
- **Coverage-gap computation** — consumer-side concern. Scanners compute their own coverage from codeprint's output; codeprint only emits the inventory.

## Key differentiators

1. **Schema-first product.** `codeprint.schema.json` is *the* deliverable; CLI is the reference producer.
2. **Versioned semver contract.** Consumers can depend on the shape. No silent breaking changes.
3. **Pipeline-native by design.** Built for CI/scanner consumption from day 1, not retrofitted.
4. **Broad but cheap.** Language + LOC + manifests + framework in a single pass; no AST, no dep resolution.
5. **Stands on go-enry.** Inherits github-linguist's 500+ language ruleset without reinventing.

## Open questions (resolve in later phases)

- **Performance target** — what's the budget for a mid-sized repo (1M LOC)? → Feasibility (1.3)
- **Monorepo handling** — flat fingerprint only, or per-workspace sub-fingerprints? → Architecture (4.1)
- **Default ignore rules** — follow `.gitignore`, also `.dockerignore`, add `.codeprintignore`? → Architecture (4.1)
- **Per-file records vs. rollups only** — per-file is richer for scanners but larger output. → Architecture (4.1)

## Risks flagged early

| Risk | Impact | Mitigation |
|---|---|---|
| Schema-first commitment is heavy — locking v1.0 too early hurts later | Forced major bumps / consumer churn | Run a long v0.x pre-1.0 phase, solicit feedback from cradar/sonar integrators before locking |
| `go-enry` is a single point of failure | Slow to pick up new languages if upstream stalls | Keep detection logic pluggable behind an interface; fall back to own rules if needed |
| Schema-first framing is wasted overhead if only 1-2 consumers adopt it | Design tax without payoff | Use internally first (cradar as first consumer), prove value, then evangelize |
| Framework detection is fuzzy — "package.json has react" ≠ "is a React app" | False positives in downstream tools | Emit `framework_hints` as a **list**, not singular. Confidence score optional |
| Lifecycle duplication with cradar's existing file-type classifier | Two sources of truth | Mark cradar's classifier as a future retirement candidate; eventually cradar imports codeprint's library |

## Synthesis

codeprint is a **fingerprint producer**. It reads a codebase and emits a stable, versioned JSON inventory. Every design decision — tech, scope, API surface, output modes — serves that single contract. It does not count better than `scc`, detect better than `linguist`, or analyze like an SBOM tool. It fills the pipeline-contract gap none of those occupy.

**Deployment scale:** ~18,000 unique repos. codeprint runs locally in each CI pipeline and writes a JSON artifact. Existing org infra (JFrog, ELK, release-ticket sync) handles transport and storage. codeprint stays a pure, offline, single-binary producer.

## Change log

- 2026-04-21: initial draft
