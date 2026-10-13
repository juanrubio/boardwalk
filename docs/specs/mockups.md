# Boardwalk TUI & CLI Mockups

**Status:** Approved by the owner on 2026-10-13; revised the same day after the beauty review (true cell geometry, key-collision and reference fixes, missing states added)
**Scope:** Reference character-cell layouts for the TUI ([tui.md](tui.md)) and CLI ([cli.md](cli.md))

These mockups implement [tui.md](tui.md), [cli.md](cli.md), and [performance.md](performance.md) (Amendment 1). Every TUI frame is drawn at its stated cell size exactly — each line is padded to the declared width — with flush header/footer bars (no application frame), one-cell column gutters, rounded card borders, and semantic-token styling as specified. CLI blocks show natural output width at an 80-column terminal.

---
## 1. Full Kanban Board — Comfortable Density (120 × 30)

*Board:* `juanrubio/1` (SnazzyEmail) · *Grouping:* `Status` · *Focused card:* `juanrubio/snazzyemail#42` in "In Progress"

```text
Boardwalk · snazzy (juanrubio/1) · Grouped by Status · 14 items                        updated 4m ago · ? Help          
  Backlog (3)                          Ready (4)                            In Progress (3)                     2 cols ▸
 ╭──────────────────────────────────╮ ╭──────────────────────────────────╮ ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓         
 │  Migrate auth flow               │ │  Refactor parser                 │ ┃ ▸Export to CSV crash             ┃         
 │   issue juanrubio/snazzyemail#12 │ │   issue juanrubio/snazzyemail#33 │ ┃   issue juanrubio/snazzyemail#42 ┃         
 │   @juanrubio  ▲ P1 · in 5d       │ │   @juanrubio  ◆ P2 · in 3d       │ ┃   @juanrubio  ▲ P0 · in 2d       ┃         
 ╰──────────────────────────────────╯ ╰──────────────────────────────────╯ ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛         
 ╭──────────────────────────────────╮ ╭──────────────────────────────────╮ ╭──────────────────────────────────╮         
 │  Webhook retry queue             │ │  CLI autocomplete test           │ │  Bulk archive endpoint           │         
 │   issue juanrubio/snazzyemail#15 │ │   issue juanrubio/snazzyemail#38 │ │   issue juanrubio/snazzyemail#44 │         
 │   @juanrubio  ◆ P2               │ │   @collaborator ▲ P1             │ │   @juanrubio  ▲ P1 · in 4d       │         
 ╰──────────────────────────────────╯ ╰──────────────────────────────────╯ ╰──────────────────────────────────╯         
 ╭──────────────────────────────────╮ ╭──────────────────────────────────╮ ╭──────────────────────────────────╮         
 │  Set up CI golden tests          │ │  Schema cache spike              │ │  ◌ Sync worker race              │         
 │   issue juanrubio/snazzyemail#18 │ │   issue juanrubio/snazzyemail#39 │ │   issue juanrubio/snazzyemail#48 │         
 │   @collaborator ◆ P2             │ │   @juanrubio  ◆ P2               │ │   @juanrubio  ▲ P0               │         
 ╰──────────────────────────────────╯ ╰──────────────────────────────────╯ ╰──────────────────────────────────╯         
                                      ╭──────────────────────────────────╮                                              
                                      │  Config TOML docs                │                                              
                                      │   issue juanrubio/snazzyemail#41 │                                              
                                      │   @juanrubio  ◆ P2               │                                              
                                      ╰──────────────────────────────────╯                                              
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
↑↓/jk card · ←→/hl column · H/L move · Enter detail · n draft · a add · / filter · d density · q quit                   
```

> **Design notes:**
> - No application frame: the header and footer are flush one-line bars
>   (tui.md §3). Columns are 36 cells with a one-cell gutter; at this
>   width three columns fit in full, so "In Review" and "Done" are
>   off-screen to the right and the header row says so (`2 cols ▸`).
>   Nothing is ever clipped mid-card.
> - Rounded card borders (comfortable density); the focused card uses a
>   thick border plus `▸`, with the marker cell reserved on every card
>   so focus never shifts a title. Every card is exactly 5 lines tall.
> - References are copyable in full (`juanrubio/snazzyemail#42`);
>   dates always use the relative form (`in 2d`, `overdue 2d`).
> - `◌` on "Sync worker race" is the in-flight write marker.

