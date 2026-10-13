# ADR 0007 — Go stack: Cobra + Charm (Bubble Tea, Lip Gloss, Bubbles)

**Status:** Final
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Date:** 2026-10-13
**Deciders:** Project owner (accepted option A in decision interview, 2026-10-13)
**Depends on:** [ADR 0001](0001-implementation-language-go.md) (Final), [ADR 0003](0003-command-grammar-noun-first.md) (Final), [ADR 0006](0006-tui-scope-full-in-mvp.md) (Final)

## Context

ADR 0001 settled Go but deferred the library choices. ADR 0006 then made
the TUI a full item-management surface in the MVP, which makes the CLI/TUI
toolkit a critical-path decision: every screen, keystroke, and styled
output will be built with it. The owner's top priorities are a beautiful
CLI/TUI experience and speed — including the speed of iterating on visual
polish.

## Decision

Build on:

- **Cobra** — command tree for the noun-first grammar (ADR 0003), flags,
  help generation, and shell completion (PRD §7).
- **Bubble Tea** — the TUI application model (Elm-style update/view) for
  `boardwalk board`.
- **Lip Gloss** — all TUI/CLI styling: layout, borders, colour, and
  adaptive (light/dark, colour-profile-aware) styles.
- **Bubbles** — reusable components (list, text input, textarea,
  spinner, paginator, viewport, help/key bindings) wherever they fit;
  Boardwalk-specific kanban components are built on Bubble Tea/Lip Gloss
  rather than forking Bubbles.

Constraints:

- One shared core: Cobra commands and the Bubble Tea app call the same
  internal client/service layer (ADR 0002, ADR 0006); neither contains
  GitHub logic of its own.
- Human CLI output is rendered through Lip Gloss only when writing to a
  terminal; piped output and `--json` carry no styling or decoration
  (PRD §9).
- Status is never conveyed by colour alone (PRD §10): styles pair colour
  with text/symbols, and components must remain usable in monochrome and
  colour-blind-safe palettes. Accessibility is a component-level
  requirement, not a theme afterthought.

## Rationale

- **Most design control for a bespoke kanban:** Bubble Tea's model plus
  Lip Gloss styling is the stack behind the polished Go TUIs cited in
  ADR 0001 (lazygit-style apps, gh-dash, Glow). It optimises exactly the
  tweak-and-look iteration loop that visual quality comes from.
- **Ecosystem coherence:** Cobra's help/completion model matches the
  noun-first grammar and scriptability requirements; Bubbles covers the
  commodity widgets so effort concentrates on the board itself.
- **Speed:** All rendering is local and incremental; nothing in this
  stack adds latency to the network-bound flows (ADR 0004 fixtures,
  including the 100+-item board, are the performance gate).

## Alternatives considered

- **Cobra + tview/tcell (option B):** Ready-made tables/forms/modals
  assemble conventional screens quickly. Rejected because the look trends
  generic and fine-grained kanban styling and keyboard choreography —
  the centrepiece under ADR 0006 — are harder to make feel crafted.
- **Minimal dependencies, hand-rolled terminal layer (option C):**
  Rejected because it re-implements rendering, input, and layout that
  this stack already provides; the slowest route to beautiful.

## Consequences

- The project takes maintained dependencies on Cobra, Bubble Tea,
  Lip Gloss, and Bubbles (plus `go-gh` per ADR 0002) and must track their
  releases; major-version Charm changes are the expected upgrade cost.
- Kanban components (columns, cards, drag/move interaction, detail
  panel) are Boardwalk code and Boardwalk's testing responsibility,
  exercised keyboard-only against the ADR 0004 fixtures.
- If a needed interaction proves impossible or unacceptably awkward in
  Bubble Tea, the fallback is a Boardwalk-built component on the same
  model — not a second TUI toolkit alongside it.

## Unresolved

- TUI design — **settled** in the
  [TUI specification](../specs/tui.md) (Final), all sections.
- Config format — **resolved:** TOML, see
  [ADR 0008](0008-config-format-toml.md) (Final).
- JSON/error contract — **resolved:** bare JSON plus documented exit
  codes, see [ADR 0009](0009-json-error-contract-bare.md) (Final).
- Auth scopes/remediation — **resolved:** see
  [ADR 0010](0010-auth-scopes-upfront.md) (Final).
