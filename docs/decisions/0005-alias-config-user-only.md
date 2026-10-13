# ADR 0005 — Alias/config scope: user config only for MVP

**Status:** Final
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Date:** 2026-10-13
**Deciders:** Project owner (accepted option A in decision interview, 2026-10-13)
**Depends on:** [ADR 0001](0001-implementation-language-go.md) (Final), [ADR 0003](0003-command-grammar-noun-first.md) (Final)

## Context

The PRD requires short, memorable local aliases and a default project
(§3, §5, §6.1) and asks (§13 question 5) whether aliases should live only
in user config or also support repository-local config. Aliases are the
muscle-memory layer of the tool — the owner's top priorities are a
beautiful experience and speed — so an alias must mean the same board
everywhere, instantly, without surprises from the current directory.

## Decision

For the MVP, aliases and the default project live **only in user config**
(located under the OS user config directory, e.g.
`~/.config/boardwalk/` on Linux; the exact file name and format are a
dependent detail for the config specification).

Project resolution order for every command:

1. Explicit reference on the command line (alias, URL, or owner/number),
   including the `--project` flag.
2. `BOARDWALK_PROJECT` environment variable.
3. The saved default project in user config.
4. Otherwise, an interactive selector or an actionable error — never a
   guess.

Repository-local config is **not** supported in the MVP. The resolution
order above is designed so it can be inserted later (most likely between
steps 2 and 3) without renaming existing aliases or changing their
meaning for current users.

The config file stores aliases and the default project only. It must
never contain GitHub tokens or credentials (PRD §6.5, ADR 0002).

## Rationale

- **Predictable speed:** The same alias resolves to the same board in
  every directory, so muscle memory is trustworthy; nothing inside a
  cloned repository can silently redirect `life-board` elsewhere.
- **No premature trust model:** Repository-local config would immediately
  require precedence, trust, and team-sharing rules before any user has
  asked for shared aliases.
- **Reversible:** Adding repo-local lookup later is backward compatible;
  shipping it in v1 and removing or reordering it later is not.

## Alternatives considered

- **User + repository-local config in the MVP (option B):** Enables
  checked-in team aliases per repo. Rejected for v1: it front-loads
  precedence and trust decisions with no demonstrated demand.
- **No aliases, explicit owner/number or URL every time (option C):**
  Simplest implementation, but contradicts the PRD's "fast for the common
  case" principle and taxes every daily command. Rejected.

## Consequences

- `boardwalk project use <alias>` writes only to user config; first-run
  flow (PRD §8 flow A) never creates files inside a repository.
- Documentation must state the resolution order explicitly, and
  `boardwalk project list` should show which project is default and how
  each alias resolves, so resolution is never mysterious.
- If repository-local config is added later, it needs its own ADR
  covering precedence, trust (explicit opt-in per repo is the expected
  shape), and which keys a repo may set — aliases yes, credentials never.

## Unresolved

- Config file name, format, and schema — **resolved:** TOML
  `config.toml` in the OS config directory, see
  [ADR 0008](0008-config-format-toml.md) (Final).
- TUI scope — **resolved:** full item management in the MVP, see
  [ADR 0006](0006-tui-scope-full-in-mvp.md) (Final). All PRD §13
  questions are now resolved.
