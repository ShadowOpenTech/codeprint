# codeprint — Code Quality

**Purpose:** define linting, formatting, static analysis, review standards, and the performance benchmark gate.
**Phase:** 5.2 Code Quality
**Status:** confirmed
**Updated:** 2026-05-30

## Toolchain

| Concern | Tool | Enforcement |
|---|---|---|
| Formatting | **gofumpt** (stricter gofmt superset) | CI gate; auto-fixable; non-negotiable |
| Import ordering | `goimports` (via golangci-lint) | CI gate |
| Vetting | `go vet` | CI gate |
| Static analysis + lint | **golangci-lint** (curated set below) | CI gate |
| Security lint | **gosec** (within golangci-lint) | CI gate |
| Vuln scanning | `govulncheck` | CI gate (also in test-plan) |

## golangci-lint set — curated balanced

Explicit enabled linters (not enable-all — avoids the funlen/wsl/godox noise that breeds `//nolint` clutter):

```
govet, staticcheck, errcheck, ineffassign, unused,
gosec, revive, misspell, gocritic, bodyclose
```

- **errcheck** — no silently dropped errors (critical for NFR-5 error tolerance).
- **gosec** — backs security-audit findings (no `unsafe`, no command exec, etc.).
- **bodyclose** — defensive even though codeprint makes no network calls.
- New `//nolint` directives require an inline justification comment, reviewed in PR.

## Formatting & style

- `gofumpt` formatting is mandatory; CI fails on diff.
- No custom style beyond gofumpt + the linter set — the toolchain *is* the style guide. Naming/structure conventions live in [conventions.md](./conventions.md) (5.3).

## Review standards

- Every change lands via PR; no direct pushes to `main` (project workflow).
- **CI must be green** (all gates below) before merge.
- `internal/*` may change freely between releases — no API promise.
- **Any change to the `pkg/codeprint` exported surface or the JSON schema requires explicit reviewer sign-off** — it is the stability contract (flows F2, F-7). Schema drift is caught mechanically (test-plan), but intent still needs human review.
- No merge with failing determinism, schema-drift, or coverage gates.

## Performance benchmark gate (NFR-1)

- `go test -bench` on the performance corpus (pinned-submodule big repos — see [test-plan.md](./test-plan.md)).
- **Hard gate: >10% throughput regression vs baseline fails the build.** This is the literal NFR-1 rule and is stable across runs.
- **Absolute ≥300k LOC/sec is tracked and reported, NOT a hard gate.** CI hardware variance would make an absolute floor flaky and produce failures unrelated to code changes. If a dedicated stable benchmark runner is provisioned later, an absolute floor can be promoted to a gate.
- Baseline stored per-branch; updated deliberately when a known-good perf change lands.

## CI gate summary

| Gate | Fails when |
|---|---|
| gofumpt | formatting diff |
| golangci-lint | any enabled linter fires |
| go vet | vet finding |
| `go test ./...` | test failure |
| coverage | pipeline packages < 85% (test-plan) |
| determinism | two runs differ in `fingerprint` |
| schema drift | generated ≠ committed schema |
| benchmark | >10% throughput regression |
| `govulncheck` | known vuln in deps |
| network-disabled build | any network call attempted |

These wire into Phase 7 `CLAUDE.md` (`/guard`, `/sec-audit`, `/perf-audit`).

## Change log

- 2026-05-30: initial draft; confirmed. gofumpt + curated golangci-lint set (10 linters); perf gate is relative-regression-only (>10%), absolute LOC/sec tracked not gated; exported-surface/schema changes require reviewer sign-off.
