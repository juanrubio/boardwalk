# Boardwalk CLI specification

**Status:** Final — amended by A1 and A2 (2026-10-13)
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Amended:** 2026-10-13 — A1, owner approval of the
[beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md);
A2, mockup beauty-review fixes ([mockups.md](mockups.md)).
Amended passages are marked *(A1)* / *(A2)*; see the amendment log at the end.
**Date:** 2026-10-13
**Scope:** The noun-first command surface (`project`, `item`, `board`)
from [ADR 0003](../decisions/0003-command-grammar-noun-first.md)
(Final), on Cobra ([ADR 0007](../decisions/0007-go-stack-cobra-charm.md),
Final), with the JSON/error contract of
[ADR 0009](../decisions/0009-json-error-contract-bare.md) (Final).
Decisions are recorded here as they settle in the decision interview;
unresolved sections stay visibly marked. The TUI has its own
specification ([tui.md](tui.md), Final).

## 1. Item references (settled 2026-10-13)

Commands that act on an existing item (`item edit`, `item move`, and
their successors) take one item reference. A reference is resolved
**within the selected project only** (ADR 0005 resolution order) and
accepts these forms:

1. **Linked-item reference:** `owner/repo#number` for a linked issue or
   pull request (for example `juanrubio/snazzyemail#42`, PRD §7).
2. **Item URL:** the GitHub URL of the issue/PR, or of the project item
   where GitHub exposes one.
3. **Item ID:** the item's node ID in full, or the short ID displayed
   for it (below). This is the only universal form — drafts have no
   `owner/repo#number`.
4. **Exact title:** the item's exact title, as a convenience form. It
   counts only when it matches exactly one item in the project.

Display rule: every list, board, and detail rendering shows the
reference a user can copy back into a command — the
`owner/repo#number` for linked items, and for drafts a short ID of the
form `draft:<id>`. *(A1)* `<id>` is the project item's numeric
`databaseId` (for example `draft:91234567`). It is short, typeable, and
**stable for the item's lifetime**. It replaces the earlier "shortest
unambiguous node-ID prefix" rule, which could not work: Projects v2
node IDs share a long common prefix, and the shortest unique prefix
changes as items are added. Full node IDs are shown in `--json` output
and item detail.

Resolution rules:

- No match: exit 4 (not found, ADR 0009) naming the reference and the
  project searched. An `owner/repo#number` that exists on GitHub but
  is **not on the selected project** is also not-found here — edit/move
  commands never implicitly add items; adding is `item add`'s job.
- More than one match (possible for titles *(A1: short IDs are now
  exact, so they cannot be ambiguous)*):
  exit 2 (usage) listing each candidate's reference, kind, and title,
  and **change nothing**. Ambiguity never guesses, including for
  mutations.
- Title matching is exact and case-sensitive; partial titles are
  search (`item list --search` / TUI `/`), never references.
- References are stable across sessions for linked items; node IDs
  *(A1)* and `draft:<databaseId>` short IDs are stable for all items.
  Row numbers or screen positions are never references.
- *(A1)* **Resolution cost** (performance.md §2):
  - Forms 1–2 resolve from the issue/PR side, through its
    `projectItems` filtered to the selected project. That is one
    request, whatever the board size.
  - Item IDs resolve with a direct node lookup. `draft:<id>` resolves
    through the cached `databaseId` → node ID map (ADR 0011), with a
    board scan only on a cache miss.
  - Exact titles are the only form that always needs the item list.
    They use the server-side item query where it can narrow the result.
    The command documents that this form is slower on large boards.

`item add` takes the same linked-item forms (1–2) for the thing being
added, or `--draft --title …` for a new draft (PRD §7); it never takes
a title reference for an existing item.

Rejected for this section: node IDs only (precise but untypeable,
forcing list-copy-paste for every edit) and row numbers from a prior
listing (meaning rots as soon as the board or filter changes — unsafe
for mutations).

## 2. Verb and flag inventory (settled 2026-10-13)

The MVP surface is deliberately small. `move` is the only
special-cased verb; every other field change goes through the generic
`edit --field` mechanism (PRD §7).

### `project`

| Command | Behaviour |
|---|---|
| `boardwalk project list [--owner <login>]` | List accessible Projects v2 (user- and org-owned), showing number, title, owner, item count, and which alias/default each matches. |
| `boardwalk project use <ref>` | Resolve `<ref>` (alias, URL, or owner/number), save it as an alias and/or default in user config (ADR 0005/0008) after confirmation of what will be written. |

