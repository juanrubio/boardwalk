# Quit confirmation and visual continuity

The owner requested Trailmark-style improvements to Boardwalk's quit flow,
status bar, detail pane, projects picker, and draft form. Requirements are
recorded in [PRD.md](../../PRD.md) and [TUI amendment A12](../specs/tui.md).

## Implemented

- Board-level `q` opens an accent-bordered confirmation with Cancel selected.
  Explicit confirmation waits for pending moves; a failed move keeps the app
  open. Nested back navigation, text entry, dirty-form protection, and the
  existing Ctrl-C escape behavior remain covered by regression tests.
- Footer keys use bold accents, labels use muted text, and messages carry
  action-defined information, success, pending, warning, or error styling.
  Truncated hints end with an ellipsis and remain within one row.
- Detail references and GitHub single-select values retain semantic colours;
  Markdown follows the active light, dark, or monochrome profile.
- Projects have accent references and borders, a full-row selected fill,
  selection/current markers, and an ellipsis for clipped titles.
- Draft/edit forms have focused labels and cursors, coloured option values,
  warning markers with a changed legend, and clear validation messages.
  Raised modal surfaces preserve their fill across nested ANSI resets and
  stop at the component boundary.

## Validation

`make check build` passed on 2026-10-16: Go vet, the pinned golangci-lint
(zero issues), the complete test suite, and the local `boardwalk` build.
The suite includes frame-performance budgets, keyboard workflows, contrast
checks, monochrome behaviour, and terminal geometry checks.

Updated the existing ANSI golden frames and added quit-confirmation frames
at 80×24 dark, 120×40 light, and 60×15 monochrome. Tests exercise cancellation,
explicit confirmation, repeated `q`, pending completion before confirmation,
multiple pending writes, write failure, nested inputs, dirty forms, and
undersized-terminal confirmation.

Rendered synthetic offline fixtures for board, detail, projects, draft, and
quit states in light and dark themes at 80×24 and 120×40. Inspected previews
for focus contrast, filled surfaces, border containment, and visible controls.
Fixture previews are temporary review artifacts rather than product assets.

No live GitHub writes were performed. Rendered previews and automated tests
do not establish colour fidelity or redraw behaviour in the owner's physical
terminal; that remains a manual acceptance step.