---
## 2. Compact Density with Collapsed Column (80 × 24)

*Board:* `juanrubio/3` (Life Board) · *Density:* Compact (`d`) · *Column "Done" collapsed (`z`)*

```text
Boardwalk · life-board (juanrubio/3) · Status · 16 items · updated just now · ? 
 Todo (6)                        In Progress (4)                 Done (6) [z]   
 ▌Fix memory leak in parser       Set up GraphQL client          collapsed      
 ▌issue …#42 @juanrubio           draft:91204843 @juanrubio                     
  Prepare v1 release notes        Refactor Bubble Tea model                     
  draft:91204812 @juanrubio       draft:91204847 @juanrubio                     
  Add large-board fixture         ◌ Offline cache spike                         
  draft:91204855 @juanrubio       draft:91204901 @juanrubio                     
  Investigate export flow         Profile memory footprint                      
  draft:91204860 @juanrubio       draft:91204910 @juanrubio                     
  Test Windows console host                                                     
  draft:91204872 @juanrubio                                                     
  Water the plants                                                              
  draft:91204880 @juanrubio                                                     
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
↑↓ card · ←→ col · H/L move · z expand · d density · / filter · q quit          
```

> **Design notes:**
> - Compact cards are exactly 2 lines with no box and no blank
>   separators; the one-cell gutter (`▌` on both lines of the focused
>   card, blank otherwise) is the only focus geometry. References that
>   cannot fit in full at this width shorten to `…#42` on the card and
>   remain full in detail, list view, and on copy (`y`).
> - The collapsed "Done" strip is 15 cells and still shows its name
>   and count. All three Status columns are on screen, so there is no
>   off-screen hint; hints (`◂ 2`, `↓ 12 more`) appear only when
>   content is actually hidden. Every card fits at this height, so no
>   vertical hint is shown either.

---
## 3. Split Detail Pane — Live Preview (120 × 28)

*State:* Detail pane open in read mode; board keeps focus and `j`/`k` move between cards while the pane follows (tui.md §2). The board occupies the left 60% and the detail pane the right 40%.

```text
Boardwalk · snazzy (juanrubio/1) · Grouped by Status · 14 items                          updated 4m ago · ? Help        
  In Progress (3)                     In Review (2)                      │ Export to CSV crash                          
 ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓ ╭─────────────────────────────────╮ │ ─────────────────────────────────────────────
 ┃ ▸Export to CSV crash            ┃ │  Add dark mode toggle           │ │ Kind:     issue · juanrubio/snazzyemail#42   
 ┃   issue juanrubio/snazzyemail#42┃ │   issue juanrubio/snazzyemail#50│ │ Status:   ● In Progress                      
 ┃   @juanrubio ▲ P0 · in 2d       ┃ │   @collaborator ◆ P2            │ │ Priority: ▲ P0                               
 ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛ ╰─────────────────────────────────╯ │ Assignee: @juanrubio                         
 ╭─────────────────────────────────╮ ╭─────────────────────────────────╮ │ Size:     M                                  
 │  Bulk archive endpoint          │ │  OAuth refresh token            │ │ Estimate: 5                                  
 │   issue juanrubio/snazzyemail#44│ │   issue juanrubio/snazzyemail#52│ │ Target:   2026-10-15 (in 2d)                 
 │   @juanrubio ▲ P1 · in 4d       │ │   @juanrubio ▲ P0               │ │                                              
 ╰─────────────────────────────────╯ ╰─────────────────────────────────╯ │ When exporting more than 1,000 rows to CSV, …
 ╭─────────────────────────────────╮                                     │ worker thread exhausts available heap memory.
 │  ◌ Sync worker race             │                                     │                                              
 │   issue juanrubio/snazzyemail#48│                                     │ Steps to reproduce:                          
 │   @juanrubio ▲ P0               │                                     │   1. Go to Settings > Export                 
 ╰─────────────────────────────────╯                                     │   2. Select a date range over 6 months       
                                                                         │   3. Click Export                            
                                                                         │                                              
                                                                         │ Expected: a streamed download.               
                                                                         │ Actual: the worker dies with an out-of-memory
                                                                         │ error after roughly 40 seconds.              
                                                                         │                                              
                                                                         │ o browser · y copy ref · e edit · x archive  
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
↑↓/jk board · Tab pane · H/L move · e edit · x archive · Esc close pane                                                 
```

