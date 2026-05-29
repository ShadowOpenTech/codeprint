# codeprint — Data Model

**Purpose:** define the `Fingerprint` entity set — the Go structs that ARE the published JSON schema.
**Phase:** 4.2 Data Model
**Status:** confirmed
**Updated:** 2026-05-30

## Principle

The library structs in `pkg/codeprint/types.go` are the single source of truth. `invopop/jsonschema` generates `schema/codeprint-v1.schema.json` from them (4.1). There is no separate "schema" artifact maintained by hand — change a struct, regenerate, CI diffs.

Determinism (NFR-2) applies to the `fingerprint` payload only. The `_meta` envelope is the sole home for volatile data and is excluded from diff comparisons.

## Output envelope

```jsonc
{
  "$schema": "https://.../codeprint-v1.schema.json",  // immutable URL at GA; floating/omitted pre-1.0 (F-7)
  "_meta": {
    "schema_version": "1.0",
    "producer": { "name": "codeprint", "version": "0.1.0" },
    "generated_at": "2026-05-30T10:00:00Z"             // RFC3339 — the ONLY timestamp, _meta-only
  },
  "fingerprint": { /* deterministic payload, see below */ }
}
```

`generated_at` lives in `_meta` so `fingerprint` stays byte-identical across runs. Consumers are told to ignore `_meta` when diffing (the NFR-2 CI test diffs `fingerprint` only).

## Fingerprint payload

```jsonc
"fingerprint": {
  "totals":          { "files": 412, "code": 38120, "comment": 6402, "blank": 3690, "bytes": 1840221 },
  "languages":       [ /* LanguageRollup, sorted by language ID */ ],
  "files":           [ /* FileRecord, sorted by path — included by default, see Decision 1 */ ],
  "build_systems":   [ /* BuildSystem, sorted by path */ ],
  "framework_hints": [ /* FrameworkHint, sorted by (ecosystem, name) */ ],
  "container":       { "dockerfile_present": true },
  "errors":          [ /* ScanError, sorted by path — non-fatal (NFR-5) */ ],
  "workspaces":      null   // reserved for v2; ALWAYS null in v1 (field present so consumers code against it now)
}
```

## Entities

### LanguageRollup
| Field | Type | Notes |
|---|---|---|
| `language` | string | enry language ID (authoritative — matches go-enry/linguist) |
| `files` | int | count of files of this language |
| `code` | int | total code LOC |
| `comment` | int | total comment LOC |
| `blank` | int | total blank LOC |
| `percent` | float | **rounded to 2 decimals**, derived from `code` / total code LOC. Convenience only; counts are source of truth |
| `comment_aware` | bool | `false` when language has no gocloc comment mapping (counts fall back to non-blank=code) — see 4.1 enry↔gocloc seam |

### FileRecord
| Field | Type | Notes |
|---|---|---|
| `path` | string | repo-relative, never absolute (NFR-8) |
| `language` | string | enry ID; empty string if undetected |
| `kind` | enum | exactly one: `source` \| `test` \| `generated` \| `vendored` \| `binary` \| `minified` (F-2 priority order) |
| `flags` | []string | non-exclusive props, e.g. `["test"]` on a generated test file, `["skipped_large"]` (NFR-4) |
| `code` / `comment` / `blank` | int | `0` for binary/vendored/minified/oversized (excluded from LOC totals per F-1) |
| `bytes` | int | file size |

### BuildSystem
| Field | Type | Notes |
|---|---|---|
| `ecosystem` | string | e.g. `npm`, `go-modules`, `maven` (F-3 catalog of 10) |
| `path` | string | repo-relative path to the marker file that triggered detection (4.1 flat-list-with-paths) |

### FrameworkHint
| Field | Type | Notes |
|---|---|---|
| `name` | string | e.g. `django`, `react`, `spring-boot` (F-4 catalog) |
| `ecosystem` | string | owning ecosystem |
| `confidence` | enum | `high` \| `medium` \| `low` |
| `evidence` | object | `{ "path": "<manifest>", "keyword": "<trigger>" }` |

Always a **list**, never a singular `framework` (hard rule from ideation/feasibility).

### ScanError
| Field | Type | Notes |
|---|---|---|
| `path` | string | repo-relative path that failed |
| `reason` | string | human-readable cause |

Non-fatal per NFR-5: unreadable individual files land here, scan continues, exit stays 0.

### Workspace (reserved, v2)
Field `workspaces` is present and `null` in v1. The v2 shape (per-workspace sub-fingerprints for monorepos) is intentionally undefined here — locking only the field name now means v2 adds population without a schema-shape break.

## Decisions

1. **Per-file records — included by default, opt-out via `--no-files` / `WithoutFileRecords`.** F-2's user story is explicitly per-file ("scanner decides which files to process or skip"), so the data must be present by default. Size-sensitive consumers (huge monorepos at fleet scale) drop to rollups-only. Resolves the ideation open question "per-file records vs rollups only."

2. **`percent` is a float rounded to 2 decimals**, derived from sorted integer counts. Deterministic; human-friendly for pretty/summary. Counts remain the source of truth — consumers needing exactness use `code`/`totals.code`.

3. **`generated_at` is `_meta`-only**; `workspaces` is `null` not absent; all `fingerprint` lists sorted by a documented stable key (declared per entity above).

## Sort keys (determinism contract)

| List | Stable sort key |
|---|---|
| `languages` | language ID (asc) |
| `files` | path (asc) |
| `build_systems` | path (asc) |
| `framework_hints` | (ecosystem, name) (asc) |
| `errors` | path (asc) |
| `flags` within a record | lexical (asc) |

## Change log

- 2026-05-30: initial draft; confirmed. Per-file default ON with opt-out; percent as 2-decimal float; reserved `workspaces` field locked as null. Resolves ideation "per-file vs rollups" open question.
