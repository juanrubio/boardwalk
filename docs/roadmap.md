# Boardwalk roadmap — what's next, and why

Boardwalk covers the core board loop — discover, file, move, reorder,
edit — as of v0.5.2 plus within-column reordering (TUI amendment A13).
What follows are the remaining features judged genuinely valuable, in
recommended build order. Deliberately excluded items are listed at the
end so the "no"s are on record.

Items are ordered by how often their absence forces a trip to the
browser, not by ease. Each one is a proposal, not a commitment: it
still needs its spec amendment, and where flagged a new ADR, before
implementation. The yardsticks are the principles in
[PRD §3](../PRD.md) and the existing record in [decisions](decisions/).

## 1. Comments in the detail pane

**Why it moves the needle:** The commonest reason to leave the
terminal today is reading and posting discussion. A card's status and
fields are already here; the conversation is not, so triage still ends
in the browser. Closing that one gap is what makes "everyday use must
not require a browser" true in practice.

**Sketch:**

- The detail pane gains the issue's comment thread, loaded lazily when
  the pane opens, newest relevant first.
- `c` composes in `$VISUAL`/`$EDITOR`; posting goes through the usual
  preview-and-confirm mutation rules, and `--dry-run`/`--yes`
  semantics where the surface has them.
- CLI read access (e.g. comments on the item-detail output) only if
  it is trivial to add; otherwise this lands TUI-first.

**Effort:** L. Main risk: scope creep into a general GitHub client —
reactions, reviews, and notifications are the adjacent requests, and
this item is comments only.

**Spec/ADR:** [PRD §6](../PRD.md) excluded "full issue triage
(comments, reactions, review workflows)" from the MVP. This is the
deliberate, scoped expansion of that exclusion — read and post
comments, nothing else — and needs a PRD §6 amendment plus a
[TUI spec](specs/tui.md) amendment to say so.

## 2. Start-work: branch from a card

**Why it moves the needle:** This is the terminal-native
differentiator: the board is where work is chosen and the terminal is
where it starts, and today the two never touch. One key that
branches, assigns, and advances the card removes the manual glue
between plan and code that every other tool leaves to the user.

**Sketch:**

- `b` on an issue card creates and checks out a branch from a
  configurable template (e.g. `<type>/<number>-<slug>`).
- Optionally move the card to In Progress and assign `@me` in the
  same action — one previewed, confirmed step, not three.
- Inverse: launching inside a repo whose checked-out branch names an
  item highlights and focuses that card.

**Effort:** M. Main risk: repo detection and template edge cases
across users' branch conventions. Must degrade cleanly outside a
work tree — unavailable with a clear message, never a guessed repo.

**Spec/ADR:** Needs a new ADR revisiting
[ADR 0005](decisions/0005-alias-config-user-only.md) (user-only
config): templates and branch-based focus must not silently
reintroduce directory-dependent behaviour. TUI spec amendment for
`b`; CLI spec amendment if it gets a non-TUI twin.

## 3. Multi-select bulk triage

**Why it moves the needle:** Grooming is batch work — close out an
iteration, sweep stale cards, re-prioritise a column — and today
everything is one card at a time. Single-card moves are already fast;
the batch is where the time goes, and where the browser currently
wins by default.

**Sketch:**

- `Space` toggles selection in the kanban and list views; the footer
  shows the count, `Esc` clears.
- Bulk actions: move (`H`/`L`), set one field via the existing edit
  form, archive. Reorder-as-block is deferred until usage demands it.
- One confirmation summary listing all items, then per-item results
  (✓/✗ lines) like today's batched field saves.
- CLI twin: item verbs accept multiple references.

**Effort:** L in the TUI, M on the CLI. Main risk: confirmation and
partial-failure UX — "3 of 10 failed, and which ones" must be
impossible to miss, without pretending the other 7 did not happen.

**Spec/ADR:** [CLI spec](specs/cli.md) §1 currently specifies one
item reference per verb; amending it to several is the core of the
CLI work. No new ADR expected — the confirmation and
partial-failure rules are spec amendments, not new architecture.

## 4. Sub-issues, read-first

**Why it moves the needle:** GitHub is reorganising around
sub-issues, and a board that cannot show "this card is 3 of 5 done"
understates real progress. Visibility comes before authoring: most
users need to see the tree long before they need to edit it from the
terminal.