> **Design notes:**
> - True 60/40 geometry at 120 cells: two full board columns stay
>   visible beside the detail pane, so opening a card never loses
>   board context (tui.md §3).
> - The pane title is the item title; the full copyable reference
>   appears once, on the Kind line. Status and Priority pair a glyph
>   with their text value, never colour alone.
> - The body is Markdown rendered with Glamour from the theme tokens;
>   long bodies scroll inside the pane after `Tab`.

---
## 4. Change Summary with Partial Failure (120 × 25)

*State:* A multi-field save on `juanrubio/snazzyemail#42` has been sent as one request; GitHub applied three fields and rejected one. The board stays visible on the left.

```text
Boardwalk · snazzy (juanrubio/1) · Edit item                                              updated 4m ago · ? Help       
  In Progress (3)                     In Review (2)                      │ Review changes — juanrubio/snazzyemail#42    
 ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓ ╭─────────────────────────────────╮ │ ─────────────────────────────────────────────
 ┃ ▸Export to CSV crash            ┃ │  Add dark mode toggle           │ │ Applied (GitHub confirmed):                  
 ┃   issue juanrubio/snazzyemail#42┃ │   issue juanrubio/snazzyemail#50│ │   ✓ Status:    Ready → In Progress           
 ┃   @juanrubio ▲ P0 · in 2d       ┃ │   @collaborator ◆ P2            │ │   ✓ Priority:  P1 → P0                       
 ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛ ╰─────────────────────────────────╯ │   ✓ Target:    2026-10-20 → 2026-10-15       
 ╭─────────────────────────────────╮ ╭─────────────────────────────────╮ │                                              
 │  Bulk archive endpoint          │ │  OAuth refresh token            │ │ Failed (nothing else was changed):           
 │   issue juanrubio/snazzyemail#44│ │   issue juanrubio/snazzyemail#52│ │   ✗ Estimate:  3 → 5                         
 │   @juanrubio ▲ P1 · in 4d       │ │   @juanrubio ▲ P0               │ │     GitHub rejected the request              
 ╰─────────────────────────────────╯ ╰─────────────────────────────────╯ │     (FORBIDDEN: no write access to Estimate).
 ╭─────────────────────────────────╮                                     │                                              
 │  Sync worker race               │                                     │ Enter  retry failed changes only             
 │   issue juanrubio/snazzyemail#48│                                     │ e      back to the form (edits kept)         
 │   @juanrubio ▲ P0               │                                     │ Esc    back (asks before discarding)         
 ╰─────────────────────────────────╯                                     │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
Enter retry failed · e back to form · Esc back (asks before discarding)                                                 
```

> **Design notes:**
> - Saves are one GraphQL request but **not atomic** (tui.md §4):
>   applied lines stay applied, failed lines keep their edits in the
>   form. The summary keys deliberately avoid the board's `d`
>   (density) and `r` (refresh): `Enter` retries, `e` returns to the
>   form, `Esc` goes back and still asks before discarding anything
>   unapplied.

---
## 5. CLI Human Terminal Output — 80 columns (`boardwalk item list`)

*Command:* `boardwalk item list --status "In Progress" --assignee @me`. The full copyable reference always fits; `KIND` and `REPOSITORY` columns are added only when the terminal is wide enough (the reference already encodes both).

```text
REF                       STATUS        PRIORITY ASSIGNEE    TARGET
juanrubio/snazzyemail#42  In Progress   P0       @juanrubio  2026-10-15 (in 2d)
juanrubio/snazzyemail#44  In Progress   P1       @juanrubio  2026-10-19 (in 6d)
draft:91204910            In Progress   -        @juanrubio  -

Showing 3 of 14 items in 'snazzy' (juanrubio/1)
```

*Mutation output:*

```text
$ boardwalk item move juanrubio/snazzyemail#42 --to Done
✓ Moved juanrubio/snazzyemail#42  In Progress → Done
```

*Piped output (e.g. `boardwalk item list | cut -f1,4`) — TSV, no header, no truncation (cli.md §5):*

