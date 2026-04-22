# Documentation Standard — codeprint

All docs under this directory follow this standard so future sessions (and teammates) can navigate consistently.

## Structure

```
docs/
  discovery/     # Phase 1: what & why
  design/        # Phase 3: flows, spec, mocks
  architecture/  # Phase 4: stack, data, API, error, security
  planning/      # Phase 5: tests, quality, conventions, env, breakdown
  launch/        # Phase 6: deploy
  index.md       # master index (created at end of Phase 7)
```

## Per-document rules

- **Front matter**: every doc starts with a one-line purpose + the phase it belongs to.
- **Status tag**: `Status: draft | confirmed | superseded`. Default `draft` while a phase is in flight, bumped to `confirmed` at phase checkpoint.
- **Date**: ISO date of last meaningful update (`Updated: YYYY-MM-DD`).
- **Cross-refs**: link to prior docs using relative paths (e.g., `../discovery/ideation.md`). Never restate decisions — link to them.
- **Concrete over abstract**: where a rule or concept is non-obvious, show a before/after or shell example. Prose-only specs rot fast.
- **Lifecycle pairing**: additions (new features, new commands, new config) must be described alongside their retirement/deprecation story if one applies.

## Writing style

- Short paragraphs. Bulleted lists when order doesn't matter, numbered when it does.
- No filler ("In this section we will..."). State the thing.
- Code blocks tagged with language.
- Tables for comparisons (name vs. name, option vs. option), not prose.

## Change log

When a confirmed doc is materially changed, append a dated line at the bottom:

```
## Change log
- 2026-04-21: initial draft
- 2026-04-25: revised scope after feasibility
```
