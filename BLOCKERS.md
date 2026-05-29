# Blockers & decisions deferred to the user

Logged during the autonomous M0→Release-0 run. None of these stop progress; each has a working interim approach. Resolve when convenient.

## 1. Test-corpus hosting & submodule conversion — OPEN

**Context:** the plan (test-plan.md) called for the correctness corpus to be a SHA-pinned **git submodule**. Building it was approved; **publishing it was not**, and the auto-mode classifier (correctly) blocked me from creating a *public* org repo since visibility was my choice and is irreversible.

**Interim approach (in effect):** the corpus is vendored as a working copy at `testdata/corpus/` (self-contained; CI works without a remote). A standalone git repo also exists locally at `~/projects/codeprint-testcorpus` (commit `e362238`), ready to become the submodule source.

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

## 2. Release publish (JFrog + real v0.1.0 tag) — DEFERRED BY DESIGN

Per your auto-mode decision: build **release-ready**, stop before the real tag/publish. JFrog credentials / GitHub-Actions secrets are not available to me, and a public tag is irreversible. The exact tag + publish steps will be in the final M5 report / deploy checklist.
