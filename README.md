# Boardwalk

**Boardwalk** is a terminal-first way to file, edit, and visualize GitHub Projects (Projects v2) work without opening a browser. It reuses your authenticated GitHub CLI (`gh`) session and never stores tokens; `project` and `repo` scopes are required upfront (`gh auth refresh -s project,repo`).

The MVP is implemented: a noun-first CLI (`project`, `item`, `board`) and a full keyboard-first Bubble Tea TUI share one core service over a Go GraphQL client (`cli/go-gh`).

## Quick start

```sh
go build -o boardwalk ./cmd/boardwalk
boardwalk project list                       # find your Projects v2 boards
boardwalk project use juanrubio/3 --as life  # save a default (alias optional)
boardwalk item list                          # table on a TTY, TSV when piped, --json for scripts
boardwalk board                              # interactive kanban/list TUI (press ? for keys)
boardwalk item move owner/repo#42 --to Done
boardwalk item edit owner/repo#42 --field "Priority=P0"
boardwalk item add owner/repo#42             # or: item add --draft --title "..."
```

Mutations print a change summary first; use `--dry-run` to preview and `--yes` to confirm in scripts (piped mutations without `--yes` refuse with exit 2). Exit codes follow ADR 0009 (`0/1/2/3/4/5`).

## Documentation

- [Product Requirements Document](PRD.md)
- [Architecture decisions](docs/decisions/) — ADR 0001: Go (Final); ADR 0002: GitHub API via `go-gh` (Final); ADR 0003: noun-first commands (Final); ADR 0004: three-board fixtures (Final, amended by A1); ADR 0005: user-only alias config (Final); ADR 0006: full TUI in MVP (Final, amended by A1); ADR 0007: Cobra + Charm stack (Final, amended by A1); ADR 0008: TOML config (Final); ADR 0009: bare JSON + exit codes (Final); ADR 0010: upfront auth scopes (Final, amended by A1); ADR 0011: local cache and snapshot (Final); ADR 0012: permanent item deletion (Final)
- [Specifications](docs/specs/) — [TUI specification](docs/specs/tui.md) (Final, amended by A1/A2), [CLI specification](docs/specs/cli.md) (Final, amended by A1), [Performance specification](docs/specs/performance.md) (Final), and [Mockups](docs/specs/mockups.md) (Approved 2026-10-13)
- [Reviews and amendments](docs/reviews/) — [Beauty and speed review](docs/reviews/2026-10-13-beauty-and-speed.md) (Amendment 1, Approved 2026-10-13)
- [VHS tapes](tapes/) — scripted recordings of the cli.md §4 human tasks; the human completion gate (owner plus 2–3 target users) remains an owner-run release step.

## Development

```sh
make check   # vet + pinned golangci-lint + full test suite (offline, fixture-driven)
make build
```

Working name: **Boardwalk**
Repository: **gh-boardwalk**
Command: **boardwalk**
