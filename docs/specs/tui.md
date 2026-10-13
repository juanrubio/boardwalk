# Boardwalk TUI specification

**Status:** Final — amended by A1 (2026-10-13)
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Amended:** 2026-10-13 — A1, owner approval of the
[beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md).
Amended passages are marked *(A1)*; see the amendment log at the end.
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
density = "comfortable"   # (A1) §3, toggled with `d`
collapsed = ["Done"]      # (A1) §3, toggled with `z`
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
  "No <field>" column rather than being hidden. *(A1)* It is always the
  **first** column, as on GitHub's own board, and it is hidden when
  empty.
- *(A1)* The remembered grouping choice is a **TUI view preference
  only**. It never changes what a CLI command reads, filters, or writes.
  `item move` and `item list --status` use the project's default board
  field (rules 2–3 above), independent of this preference (cli.md §2).
  The TUI picker is the only place the preference is changed.

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
| `Enter` | Open the detail pane for the focused card |
| `Tab` | *(A1)* Move focus between the board and an open detail pane |
| `Esc` / `q` in a panel | Close the panel and return focus to the board |
| `q` on the board | Quit (see "Quitting" below) |
| `?` | Toggle the help overlay listing all keys |

Arrow keys and `hjkl` are both active wherever navigation is possible.
*(A1)* The single exception is a **focused text input** (search prompt,
title, body, text/number/date fields, reference prompt). It captures
every printable key, so `h j k l q e n` type characters. Inside a form,
`Tab`/`Shift-Tab` and `↑`/`↓` move between fields, and `Esc` leaves the
input (§4). A visible footer shows the available keys for the current
context (board, detail, form, picker, input), and `?` shows the full
set. The interface must be usable without memorising this table.

*(A1)* **Live preview.** While the detail pane is open in read mode,
board focus stays on the board: `j/k/h/l` keep moving between cards and
the pane follows the focused card. `Tab` moves focus into the pane (to
scroll a long body) and back. Bodies load lazily and are prefetched for
neighbours (performance.md §2), so the pane stays responsive.

### Moving a card between columns

| Keys | Action |
|---|---|
| `H` (shift+`h`) | Move focused card to the previous column |
| `L` (shift+`l`) | Move focused card to the next column |
| `u` | *(A1)* Undo the last move made in this session |

- `H`/`L` act on the grouping field (§1): the card's grouping-field
  value becomes the adjacent column's option. Focus stays on the card
  in its new column.
- The move is **optimistic**: the card moves instantly, the mutation is
  sent, and on failure the card returns to its original column with the
  error shown in place (categories and remediation per ADR 0009/0010).
  A move is a single-field change, so it never opens a confirmation
  step; multi-field edits follow §4's preview/confirm rules.
- *(A1)* **Coalescing.** Repeated presses on the same card move it on
  screen at once, every time, but Boardwalk sends **one** mutation with
  the final target after about 250 ms with no further presses. Writes
  for an item are sent one at a time, and the last one wins. A rollback
  returns the card to the last value GitHub confirmed. So `L L L` is a
  cheap, safe way to jump several columns, and writes never race.
- *(A1)* **Sync state.** A card with a write in flight shows a pending
  marker (`◌`) next to its title. A failed write shows a failure marker
  (`✗`) and its reason in place until the next action on that card.
  Neither relies on colour alone.
- *(A1)* **Stale cards.** A card shown only from the startup snapshot
  (ADR 0011) cannot be moved until its live data has arrived. The footer
  says so instead of acting.
- *(A1)* **Undo.** `u` reverses the most recent confirmed move by moving
  the card back. It is an ordinary move with the same coalescing and
  rollback rules. Undo covers moves only, not §4 edits.
- At the first/last column, `H`/`L` do nothing and the footer says so —
  no wrap-around.
- The detail panel offers the same move as a grouping-field edit, for
  picking a target by name. There is no separate column-picker mode on
  the board itself.

### Other primary actions