### `item`

| Command | Behaviour |
|---|---|
| `boardwalk item list` | Table of the selected project's items in the TUI list-view row model (tui.md §6): reference (§1), title, kind, repository, grouping/status value, assignee, then salient fields. Filters: `--status <option>`, `--assignee <login\|@me>`, `--repo <owner/repo>`, `--type <issue\|pr\|draft>`, `--search <text>`. Pagination is transparent (PRD §6.2); ordering matches the TUI list (grouping order, then board order). *(A1)* `--status` filters on the project's **default board field** (defined below), never on the TUI grouping preference. `--search` accepts free text plus **GitHub's project filter syntax** (for example `assignee:@me is:draft "label:needs triage"`), the same language as the TUI `/` (tui.md §2). All filters are combined into one query. Whatever GitHub can evaluate is sent as the server-side item query, so only matching items are downloaded (performance.md B9). The rest is filtered locally, with identical results. |
| `boardwalk item add <issue/PR ref>` | Add an existing issue/PR (§1 forms 1–2) to the selected project. Previews the resolved item, then adds and prints the resulting item reference, ID, and link (PRD §8 flow B). Already-present items are reported, not duplicated. |
| `boardwalk item add --draft --title <t> [--body <b> \| --body-file <path\|-> \| --editor]` | Create a draft item. Title is required and non-empty. *(A1)* `--body-file -` reads the body from stdin. `--editor` opens `$VISUAL`/`$EDITOR` to write it. The three body sources are mutually exclusive. |
| `boardwalk item edit <ref> [--title <t>] [--body <b> \| --body-file <path\|-> \| --editor] [--field "Name=Value"]…` | Edit title/body where GitHub permits, and set project fields via repeatable `--field`. Field names and values validate against the live schema before any mutation; unknown fields, unknown single-select options, and invalid date/number values fail with exit 2 and the valid choices listed. Interactive use shows the same change summary (`Field: old → new`) and confirmation as the TUI (tui.md §4); `--dry-run` prints that summary without applying (ADR 0009). *(A1)* `--editor` opens the current body in `$VISUAL`/`$EDITOR`. All changes go out in one request (performance.md §2). Because GitHub applies them one at a time, the result marks each line `✓ applied` or `✗ failed`. Any failure exits with that failure's ADR 0009 code, and `--json` reports per-field status. |
| `boardwalk item move <ref> --to <option> [--field <name>]` | *(A1)* Set the item's **default board field** to `<option>`. The default board field is the single-select field named `Status`, or else the project's first single-select field (tui.md §1 rules 2–3). It **never** depends on the remembered TUI grouping preference, so a script's meaning cannot change when someone regroups the board. `--field` names another single-select field explicitly. The option validates against that field's live options; an invalid option fails with exit 2 listing the valid options. Prints `Field: old → new`. This verb exists because moving is the most common board action; it must behave exactly as `item edit --field "<Field>=<option>"` would, with no divergent semantics. |
| `boardwalk item archive <ref>` | Archive the item. Requires explicit confirmation interactively; scripts must pass `--yes`. Reversible on GitHub; prefer it over `delete` for "off my board". |
| `boardwalk item delete <ref>` | *(ADR 0012)* Permanently delete the project item. A linked issue/PR itself is untouched; a draft's content is permanently gone. Requires **double confirmation** interactively (two successive prompts naming the item); scripts must pass `--yes`; `--dry-run` prints the plan and sends nothing. |
| `boardwalk item convert <ref> [--repo <owner/repo>]` | *(A7)* Convert a draft item into a real issue in the given repository, keeping the item's board position and field values. Only drafts convert; anything else fails with exit 2. `--repo` may be omitted when every linked item on the board uses one repository (that repo is used and named in the confirmation); otherwise it is required (exit 2). Confirmation, `--yes`, `--dry-run`, and `--json` behave as for other mutations. Prints the new issue reference. |

### `board`

| Command | Behaviour |
|---|---|
| `boardwalk board [--project <ref>] [--list]` | Open the TUI (tui.md) for the selected project; `--list` starts in list view instead of kanban where the project permits both. |

### Rules applying to the whole inventory

