# Boardwalk spec review: beauty and speed

**Reviewed:** [PRD.md](../../PRD.md), ADRs 0001–0010, [cli.md](../specs/cli.md), [tui.md](../specs/tui.md)
**Status:** Approved by the owner on 2026-10-13. Applied as **Amendment 1 (A1)** across the PRD, ADRs, and specs; see each document's amendment log.
**Lens:** (1) a beautiful CLI and, above all, a beautiful TUI; (2) fast performance.

## Verdict

The specs are unusually disciplined about **correctness and safety**: references, exit codes, previews, conflicts, and accessibility are all well covered. But they are **thin on the two things you care about most**:

- **Speed has no numbers.** No document sets a latency budget, defines a cache, or counts round trips. The only "performance gate" is a 100-item board, which is tiny. GitHub Projects now allows up to 50,000 items per project.
- **Beauty is mostly described by what to avoid** ("no gradients, no glow, no emoji"). There is no positive design system, no visual acceptance test, and several layout behaviours that will look janky in practice are left open.

Everything was marked **Final** on the same day, before any prototype. A few of the locked-in decisions (below) will not survive contact with real data, so plan an amendment pass after a short spike.

---

## 1. Performance

### P1 — No performance budgets *(critical)*
ADR 0001 rightly says perceived speed depends on "caching, optimistic updates, asynchronous loading, and avoiding redundant fetches". But no spec turns that into requirements. Suggested budgets, measured offline on fixtures plus one live smoke run:

| Metric | Budget (proposal) |
|---|---|
| CLI cold start to first output (`--help`, local config) | < 50 ms |
| `item move` / `item edit` with a warm cache | 1 network round trip for the write, ≤ 1 for the lookup |
| `board`: launch to first meaningful paint (warm cache) | < 100 ms (from snapshot, see P3) |
| `board`: launch to first live page (cold) | 1 round trip (see P4) |
| Keypress to frame, 1,000-item board | < 16 ms p95 |
| Keypress to frame, 10,000-item board | < 50 ms p95 |
| Memory for a 10,000-item board | < 150 MB RSS |

Also add synthetic **1k and 10k** fixtures to [ADR 0004](../decisions/0004-acceptance-fixtures-three-boards.md). At 100 items you won't find a single real performance bug.

### P2 — No cache, so every CLI mutation pays 4–5 sequential round trips *(critical)*
Here is what `boardwalk item move juanrubio/snazzyemail#42 --to Done` needs if nothing is cached:

1. Scope check ([ADR 0010](../decisions/0010-auth-scopes-upfront.md)). This is a separate call if implemented literally, and go-gh may also start a `gh auth token` subprocess.
2. Resolve alias → project node ID.
3. Fetch the field schema to get the field/option IDs.
4. Resolve the item reference within the project.
5. The mutation itself.

At about 300–600 ms per Projects v2 round trip, that is **1.5–3 s for the most common command**. Recommendations:

- **Schema cache** in `os.UserCacheDir()/boardwalk/`, keyed by project: project ID, field IDs, option IDs and colours, and iteration IDs. Revalidate lazily. If validation fails against the cache ("unknown option `Done`"), refetch the schema once before reporting exit 2. This is safe because it happens before any mutation.
- **Never do a separate scope round trip.** Read `X-OAuth-Scopes` from the first real response, and only classify as exit 3 when that response or an error says so. Fetch the token once per process.
- **Resolve `owner/repo#N` from the issue side** (`repository.issue(number).projectItems`). That is O(1). Scanning the board is O(n) and costs 100+ sequential pages on a large project.
- **Put the fields you need in the mutation's response** so the TUI does not need the separate refetch that §5 currently requires.

### P3 — Stale-while-revalidate board snapshot *(high; changes TUI §5)*
The biggest perceived-speed win available is to **paint the last board snapshot instantly**, then revalidate once with a visible "updated 3h ago · refreshing…" in the header. That is how the fastest TUIs feel instant. TUI §5 currently forbids "background magic" and says the screen paints from the network. Proposed amendment:

