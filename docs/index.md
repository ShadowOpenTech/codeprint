# codeprint — Master Index & Project Kickoff

**Status:** planning complete — ready for development (M0).
**Updated:** 2026-05-30

## What codeprint is

A produce-only Go **CLI + library** that emits a stable, versioned JSON **fingerprint** of a codebase — languages, LOC (code/comment/blank), file classification, build systems, and framework hints. It is the **scanner-agnostic ground-truth denominator** for coverage verification across ~18k repos: one fingerprint, stable schema, consumed by many tools (SonarQube, cradar, Nexus IQ, the custom API security tool, …).

It is **not** a faster `scc` — it's the stable JSON contract between a repo and N scanners.

## Confirmed scope

- **Schema-first**: Go structs in `pkg/codeprint` are the source of truth; the published schema is the contract.
- **Release 0 = v0.1.0 = full feature set** (M0–M5), incl. framework hints + complete CLI UX. Semver lock deferred to v1.0.0 GA after consumer feedback.
- **Invariants**: deterministic, produce-only/offline, no CGo, single static binary, content never emitted.

## Document map

| Phase | Doc | Status |
|---|---|---|
| 1 Discovery | [discovery/ideation.md](discovery/ideation.md) · [user-personas.md](discovery/user-personas.md) · [feasibility-study.md](discovery/feasibility-study.md) | ✓ |
| 2 Brand | [discovery/brand-identity.md](discovery/brand-identity.md) · `assets/logo.svg` | ✓ confirmed |
| 3 Design | [design/user-flows.md](design/user-flows.md) · [design/product-spec.md](design/product-spec.md) | ✓ confirmed |
| 4 Architecture | [architecture/system-architecture.md](architecture/system-architecture.md) · [data-model.md](architecture/data-model.md) · [api-contracts.md](architecture/api-contracts.md) · [error-strategy.md](architecture/error-strategy.md) · [security-audit.md](architecture/security-audit.md) | ✓ confirmed |
| 5 Planning | [planning/test-plan.md](planning/test-plan.md) · [code-quality.md](planning/code-quality.md) · [conventions.md](planning/conventions.md) · [env-setup.md](planning/env-setup.md) · [task-breakdown.md](planning/task-breakdown.md) | ✓ confirmed |
| 6 Launch | [launch/deploy-plan.md](launch/deploy-plan.md) | ✓ confirmed |
| — | [STANDARD.md](STANDARD.md) — doc standard | ✓ |

## Tech stack (locked)

Go 1.23+ · `go-enry/v2` (detection) · `gocloc` (LOC) · `cobra` (CLI) · `invopop/jsonschema` (schema) · `denormal/go-gitignore` (ignore rules) · `BurntSushi/toml` · `fatih/color`+`go-isatty`. **No CGo.** Module: `github.com/ShadowOpenTech/codeprint`.

## Key risks & mitigations

| Risk | Mitigation |
|---|---|
| Schema locked too early | Long v0.x pre-1.0; cradar as first consumer before v1.0.0 lock |
| enry single point of failure | `Detector` interface — detection swappable |
| Framework detection fuzzy | Hints are a **list** with confidence; scope-lever feature |
| Gradle Groovy DSL parsing | Regex fallback, documented partial coverage |
| Determinism regressions | Map-ranging ban + twice-and-diff CI gate + `/check-determinism` |

## Milestone 1 (where development starts)

After **M0** (scaffold, schema-first CI, corpus submodules), **M1** delivers the MVP core: `walk` (+ignore +symlink containment) → `detect` → `loc` → `Scan` collector → deterministic JSON. A working `codeprint .` emitting a correct language/LOC fingerprint. Full task list: [planning/task-breakdown.md](planning/task-breakdown.md).

## Enforcement (active via root `CLAUDE.md`)

- Before commit: `/guard` / `make verify`
- After feature: `/sec-audit`, `/perf-audit` (pipeline changes)
- Per milestone: `/quality-report`
- Project skills: `/add-ecosystem`, `/sync-schema`, `/check-determinism`, `/add-fixture`

## Repo & releases

- GitHub: `github.com/ShadowOpenTech/codeprint`
- CI: GitHub Actions (all quality/security gates)
- Release: goreleaser on manual semver tag → JFrog (primary) + GitHub Releases (mirror) + Go proxy
- Contributors: nk-sentinel, blizesg
