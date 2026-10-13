# Boardwalk TUI specification

**Status:** Final
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Date:** 2026-10-13
**Scope:** `boardwalk board` — full item management in the MVP
([ADR 0006](../decisions/0006-tui-scope-full-in-mvp.md), Final) on the
Cobra + Charm stack ([ADR 0007](../decisions/0007-go-stack-cobra-charm.md),
Final). Decisions are recorded here as they settle in the decision
interview; unresolved sections stay visibly marked.

## 1. Kanban grouping field (settled 2026-10-13)

The board groups items by a **selected single-select field**
(PRD §6.2). The selection rule is:

1. If the user has previously chosen a grouping field for this project,
   use that remembered choice.
2. Otherwise, if the project has a single-select field named `Status`,
   use it.
3. Otherwise, use the project's first single-select field.
4. If the project has **no** single-select field, open in list view and
   state plainly that kanban grouping is unavailable for this project
   and why — never an error exit.

A grouping picker inside the TUI lists the project's single-select
fields and lets the user switch at any time. Switching updates the board
immediately and is **remembered per project** in user config
(`config.toml`, ADR 0008) under a UI-preferences section keyed by the
project reference, for example:

```toml
[ui.projects."juanrubio/1"]
group_by = "Status"
```

Rules:

- The remembered value is a field *name* as last seen. If that field has
  been renamed or deleted on GitHub, fall back to rules 2–4 and update
  the remembered value on the next explicit picker choice.
- Field names, options, and their order always come from the live
  project schema (PRD §9) — including option order as columns, left to
  right. Nothing about `Status` labels is hard-coded; the two fixture
  boards (ADR 0004) use different Status vocabularies
  (Backlog/Ready/In progress/In review/Done vs. Todo/In Progress/Done)
  and both must render correctly.
- Items with no value for the grouping field appear in an explicit
  "No <field>" column rather than being hidden.
- The same grouping choice applies to the CLI list's optional grouping
  display, if any; the TUI picker is the only place it is changed in
  the MVP.

Rejected for this section: grouping by the literal name `Status` only
(fails renamed fields, PRD §9) and asking on every launch (taxes the
most common action in the tool).

## 2. Navigation and move interaction (settled 2026-10-13)

Keyboard-first, modal-lite. Every TUI action — including moving a
card — must be completable keyboard-only (PRD §9); mouse support is an
addition, never a requirement.

### Navigation

| Keys | Action |
|---|---|
| `↑`/`k`, `↓`/`j` | Previous / next card in the focused column |
| `←`/`h`, `→`/`l` | Previous / next column (focus follows to nearest card) |
| `Enter` | Open the focused card's detail panel |
| `Esc` / `q` in a panel | Close the panel and return focus to the board |
| `q` on the board | Quit |
| `?` | Toggle the help overlay listing all keys |

Arrow keys and `hjkl` are always both active; there is no mode where
one set stops working. A visible footer shows the available keys for
the current context (board, detail, form, picker), and `?` shows the
full set — the interface must be usable without memorising this table.

### Moving a card between columns

| Keys | Action |
|---|---|
| `H` (shift+`h`) | Move focused card to the previous column |
| `L` (shift+`l`) | Move focused card to the next column |

- `H`/`L` act on the grouping field (§1): the card's grouping-field
  value becomes the adjacent column's option. Focus stays on the card
  in its new column.
- The move is **optimistic**: the card moves instantly, the mutation is
  sent, and on failure the card returns to its original column with the
  error shown in place (categories and remediation per ADR 0009/0010).
  A move is a single-field change, so it never opens a confirmation
  step; multi-field edits follow §4's preview/confirm rules.
- At the first/last column, `H`/`L` do nothing and the footer says so —
  no wrap-around.
- The detail panel offers the same move as a grouping-field edit, so a
  deliberate, non-adjacent jump (for example Backlog → Done) is done by
  editing the field there rather than pressing `L` repeatedly. There is
  no separate column-picker mode on the board itself.

### Other primary actions

| Keys | Action |
|---|---|
| `/` | Search/filter prompt |
| `e` | Edit the focused card (opens detail edit; see §4) |
| `n` | New draft item (see §4) |
| `g` | Grouping-field picker (§1) |
| `r` | Refresh the board from GitHub |

### Mouse