| Keys | Action |
|---|---|
| `/` | Search/filter prompt (see "Search and filter" below) |
| `e` | Edit the focused card (opens detail edit; see §4) |
| `n` | New draft item (see §4) |
| `a` | *(A1)* Add an existing issue/PR by reference (see §4) |
| `x` | *(A1)* Archive the focused item, after explicit confirmation (ADR 0006) |
| `o` | *(A1)* Open the focused item on GitHub in the browser |
| `y` | *(A1)* Copy the focused item's reference (cli.md §1); `Y` copies its URL |
| `g` | Grouping-field picker (§1) |
| `v` | List/kanban toggle (§6) |
| `z` | *(A1)* Collapse/expand the focused column (§3) |
| `d` | *(A1)* Toggle card density, comfortable/compact (§3) |
| `r` | Refresh the board from GitHub |

*(A1)* **Archive** shows a confirmation naming the item (reference,
kind, title). It is applied only when the user presses `y` at that
prompt. Any other key cancels. Archive is not optimistic: the card stays
until GitHub confirms, then it is removed and focus moves to the nearest
neighbour.

*(A1)* **Clipboard** uses OSC 52 where the terminal supports it, and the
system clipboard otherwise. If neither works, the footer shows the
reference so it can be copied by hand.

### Search and filter *(A1)*

- `/` opens a one-line prompt in the header. Filtering is
  **incremental**: loaded cards are filtered on every keystroke.
  Non-matching cards are hidden, and the header shows the active filter
  and the match count.
- The prompt accepts free text plus **GitHub's project filter syntax**
  (for example `status:"In Progress" assignee:@me is:draft
  repo:owner/name`), the same language as the web board and CLI
  `--search` (cli.md §2). This fills ADR 0006's status/assignee/
  repository/type filtering without a separate filter UI.
- Expressible qualifiers are also sent to GitHub as the server-side
  item query, debounced (performance.md §2). Results are the same either
  way.
- While pages are still loading, the count reads
  `matching 42 of 300 loaded`, never a silently partial answer.
- `Enter` keeps the filter and returns to the board. `Esc` clears it.
  Matches are highlighted in card titles with weight or underline, not
  only colour.

### Quitting *(A1)*

`q` (board) and `Ctrl-C` (anywhere) quit at once **unless** a write is
in flight or a form is dirty:

- In-flight writes: the footer shows `saving 1 change…` and Boardwalk
  quits when the writes settle. A second `Ctrl-C` quits at once and
  prints the unconfirmed changes on exit, so their outcome is never
  silently unknown.
- A dirty form follows §4's discard rule.

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
*(A1)* Full character-cell layouts for all states and densities are
recorded in [mockups.md](mockups.md).

### Layout

- **Header (1 line):** project title/alias, grouping field (§1), item
  count, active filter/search state, and *(A1)* data age
  (`updated 3h ago`, `refreshing…`, ADR 0011). Nothing else lives here.
- **Columns** fill the available width in field-option order (§1),
  each with a header showing the option name and its item count.
  Columns have a minimum readable width, *(A1)* initially 26 cells, to
  be confirmed in the layout spike. When more columns exist than
  fit, the board scrolls horizontally and the focused column is always
  fully visible. Column bodies scroll vertically and independently.
- *(A1)* **Off-screen hints.** Hidden columns are shown at the board
  edges as `◂ 2` / `3 ▸` (the number of columns off-screen). A column
  with cards below the fold ends with `↓ 12 more`. Nothing off-screen is
  ever invisible without a hint.
- *(A1)* **Collapsed columns.** `z` collapses the focused column into a
  narrow strip that shows only its name, truncated, and its count (for
  example a long Done column). Collapsed columns are remembered per
  project in the same config section as `group_by`:
  `collapsed = ["Done"]`. Focusing a collapsed column and pressing `z`
  expands it.
- **Detail pane:** `Enter` (§2) opens detail as a right-hand pane at
  roughly 40% of the width on wide terminals; the board remains visible
  and focused-column context is preserved behind it. Below the wide
  breakpoint the pane takes the full screen. The exact breakpoint is an
  implementation detail to validate at 80, 120, and 200 columns against
  the ADR 0004 fixtures; the rule — split when readable, full-screen
  when not — is the settled contract. *(A1)* In split mode the pane is a
  live preview that follows board focus (§2).
- **List view** (the §1 fallback, and a user-selectable view) uses the
  same header/footer and card fields rendered as rows.
