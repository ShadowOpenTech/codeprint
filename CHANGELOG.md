# Changelog

All notable changes to codeprint are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/). The schema contract is **not**
locked until v1.0.0 — during v0.x it may change between releases.

Per-release notes are also generated automatically on GitHub Releases from
Conventional Commits; this file is the curated human summary.

## [Unreleased]

### Planned (tracked as GitHub issues)
- #14 — symlink resolved-path dedup (fix in-tree symlink LOC double-count; enable safe symlinked-dir following).
- #16 — decide whether `.github/` workflows classify as `source` vs `vendored`.

## [0.1.0] - 2026-05-30

First release — the full v1 feature set (M0–M5).

### Added
- **Scan engine** — `Scan(ctx, root, opts...)` producing a deterministic `*Fingerprint`: bounded worker pool, single sorting collector, per-file panic recovery, clean ctx cancellation.
- **Language + LOC** — go-enry detection + gocloc comment-aware counts, with a non-comment-aware fallback (`comment_aware: false`).
- **File classification** — `source` / `test` / `generated` / `minified` / `vendored` / `binary`, plus non-exclusive `flags` (e.g. `skipped_large`). Priority: `binary > minified > vendored > generated > test > source`.
- **Build-system detection** — flat full-tree marker scan with paths across 10 ecosystems + `container.dockerfile_present`.
- **Framework hints** — per-ecosystem manifest parsers (npm, pip, maven, gradle, bundler, cargo, composer), emitted as a list with confidence + evidence.
- **Symlink report** — `symlinks: { total, followed_file, skipped[{path,reason}] }`; makes traversal gaps auditable. Escaping symlinks are contained (NFR-8); symlinked directories are not descended (loop-safe).
- **Output modes** — canonical JSON (`$schema`/`_meta`/`fingerprint` envelope), human `pretty` report, one-line `summary`; TTY auto-detection; control-char sanitization of untrusted filenames.
- **CLI** — cobra-based `codeprint [path]` with `--format/--out/--pretty-json/--no-files/--no-hidden/--ignore-file/--max-file-size/--concurrency/--strict-io/--verbose/--debug/--no-color`, `completion` subcommand, sysexits exit codes (0/64/66/70/74) + signal handling, and a TTY-gated progress spinner.
- **Library API** — functional options (`WithConcurrency`/`WithIgnoreFile`/`WithIncludeHidden`/`WithMaxFileSize`/`WithoutFileRecords`); `*Fingerprint` mirrors the JSON schema 1:1.
- **Published schema** — `schema/codeprint-v1.schema.json`, generated from the Go types with a CI drift gate.
- **Release tooling** — goreleaser (5 static platforms, `CGO_ENABLED=0`), GitHub Actions CI (lint, vet, test, coverage ≥85%, determinism, schema drift, govulncheck, benchmark) and release workflows.

### Guarantees
- Deterministic byte-identical `fingerprint` for identical input.
- Offline — zero network calls (build-time verified). No CGo; single static binary.
- File contents read but never emitted; paths are repo-relative.

[Unreleased]: https://github.com/ShadowOpenTech/codeprint/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ShadowOpenTech/codeprint/releases/tag/v0.1.0