```text
juanrubio/snazzyemail#42	In Progress
juanrubio/snazzyemail#44	In Progress
draft:91204910	In Progress
```

---
## 6. List View (120 × 24)

*Toggle:* `v` from the kanban — focus, filters, and grouping carry across (tui.md §6).

```text
Boardwalk · snazzy (juanrubio/1) · List · Grouped by Status · 14 items                updated 4m ago · ? Help           
 REFERENCE                  KIND  TITLE                    STATUS         PRIORITY ASSIGNEE      TARGET                 
 juanrubio/snazzyemail#12   issue Migrate auth flow        ● Backlog      ▲ P1     @juanrubio    2026-10-18 (in 5d)     
 juanrubio/snazzyemail#15   issue Webhook retry queue      ● Backlog      ◆ P2     @juanrubio    -                      
 juanrubio/snazzyemail#33   issue Refactor parser          ● Ready        ◆ P2     @juanrubio    2026-10-16 (in 3d)     
▌juanrubio/snazzyemail#42   issue Export to CSV crash      ● In Progress  ▲ P0     @juanrubio    2026-10-15 (in 2d)     
 juanrubio/snazzyemail#44   issue Bulk archive endpoint    ● In Progress  ▲ P1     @juanrubio    2026-10-17 (in 4d)     
 draft:91204910             draft Profile memory footprint ● In Progress  -        @juanrubio    -                      
 juanrubio/snazzyemail#50   issue Add dark mode toggle     ● In Review    ◆ P2     @collaborator -                      
 juanrubio/snazzyemail#52   issue OAuth refresh token      ● In Review    ▲ P0     @juanrubio    2026-10-11 (overdue 2d)
  ↓ 4 more…                                                                                                             
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
                                                                                                                        
↑↓ row · Enter detail · H/L move · e edit · v kanban · / filter · q quit                                                
```

> **Design notes:**
> - Same row model as the CLI table (§5) with wider columns; ordering
>   is grouping-option order, then board order. The focused row keeps
>   the `▌` gutter so scanning never depends on highlight alone.
> - `H`/`L`, `Enter`, `e`, `n`, `a`, `/`, `g`, `r` behave exactly as
>   on the kanban — the list is not a second-class surface.

---
## 7. Edit Form with Dirty Fields (120 × 28)

*State:* `e` on the focused card. Three fields changed; nothing has been sent yet (tui.md §4).

```text
Boardwalk · snazzy (juanrubio/1) · Edit item                                              updated 4m ago · ? Help       
  In Progress (3)                     In Review (2)                      │ Edit — Export to CSV crash  (3 changes)      
 ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓ ╭─────────────────────────────────╮ │ ─────────────────────────────────────────────
 ┃ ▸Export to CSV crash            ┃ │  Add dark mode toggle           │ │ Title:     Export to CSV crash               
 ┃   issue juanrubio/snazzyemail#42┃ │   issue juanrubio/snazzyemail#50│ │ Status:    ● In Progress → Done ●            
 ┃   @juanrubio ▲ P0 · in 2d       ┃ │   @collaborator ◆ P2            │ │ Priority:  ▲ P0                              
 ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛ ╰─────────────────────────────────╯ │ Assignee:  @juanrubio                        
 ╭─────────────────────────────────╮ ╭─────────────────────────────────╮ │ Size:      M                                 
 │  Bulk archive endpoint          │ │  OAuth refresh token            │ │ Estimate:  5 → 8 ●                           
 │   issue juanrubio/snazzyemail#44│ │   issue juanrubio/snazzyemail#52│ │ Target:    2026-10-20 → 2026-10-15 ●         
 │   @juanrubio ▲ P1 · in 4d       │ │   @juanrubio ▲ P0               │ │                                              
 ╰─────────────────────────────────╯ ╰─────────────────────────────────╯ │ Body (Ctrl-E opens $EDITOR):                 
 ╭─────────────────────────────────╮                                     │   When exporting more than 1,000 rows to CSV,
 │  Sync worker race               │                                     │   the worker thread exhausts available heap… 
 │   issue juanrubio/snazzyemail#48│                                     │                                              
 │   @juanrubio ▲ P0               │                                     │ ● = changed · Ctrl-S review & save · Esc back
 ╰─────────────────────────────────╯                                     │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
                                                                         │                                              
Tab/↑↓ field · Enter pick · Ctrl-S review & save · Ctrl-E editor · Esc back (asks if dirty)                             
```

