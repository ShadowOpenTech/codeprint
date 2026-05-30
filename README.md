# codeprint

> A fast, produce-only Go CLI + library that emits a stable, versioned JSON **fingerprint** of a codebase — languages, LOC, file classification, build systems, and framework hints.

codeprint is the scanner-agnostic **ground-truth inventory** of "what's in this repo." It is built for CI pipelines and security/coverage tooling: one fingerprint, one stable schema, consumed by many tools. It is **not** a faster `scc` — it's the stable JSON contract between a repo and N scanners.

## What it produces

```jsonc
{
  "_meta": { "schema_version": "1.0", "producer": { "name": "codeprint", "version": "0.1.0" } },
  "fingerprint": {
    "totals":         { "files": 412, "code": 38120, "comment": 6402, "blank": 3690, "bytes": 1840221 },
    "languages":      [ { "language": "Go", "files": 284, "code": 38120, "percent": 62.3, "comment_aware": true } ],
    "files":          [ { "path": "main.go", "language": "Go", "kind": "source", "flags": [], "code": 42 } ],
    "build_systems":  [ { "ecosystem": "go-modules", "path": "go.mod" } ],
    "framework_hints":[ { "name": "gin", "ecosystem": "go-modules", "confidence": "high", "evidence": {...} } ],
    "container":      { "dockerfile_present": true },
    "errors":         [],
    "workspaces":     null
  }
}
```

## Usage

```bash
codeprint                 # scan the current directory (pretty output at a TTY)
codeprint /path/to/repo   # scan a specific path
codeprint . | jq '.fingerprint.languages'   # piped → canonical JSON
codeprint . --out codeprint.json             # write JSON to a file
codeprint . --format=summary                 # one-line digest
```

Output auto-detects: a terminal gets a colored summary; a pipe or `--out` file gets canonical JSON.

### Key flags

| Flag | Effect |
|---|---|
| `--format json\|pretty\|summary` | force output mode (default: auto by TTY) |
| `--out <file>` | write JSON to a file |
| `--no-files` | rollups only (smaller output) |
| `--no-hidden` | ignore dotfiles |
| `--max-file-size <n>` | per-file ceiling (default 10MB) |
| `--concurrency <n>` | worker count (default: NumCPU) |
| `--strict-io` | exit non-zero if any file was unreadable |
| `--verbose` / `--debug` | progress / per-file diagnostics on stderr |

Exit codes follow `sysexits.h`: `0` ok, `64` usage, `66` no input, `70` software, `74` I/O (with `--strict-io`).

## Library

```go
import "github.com/ShadowOpenTech/codeprint/pkg/codeprint"

fp, err := codeprint.Scan(ctx, "/path/to/repo",
    codeprint.WithConcurrency(8),
    codeprint.WithoutFileRecords(),
)
```

`Scan` returns a `*Fingerprint` that mirrors the JSON schema 1:1 — no file round-trip needed.

## Guarantees

- **Deterministic** — byte-identical `fingerprint` for identical input.
- **Offline** — zero network calls, ever.
- **Single static binary** — `CGO_ENABLED=0`, no runtime dependencies.
- **Stable schema** — `schema/codeprint-v1.schema.json`, generated from the Go types and semver-governed (locked at v1.0.0).

## Install

Download a binary from [Releases](https://github.com/ShadowOpenTech/codeprint/releases), or:

```bash
go install github.com/ShadowOpenTech/codeprint/cmd/codeprint@latest
```

## Status

Pre-1.0 (v0.x): the schema may change between releases until v1.0.0 GA. See [`docs/`](docs/) for the full design, architecture, and planning record, and [`docs/index.md`](docs/index.md) for the map.

## Development

```bash
git clone --recurse-submodules https://github.com/ShadowOpenTech/codeprint
make build      # CGO_ENABLED=0 static binary → bin/codeprint
make verify     # everything CI runs: fmt, vet, lint, schema-drift, test, coverage, vuln
make bench      # throughput benchmark
```

The correctness test corpus is a submodule ([codeprint-testcorpus](https://github.com/ShadowOpenTech/codeprint-testcorpus)) — clone with `--recurse-submodules`.

## License

[Apache-2.0](LICENSE) © ShadowOpenTech.