- One revalidation at startup and on `r`. Still **no polling**.
- **Mutations are disabled on a card until its live data has arrived.** This keeps the blind-write `H`/`L` move from overwriting someone else's change based on stale data.
- The cache contains only project data, never tokens. It should stay within PRD §6.5, but say so explicitly.

### P4 — First paint is two round trips where one would do *(high)*
§5 says "project schema and column headers first, then cards". Send **schema plus the first item page in a single GraphQL document**. Also:

- Use a smaller first page (for example 30–50) to paint sooner, then 100 per page after that. Tune this against live latency.
- **Never fetch bodies in list or board queries.** Load the body lazily when detail opens, and prefetch it for the focused card and its neighbours. Bodies are the heaviest payload.
- `fieldValues(first: N)`: request only the fields the cards and rows show. The detail view can fetch the rest.

### P5 — Use server-side filtering *(high)*
`ProjectV2.items` accepts a `query:` argument with the **same syntax as the web UI's filter bar** (`status:"In Progress" assignee:@me is:draft`). This changes the design in two ways:

- `item list --status/--assignee/--repo/--type/--search` can push filtering to the server instead of downloading the whole board on every scripted call.
- TUI `/` and CLI `--search` can share **one filter language that users already know from github.com**. That also fills the gap where ADR 0006 requires structured status/assignee/repo/type filtering in the TUI, but tui.md only defines a free-text `/`.

### P6 — Rendering must be virtualised *(high)*
Bubble Tea redraws `View()` as a whole string every frame. Styling 1,000+ cards with Lip Gloss each frame will miss the frame budget. The spec should require:

- Rendering only visible cards and rows.
- **Memoising rendered cards**, keyed by (item version, width, focused, theme).
- **Batching streamed pages** into one model update per page, not per item.
- Markdown bodies rendered with Glamour **off the UI loop** and cached per (item, width, theme). Glamour is slow enough to matter.

### P7 — Rapid `H`/`L` presses race *(high; correctness and speed)*
Pressing `L L L` sends three optimistic mutations. They can finish out of order, and "roll back on failure" has no defined meaning when the middle one fails. Specify **per-item move coalescing**: the card moves instantly on each press, and a single mutation goes out with the *final* target after about 250 ms idle. Requests are serialised per item, and the last write wins. This also:

- Makes non-adjacent jumps cheap, which removes the need to "go edit the field in detail" (§2) for Backlog → Done.
- Stays under GitHub's secondary rate limits on bursts of writes.

### P8 — Multi-field saves are non-atomic, and the spec does not say so *(medium)*
GitHub needs one mutation per field. Title/body also use different mutations for issues/PRs and for drafts. Send all of an item's mutations as **aliased fields in one GraphQL request**: one round trip, executed serially. Then specify **partial-failure UX** in TUI §4 and CLI `item edit`. Mark each change in the summary as ✓ applied or ✗ failed, and offer retry for only the failed ones.

---

## 2. Beauty: TUI

### B1 — No positive design system *(critical for the beauty goal)*
Add a "Visual language" section to tui.md that defines:

- **Semantic colour tokens:** `fg`, `muted`, `subtle`, `border`, `borderFocus`, `accent`, `success`, `warning`, `danger`, `selectionBg`. Each needs a light, dark, 256-colour, and monochrome value.
- **Spacing:** column gutter, card padding, and header/footer padding, all in cells.
- **Border vocabulary:** for example rounded for cards, none for columns, a thick or double border reserved for focus.
- **Typography roles:** title, card title, meta, key hint, error. Map each to a weight and token.

### B2 — Use GitHub's own option colours *(high, cheap, very visible)*
Single-select options in Projects v2 carry a `color` (GRAY, BLUE, GREEN, YELLOW, ORANGE, RED, PINK, PURPLE) and a `description`. Map them to a curated, adaptive palette for column headers and status/priority chips. The board then **looks like the user's web board**, with no configuration. Keep the never-colour-alone rule.

### B3 — Make "beautiful" testable *(high)*
The release gate tests usability, not looks. Add:

- **Golden-frame snapshot tests** with `teatest` for each fixture at 80×24, 120×40, and 200×50, in truecolour, 256-colour, and monochrome.
- **VHS tapes** that record the key flows as GIFs, reviewed by the owner before each release. They also double as README and demo material.