> **Design notes:**
> - Changed fields show `old → new ●` and the header counts them;
>   untouched fields are never rewritten. `Ctrl-S` opens the §4
>   change summary — the form itself never sends anything.
> - Single-select/iteration/people fields open pickers from the live
>   schema; date/number inputs validate in place before they can be
>   left. Read-only fields would be named with their reason, never
>   hidden.

---
## 8. Help Overlay and Active Search (80 × 24)

*State:* `/export` filter active (match count in the header), `?` help open as a rounded modal over the dimmed board (tui.md §2–§3).

```text
Boardwalk · snazzy · /export · matching 3 of 14 loaded · updated 4m ago · ?     
  Export to CSV crash        Investigate export flow                            
  … board dimmed behind the overlay …                                           
                  ╭─ Keys ────────────────────────────────────╮                 
                  │ ↑↓/jk card     ←→/hl column  H/L move     │                 
                  │ Enter detail   Tab pane      e edit       │                 
                  │ n draft        a add issue   x archive    │                 
                  │ / filter       g grouping    v list/board │                 
                  │ z collapse     d density     r refresh    │                 
                  │ o browser      y/Y copy      u undo move  │                 
                  │ ? this help    q quit        Esc back     │                 
                  ╰───────────────────────────────────────────╯                 
    (matches stay highlighted on the board behind)                              
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
? close help · Esc clear filter · Enter keep filter                             
```

> **Design notes:** the overlay never covers the footer, so the way
> out is always visible. Search matches are weighted/underlined in
> card titles, not colour alone, and the count reads `matching N of
> M loaded` until loading finishes — never a silently partial answer.

---
## 9. Archive Confirmation (80 × 16)

*State:* `x` on the focused card. The same rounded-modal pattern hosts the grouping picker (`g`) and single-select pickers.

```text
Boardwalk · snazzy · Status · 14 items · updated 4m ago · ?                     
                                                                                
              ╭─ Archive item? ─────────────────────────╮                       
              │ juanrubio/snazzyemail#42               │                        
              │ issue · Export to CSV crash            │                        
              │                                        │                        
              │ It leaves the board; the issue itself  │                        
              │ stays open on GitHub.                  │                        
              │                                        │                        
              │        [y] Archive    [any key] Cancel │                        
              ╰────────────────────────────────────────╯                        
                                                                                
                                                                                
                                                                                
                                                                                
y archive (not optimistic) · any other key cancels                              
```

> **Design notes:** the confirmation names the item by reference,
> kind, and title, and says what does *not* happen (the issue stays
> open). Archive is not optimistic: the card leaves only after GitHub
> confirms (tui.md §2).

---
## 10. Terminal Too Small (50 × 12) and Monochrome (80 × 12)

Below 60×15 nothing renders except the centred size message (tui.md §3):

```text
                                                  
                                                  
            Terminal too small                    
        Boardwalk needs 60 × 15 cells             
          this terminal is 50 × 12                
                                                  
         Resize to continue · q quit              
                                                  
                                                  
                                                  
                                                  
                                                  
```

With `NO_COLOR` (or a monochrome profile), the same board region relies on weight, borders, and markers only — no meaning is lost:

```text
Boardwalk · snazzy (juanrubio/1) · Grouped by Status · 14 items · updated 4m ago
 Backlog (3)                     In Progress (3)          2 cols ▸              
 Migrate auth flow               ▸Export to CSV crash                           
 issue …#12 @juanrubio P1 in 5d    issue …#42 @juanrubio P0 in 2d               
 Webhook retry queue             Bulk archive endpoint                          
 issue …#15 @juanrubio P2          issue …#44 @juanrubio P1 in 4d               
 Set up CI golden tests          Sync worker race                               
 issue …#18 @collaborator P2       issue …#48 @juanrubio P0                     
                                                                                
                                                                                
                                                                                
jk card · hl column · H/L move · Enter detail · q quit                          
```
> **Design notes:** every frame in this file must survive the
> monochrome and light-theme golden renders in tui.md §7 unchanged in
> meaning; colour is an enhancement layer over text and geometry,
> never the carrier of status.
