---
description: Scaffold a new build-system ecosystem — marker detection, framework parser, hint catalog, fixture, and ground-truth label
---

Add full support for a new ecosystem to codeprint. Ask for the ecosystem name (e.g. `gradle`, `swift-pm`) and its marker file(s) if not given, then:

1. **Build-system detection** (`internal/buildsys`): register the marker file(s) so the ecosystem appears in `build_systems[]` with its repo-relative path (flat full-tree scan — see `docs/architecture/system-architecture.md`). Detection is presence-only, no parsing.

2. **Framework parser** (`internal/framework`): add a per-ecosystem manifest parser (stdlib `encoding/json`/`encoding/xml` or `BurntSushi/toml`; regex fallback for Gradle Groovy DSL — document partial coverage). Emit `framework_hints[]` entries as a **list** with `name`, `ecosystem`, `confidence`, and `evidence` {path, keyword}. Never assert a singular framework.

3. **Hint catalog**: add the known framework keywords for this ecosystem (see the F-4 catalog in `docs/design/product-spec.md`).

4. **Fixture + label**: add a representative manifest to the correctness corpus and update its ground-truth labels (use `/add-fixture`).

5. **Tests**: table-driven unit tests for the parser; assert the fixture yields the expected hints with correct confidence. Verify TP/FP/FN against labels.

Follow `docs/planning/conventions.md`. Run `/sync-schema` if any struct changed, then `make verify`.