- *(A1)* **Minimum size and resize.** Below 60×15 cells Boardwalk shows
  a centred "terminal too small" message with the current and required
  size, and nothing else. Resizing reflows immediately and keeps focus
  on the same item. Boardwalk always restores the terminal on exit,
  including after a panic.

### Cards

*(A1)* Two densities, toggled with `d` and remembered per project
(`density = "comfortable" | "compact"`):

**Comfortable (default, 3 lines, in order)**

1. **Title** — truncated with an ellipsis, never wrapped onto line 2.
2. **Kind + reference** — item kind as text (`issue`, `pr`) plus the
   copyable reference (cli.md §1), for example
   `issue juanrubio/snazzyemail#42`. *(A1)* Drafts show their
   `draft:<id>` reference, which already names the kind.
3. **Meta row** — assignee, then the project's other salient fields in
   field order (for example priority, estimate, dates, iteration) as
   short `label value` pairs, truncated to fit.

**Compact (2 lines)** *(A1)* — line 1 is the title, and line 2 is the
reference followed by the assignee. Compact cards have no box. They use a
one-cell left gutter instead, so roughly twice as many cards fit on an
80×24 screen.

The focused card is unmistakable without colour: a drawn border plus a
focus marker in addition to any highlight. Cards carry no decoration
that does not encode data. *(A1)* **Focus never reflows the layout.**
Every comfortable card reserves its border cells, drawn in the `subtle`
token when unfocused. Every compact card reserves its gutter cell, blank
when unfocused and `▌` when focused. Moving focus changes only the
style, never a card's size.

### Colour and symbols

- The grouping field's option may be tinted per column, but the option
  **name is always printed as text** in the column header — status is
  never conveyed by colour alone (PRD §10). Priority and similar fields
  likewise pair any symbol with its text value on the card or detail.
- *(A1)* **GitHub option colours.** Tints come from the option's own
  colour in the live schema (`GRAY`, `BLUE`, `GREEN`, `YELLOW`,
  `ORANGE`, `RED`, `PINK`, `PURPLE`), mapped to Boardwalk's curated
  palette (Visual language, below). The board therefore matches the
  user's web board with no configuration. The same mapping colours
  single-select chips (for example Priority) on cards, rows, and detail.
  An option's `description` is shown in the grouping picker and in
  detail.
- Styles adapt to the terminal's colour profile (truecolour, 256, and
  monochrome must all remain legible) and to light/dark terminals.
  In monochrome, hierarchy comes from weight, borders, and markers.
  *(A1)* `NO_COLOR` forces monochrome. The light/dark choice comes from
  the terminal's reported background (Bubble Tea v2, ADR 0007) and falls
  back to dark when the terminal does not report one.
- Hierarchy is typographic: project/column titles strongest, card
  titles next, meta text muted but never low-contrast. No gradients,
  no decorative glow, no emoji as icons; symbols are limited to plain
  text markers (for example `●`, `○`, `▸`) that survive any terminal.
- Motion is functional only (loading spinner, optimistic-move
  feedback); there is no decorative animation.

### Visual language *(A1)*

All styling goes through one theme package. Components never use raw
colours or ad-hoc padding.

- **Semantic colour tokens:** `fg`, `muted`, `subtle`, `border`,
  `borderFocus`, `accent`, `selectionBg`, `success`, `warning`,
  `danger`, plus the eight option-colour tokens above. Each token has a
  light, dark, 256-colour, and monochrome value (monochrome falls back
  to bold, faint, reverse, or underline). Every foreground/background
  pair used for text meets WCAG AA contrast (4.5:1) in truecolour and
  256-colour, in both light and dark.
- **Spacing:** a one-cell gutter between columns, one cell of
  horizontal padding inside cards and the detail pane, and no vertical
  padding inside cards. The header and footer are flush, one line each.
- **Borders:** rounded borders on comfortable cards and the detail pane;
  no column boxes, because columns are separated by gutter and header.
  A thick border plus `▸` marks focus. Modal overlays (help, archive
  confirm, pickers) use a rounded border drawn over a dimmed board.
- **Typography roles:** `title` (bold, `fg`), `columnHeader` (bold,
  option tint plus name), `cardTitle` (`fg`, bold when focused), `meta`
  (`muted`), `keyHint` (key in `accent`, label in `muted`), `error`
  (`danger` plus `✗`), and `success` (`success` plus `✓`).