**Sketch:**

- Cards show `[3/5]` next to the title when the issue has
  sub-issues.
- The detail pane lists children with open/closed state; focus can
  jump to a child that is already on the board.
- Creating or relating sub-issues is explicitly deferred and
  usage-gated.

**Effort:** M. Main risk: the GraphQL shape — extra per-item data on
large boards must not break the startup and frame budgets in
[performance](specs/performance.md) — plus card real estate on
narrow terminals.

**Spec/ADR:** CLI and TUI spec amendments for the new display data
(including `--json`). No ADR unless fetching children changes the
snapshot contract in
[ADR 0011](decisions/0011-local-cache-and-snapshot.md) — check
before adding queries.

## 5. Honour GitHub's saved Views

**Why it moves the needle:** Team boards define their shared triage
views on GitHub, but Boardwalk keeps its own `group_by` preference,
so "the triage view" means different things in the browser and the
terminal. On any board with more than one user, the team's shared
definition matters more than Boardwalk's local default.

**Sketch:**

- Read the project's ProjectV2 views; `V` opens a picker.
- Choosing a view applies its filter/group/sort. Boardwalk's local
  preferences remain the default when no view is chosen.
- Swimlanes (a second grouping dimension) only if real views demand
  it — not speculatively.

**Effort:** L. Main risk: coverage — GitHub's view query/group
syntax is broader than Boardwalk's filter language. Anything that
cannot be mapped must degrade honestly, saying what was ignored, and
never silently mis-filter.

**Spec/ADR:** TUI spec amendment for the picker and the precedence
rules; a small ADR if "chosen view persists per project" touches the
config contract. Views are read from GitHub, never forked into a
local parallel definition — that is "respect GitHub's model"
([PRD §3](../PRD.md)) applied to views.

## 6. Field & state gap bundle

**Why it moves the needle:** None of these is individually
strategic, but each is a papercut that sends the user to the browser
for a five-second job. Batched, they are cheap filler between the
larger items, and they mostly extend surfaces that already exist.

**The bundle** (one line each; **Effort** S each):

- Set people fields / assignees from the edit form and the CLI.
- Edit an issue's labels.
- Clear-a-field ergonomics in the edit form — an explicit clear
  action, not a magic empty string.
- State glyphs on cards: issue open/closed, PR merged, CI and review
  status (queries fetch none of these today).
- Unarchive, and a way to view archived items.
- `Iteration=@current` / `@next` in the edit form and the CLI.
- A CLI twin for A13 reordering — e.g. `item move --before <ref>` /
  `--after <ref>` (or `--position`) — so scripts can order a column
  too.

**Spec/ADR:** Each entry is a spec amendment to
[cli.md](specs/cli.md) and/or [tui.md](specs/tui.md); none is
expected to need an ADR. Ship in any order; none blocks another.

## Deliberately not planned

- **Roadmap/timeline view** — terminal-hostile to render honestly,
  and used too rarely to justify the attempt.
- **Project/field administration** (creating projects, editing
  fields and workflows) — rare, one-off setup; the web UI is fine
  for it ([PRD §6](../PRD.md)).
- **Reactions and notifications** — GitHub Issues territory.
  Boardwalk manages board state, not an inbox.
- **Offline editing** — contradicts GitHub as the source of truth;
  the local snapshot stays read-only
  ([ADR 0011](decisions/0011-local-cache-and-snapshot.md)).
- **A `--jq` flag** — `--json` already exists; pipe it to `jq`.
- **Replacing `gh issue` flows** — close/reopen and label
  administration beyond the bundle above stay with `gh`. Boardwalk
  borrows its auth; it should not re-grow it.

## Suggested sequence

Build in the order listed — comments (1), start-work (2), bulk
triage (3), sub-issues (4), saved Views (5) — with the field & state
bundle (6) interleaved as filler whenever a small change is wanted
between the larger ones. Each item needs its spec amendment, and an
ADR where flagged above, before implementation, per repo practice.
And the order is a judgement, not a schedule: as the
[CLI spec](specs/cli.md) puts it, new verbs require usage evidence —
someone stuck without one of these on a real board should reorder
this list without ceremony.
