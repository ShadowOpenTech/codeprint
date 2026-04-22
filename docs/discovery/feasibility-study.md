# codeprint — Feasibility Study

**Purpose:** assess v1 feature viability against real Go libraries and APIs.
**Phase:** 1.3 Feasibility
**Status:** draft
**Updated:** 2026-04-22

## Summary

| # | Feature | Verdict | Note |
|---|---|---|---|
| 1 | Language detection | 🟢 Green | `go-enry/v2` — active, 2× linguist, library-first |
| 2 | LOC counting (code/comment/blank) | 🟢 Green | `gocloc` — library-embeddable with callbacks |
| 3 | File classification (source/test/gen/vendored/binary/minified) | 🟢 Green | enry handles vendored+binary; rest is heuristics |
| 4 | Build system detection (10 ecosystems) | 🟢 Green | Marker-file presence-check |
| 5 | Framework hints from manifests | 🟡 Yellow | Per-ecosystem parsing; potential shortcut via `git-pkgs/manifests` (24+ managers, unified output) — needs vetting |
| 6 | JSON + pretty + summary output | 🟢 Green | stdlib + small custom printer |
| 7 | Library API + thin CLI | 🟢 Green | Standard Go `pkg/` + `cmd/` layout |
| 8 | Published JSON Schema with semver | 🟢 Green | `invopop/jsonschema` generates from Go types |
| 9 | Produce-only, offline, single static binary | 🟢 Green | Native Go default; **no CGo** (see below) |
| 10 | Performance at 18k-repo scale | 🟢 Green (contingent) | Depends on per-run budget — see Open Questions |

No red-flagged features. One yellow (framework detection depth) and one contingent green (performance).

---

## Feature-by-feature assessment

### 1. Language detection — 🟢 Green