- **Relative time:** dates and iterations render with relative labels
  next to the value, for example `Target 2026-10-20 (in 7d)`,
  `overdue 2d`, `@current` for the current iteration. Urgency is carried
  by the label text first and the colour second.

### Detail pane content

Title, kind/repository link reference, body (scrollable), then every
project field with its current value in field order — the same data
the §4 edit form manipulates. Field values render with
the same text-plus-symbol rule as cards. *(A1)* The body is rendered as
Markdown with Glamour, styled from the same theme tokens. Rendering runs
off the UI loop and is cached per (item version, width, theme)
(performance.md §3). Long bodies scroll inside the pane.

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
  in field order with `Tab`/`Shift-Tab` or `↑`/`↓` *(A1: not `j`/`k`,
  which type into a focused input, §2)*. The pane footer lists the keys
  valid for the focused control.
- *(A1)* **Two-level `Esc`.** In a focused input, `Esc` leaves the input
  and keeps its value. On the form itself, `Esc` leaves edit mode, which
  triggers the discard rule below if the form is dirty. The save key is
  `Ctrl-S`. It is shown in the footer whenever the form is dirty.
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
    pane. *(A1)* `Ctrl-E` on the body hands off to `$VISUAL`/`$EDITOR`
    (falling back to the pane editor if neither is set). The TUI is
    suspended while the editor runs, and the saved text comes back as a
    dirty body change. Fields GitHub does not permit editing for the item kind
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
- *(A1)* **One request, results per field.** GitHub applies field
  changes one mutation at a time, so a save is **not atomic**. All of an
  item's changes go out as one GraphQL request (performance.md §2), and
  the summary then marks each line `✓ applied` or `✗ failed` with its
  reason. Retry resends only the failed lines. Discard drops only the
  unapplied ones. Applied values are never rolled back behind the user's
  back.
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
  the project is reported, not duplicated. *(A1)* The prompt opens
  with `a` (§2) and appears in the footer whenever the board has focus.
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
single field update). *(A1)* Painting a local snapshot is not
background magic, provided its age is always shown and stale data can
never be written through ([ADR 0011](../decisions/0011-local-cache-and-snapshot.md)).

### Loading

- *(A1)* **Snapshot first.** If a board snapshot exists (ADR 0011), the
  board paints it immediately with zero network requests
  (performance.md B4). The header shows its age, and cards from the
  snapshot cannot be written to until their live data arrives (§2). One
  revalidation then runs as below. A missing or corrupt snapshot just
  means a cold start.
- The board paints progressively: project schema and column headers
  (§1, §3) first, then cards as item pages arrive. *(A1)* The schema
  and the first item page come back in **one** request
  (performance.md §2), so column headers and the first cards appear
  together. A spinner appears
  only in regions whose data is still missing (header count, a column
  body), never as a full-screen blocker once any data can be shown.
- *(A1)* **Stable focus while streaming.** Arriving pages and
  revalidated data never move the focused card on screen. Inserts above
  it shift the content outside the viewport, not the card under the
  cursor. Column counts read `12+` until the column is fully loaded.
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
  on failure with the error shown in place. *(A1)* Repeated presses are
  coalesced into one write per item (§2).
- **Confirmed edits/creates (§4)** are not optimistic: the summary is
  confirmed, the mutation is sent with a progress indication in the
  pane, and on success the affected item is updated in place. *(A1)*
  The data comes from the mutation's own response (performance.md §2).
  One targeted item query is allowed only where a payload cannot carry
  it. A full-board refetch for a single-item change is forbidden
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
fastest way to scan large boards (ADR 0004's large fixtures, *(A1)* 1,000
and 10,000 items) and it doubles as the PRD §10 plain-list fallback.

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
  *(A1)* reference (cli.md §1, which includes the repository for linked
  items), grouping-field value, assignee, then the project's
  remaining salient fields in field order as space allows (§3 card
  meta rule). Columns appear/disappear based on the live schema only —
  a field the project does not have never renders as an empty column.
