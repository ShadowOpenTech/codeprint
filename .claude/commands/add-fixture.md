---
description: Add a corpus fixture with ground-truth labels and wire it into the golden TP/FP/FN tests
---

Add a test fixture to the correctness corpus (CipherRadarTestProj submodule) with known ground truth, so detection/classification accuracy stays measurable (`docs/planning/test-plan.md`).

Ask what the fixture should exercise if not given (a language, a `kind`, an ecosystem, an edge case like minified/generated/oversized/symlink), then:

1. **Add the file(s)** to the appropriate corpus directory. For edge cases, make the trigger explicit:
   - `generated` → include a `// Code generated ... DO NOT EDIT.` header
   - `minified` → `.min.js` or a single long line (>500 avg, low newline ratio)
   - `vendored` → under a vendored path enry recognizes
   - `binary` / oversized / symlink (in-tree vs escaping) as needed

2. **Update ground-truth labels** (`expected.json` / per-dir labels): declare the expected `language`, `kind`, `flags`, and any `build_systems`/`framework_hints` this fixture should produce.

3. **Wire into golden tests**: ensure the fixture is covered by the determinism + accuracy suite; regenerate golden output via the `-update` flag, then review the diff to confirm it matches the intended ground truth (don't blindly accept).

4. Run the TP/FP/FN comparison and `make test`. A drop in precision/recall is a regression, not a new baseline.

Note: corpus changes are submodule changes — commit in the submodule and bump the pinned SHA in codeprint.
