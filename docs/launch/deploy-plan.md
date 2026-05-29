# codeprint — Deploy Plan

**Purpose:** define the release, distribution, and CI/CD strategy. codeprint ships as a binary + Go library, not a hosted service.
**Phase:** 6.1 Deploy Plan
**Status:** confirmed
**Updated:** 2026-05-30

## "Deploy" = release & distribution

codeprint has no server, no runtime infra to host. Deployment means **producing release artifacts and distributing them** to the org's pipelines and to library consumers. There is nothing running between releases.

## CI/CD platform — GitHub Actions

codeprint's own repo is on GitHub (`ShadowOpenTech/codeprint`), so its build/release CI is GitHub Actions. (The ~18k *consumer* repos live on Bitbucket — but that is consumption of the published binary, not codeprint's build.)

### CI workflow (every PR + push)

```
lint (gofumpt + golangci-lint) → go vet → test (race + coverage)
  → coverage gate (85% pipeline) → determinism (twice, diff fingerprint)
  → schema-drift (regenerate vs committed) → govulncheck
  → network-disabled build (proves zero network calls, NFR-8)
```

Every gate from [../planning/code-quality.md](../planning/code-quality.md) and [../planning/test-plan.md](../planning/test-plan.md). Submodule corpora fetched before test/bench. Benchmark job runs the perf-regression gate (>10%).

### Release workflow (on version tag)

```
git tag vX.Y.Z → push → workflow:
  goreleaser → cross-compile 5 platforms (CGO_ENABLED=0)
    → checksums → changelog (from Conventional Commits)
    → publish to JFrog (primary) + GitHub Releases (mirror)
```

## Release artifacts (NFR-3)

| Platform | Built |
|---|---|
| linux/amd64, linux/arm64 | `CGO_ENABLED=0` static |
| darwin/amd64, darwin/arm64 | static |
| windows/amd64 | static |

Plus SHA256 checksums. Binary target <30MB stripped (NFR-3). Code-signing is a forward path (not Release 0).

**Container image: NOT shipped for Release 0.** Binary-only matches the established consumption path (the org's scan plugin pulls the binary from JFrog, flows F1). A container adds a registry + image-scan surface with no current consumer; revisit if a containerized pipeline asks.

## Distribution channels

| Channel | Role | Mechanism |
|---|---|---|
| **JFrog** | Primary — org scan plugin pulls the binary | goreleaser upload (flows F1) |
| **GitHub Releases** | Public mirror | goreleaser |
| **Go module proxy** (`go get`) | Library consumers (cradar, in-process) | automatic on tag via `proxy.golang.org` |
| **Schema** | `schema/codeprint-v1.schema.json` at tag-pinned raw URL; schemastore registration post-v1.0.0 GA | F-7 |

## Versioning & release trigger

- **Semver.** v0.x is the pre-1.0 phase — schema may change between releases (F-7). Lock at v1.0.0 GA.
- **Release trigger: manual semver tag.** A human pushes `vX.Y.Z`; the tag fires the release workflow. Conventional Commits (5.3) auto-generate the changelog, but **version timing stays deliberate** — important in v0.x when schema changes should be batched into a considered release, not auto-cut on every merge.
- Breaking schema change in v0.x → minor bump + changelog warning; post-1.0 → major bump.

## Operational concerns

- **Rollback:** releases are immutable artifacts; "rollback" = consumers pin a prior version. Stored fingerprints from any version remain valid forever (F-7 backward-compat).
- **Provenance:** each release's `_meta.producer.version` lets consumers trace which binary produced a fingerprint (flows F2).
- **No telemetry, no phone-home** (NFR-8). Release health is observed via GitHub Releases / JFrog download stats only.
- **Schema hosting:** the tag-pinned raw GitHub URL is the immutable reference at GA; pre-1.0 uses a floating/branch URL or omits `$schema` (F-7 open question, resolved per spec).

## Release 0 checklist (ties to M5, [../planning/task-breakdown.md](../planning/task-breakdown.md))

- [ ] GitHub Actions CI workflow (all gates green)
- [ ] goreleaser config (5 platforms + checksums + JFrog + GH Releases)
- [ ] Tag `v0.1.0`
- [ ] Verify scan plugin can pull v0.1.0 binary from JFrog
- [ ] Verify `go get` of the library at v0.1.0
- [ ] Schema published at the pre-1.0 URL

## Change log

- 2026-05-30: initial draft; confirmed. GitHub Actions CI + goreleaser release; binary-only (no container) for Release 0; JFrog primary + GH Releases mirror + Go proxy; manual semver-tag trigger (deliberate versioning in v0.x); immutable artifacts, no telemetry.