Mouse clicks may focus a column/card and open a card on double-click,
where the terminal supports it. There is **no drag-and-drop** in the
MVP, and no action is mouse-only. Behaviour must degrade cleanly in
terminals without mouse reporting.

Rejected for this section: mouse-first drag-and-drop as the primary
move gesture (unreliable across terminals, and the keyboard equivalent
would still need designing) and pane-and-picker moving (several steps
for the most common board action).

## 3. Visual design: compact board, split detail (settled 2026-10-13)

The board is a compact kanban with one slim header and the §2
contextual footer as its only chrome. Detail opens beside the board so
context is never lost while scanning, opening, editing, or moving.

### Layout

- **Header (1 line):** project title/alias, grouping field (§1), item
  count, and active filter/search state. Nothing else lives here.
- **Columns** fill the available width in field-option order (§1),
  each with a header showing the option name and its item count.
  Columns have a minimum readable width; when more columns exist than
  fit, the board scrolls horizontally and the focused column is always
  fully visible. Column bodies scroll vertically and independently.
- **Detail pane:** `Enter` (§2) opens detail as a right-hand pane at
  roughly 40% of the width on wide terminals; the board remains visible
  and focused-column context is preserved behind it. Below the wide
  breakpoint the pane takes the full screen. The exact breakpoint is an
  implementation detail to validate at 80, 120, and 200 columns against
  the ADR 0004 fixtures; the rule — split when readable, full-screen
  when not — is the settled contract.
- **List view** (the §1 fallback, and a user-selectable view) uses the
  same header/footer and card fields rendered as rows.

### Cards (3 lines, in order)

1. **Title** — truncated with an ellipsis, never wrapped onto line 2.
2. **Type + repository** — item kind as text (`issue`, `pr`, `draft`)
   plus `owner/repo` for linked items; drafts show `draft` alone.
3. **Meta row** — assignee, then the project's other salient fields in
   field order (for example priority, estimate, dates, iteration) as
   short `label value` pairs, truncated to fit.

The focused card is unmistakable without colour: a drawn border plus a
focus marker in addition to any highlight. Cards carry no decoration
that does not encode data.

### Colour and symbols

- The grouping field's option may be tinted per column, but the option
  **name is always printed as text** in the column header — status is
  never conveyed by colour alone (PRD §10). Priority and similar fields
  likewise pair any symbol with its text value on the card or detail.
- Styles adapt to the terminal's colour profile (truecolour, 256, and
  monochrome must all remain legible) and to light/dark terminals.
  In monochrome, hierarchy comes from weight, borders, and markers.
- Hierarchy is typographic: project/column titles strongest, card
  titles next, meta text muted but never low-contrast. No gradients,
  no decorative glow, no emoji as icons; symbols are limited to plain
  text markers (for example `●`, `○`, `▸`) that survive any terminal.
- Motion is functional only (loading spinner, optimistic-move
  feedback); there is no decorative animation.

### Detail pane content

Title, kind/repository link reference, body (scrollable), then every
project field with its current value in field order — the same data
the §4 edit form manipulates. Field values render with
the same text-plus-symbol rule as cards.

Rejected for this section: spacious tall cards with full-screen-only
detail (loses board context on every open and slows scanning of large
boards) and a minimal grouped-list-only look (gives up the crafted
board feel that justified the full TUI, ADR 0006).

## 4. Create/edit forms (settled 2026-10-13)

Editing happens **in the detail pane** (§3), in context. The single
exception to this section is already settled in §2: an `H`/`L` column
move is one field, applied instantly and optimistically, with no
confirmation step. Everything else follows the rules below, honouring
the PRD's preview/pending-changes contract (§3, §8 flow C).

### Editing an existing item

- `e` (§2) switches the detail pane to edit mode. Fields are visited
  in field order with the §2 navigation keys; the pane footer lists
  the keys valid for the focused control.