- **Library:** [`github.com/go-enry/go-enry/v2`](https://github.com/go-enry/go-enry)
- **Status:** actively maintained (latest April 2026). Go port of github-linguist.
- **Performance:** ~2× Ruby linguist. Optional CGo + Oniguruma for another 1.5–2× — **we decline CGo** to preserve the single-static-binary invariant.
- **Coverage:** 500+ languages, inherits linguist's vendored-file and generated-file heuristics for free.
- **API fit:** `enry.GetLanguage(filename, content)` returns language ID. Works on file content without a git repo.
- **Risk:** dependence on upstream for new-language support. Mitigation: detection layer pluggable behind an interface so a custom ruleset can be added later.

### 2. LOC counting — 🟢 Green

- **Library:** [`github.com/hhatto/gocloc`](https://github.com/hhatto/gocloc)
- **Status:** mature, library-friendly design.
- **API fit:** offers `ClocOptions` with `OnCode`, `OnBlank`, `OnComment` callbacks — we can stream counts without running its own file walker. Returns `ClocLanguage{Name, FilesCount, Code, Comments, Blanks}`.
- **Integration note:** gocloc uses its own language table separate from enry's. Resolution: enry identifies the language; we map that to gocloc's language name for comment-syntax counting. The two libraries' language universes overlap heavily but not 100% — unmapped languages fall back to "count non-blank lines as code" without comment awareness. Acceptable for v1; document the gap.
- **Alternative considered:** forking `scc` as a library. Rejected — `scc` is CLI-first, not designed for embedding.

### 3. File classification — 🟢 Green

- **source / test / generated / vendored / binary / minified**
- `enry.IsVendor(path)`, `enry.IsBinary(content)`, `enry.IsGenerated(path, content)` — handled by linguist rules.
- **Test detection:** path heuristics (`_test.go`, `test/`, `tests/`, `*.spec.ts`, `*Test.java`, etc.). Per-ecosystem table.
- **Minified detection:** trivial heuristic — extension `.min.js`/`.min.css` OR `(avg_line_length > 500 AND newline_ratio < 0.01)`.
- **Risk:** low. All heuristics are conservative — can be refined post-v1.

### 4. Build system detection — 🟢 Green

- Presence check only. `package.json` → JS/TS; `go.mod` → Go; `pom.xml` / `build.gradle(.kts)` → Java/Kotlin; etc.
- Zero parsing needed for detection. Parsing happens in feature 5.
- **Risk:** negligible.

### 5. Framework hints — 🟡 Yellow

- **Challenge:** each ecosystem has its own manifest format (JSON, TOML, XML, Groovy DSL for older Gradle, Kotlin DSL for newer Gradle, etc.).
- **Option A — Write per-ecosystem parsers** using stdlib (`encoding/json`, `encoding/xml`) + `BurntSushi/toml` for TOML. Framework detection is a lookup table (`react` in deps → hint `react`). Maintainable; 10 ecosystems = ~10 small modules.
- **Option B — Use [`git-pkgs/manifests`](https://awesome-go.com/)** (Andrew Nesbitt, 2026). 24+ package managers parsed into a unified normalized dependency graph with PURLs. Big shortcut if it fits.
- **Recommendation:** start with Option A for the known 10 ecosystems (predictable, no external maintenance dependency). Evaluate git-pkgs in architecture phase — if it's solid and the license fits, switch and get broader coverage for free.
- **Hard rule:** framework hints always emitted as a **list** with optional confidence scores — never assert a single "framework" per repo.
- **Risk:** Gradle's Groovy DSL is notoriously hard to parse without actually running Gradle. Mitigation: for `build.gradle` (Groovy), use regex pattern-matching for known framework imports; accept that coverage will be partial and document it. Kotlin DSL (`build.gradle.kts`) is easier (parseable-ish).

### 6. Output modes — 🟢 Green

- **JSON:** `encoding/json` — trivial.
- **Pretty-print:** ~100 lines of code; align columns, color optional via `isatty` check.
- **Summary line:** one `fmt.Printf` format call.
- **Risk:** none.

### 7. Library API + CLI — 🟢 Green

- Standard Go layout: `pkg/codeprint` (library), `cmd/codeprint` (CLI).
- CLI framework: `cobra` or `urfave/cli` — small preference for `cobra` for subcommand ergonomics (useful if v2 adds `codeprint diff`).
- **Risk:** none.

### 8. JSON Schema generation — 🟢 Green

- [`github.com/invopop/jsonschema`](https://github.com/invopop/jsonschema) generates JSON Schema 2020-12 from Go struct tags via reflection. Actively maintained.
- Alternative: Google's new JSON Schema package ([opensource.googleblog.com Jan 2026](https://opensource.googleblog.com/2026/01/a-json-schema-package-for-go.html)) — worth vetting in architecture phase.
- **Workflow:** Go structs in the library are the source of truth; CI step generates `codeprint.schema.json` from them and asserts no drift. Prevents schema/code divergence.
- **Risk:** low.

### 9. Produce-only, offline, single static binary — 🟢 Green

- Pure Go + zero CGo = static `CGO_ENABLED=0 go build`.
- No network calls, no config files required, no runtime deps.
- Binary size estimate: 15–25 MB stripped (reasonable for CI distribution).
- **Risk:** none.

### 10. Performance at scale — 🟢 Green (contingent)

- **Approach:** single-pass filesystem walk + bounded worker pool (`runtime.NumCPU()`) + per-file pipeline (enry detect → gocloc-style count → manifest parse if marker file).
- **Reference performance:** `scc` does 1M LOC in ~100–300 ms on modern hardware. Our overhead on top of a cloc-style counter: language detection via enry (~10–30% overhead), optional manifest parse (cheap, only on marker-file matches).
- **Rough budget estimate:** a 1M-LOC repo should complete in **under 5 seconds** on CI runner hardware. Smaller repos (~100k LOC) under 1s.
- **Contingent on:** agreed performance budget — see Open Questions.
- **Scale math:** 18k repos × seconds per run = aggregate CI cost. At 3s/run average, that's ~15 CPU-hours per full fleet refresh. Acceptable for nightly or per-commit runs.

---

## Decisions made

1. **Performance budget — Moderate.** Target ≥300–500k LOC/sec throughput on a typical CI runner.
   - 50k LOC → 100–170ms
   - 500k LOC → 1–1.7s
   - 5M LOC → 10–17s
   - Rationale: Tight (≥1M LOC/sec) matches pure-LOC tools like `scc`; codeprint's enry + manifest overhead makes Tight costly without mmap/alloc-tuning/CGo. Moderate is single-digit seconds for ~99% of repos with clean implementation.

2. **CGo policy — No.** Single static binary is mandatory. Forgoes 1.5–2× language-detection speedup from Oniguruma.

## Open questions (resolve in architecture phase)

- **Dependency license constraints.** Any restrictions on GPL/LGPL/AGPL in the org? enry is Apache 2.0, gocloc MIT, invopop/jsonschema MIT — all safe. `git-pkgs/manifests` license TBD.
- **Schema ownership / publication location.** Alongside code, public registry (Schema Store), or internal-only? Decide in architecture phase.

---

## Risks (carried forward to architecture & planning)

| Risk | Area | Action |
|---|---|---|
| Gradle Groovy DSL parsing is hard | Feature 5 | Regex fallback; document partial coverage |
| enry ↔ gocloc language-table mismatch | Features 1+2 | Map table; fallback to non-comment-aware count |
| git-pkgs may not fit (license/stability) | Feature 5 | Default to Option A (per-ecosystem parsers) |
| Binary size creep as ecosystem parsers pile up | Feature 9 | Benchmark & monitor at each minor release |

## Change log

- 2026-04-22: initial draft