- Every mutation previews its effect interactively and honours
  `--dry-run` (print the plan, change nothing) and `--yes` (skip the
  confirmation the user explicitly opted out of) per ADR 0009 and
  tui.md §4. `--yes` never bypasses validation or ambiguity checks
  (§1).
- No other verbs exist in the MVP. In particular there are no
  `item assign` / `item close` / `item remove` verbs: assignees and
  state are fields/items concerns handled through `edit --field` where
  the API permits, and closing an issue/PR itself is GitHub Issues
  territory, out of scope (PRD §6).
- New verbs require usage evidence and a spec amendment; aliases for
  the verbs above follow ADR 0003 (deferred until usage earns them).

Rejected for this section: omitting `move` (the most frequent action
would pay full generic `--field` syntax every time, asymmetric with
the TUI's `H`/`L`) and a wider day-one verb set (`remove`, `assign`,
`close`, …) that grows the surface — including its destructive-safety
design — before any usage data justifies it.

## 3. Global flags and script behaviour (settled 2026-10-13)

Four flags are **persistent globals**, accepted on every command
(Cobra persistent flags; position-independent for the user, so
`boardwalk item list --json` and `boardwalk --json item list` behave
identically). *(A1)* A fifth, `--no-cache`, was added:

| Flag | Meaning |
|---|---|
| `--project <ref>` | Select the project (alias, URL, or owner/number), overriding `BOARDWALK_PROJECT` and the saved default (ADR 0005). |
| `--json` | Bare JSON on stdout, JSON error object on stderr (ADR 0009). |
| `--dry-run` | Print the planned mutation/change summary; change nothing. Valid on every mutation; harmless elsewhere. |
| `--yes` | Skip interactive confirmation for a mutation the user has explicitly authorised. Never skips validation or ambiguity checks (§1). |
| `--no-cache` | *(A1)* Bypass the local schema cache and board snapshot for this run ([ADR 0011](../decisions/0011-local-cache-and-snapshot.md)); `BOARDWALK_NO_CACHE=1` is equivalent. |

Script-safety rules:

- Confirmation prompts are shown **only on a real terminal**. When
  stdin/stdout is not a terminal, a mutation without `--yes` prints
  its change summary and refuses with **exit 2**, naming both escapes:
  `--yes` to proceed, `--dry-run` to preview. It must never block
  reading input that cannot arrive, and never proceed unconfirmed.
- `--dry-run` takes precedence over `--yes` if both are passed:
  nothing is applied, and the output says the run was a dry run.
- Non-mutating commands never prompt; in scripts they simply work,
  with decoration off when piped (ADR 0009).
- There is **no `--non-interactive` flag**: TTY detection plus
  `--yes`/`--dry-run` already express script behaviour, and a second
  overlapping mechanism would eventually disagree with them.
- Standard Cobra behaviour applies everywhere else: `--help` on every
  command and subcommand, and shell completion for bash, zsh, fish,
  and PowerShell (PRD §7).

Rejected for this section: per-command flag definitions (scripts would
have to memorise which commands accept `--json`/`--project`, and the
surface would drift) and an explicit `--non-interactive` mode
(redundant with TTY detection plus `--yes`/`--dry-run`).

## 4. Help and completion validation (settled 2026-10-13)

Help and shell completion are part of the shipped surface, and they
are validated like code — this is the PRD §7 usability requirement
made concrete. The MVP is not ready until both halves pass.

### Human task check (release gate)

- **Participants:** the owner plus two or three target users (people
  who use GitHub Projects from the terminal). Not the implementer
  alone.
- **Tasks** (against the ADR 0004 fixture boards, using only
  `--help`, completion, and the TUI's `?`/footer — no other docs, no
  coaching):
  1. First run: select a project and save it as the default.
  2. Add an existing issue to the project.
  3. Move an item to another status/column from the CLI.
  4. Edit a non-status field (for example Priority) from the CLI.
  5. In the TUI: find an item by search, open it, and move it with
     `H`/`L`.
- **Gate:** every participant completes every task without outside
  help. A stumble is logged as a wording/grammar/help bug against this
  specification and fixed; it is never recorded as user error. Fixes
  that change command grammar require amending §2 and re-running the
  affected task.
- Timing is observed (the PRD §5 five-minute first-run measure uses
  task 1) but completion without help is the pass/fail line.

### Automated checks (every change)