### B4 — Layout behaviours left open that cause visual jank *(high)*
- **Focus must not reflow.** §3 says the focused card has "a drawn border". If unfocused cards have no border, card heights change as focus moves. Require every card to reserve the border cells, for example with a hidden or `subtle` border.
- **Card density.** At 3 lines plus a border, a card takes 5 rows. An 80×24 terminal shows about 4 cards per column. Add a **compact density (1–2 lines) toggle** and remember it per project, like `group_by`.
- **Minimum column width and off-screen hints.** SnazzyEmail has 5 Status columns. At 80 columns that is about 16 characters each. Pin a minimum (about 26) and specify scroll indicators (`◂ 2` / `3 ▸`) plus per-column "↓ 12 more" hints. Currently nothing tells the user that content is off-screen.
- **Column placement and collapsing.** Specify where the "No ‹field›" column goes (suggest first or last, consistently). Allow **collapsing columns** (Done grows without bound): `z` collapses to a narrow strip showing the count.
- **Too-small terminal.** Define a minimum size and a styled "terminal too small" screen. Define resize behaviour, and terminal restore on panic.

### B5 — Detail pane should follow focus *(medium-high)*
§3 says board context is preserved, but not whether `j/k` keep moving on the board while the pane is open. Make the split pane a **live preview**, lazygit-style: navigation stays on the board, and the pane updates to the focused card. Pair this with lazy body prefetch (P4). This gives the "crafted" feel at almost no cost.

### B6 — Key-binding contradictions *(high)*
- §2 says "Arrow keys and hjkl are always both active; there is no mode where one set stops working." But §4 has text inputs (title, body, date, search) where `h j k l q e n` must type characters. Amend this: **a focused text input captures printable keys.** Field-to-field movement uses `Tab`/`Shift-Tab`/arrows, and `Esc` leaves the input.
- In edit mode, `Esc` is overloaded (leave field? leave form?). Define two levels. `Ctrl-C` with a dirty form or pending mutations must follow the discard rule as well.
- **Quitting with an in-flight optimistic move** is not specified. Flush with a short "saving 1 change…" or warn. Never exit silently with the outcome unknown.

### B7 — Missing TUI affordances *(medium)*
- **Archive** in the TUI is required by ADR 0006, but tui.md assigns it no key or flow.
- The **add-existing key** is still unassigned. Suggest `a`.
- `o` **open in browser** and `y` **copy reference/URL**. Expected escape hatches, and trivial to build.
- `u` **undo last move**. It is just another move, and makes optimistic moves feel safe.
- **`$EDITOR` handoff** for the body (`tea.ExecProcess`), as `gh` does. Bubbles' textarea is poor for long Markdown. Add `--body-file -` and `$EDITOR` to CLI `item edit`/`add` too.
- **Search semantics:** incremental as you type? Highlight matches? Hide or dim non-matches? Behaviour while pages are still loading ("matching 300/1,200 loaded")?
- **Per-card sync state:** a pending glyph while a mutation is in flight, and a failed state with the reason. §3 mentions "optimistic-move feedback" but does not define it.
- **Iteration and date fields** are a chance for visual richness: show `@current` or `in 3d` / `overdue 2d` relative labels, with text, not colour alone.

### B8 — Pin the Charm major version *(medium)*
ADR 0007 says major Charm upgrades are "the expected upgrade cost". **Start on Bubble Tea v2 and Lip Gloss v2** to avoid paying that cost immediately. They bring the new cell-diffing renderer, layer compositing (useful for the help overlay and modals), better key disambiguation (`Shift`+letter, which `H`/`L` relies on), and explicit background-colour detection (`tea.BackgroundColorMsg`). v1's implicit detection is a known source of light/dark misdetection and startup delays under tmux.

---

## 3. Beauty: CLI

[cli.md](../specs/cli.md) is all grammar and almost nothing about how output looks. Add a short "Human output" section:

