# codeprint — Project Instructions

codeprint is a produce-only Go CLI + library that emits a stable, versioned JSON fingerprint of a codebase (languages, LOC, file classification, build systems, framework hints). It is **schema-first**: the Go structs in `pkg/codeprint` are the source of truth, and the published JSON schema is the product's contract.

Read the planning docs before non-trivial work — they hold the decisions and their rationale:
- Product: `docs/design/product-spec.md`, `docs/design/user-flows.md`
- Architecture: `docs/architecture/{system-architecture,data-model,api-contracts,error-strategy,security-audit}.md`
- Planning: `docs/planning/{test-plan,code-quality,conventions,env-setup,task-breakdown}.md`
- Release: `docs/launch/deploy-plan.md`
- Master index: `docs/index.md`

## Non-negotiable invariants

These define the product. Breaking one is a bug, not a tradeoff:

- **Determinism** — byte-identical `fingerprint` for identical input. **Never range a Go map into output**; collect into a slice and sort by the documented stable key (`docs/architecture/data-model.md`). Enforced by the determinism test.
- **Produce-only / offline** — no network calls, ever. Verified by the network-disabled build test.
- **No CGo** — `CGO_ENABLED=0`; single static binary.
- **Content never emitted** — file contents are read but never appear in output, errors, or logs (only metadata). `errors[].reason` is the OS error, never file bytes.
- **Schema = struct** — change a struct → regenerate schema (`make schema`); CI fails on drift.
- **Stable contract** — any change to the `pkg/codeprint` exported surface or the schema requires explicit reviewer sign-off.

## Quality & security enforcement

### Before every commit
- Run `/guard` (or `make verify`) — lint, format, vet, tests, determinism, schema-drift, vuln scan.
- Do NOT commit if anything fails. Fix first.
- Commit messages: **Conventional Commits** (`feat:`/`fix:`/`docs:`/…); no `Co-Authored-By` trailers.

### After completing a feature
- Run `/sec-audit` on changed files; address critical/high findings before merging.
- Run `/perf-audit` if the change touches the scan pipeline (walk/detect/loc/classify) — see `docs/planning/code-quality.md`.

### Before every milestone (M0–M5)
- Run `/quality-report`. All scores B+ to proceed.

### Standards references
- Quality + perf benchmarks: `docs/planning/code-quality.md`
- Testing + corpora: `docs/planning/test-plan.md`
- Security: `docs/architecture/security-audit.md`
- Conventions: `docs/planning/conventions.md`

## Workflow

- Each lifecycle phase / milestone is developed on its own branch; push it; **never merge to `main` without explicit approval**.
- Every change lands via PR with green CI.

## Non-negotiable (security)
- No hardcoded secrets, ever.
- No lint errors at commit time.
- No known vulnerabilities in dependencies (`govulncheck`).
- Pipeline-package test coverage ≥ 85% (`docs/planning/code-quality.md`).
- All untrusted input (file names, contents, manifests, symlink targets) treated as hostile; control chars sanitized in TTY output; symlinks contained to the workspace root.
