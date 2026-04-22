# codeprint — Brand Identity

**Purpose:** define name, logo, colors, and typography for codeprint.
**Phase:** 2 Brand
**Status:** confirmed
**Updated:** 2026-04-22

## Name

**codeprint** — all lowercase. Chosen over `stacksnap`, `codecensus`, `srcsight`, `repoprint`, `polyglot`.

**Rationale:** the "fingerprint" metaphor captures the product's essence — produce a compact, stable identity of a codebase. Short, ownable, unambiguous in the CLI/CI context. No collision with well-known tools.

## Logo

**Concept:** two curly braces `{` and `}` with a horizontal scan line across the middle.

- Braces signal "code."
- The scan line (accent color) signals the tool's action — inventory, produce, scan.
- Minimal, favicon-friendly, scales to 16×16.

**Canonical file:** `assets/logo.svg` — 64×64 viewBox, two-color, stroke-based.

**Image-generation prompt** (for Midjourney/DALL-E-style variants if needed):

> Minimalist logo: two rounded curly braces facing each other, deep teal color, with a short horizontal amber accent line passing through the center between them. Flat 2D vector, clean strokes, generous padding, no shadows, no gradients, white background. Suitable for a developer CLI tool called "codeprint." Square composition, 1:1 aspect ratio.

## Color palette

| Role | Hex | Usage |
|---|---|---|
| Primary | `#0D5E6B` | Braces, headings, brand marks, CLI accents |
| Accent | `#F59E0B` | Scan line, highlights in pretty-print output, emphasis only |
| Neutral dark | `#2D3748` | Body text, secondary UI |
| Neutral light | `#F7FAFC` | Backgrounds (web) |
| Text muted | `#718096` | Captions, helper text |
| Success | `#10B981` | Green-light output (ok, found) |
| Warning | `#F59E0B` | Same as accent |
| Error | `#EF4444` | Error output, non-zero exit indicators |

**Rules:**

- The **amber accent is rare** — use on one element at a time for emphasis. Never across multiple elements in the same view. Overuse flattens its meaning.
- Primary teal is the dominant color in all brand touchpoints.
- Avoid heavy gradients; codeprint's identity is clean and flat.

## Typography

- **Brand mark / logo:** no text in the mark itself.
- **Wordmark (when "codeprint" is typeset):** monospace — `ui-monospace`, `SFMono-Regular`, `Menlo`, `Monaco`, `Consolas`, fallback `monospace`. All lowercase.
- **CLI pretty-print:** inherits the user's terminal font. No custom fonts shipped.
- **Docs / README:** default GitHub-flavored markdown fonts. No custom webfont.

## Out of scope for v1

- Brand guidelines PDF
- Animation / motion
- Icon family (alternate sizes / variants) — single 64×64 SVG is sufficient
- Social / marketing assets

## Change log

- 2026-04-22: initial draft; logo-3 (braces + scan line) chosen from 5 candidates.
