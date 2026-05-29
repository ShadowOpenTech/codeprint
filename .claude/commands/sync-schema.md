---
description: Regenerate the JSON schema from Go structs, diff against the committed schema, and classify the change
---

Keep `schema/codeprint-v1.schema.json` in sync with the `pkg/codeprint` structs (the schema-first contract — see `docs/design/product-spec.md` F-7).

1. Run `make schema` to regenerate from struct tags (`invopop/jsonschema`, JSON Schema 2020-12).

2. Diff the regenerated schema against the committed one. If identical, report "no drift" and stop.

3. If changed, **classify the change**:
   - **Additive** (new optional field, new enum value) → safe; minor/patch in v0.x.
   - **Breaking** (removed/renamed field, type change, narrowed constraint, newly-required field) → flag loudly. In v0.x: minor bump + changelog warning. Post-1.0: **major bump**.

4. Summarize exactly what changed (field-by-field) and the recommended version impact per `docs/launch/deploy-plan.md`.

5. Remind: any exported-surface or schema change needs reviewer sign-off (`docs/planning/code-quality.md`). Commit the regenerated schema alongside the struct change so CI's drift gate passes.
