# codeprint — User Flows

**Purpose:** invocation and integration patterns for codeprint across its user personas.
**Phase:** 3.1 Flow
**Status:** confirmed
**Updated:** 2026-04-22

## Overview

codeprint is a CLI + library, not a GUI app, so "flows" here mean **invocation and integration patterns**, not clickstreams. Four flows, two primary.

| # | Flow | Priority | Persona(s) |
|---|---|---|---|
| F1 | CI pipeline integration | Primary | P3 DevOps (setup), P1 AppSec (consumes) |
| F2 | Scanner integration via library | Primary | P2 Scanner Maintainer |
| F3 | Local developer use | Secondary | Any engineer |
| F4 | Schema consumption via stored JSON | Primary | P2 (retrieves from JFrog/ELK) |

---

## F1 — CI pipeline integration

**Steps:**

1. **Checkout** — pipeline clones the repo (existing step).
2. **Ensure binary** — `codeprint` binary becomes available on the runner.
3. **Run** — `codeprint /workspace --out codeprint.json`.
4. **Push** — existing pipeline infra uploads `codeprint.json` to JFrog/ELK.
5. **Release-ticket sync** — existing infra reads the stored JSON and attaches to ticket.

Steps 1, 4, 5 already exist. codeprint owns only steps 2–3.

**Decisions:**

| Decision | Choice |
|---|---|
| Binary distribution | Pulled from JFrog by the org's **custom scan plugin**; codeprint becomes one of the scanners the plugin orchestrates |
| Exit codes (codeprint) | sysexits.h-aligned — see [product-spec.md NFR-6](./product-spec.md#nfr-6--exit-codes-sysexitsh-aligned). `0` success (incl. empty workspace), `64` usage, `66` no input, `70` software, `74` I/O (only with `--strict-io`) |
| Fail-the-build policy | **Lives in the CI plugin, not codeprint.** Default: plugin ignores codeprint exit, logs warning on non-zero, never fails the job. Plugin's `--strict` flag propagates non-zero to fail |
| Determinism | **Hard requirement.** Byte-identical output for identical input. Sorted lists, sorted JSON keys, no timestamps in the `fingerprint` data. Enforced by test (`go test` runs codeprint twice, asserts equal) |
| Monorepo handling | v1 = flat single-tree fingerprint. v2 adds per-workspace via reserved optional `workspaces: []` field (null in v1) |
| Stdout vs file | JSON goes to `--out` file OR stdout (pipe). Logs always to stderr. Never mixed |

---

## F2 — Scanner integration via library

**Steps:**

1. Scanner adds `github.com/ShadowOpenTech/codeprint` as a module dependency.
2. Calls `codeprint.Scan(root, opts...)` in-process.
3. Receives `*Fingerprint` struct (mirrors the JSON schema 1:1).
4. Uses fields directly — no file round-trip, no unmarshal.
5. Scanner writes its own report; never re-emits codeprint's output.

**Decisions:**

| Decision | Choice |
|---|---|
| API shape | **Functional options pattern.** `codeprint.Scan(root string, opts ...Option) (*Fingerprint, error)`. Non-breaking growth; new options don't change signature. 10+ years community-validated idiom |
| Reference comparables | [`anchore/syft`](https://github.com/anchore/syft) + [`anchore/grype`](https://github.com/anchore/grype) use the same library-and-CLI shape for adjacent problems — strong validation |
| Library vs CLI source of truth | **Library is source of truth.** CLI is thin `cmd/codeprint` wrapper; Go struct tags drive the published JSON schema |
| Version stamps | **In a `_meta` envelope, never inside `fingerprint` data.** Output = `{ "_meta": { "schema_version": "1.0", "producer": {...} }, "fingerprint": {...} }`. Meta can change without triggering spurious diffs in v2 diff-mode |
| Go version floor | `go 1.23` — N-2 from current latest (Go 1.26). Matches community best practice for libraries |
| Cross-reference in scanner output | Scanner reports include `codeprint._meta.producer.version` + `schema_version` so downstream knows provenance |

---

## F3 — Local developer use

**Steps:**

1. Engineer runs `codeprint` (zero args) or `codeprint /some/path` in their shell.
2. Sees a pretty-printed, colored summary.
3. Optionally pipes: `codeprint . | jq '.fingerprint.languages'`.

**Decisions:**

| Decision | Choice |
|---|---|
| Format auto-detection | stdout TTY → pretty-print (color); stdout piped/redirected → canonical JSON |
| Color | Auto-on for TTY, off for non-TTY; honors `NO_COLOR` env var universally |
| Zero-arg invocation | `codeprint` = `codeprint .` (matches `prettier`, `eslint`, `scc`, `tokei`) |
| Config file | None. Flags only |
| Subcommands | Only one: `codeprint completion [bash\|zsh\|fish\|powershell]` (Cobra boilerplate). v2 will add `codeprint diff` |
| Explicit format override | `--format=json` or `--format=pretty` always wins over TTY detection |
| TTY/color libraries | [`fatih/color`](https://github.com/fatih/color) + `mattn/go-isatty` |

---

## F4 — Schema consumption via stored JSON

**Steps:**

1. Consumer (scanner, dashboard, release-ticket system) retrieves `codeprint.json` from JFrog or ELK using existing infra.
2. Optional: validate JSON against hosted schema.
3. Parse into Go struct (reuse exported `codeprint.Fingerprint`) or a language-native type.
4. Use fields.

**Decisions:**

| Decision | Choice |
|---|---|
| Schema canonical location | `schema/codeprint-v1.schema.json` in the repo, generated from Go structs via [`invopop/jsonschema`](https://github.com/invopop/jsonschema); CI asserts no drift |
| Immutable hosted URL | `https://raw.githubusercontent.com/ShadowOpenTech/codeprint/v1.0.0/schema/codeprint-v1.schema.json` (tag-pinned) |
| Public registry | [schemastore.org](https://www.schemastore.org/) — register post-v1.0 GA, not day 1 |
| Version pointers in output | `$schema` field = immutable URL; `_meta.schema_version = "1.0"` quick check |
| Go validator for consumers | [`santhosh-tekuri/jsonschema/v6`](https://github.com/santhosh-tekuri/jsonschema) — active, supports draft 2020-12. Reject `xeipuuv/gojsonschema` (stale) |
| Backward-compat policy | Files stored at v1 remain valid v1 forever. v2 release never rewrites stored v1 files. v1.x patch support continues after v2 ships |
| Pre-1.0 discipline | v0.x may change freely; semver lock starts at v1.0.0 |

---

## Cross-flow design implications

| Implication | Drives design in |
|---|---|
| Determinism is a hard requirement | `Fingerprint` serialization, file-walker ordering, concurrency collector |
| `_meta` envelope vs `fingerprint` data split | JSON struct layout, CI diff-test, diff-mode (v2) |
| Functional options pattern | Public API surface |
| One entry point (`Scan`) | Library design; keeps deprecation blast radius small |
| Schema file generated from Go structs | Release workflow; schema validation in CI |
| Backward-compat for stored files across majors | Release policy; long-term binary tagging |

## Change log

- 2026-04-22: initial draft; all four flows confirmed.
- 2026-05-27: corrected F1 exit-code row — replaced restated `0/2/3` values with a link to NFR-6 (sysexits `0/64/66/70/74`). The restated values conflicted with the spec and violated the "link, don't restate" rule in STANDARD.md.