- **Table style** for `item list` / `project list`: header weight, status chips using the option colours (B2), and truncation rules for TTYs.
- **Piped, non-JSON output:** untruncated and tab-separated, like `gh`. This is currently undefined.
- **Progress:** a spinner on **stderr only**, shown only after about 150–200 ms so fast commands never flicker. PRD §10 requires progress, and cli.md never mentions it.
- **Success lines:** a consistent shape, for example `✓ Moved snazzyemail#42  In review → Done`.
- **OSC 8 hyperlinks** on references and URLs where the terminal supports them, with a plain-text fallback.
- **Interactive prompts** (first-run selector, confirmations): use **Huh** from the Charm stack so CLI prompts share the TUI's look. Add it to ADR 0007.
- **Environment:** honour `NO_COLOR`, `CLICOLOR_FORCE`, and `TERM=dumb`. Consider `--color=auto|always|never`.

---

## 4. Correctness issues that also hurt UX

- **`draft:<prefix>` short IDs are broken.** cli.md §1 says to use "the shortest prefix of the item's node ID (≥ 8 chars)". Projects v2 item node IDs (`PVTI_lAD…`) share a long common prefix that encodes the type and project. The "shortest unambiguous prefix" would therefore be long and ugly. It also **changes as items are added**, so a reference that worked yesterday can become ambiguous. Use a short stable value instead, for example the item's numeric `databaseId` (`draft:91234567`), or a short hash.
- **`item move` depends on a TUI view preference.** cli.md says `move` targets the grouping field from tui.md §1, which includes the *remembered TUI choice*. Changing the grouping in the TUI would silently change what a scripted `item move --to X` writes. Have `move` always target Status (or the first single-select field) and add an explicit `--field`. `--status` on `item list` has the same ambiguity.
- **Exact-title and short-prefix references need a full board scan.** That is acceptable, but the spec should state the cost and use the cache or server-side `query:` (P5).
- **Documents disagree with each other:**
  - ADR 0006 says moves have "pending changes shown and applied on confirmation"; tui.md §2 says moves are instant with no confirmation.
  - ADR 0006 mentions "archive/remove"; cli.md says there is no remove.
  - ADR 0007 mentions a "drag/move interaction"; tui.md says there is no drag.
  - tui.md §6 contains a garbled clause: "when piped concepts are described".

---

## 5. Proposed amendments, in priority order

| # | Amendment | Docs touched | Goal |
|---|---|---|---|
| 1 | Add performance budgets plus 1k/10k synthetic fixtures | PRD §10, ADR 0004, new `docs/specs/performance.md` | Speed |
| 2 | Schema/ID cache, scope check from response headers, issue-side reference resolution | ADR 0002, ADR 0010, cli.md §1 | Speed |
| 3 | Stale-while-revalidate snapshot; combined first query; lazy bodies | tui.md §5 | Speed |
| 4 | Server-side `query:` filtering and one shared filter syntax | cli.md §2, tui.md §2 | Speed + UX |
| 5 | Move coalescing; batched multi-field writes with partial-failure UX | tui.md §2/§4, cli.md §2 | Speed + safety |
| 6 | Visual language section, GitHub option colours, golden/VHS tests | tui.md §3, cli.md §4 | Beauty |
| 7 | Layout fixes: no reflow on focus, density toggle, min width and scroll hints, column collapse | tui.md §3 | Beauty |
| 8 | Key-binding fixes for text inputs, quit with pending writes, archive/add/open/copy/undo keys, `$EDITOR` | tui.md §2/§4 | Beauty + UX |
| 9 | CLI human-output section; Huh for prompts; Charm v2 | cli.md, ADR 0007 | Beauty |
| 10 | Fix `draft:` short IDs, decouple `move` from the UI preference, reconcile cross-doc inconsistencies | cli.md §1/§2, ADR 0006/0007 | Correctness |

> [!TIP]
> Before writing more spec text, do a **one-day spike**: render the SnazzyEmail and Life Board fixtures, plus a synthetic 1k board, in Bubble Tea v2 at 80/120/200 columns. Time a live `item move`. That will settle card density, minimum column width, the split breakpoint, and the real round-trip cost. Those are the decisions most likely to need amending.
