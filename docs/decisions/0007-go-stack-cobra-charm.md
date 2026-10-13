# ADR 0007 — Go stack: Cobra + Charm (Bubble Tea, Lip Gloss, Bubbles)

**Status:** Final — amended by A1 (2026-10-13)
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Amended:** 2026-10-13 — A1, owner approval of the
[beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md)
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
- **Bubble Tea v2** *(A1)* — the TUI application model (Elm-style update/view)
  for `boardwalk board`, using the cell-diffing renderer, frame-rate controls,
  and explicit background querying (`tea.BackgroundColorMsg`).
- **Lip Gloss v2** *(A1)* — all TUI/CLI styling: layout, borders, colour,
  canvas/layer compositing, and adaptive styles.
- **Bubbles + Huh** *(A1)* — reusable components (list, text input, textarea,
  spinner, viewport, key bindings) plus **Huh** for shared interactive
  prompts in the CLI (cli.md §5).
- **Glamour** *(A1)* — styled Markdown rendering for issue/PR bodies,
  rendered off the main UI loop and cached per item/theme (tui.md §3).
- Boardwalk-specific kanban components are built on Bubble Tea/Lip Gloss
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
- Kanban components (columns, cards, keyboard move interaction
  *(A1: mouse drag-and-drop is out of MVP, tui.md §2)*, detail panel)
  are Boardwalk code and Boardwalk's testing responsibility,
  exercised keyboard-only against the ADR 0004 fixtures.
- If a needed interaction proves impossible or unacceptably awkward in
  Bubble Tea, the fallback is a Boardwalk-built component on the same
  model — not a second TUI toolkit alongside it.

## Amendment log

- **A1 (2026-10-13)** — owner approval of the
  [beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md):
  Adopted Bubble Tea v2 and Lip Gloss v2 from the start to take advantage of
  cell-diffing rendering, explicit background query, and layer compositing;
  added Huh for CLI interactive prompts and Glamour for off-loop Markdown
  rendering; clarified that move interaction is keyboard-only (no mouse drag-and-drop).
