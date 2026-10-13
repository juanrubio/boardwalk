# Boardwalk

**Boardwalk** is a proposed terminal-first way to file, edit, and visualize GitHub Projects work without opening a browser.

This repository currently contains the product requirements document and the accepted architecture and specification decisions; implementation has not started yet.

- [Product Requirements Document](PRD.md)
- [Architecture decisions](docs/decisions/) — ADR 0001: Go (Final); ADR 0002: GitHub API via `go-gh` (Final); ADR 0003: noun-first commands (Final); ADR 0004: three-board fixtures (Final, amended by A1); ADR 0005: user-only alias config (Final); ADR 0006: full TUI in MVP (Final, amended by A1); ADR 0007: Cobra + Charm stack (Final, amended by A1); ADR 0008: TOML config (Final); ADR 0009: bare JSON + exit codes (Final); ADR 0010: upfront auth scopes (Final, amended by A1); ADR 0011: local cache and snapshot (Final)
- [Specifications](docs/specs/) — [TUI specification](docs/specs/tui.md) (Final, amended by A1), [CLI specification](docs/specs/cli.md) (Final, amended by A1), [Performance specification](docs/specs/performance.md) (Final), and [Mockups](docs/specs/mockups.md) (Approved 2026-10-13)
- [Reviews and amendments](docs/reviews/) — [Beauty and speed review](docs/reviews/2026-10-13-beauty-and-speed.md) (Amendment 1, Approved 2026-10-13)

Working name: **Boardwalk**  
Repository: **gh-boardwalk**  
Planned command: **boardwalk**