- Every example shown in any `--help` output is executed in tests
  against the ADR 0004 fixtures and must succeed with the documented
  exit code (ADR 0009). Help text is generated from the same command
  definitions that run, so examples cannot drift from behaviour.
- Generated shell completions for bash, zsh, fish, and PowerShell are
  smoke-tested (they install and complete the §2 verbs, §1 references
  where dynamic completion is offered, and the §3 global flags).
- `--help` for every command states its purpose, its reference/flag
  syntax, at least one example, and its applicable exit codes.

Rejected for this section: automated checks only, with human
usability deferred to post-release feedback (the first grammar test
would happen on users who never volunteered), and deferring
validation entirely (quietly deletes PRD §7).

## 5. Human output and styling *(A1, 2026-10-13)*

To meet the beautiful CLI goal, human-readable terminal output follows
a consistent visual standard across all commands (ADR 0007).

### Tables and listings

- `boardwalk item list` and `project list` format rows as an aligned,
  space-padded table on TTYs. Header text is bold (`fg`). Status and
  single-select fields render with option tint chips matching GitHub's
  palette (tui.md §3).
- **Truncation on TTY:** columns truncate with `…` to fit terminal width
  without hard wrapping onto subsequent lines. *(A2)* The reference
  column is **never truncated**: it must stay copyable back into a
  command (§1). At narrow widths (80 columns) the `KIND` and
  `REPOSITORY` columns are dropped first — the reference already encodes
  both — before any other column truncates (see
  [mockups.md](mockups.md) §5).
- **Piped output (stdout not a TTY, without `--json`):** tab-separated
  values (TSV), no headers, no truncation, no color codes, compatible
  with standard Unix tools (`awk`, `cut`, `grep`), matching `gh`'s
  piping conventions (ADR 0009).

### Progress and spinners

- Network operations display a subtle spinner on **stderr only**
  (never polluting stdout or piped output).
- **Debounced display:** the spinner appears only if the operation takes
  longer than **150 ms**, preventing visual flicker on fast cached reads
  or fast connections (performance.md B1–B3).

### Success lines and feedback

- Mutations print a structured, single-line summary on success:
  `✓ Moved <ref>  <old_value> → <new_value>` or
  `✓ Created draft <ref> "<title>"`.
- Errors and warnings print to **stderr** with `✗` or `!`, using the
  remediation format defined in ADR 0009 and ADR 0010.

### Hyperlinks (OSC 8)

- On terminals supporting OSC 8 hyperlink sequences, issue/PR references
  and URLs are emitted as clickable links pointing to the GitHub web
  item, with the visible text formatted as `owner/repo#number`. Terminals
  without OSC 8 support receive plain text without escape sequences.

### Interactive prompts

- Interactive prompts on TTYs (first-run project selection, confirmation
  prompts for archive or multi-field edits) use **Huh** (ADR 0007),
  sharing the exact typography, borders, and color tokens of the TUI theme.

### Color and environment

- Follows the CLI color standard:
  - If `NO_COLOR` is set (regardless of value), all color and style escapes
    are omitted.
  - If `CLICOLOR_FORCE` is set to a non-zero value, styling is emitted
    even when stdout/stderr is redirected.
  - If `TERM=dumb`, styling is disabled.
- Global flag `--color=auto|always|never` overrides environment detection.

## Specification status

All sections of this specification were settled in the decision
interview and accepted as **Final** by the owner on 2026-10-13.
Changes from here require a spec amendment, not a silent edit.

## Amendment log

- **A1 (2026-10-13)** — owner approval of the
  [beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md):
  - §1: `draft:<id>` uses stable numeric `databaseId`; resolution cost
    rules added (issue-side O(1) lookup, cached databaseId map, server-side query).
  - §2: `item list` server-side query filtering and `--status` on default
    board field; `--body-file` and `--editor` options for `add`/`edit`;
    one-request batched saves with per-field status; `item move` decoupled
    from TUI grouping preference and accepts `--field`.
  - §3: `--no-cache` added as fifth persistent global flag (ADR 0011).
  - §5: new — human output standards, table truncation, piped TSV format,
    debounced stderr progress spinner, OSC 8 hyperlinks, Huh prompts,
    and NO_COLOR / `--color` handling.
- **A2 (2026-10-13)** — mockup beauty-review fixes:
  - §5: the reference column never truncates; at 80 columns `KIND` and
    `REPOSITORY` are dropped before anything else truncates.
