# codeprint — Coding Conventions

**Purpose:** define naming, structure, error, doc, and commit conventions for the codebase.
**Phase:** 5.3 Convention
**Status:** confirmed
**Updated:** 2026-05-30

## Structure

- **Feature-based `internal/` packages** (4.1) — `detect`, `loc`, `classify`, `walk`, `buildsys`, `framework`, `emit`, `schemagen`. No `utils`/`common`/`helpers` grab-bag packages.
- Public surface is exactly `pkg/codeprint`; CLI is `cmd/codeprint`. Nothing else exported.
- One concern per package; cross-package types that are part of the contract live in `pkg/codeprint/types.go`.

## Naming

- Go-standard naming. Exported identifiers documented; unexported for everything not in the contract.
- **JSON tags are snake_case** (`BuildSystems` → `json:"build_systems"`), matching the schema and data-model field names.
- Functional options are `WithX` (4.3). The unexported config struct is `config`.

## API shape

- `ctx context.Context` is always the **first parameter** of any function that does I/O or long work (4.3).
- **No package-level mutable state** in `pkg/codeprint` — `Scan` must be safe to call concurrently from multiple goroutines (in-process consumers like cradar may do so).

## Determinism discipline (codeprint-specific, non-negotiable)

> **Never range a Go map directly into output.** Map iteration order is randomized by the runtime. Any data destined for the `fingerprint` payload must first be collected into a slice and **sorted by its documented stable key** (see [../architecture/data-model.md](../architecture/data-model.md) sort-key table) before serialization.

This is the single easiest way to break NFR-2. It is enforced in code review and caught mechanically by the determinism test (run twice, diff). Treat any `for k := range someMap` that feeds output as a bug.

## Errors

- Wrap with context: `fmt.Errorf("reading %s: %w", path, err)`. Never return a bare lower-level error across the package boundary.
- Exported sentinels (`ErrUsage`, `ErrUnreadableRoot`) for fatal cases consumers branch on (4.4); use `errors.Is`.
- Per-file failures go to `fingerprint.errors`, not returned (4.4 / NFR-5).

## Documentation

- Doc comment on **every exported symbol** (godoc convention), enforced by `revive` (5.2).
- Package-level doc comment on `pkg/codeprint` stating the contract and the determinism guarantee.
- Comments explain *why*, not *what*; match surrounding density.

## Tests

- Table-driven tests where natural.
- Golden files updated via a `-update` flag convention (`go test -run X -update`).
- `testdata/` for local hand-crafted fixtures; the correctness/performance corpora are submodules (5.1).

## Commits & PRs

- **Conventional Commits** — `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`, `perf:`, `build:`, `ci:`. Enables automated changelog + semver inference, which matters for a semver-contract product (the schema version follows releases — F-7).
  - Breaking changes: `feat!:` or a `BREAKING CHANGE:` footer → drives the major bump.
  - Lightly enforced via a commit-lint CI check.
- No `Co-Authored-By` trailers.
- Every change via PR; no direct pushes to `main` (project workflow); CI green required (5.2).

## Change log

- 2026-05-30: initial draft; confirmed. Feature-based packages, snake_case JSON tags, ctx-first, no package-level mutable state, the map-ranging determinism rule, error-wrapping, godoc on exported symbols, Conventional Commits.
