# Boardwalk TUI & CLI Mockups

**Status:** Approved by the owner on 2026-10-13
**Scope:** Reference character-cell layouts for the TUI ([tui.md](tui.md)) and CLI ([cli.md](cli.md))

These mockups reflect the exact specifications in [tui.md](tui.md), [cli.md](cli.md), and [performance.md](performance.md) (Amendment 1).

All views assume a monospace terminal font and adhere to fixed cell coordinates, box-drawing characters, and semantic token styling.

---

## 1. Full Kanban Board — Comfortable Density (120 × 30)

*Board:* `juanrubio/1` (SnazzyEmail) · *Grouping:* `Status` · *Focused card:* `snazzyemail#42` in "In Progress"

```text
┌ Boardwalk · snazzy (juanrubio/1) · Grouped by: Status · 14 items ─────────────── updated 4m ago ─ [?] Help ┐
│                                                                                                             │
│  Backlog (3)             Ready (4)               In Progress (3)         In Review (2)           Done (2)   │
│ ┌──────────────────────┐┌──────────────────────┐┌──────────────────────┐┌──────────────────────┐┌──────────│
│ │ Migrate auth flow    ││ Refactor parser      ││▸Export to CSV crash  ││ Add dark mode toggle ││ Fix typos │
│ │ issue snazzyemail#12 ││ issue snazzyemail#33 ││ issue snazzyemail#42 ││ issue snazzyemail#50 ││ issue #10 │
│ │ @juanrubio P1 5d     ││ @juanrubio P2 3d     ││ @juanrubio P0 (in 2d)││ @collaborator P2     ││ @juanrubio│
│ └──────────────────────┘└──────────────────────┘└──────────────────────┘└──────────────────────┘└──────────│
│ ┌──────────────────────┐┌──────────────────────┐┌──────────────────────┐┌──────────────────────┐            │
│ │ Webhook retry queue  ││ CLI autocomplete test││ Bulk archive endpoint││ OAuth refresh token  │            │
│ │ issue snazzyemail#15 ││ issue snazzyemail#38 ││ issue snazzyemail#44 ││ issue snazzyemail#52 │            │
│ │ @juanrubio P2        ││ @collaborator P1     ││ @juanrubio P1 4d     ││ @juanrubio P0        │            │
│ └──────────────────────┘└──────────────────────┘└──────────────────────┘└──────────────────────┘            │
│ ┌──────────────────────┐┌──────────────────────┐┌──────────────────────┐                                    │
│ │ Setup CI golden tests││ Schema cache spike   ││ ◌ Sync worker race   │                                    │
│ │ issue snazzyemail#18 ││ issue snazzyemail#39 ││ issue snazzyemail#48 │                                    │
│ │ @collaborator P3     ││ @juanrubio P2        ││ @juanrubio P0        │                                    │
│ └──────────────────────┘└──────────────────────┘└──────────────────────┘                                    │
│                         ┌──────────────────────┐                                                            │
│                         │ Config TOML docs     │                                                            │
│                         │ issue snazzyemail#41 │                                                            │
│                         │ @juanrubio P3        │                                                            │
│                         │                      │                                                            │
│                         └──────────────────────┘                                                            │
│                                                                                                             │
└ [↑↓/jk] Card  [←→/hl] Column  [H/L] Move  [Enter] Detail  [n] Draft  [a] Add  [/] Filter  [d] Density  [q] Quit ┘
```

> **Design notes:**
> - Header carries project alias, full reference, grouping field, item count, and data age (`updated 4m ago`).
> - Focused card has thick/distinct border (`▸`) and focus marker without shifting geometry.
> - Card 3 in "In Progress" shows in-flight sync glyph (`◌`).
> - Unfocused cards maintain subtle border reserving space to avoid reflow.

---

## 2. Compact Density with Collapsed Column & Off-screen Hints (80 × 24)

*Width:* 80 columns · *Density:* Compact (`d`) · *Column "Done" collapsed (`z`)* · *Scroll indicator active*

```text
┌ Boardwalk · life-board · Status · 16 items · updated just now ───────── [?] ┐
│◂ 1 col   Todo (6)              In Progress (4)         Done (6) [z]         │
│▌Fix memory leak in parser     Setup GraphQL client    │ (6 items)           │
│ issue snazzyemail#42 @juan    issue snazzyemail#43    │                     │
│                                                       │                     │
│ Prepare v1 release notes      Refactor Bubble Tea v2  │                     │
│ draft:91204812 @juan          issue snazzyemail#47    │                     │
│                                                       │                     │
│ Add 10k synthetic fixture     ◌ Offline cache spike   │                     │
│ draft:91204855 @juan          draft:91204901 @juan    │                     │
│                                                       │                     │
│ Investigate export flow       Profile memory RSS      │                     │
│ draft:91204860 @me            draft:91204910 @me      │                     │
│                                                       │                     │
│ Test Windows conhost          ↓ 2 more…               │                     │
│ draft:91204872                                        │                     │
│                                                       │                     │
│ ↓ 1 more…                                             │                     │
└ [↑↓] Card [←→] Col [H/L] Move [z] Expand [d] Density [/] Filter [q] Quit ────┘
```

