# Blockers & decisions deferred to the user

Logged during the autonomous M0→Release-0 run. None of these stop progress; each has a working interim approach. Resolve when convenient.

## 1. Test-corpus hosting & submodule conversion — RESOLVED (2026-05-30)

Published `ShadowOpenTech/codeprint-testcorpus` (public) and converted `testdata/corpus/` into a SHA-pinned submodule (pinned to `9e3f476`). CI checks out submodules recursively. Original context retained below.

**Context:** the plan (test-plan.md) called for the correctness corpus to be a SHA-pinned **git submodule**. Building it was approved; **publishing it was not**, and the auto-mode classifier (correctly) blocked me from creating a *public* org repo since visibility was my choice and is irreversible.

**Decision needed from you:**
- Push `~/projects/codeprint-testcorpus` to `ShadowOpenTech` — **public or private?**
  - Public → simplest: codeprint's public-repo CI can fetch it as a normal submodule.
  - Private → CI needs a token/secret to fetch the submodule (extra setup).
- Once decided, convert `testdata/corpus/` into a pinned submodule (or keep it vendored — also valid; just resync manually).

**Command if you choose public:**
```bash
cd ~/projects/codeprint-testcorpus
gh repo create ShadowOpenTech/codeprint-testcorpus --public --source=. --push
# then in codeprint:
git rm -r testdata/corpus
git submodule add https://github.com/ShadowOpenTech/codeprint-testcorpus testdata/corpus
```

## 2. JFrog / release publish — RESOLVED (2026-05-30)

codeprint is open source: distribution is **GitHub Releases + Go module proxy** (free/public). JFrog dropped (was for internal air-gapped runners). The real v0.1.0 tag is still the user's to push — see RELEASE.md.

_Original note:_

Per your auto-mode decision: build **release-ready**, stop before the real tag/publish. JFrog credentials / GitHub-Actions secrets are not available to me, and a public tag is irreversible. The exact tag + publish steps will be in the final M5 report / deploy checklist.

## 3. Known issue (tech debt, non-blocking) — in-tree symlink double-counts

An in-tree symlink to a regular file is followed and counted, so its target's
LOC is counted twice (once via the real path, once via the link). Deferred:
dedup by resolved path/inode in a later milestone. Escaping symlinks are
correctly skipped (NFR-8). Symlinked directories are not descended.

Now auditable: the fingerprint's `symlinks` report exposes `followed_file`
(where this double-count lives) and `skipped[]` (dirs/escaping not walked), so
consumers can see the traversal gaps. The resolved-path dedup that would fix the
double-count is the same mechanism that would enable safe symlinked-dir following.

Tracked as GitHub issue #14 (label: future-improvement).

## 4. Minified classification priority — RESOLVED (2026-05-30): kept; spec F-2 amended

Spec F-2 ordered file-kind priority as binary > vendored > generated > minified.
In practice enry/linguist folds minified files into "vendored" (.min.* extension)
and "generated" (very long lines), which made the dedicated `minified` kind
unreachable. Since `minified` is the more specific, actionable signal for
scanners (skip built assets), I refined the priority to:

    binary > minified > vendored > generated > test > source

`minified` = explicit `.min.js`/`.min.css` extension OR the long-line heuristic.
Decision: **kept**. product-spec.md F-2 amended to document this order so doc and code agree.

## 5. LICENSE — RESOLVED (2026-05-30): Apache-2.0 committed

codeprint has no LICENSE file yet. A public OSS release needs one. Dependencies
are Apache-2.0 (go-enry, cobra) and MIT (gocloc, invopop, denormal, toml,
fatih/color) — all permissive, compatible with either choice.

Recommendation: **Apache-2.0** (matches go-enry; patent grant suits a tool with
external consumers) or **MIT** (simpler). Your call — add a `LICENSE` file and
commit. goreleaser archives already include `LICENSE*`. Until then, M5 is
"release-ready" but the tag is held (see RELEASE.md).
