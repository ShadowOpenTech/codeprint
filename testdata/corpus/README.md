# codeprint-testcorpus

Purpose-built correctness fixture for [codeprint](https://github.com/ShadowOpenTech/codeprint). Consumed as a **SHA-pinned git submodule** by codeprint's golden / TP-FP-FN tests (see codeprint `docs/planning/test-plan.md`).

This corpus has **known ground truth** declared in [`expected.json`](./expected.json). codeprint's output is compared against it to measure precision/recall on:

- **Language detection** — Go, Python, JavaScript, TypeScript, Java, Ruby, Rust
- **File classification** — `source`, `test`, `generated` (header), `minified` (`.min.js`), `vendored` (`vendor/`, `node_modules/`), `binary`
- **Build-system detection** — go-modules, npm, pip, maven, cargo, bundler (flat full-tree, each with its marker path)
- **Framework hints** — react, express (npm); django, flask (pip); spring-boot (maven); actix (cargo); rails (bundler)
- **Container** — `docker/Dockerfile`
- **Symlinks** — an in-tree symlink (followed) and an escaping symlink (must be skipped per codeprint NFR-8 containment)

It contains **no secrets** — every file is synthetic. Do not add real credentials, keys, or proprietary code.

## Layout

```
src/<lang>/        normal source + test files per language
gen/               generated file (DO NOT EDIT header)
testfiles/         minified, binary, escaping symlink
vendor/, node_modules/   vendored trees
manifests/<eco>/   build-system manifests with framework deps
docker/Dockerfile  container signal
expected.json      ground-truth oracle
```

## Updating

Use codeprint's `/add-fixture` skill. When you change fixtures, update `expected.json` in the same commit, then bump the pinned submodule SHA in codeprint.

Oversized-file behavior (`skipped_large`) is tested in codeprint with a low `WithMaxFileSize`, not with a committed >10MB file.
