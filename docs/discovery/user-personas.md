# codeprint — User Personas

**Purpose:** define target users for codeprint to ground design decisions.
**Phase:** 1.2 Persona
**Status:** draft
**Updated:** 2026-04-22

## Summary

Three personas — two primary, one secondary. Note: in the target org, one person often wears all three hats. The product must work cohesively across those workflows without forcing context-switching between "modes."

| # | Persona | Priority | Primary interaction |
|---|---|---|---|
| 1 | Platform / AppSec Engineer | Primary | Deploys codeprint in CI; integrates outputs with scanners |
| 2 | Scanner Tool Maintainer | Primary | Consumes codeprint JSON via library/file retrieval; depends on schema stability |
| 3 | DevOps / SRE Engineer | Secondary | Slots codeprint into existing CI pipelines; cares about packaging, perf, exit codes |

**Not a persona:** the solo developer curious about their repo. `scc` and `tokei` serve that better; codeprint is not optimized for it.

---

## Persona 1 — Platform / AppSec Engineer (primary)

**Role:** owns the security scanning platform across the org's ~18,000 repos.

**Goals:**
- Answer "what's in this app?" reliably, without human effort.
- Verify that scanners actually covered what they claimed to cover.
- Detect false deployment attestations ("config only" claims that actually ship code).
- Produce portfolio-level metrics on language distribution, framework usage, coverage gaps.

**Environment:**
- Linux, CLI-heavy, CI/CD-centric workflow.
- Integrates with JFrog, ELK, release-ticket system (all already in place).
- Comfortable with Go, JSON, shell. Doesn't touch Ruby.

**Today's workflow (the pain):**
- Nothing answers "what's in this app." Ad-hoc `cloc`/`find`/manual checks.
- Someone manually opens Bitbucket to verify "config only" deployment claims — doesn't scale past a few dozen a day, org has 18k repos.
- SonarQube LOC results are untrusted because app teams freely exclude directories; no independent denominator.

**What codeprint enables:**
- One JSON per repo per CI run → pushed by existing infra to JFrog/ELK.
- Scanners retrieve the JSON and compute their own coverage math against a trusted denominator.
- Release-ticket system gets an automated, non-human attestation of "what changed."

**Success signal:** stops writing glue scripts. Starts writing coverage checks *against* the stable schema instead of parsing scanner outputs.

---

## Persona 2 — Scanner Tool Maintainer (primary)

**Role:** builds or extends scanners/security tools that consume codeprint's JSON.

**Known consumers in the stack:**
| Scanner | Type | Primary need from codeprint |
|---|---|---|
| SonarQube | SAST | Language, LOC, file classification |
| cradar | SAST (internal) | Language, LOC, file classification, manifest presence |
| Nexus IQ | SCA | Manifest presence, ecosystem identifier |
| Twistlock (Prisma Cloud) | Container / cloud | Dockerfile detection, base-image hints |
| Contrast IAST | Runtime instrumentation | Framework detection (Spring, Express, Django, Flask…) |
| zScan | Mobile | Kotlin/Swift signals, Android/iOS project hints |
| Custom API security tool | API (in development, 1 month – 1 quarter) | Framework + manifest (FastAPI, Spring, Express, Flask, Django) |
| Venari | DAST | Framework hints for crawl strategy; manifest presence for endpoint discovery context |

**Goals:**
- Integrate codeprint **once** and not re-integrate on every codeprint release.
- Treat the JSON schema as a contract: types, field semantics, absence semantics.
- Ship scanner updates independently of codeprint updates.

**Environment:**
- Retrieves codeprint output **from JFrog/ELK**, not by invoking codeprint directly.
- May also consume the Go library in-process (for tools also written in Go, e.g., cradar).

**What codeprint must provide:**
- Published `codeprint.schema.json` with semver discipline.
- Field-level documentation — what each field means, when it's null/absent, what "confidence" means.
- Stable identifier keys across versions (e.g., language IDs match go-enry, not reinvented).
- Output is safe to retrieve months or years after production (no implicit time dependencies).

**Success signal:** adds a new scanner (e.g., the custom API security tool) in days, not weeks, by consuming the same schema already used by other scanners.

---

## Persona 3 — DevOps / SRE Engineer (secondary)

**Role:** owns CI/CD pipelines, runners, build tooling.

**Goals:**
- Add codeprint to existing pipelines with minimal config.
- Keep build times predictable across 18k pipelines.
- Minimal runtime dependencies, minimal surface area for breakage.

**Environment:**
- Bitbucket + CI runners (locked down; often no network egress).
- Existing push/retrieval plumbing for JFrog/ELK.
- Already slotted tools: linters, SAST scanners, SBOM generators.

**What codeprint must provide:**
- Single static Go binary, no runtime deps.
- Zero network calls by default.
- Deterministic exit codes (0 success, non-zero with documented meaning).
- Predictable performance on large repos.
- Output to stdout OR file; never both unless asked.
- No mandatory config file — flags-and-defaults only.

**Success signal:** added codeprint once to a pipeline template; it runs forever without tickets.

---

## Cross-persona design implications

| Design decision | Driven by |
|---|---|
| Produce-only, no network | P1 (scale), P3 (locked CI runners) |
| Single static Go binary | P3 (packaging) |
| Stable versioned JSON schema | P2 (consumers depend on it for years) |
| Library API (Go package) | P2 (cradar and in-process consumers) |
| Framework hints as a list (not singular) | P2 (different scanners need different confidence thresholds) |
| Deterministic exit codes + stdout/file separation | P3 (CI hygiene) |
| Diff mode (v2) | P1 (deployment change-type attestation) |
| 10 ecosystems in v1 | P2 (covers all named consumers on day 1) |

## Open questions

- **Custom API security tool delivery window is 1 month – 1 quarter.** codeprint v1 should land inside that window so the API tool ships consuming codeprint from day 1 — sets the v1 target timeline.
- DAST (Venari) consumption is shallower than SAST consumption — codeprint's framework-hint design should stay useful to both without special-casing either.

## Change log

- 2026-04-22: initial draft
