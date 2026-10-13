# Human task gate — session run-sheet (cli.md §4)

**Gate:** the owner plus two or three target users (people who use GitHub
Projects from the terminal), each completing all five tasks using only
`--help`, shell completion, and the TUI footer/`?` overlay. No other docs,
no coaching. A stumble is a wording/grammar/help bug against the specs —
never user error — and is fixed and re-run before release.

## Setup (per participant)

1. Install: `go build -o ~/.local/bin/boardwalk ./cmd/boardwalk`
   (or copy a built binary onto PATH; check `boardwalk --help`).
2. Auth: `gh auth status` must show the `project` and `repo` scopes
   (`gh auth refresh -s project,repo` if not).
3. Boards: the ADR 0004 boards `juanrubio/1` (SnazzyEmail) and
   `juanrubio/3` (Life Board), or a scratch copy if the participant will
   run the add/move/edit tasks for real. Start from a clean config:
   `mv ~/.config/boardwalk/config.toml ~/.config/boardwalk/config.toml.bak`
   (restore it afterwards).
4. Observer records: task, participant, completed unassisted (Y/N),
   time, exact stumble point and quote, fix filed.

## Tasks

1. **First run:** select a project and save it as the default.
   (Expected path: `boardwalk project list`, then
   `boardwalk project use <ref>`.) Observer also times this one —
   the PRD §5 five-minute first-run measure uses it.
2. **Add an existing issue** to the project.
   (`boardwalk item add owner/repo#number`.)
3. **Move an item** to another status/column from the CLI.
   (`boardwalk item move <ref> --to <option>`.)
4. **Edit a non-status field** (for example Priority) from the CLI.
   (`boardwalk item edit <ref> --field "Priority=P0"`.)
5. **In the TUI:** find an item by search, open it, and move it with
   `H`/`L`. (`boardwalk board`, `/`, `Enter`, `L`.)

## Results template

| # | Task | Participant | Unassisted | Time | Stumble / fix filed |
|---|---|---|---|---|---|
| 1 | First run + save default | | | | |
| 2 | Add existing issue | | | | |
| 3 | Move item (CLI) | | | | |
| 4 | Edit field (CLI) | | | | |
| 5 | TUI search, open, H/L move | | | | |

**Pass:** every participant completes every task without outside help.
Fixes that change command grammar require amending cli.md §2 and
re-running the affected task (cli.md §4).

## Automated rehearsal already in place

- The same five tasks run against the offline fixtures in
  `internal/cli/help_test.go` (`TestHumanTaskEquivalents`) and the TUI
  tests, so the flows are known-good before a human sits down.
- VHS rehearsal tapes for all five tasks plus the large board are in
  [tapes/](../../tapes/README.md) — note the mutation warning there
  before rendering.