- **Ordering:** by grouping-option order (§1), then by the project's
  own item order within each group. The grouping value is printed as
  text on every row (§3's never-colour-alone rule), so the list stays
  meaningful in monochrome. *(A1: replaces a garbled clause.)*
- **Shared behaviour — no second-class surface:** everything from
  §2/§4/§5 works identically in list view: `↑`/`↓` (and `j`/`k`) move
  between rows, `Enter` opens the same split detail pane, `e`/`n` and
  the add-existing prompt use the same forms, `/` filters, `g` changes
  grouping (regrouping the rows), `r` refreshes preserving context,
  and `H`/`L` change the focused row's grouping value with the same
  optimistic-move-and-rollback semantics. Loading is progressive by
  page exactly as on the board (§5). *(A1)* `a`, `x`, `o`, `y`/`Y`, and
  `u` behave as on the board, and `z` collapses a group. Rows are
  virtualised exactly like cards (performance.md §3).
- **CLI relationship:** `boardwalk item list` (ADR 0003) prints this
  same row model as a table for scripts/humans outside the TUI; field
  selection flags there follow the CLI specification. The TUI list and
  the CLI list must agree on ordering and field meaning.

Rejected for this section: list as a read-mostly fallback only (users
on normal boards would lose the best scanning view, and the §10
fallback would stay second-class) and list reachable only via a launch
flag with no in-place toggle (switching views would mean quitting and
relaunching, against the speed priority).

## 7. Visual acceptance (A1, 2026-10-13)

"Beautiful" is a release gate, so it must be testable.

- **Golden frames.** `teatest` snapshot tests render each ADR 0004
  fixture at 80×24, 120×40, and 200×50, in truecolour, 256-colour, and
  monochrome, and in light and dark, for these states: board, focused
  card, split detail, edit form with dirty fields, change summary with a
  partial failure, help overlay, search with matches, snapshot-stale
  header, and "terminal too small". Any frame diff fails CI until the
  golden file is deliberately updated.
- **Contrast check.** A unit test asserts the WCAG AA rule for every
  text token pair in the theme (§3 Visual language).
- **VHS tapes.** Scripted VHS recordings of the five CLI spec §4 human
  tasks, plus a "large board" scroll/search tape, are generated in CI.
  The owner reviews them before each release. They also serve as README
  and demo media.
- **Performance.** Keypress-to-frame and memory budgets are in
  [performance.md](performance.md). Visual acceptance and performance
  acceptance are both required for the MVP.

## Specification status

All sections of this specification were settled in the decision
interview and accepted as **Final** by the owner on 2026-10-13.
Changes from here require a spec amendment, not a silent edit.
Implementation details deliberately left open here (exact wide/narrow
breakpoint, *(A1)* exact minimum column width, spinner glyphs) must stay
consistent with the settled rules and be validated against the ADR 0004
fixtures at 80, 120, and 200 columns, keyboard-only. *(A1)* A one-day
layout and latency spike on Bubble Tea v2 fixes these values before the
build starts. The results are recorded as an amendment.

## Amendment log

- **A1 (2026-10-13)** — owner approval of the
  [beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md):
  - §1: the "No ‹field›" column is first and hidden when empty; the
    grouping preference is TUI-only and never affects CLI commands.
  - §2: text inputs capture printable keys; live-preview detail pane;
    move coalescing, sync markers, stale-card guard, `u` undo; new keys
    `a` add, `x` archive, `o` open, `y`/`Y` copy, `z` collapse,
    `d` density; search/filter semantics in GitHub filter syntax;
    quitting with in-flight writes.
  - §3: data age in header; minimum column width; off-screen hints;
    collapsed columns; minimum terminal size and resize; comfortable and
    compact densities with no focus reflow; cards show the copyable
    reference; GitHub option colours; `NO_COLOR`; new Visual language
    subsection; Markdown bodies via Glamour.
  - §4: Tab/arrow field navigation, two-level `Esc`, `Ctrl-S` save,
    `$EDITOR` handoff, one-request saves with per-field results, `a` for
    add-existing.
  - §5: snapshot-first paint (ADR 0011), one-request first page,
    focus-stable streaming, mutation payloads instead of refetches.
  - §6: larger fixtures, reference column, garbled clause fixed, A1 keys.
  - §7: new — visual acceptance gate.