- Each field type gets a type-appropriate control, with options always
  from the live project schema (PRD §9):
  - **Single-select:** a picker over the field's options, plus a
    "clear" choice where GitHub permits an empty value.
  - **Iteration:** a picker over the field's iterations (current and
    upcoming first), plus "clear" where permitted.
  - **Date:** a text input validated as a real calendar date before it
    can be left; invalid input is rejected in place with the reason.
  - **Number:** a text input validated as a number in place.
  - **Text:** a single/multi-line text input, as the field's use implies.
  - **People/assignees:** a picker over project collaborators where the
    API exposes them; if it cannot supply candidates, the field shows
    as unsupported in the TUI (and remains editable via `item edit`).
  - **Title/body:** text inputs; body uses a multi-line editor in the
    pane. Fields GitHub does not permit editing for the item kind
    (issue vs. PR vs. draft) are shown read-only with the reason, never
    silently hidden.
- **Dirty state:** every changed field is visibly marked, and the pane
  header shows a change count. Untouched fields are never rewritten.
- **Save:** the save key first replaces the form with a **change
  summary** — one line per change, `Field: old → new` (long values
  truncated) — and applies only on an explicit confirm keypress. After
  applying, the pane returns to read mode showing the saved values; on
  failure nothing is applied silently: the summary stays, the error is
  shown with its ADR 0009 category and remediation, and the user can
  fix, retry, or discard.
- **Discard:** leaving edit mode with unsaved changes always asks
  first. There is no path that silently drops a dirty form.
- Validation happens before the summary; a form containing an invalid
  field cannot reach the confirm step.

### Creating work

- `n` opens the same pane as a **draft-item form**: title (required,
  non-empty) and optional body, then the same change-summary confirm
  (shown as the item to be created) before anything is sent.
- Adding an **existing issue/PR** uses a reference prompt in the pane:
  the user enters `owner/repo#number` or a URL; Boardwalk resolves and
  previews the item (title, kind, repository, current project state if
  already present) and adds it only on confirmation. An item already on
  the project is reported, not duplicated. The key for this prompt is
  part of the §2 key table's forms extension and must appear in the
  footer whenever the board has focus — final key assignment is an
  implementation detail to keep consistent with §2, not a new mode.
- Newly created/added items appear in their grouping column (§1) once
  confirmed by GitHub, focused, with the resulting item ID/link
  reference shown in the pane (PRD §8 flow B).

### Relationship to the CLI

The same validation, summary, and error rules apply to
`boardwalk item edit` / `item add` in interactive use; `--dry-run`
prints the identical change summary without applying (ADR 0009), and
`--yes` skips the confirm step only when the user passed it explicitly.
TUI and CLI must never diverge in what they would send (ADR 0006,
ADR 0007).

Rejected for this section: applying every edit immediately (a mistake
is live on GitHub before it is seen, and it breaks the PRD's
preview contract for all but moves) and a full-screen step-by-step
form wizard (hides the board for the whole edit; fixing one field
becomes a multi-screen trip).

## 5. Loading, refresh, and conflicts (settled 2026-10-13)

Progressive and explicit: the screen is never blank waiting for
GitHub, the only optimistic mutation is the §2 column move, and
staleness is handled by an explicit refresh and a defined save-conflict
flow — never by background magic. This implements PRD §8 flow C and
§10 (progress for network work, no repeated full-board fetches for a
single field update).

### Loading

- The board paints progressively: project schema and column headers
  (§1, §3) first, then cards as item pages arrive. A spinner appears
  only in regions whose data is still missing (header count, a column
  body), never as a full-screen blocker once any data can be shown.
- The board is usable as soon as the first page renders: navigation,
  search, and already-loaded cards work while later pages stream in.
  Actions that need not-yet-loaded data say so in the footer rather
  than failing.
- A failed page load leaves loaded data on screen, marks the missing
  region, and offers retry (`r`); it never discards what already
  arrived. Startup failures that prevent any board (auth/scopes per
  ADR 0010, project not found) exit before the TUI starts, using the
  ADR 0009 messages and exit codes.

### Mutations and refresh

- **`H`/`L` moves** are optimistic (§2): instant on screen, rolled back
  on failure with the error shown in place.
- **Confirmed edits/creates (§4)** are not optimistic: the summary is
  confirmed, the mutation is sent with a progress indication in the
  pane, and on success only the affected item is refetched and updated
  in place. A full-board refetch for a single-item change is forbidden
  (PRD §10).
- **`r` refresh** refetches the board (schema, then items) while
  preserving the grouping field, filters/search, focused item, and
  scroll context as far as the refreshed data allows. If the focused
  item vanished, focus lands on its nearest surviving neighbour and
  the footer says the item is gone.