> **Design notes:**
> - Density toggle (`d`) reduces cards to 2 lines with a left indicator gutter (`▌`), fitting roughly double the items.
> - "Done" column is collapsed (`z`), taking only 15 cells while preserving count.
> - `◂ 1 col` indicates off-screen column to the left; `↓ 2 more…` indicates cards below vertical fold.

---

## 3. Split Detail Pane — Live Preview Mode (120 × 28)

*State:* Detail pane open (`Enter` or `Tab`), split view at ~40% width. Board on left keeps focus during `j`/`k`.

```text
┌ Boardwalk · snazzy (juanrubio/1) · Status · 14 items ──────────── [?] Help ─┐
│  In Progress (3)         │ snazzyemail#42 · Export to CSV crash             │
│ ┌──────────────────────┐ │ ──────────────────────────────────────────────── │
│ │▸Export to CSV crash  │ │ Kind:     Issue · juanrubio/snazzyemail#42       │
│ │ issue snazzyemail#42 │ │ Status:   ● In Progress                          │
│ │ @juanrubio P0 (in 2d)│ │ Priority: ▲ P0 (Critical)                        │
│ └──────────────────────┘ │ Assignee: @juanrubio                             │
│ ┌──────────────────────┐ │ Size:     M                                      │
│ │ Bulk archive endpoint│ │ Estimate: 5                                      │
│ │ issue snazzyemail#44 │ │ Target:   2026-10-15 (in 2d)                     │
│ │ @juanrubio P1 4d     │ │                                                  │
│ └──────────────────────┘ │ Description (Markdown):                          │
│ ┌──────────────────────┐ │ When exporting more than 1,000 rows to CSV,      │
│ │ ◌ Sync worker race   │ │ the worker thread exhausts available heap memory.│
│ │ issue snazzyemail#48 │ │                                                  │
│ │ @juanrubio P0        │ │ Steps to reproduce:                              │
│ └──────────────────────┘ │ 1. Go to settings > export                       │
│                          │ 2. Select date range > 6 months                  │
│                          │ 3. Click export                                  │
│                          │                                                  │
│                          │ [o] Browser  [y] Copy ref  [e] Edit  [x] Archive │
└ [↑↓/jk] Board [Tab] Pane [H/L] Move [e] Edit [x] Archive [Esc] Close ───────┘
```

> **Design notes:**
> - Detail pane displays live schema fields with badges and Markdown formatting (rendered via Glamour).
> - Pressing `j`/`k` on board updates detail pane instantly via prefetch.
> - Status and Priority display text labels paired with glyphs (`●`, `▲`).

---

## 4. Edit Form in Detail Pane (Partial Failure / Change Summary)

*State:* Editing fields on `snazzyemail#42`. Multi-field save submitted, reporting one-request batched outcome.

```text
┌ Boardwalk · snazzy (juanrubio/1) · Edit Item ──────────────────── [?] Help ─┐
│                          │ snazzyemail#42 · Review Changes (1 failed)       │
│                          │ ──────────────────────────────────────────────── │
│                          │ Applied changes:                                 │
│                          │   ✓ Status:    Ready → In Progress               │
│                          │   ✓ Priority:  P1 → P0                           │
│                          │   ✓ Target:    2026-10-20 → 2026-10-15           │
│                          │                                                  │
│                          │ Failed changes:                                  │
│                          │   ✗ Estimate:  3 → 5                             │
│                          │     Reason: Field value exceeds permitted bounds │
│                          │                                                  │
│                          │ Actions:                                         │
│                          │   [r] Retry failed changes only                  │
│                          │   [e] Back to form to edit invalid fields        │
│                          │   [d] Discard unapplied changes                  │
│                          │                                                  │
│                          │                                                  │
│                          │                                                  │
│                          │                                                  │
└ [r] Retry failed  [e] Edit form  [d] Discard failed  [Esc] Back ────────────┘
```

> **Design notes:**
> - Multi-field saves batch into 1 GraphQL query; non-atomic partial failures clearly show `✓` and `✗`.
> - User can retry only failed fields without reverting applied ones.

---

## 5. CLI Human Terminal Output (`boardwalk item list`)

*Command:* `boardwalk item list --status "In Progress" --assignee @me`

```text
REF                    KIND   REPOSITORY             STATUS          PRIORITY  ASSIGNEE    TARGET
juanrubio/snazzy#42    issue  juanrubio/snazzyemail  ● In Progress   ▲ P0      @juanrubio  2026-10-15 (in 2d)
juanrubio/snazzy#44    issue  juanrubio/snazzyemail  ● In Progress   ◆ P1      @juanrubio  2026-10-19 (in 6d)
draft:91204910         draft  -                      ● In Progress   -         @juanrubio  -

Showing 3 of 14 items in 'snazzy' (juanrubio/1)
```

*Mutation Output:*
```text
$ boardwalk item move juanrubio/snazzyemail#42 --to Done
✓ Moved juanrubio/snazzyemail#42  In Progress → Done
```

*Piped Output (e.g. `boardwalk item list | cut -f1,4`):*
```text
juanrubio/snazzyemail#42	In Progress
juanrubio/snazzyemail#44	In Progress
draft:91204910	In Progress
```
