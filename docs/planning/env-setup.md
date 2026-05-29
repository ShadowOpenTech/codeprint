# codeprint — Environment Setup

**Purpose:** define dev/build requirements, setup steps, and the local development workflow.
**Phase:** 5.4 Env Setup
**Status:** confirmed
**Updated:** 2026-05-30

## No runtime config — by design

codeprint has **no config file and no environment variables** for its own behavior (flags-only, per spec NFR / F-3). The one env var it *honors* is `NO_COLOR` (disables color). Therefore **there is no `.env.example` for the tool** — its absence is intentional, not an omission. Everything below concerns the **developer/build environment**, which does not ship.

## Requirements

| Tool | Purpose | Version |
|---|---|---|
| Go | build + test | 1.23+ (module floor, F-6) |
| git | clone + submodule corpora | recent |
| make | task runner (dev/CI convenience; not shipped) | any |
| golangci-lint | lint gate | pinned (`make tools`) |
| gofumpt | formatting gate | pinned |
| govulncheck | vuln gate | pinned |

The **shipped binary** is produced by `CGO_ENABLED=0 go build` — no CGo, no runtime deps (NFR-3). The Makefile only wraps that command; it is not part of the artifact.

## Setup

```bash
git clone --recurse-submodules git@github.com:ShadowOpenTech/codeprint.git
cd codeprint
go mod download
make tools     # install golangci-lint, gofumpt, govulncheck at pinned versions
make build     # → ./bin/codeprint (static, CGO_ENABLED=0)
```

Already cloned without submodules? `git submodule update --init --recursive` (pulls the correctness + performance corpora — see [test-plan.md](./test-plan.md)).

## Makefile targets

```
make build        CGO_ENABLED=0 go build -o bin/codeprint ./cmd/codeprint
make test         go test ./... (race + coverage)
make lint         gofumpt -l (check) + golangci-lint run
make fmt          gofumpt -w (apply)
make bench        go test -bench on the performance corpus (NFR-1)
make determinism  scan the corpus twice, diff fingerprint (NFR-2)
make schema       regenerate schema/codeprint-v1.schema.json from structs
make vuln         govulncheck ./...
make verify       run everything CI runs — the local /guard equivalent
make tools        install pinned dev tools
make clean        rm -rf bin/ coverage output
```

`make verify` is the pre-push gate: it mirrors CI exactly so a green local run means a green CI run.

## `.gitignore` additions

```
/bin/
*.out          # coverage profiles
```

(The repo already has a base `.gitignore`; these are appended.)

## Release builds (forward pointer)

Cross-compilation for the 5 target platforms (NFR-3: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64) is **goreleaser's** job, defined in the Phase 6 deploy plan — not the Makefile. `make build` is for local single-platform builds only.

## Change log

- 2026-05-30: initial draft; confirmed. Make-based dev workflow; no `.env.example` (tool has no runtime config by design — honors only NO_COLOR); submodule-aware clone; `make verify` as the local CI mirror. Release cross-compile deferred to Phase 6 (goreleaser).