- **No background polling in the MVP.** The board states its data
  age (for example in the header after refresh) rather than pretending
  to be live. Auto-refresh, if ever wanted, needs its own decision: it
  spends GraphQL rate budget while idle (PRD §12) and multiplies the
  conflict cases below.

### Conflicts and errors

GitHub Projects v2 has no item versioning Boardwalk can rely on, so
conflicts are detected, not prevented:

- If a confirmed save fails — or the pre-save refetch shows the item's
  current values no longer match the values the form was opened with —
  Boardwalk does **not** overwrite silently. It refetches the item,
  then shows the §4 change summary side by side with the new current
  values (`Field: theirs-now → yours-pending`) and offers exactly two
  choices: re-apply onto the new values, or discard the pending
  changes. The form's edits are kept until the user chooses.
- A rolled-back optimistic move (§2) is the same idea at card scale:
  original column restored, error in place, no data lost.
- All errors use the ADR 0009 categories and remediation text
  (auth/scope → exit-3 wording shown in the footer/pane, permission,
  not-found, rate-limit with retry information). Rate-limited or
  transient *reads* retry with bounded backoff before surfacing
  (ADR 0009); mutations are never silently retried after an uncertain
  outcome (PRD §10, §12) — the user retries explicitly from the error
  state.

Rejected for this section: blocking pessimistic loading/mutation
(blank spinner, screen changes only after GitHub answers — contradicts
§2 and feels slow on the ADR 0004 large board) and a live polling
board (idle rate-limit spend plus a conflict surface the PRD scopes
out of the MVP, §6).

## 6. List view (settled 2026-10-13)

The list view is a **first-class view**, not only a fallback. It is the
fastest way to scan large boards (ADR 0004's 100+-item fixture) and it
doubles as the PRD §10 plain-list fallback.

- **Toggle:** `v` switches between kanban and list in place, without
  leaving the TUI or losing state. Focus (the same item), filters/
  search, grouping field, and scroll context carry across the switch
  as far as the target view allows.
- **Entry rules:** `boardwalk board` opens in kanban unless §1 forces
  list view (no single-select field), in which case it opens in list
  and says why. A launch flag may request list first (CLI spec detail);
  the in-place toggle exists regardless.
- **Rows:** one row per item with dynamic columns in this priority:
  title (widest, truncated with ellipsis), kind (`issue`/`pr`/`draft`),
  repository, grouping-field value, assignee, then the project's
  remaining salient fields in field order as space allows (§3 card
  meta rule). Columns appear/disappear based on the live schema only —
  a field the project does not have never renders as an empty column.
- **Ordering:** by grouping-option order (§1), then by the project's
  own item order within each group. The grouping value is printed as
  text on every row (§3's never-colour-alone rule), so the list stays
  meaningful in monochrome and when piped concepts are described.
- **Shared behaviour — no second-class surface:** everything from
  §2/§4/§5 works identically in list view: `↑`/`↓` (and `j`/`k`) move
  between rows, `Enter` opens the same split detail pane, `e`/`n` and
  the add-existing prompt use the same forms, `/` filters, `g` changes
  grouping (regrouping the rows), `r` refreshes preserving context,
  and `H`/`L` change the focused row's grouping value with the same
  optimistic-move-and-rollback semantics. Loading is progressive by
  page exactly as on the board (§5).
- **CLI relationship:** `boardwalk item list` (ADR 0003) prints this
  same row model as a table for scripts/humans outside the TUI; field
  selection flags there follow the CLI specification. The TUI list and
  the CLI list must agree on ordering and field meaning.

Rejected for this section: list as a read-mostly fallback only (users
on normal boards would lose the best scanning view, and the §10
fallback would stay second-class) and list reachable only via a launch
flag with no in-place toggle (switching views would mean quitting and
relaunching, against the speed priority).

## Specification status

All sections of this specification were settled in the decision
interview and accepted as **Final** by the owner on 2026-10-13.
Changes from here require a spec amendment, not a silent edit.
Implementation details deliberately left open here (exact wide/narrow
breakpoint, the add-existing key, spinner glyphs) must stay consistent
with the settled rules and be validated against the ADR 0004 fixtures
at 80, 120, and 200 columns, keyboard-only.
