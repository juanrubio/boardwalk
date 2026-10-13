# ADR 0001 — Implementation language: Go

**Status:** Final
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Date:** 2026-10-13
**Deciders:** Project owner (accepted in decision interview, 2026-10-13)

## Context

The PRD ([PRD.md](../../PRD.md) §13, question 1) left the implementation
language open between Go, Rust, and TypeScript. The owner's stated top
priorities for Boardwalk are:

1. A really nice, beautiful CLI/TUI experience.
2. Speed.

Boardwalk is I/O-bound: every user-visible operation is dominated by a
GitHub GraphQL round trip (typically hundreds of milliseconds), not by
local CPU work. Perceived speed therefore depends on caching, optimistic
updates, asynchronous loading, and avoiding redundant fetches rather than
on raw language throughput.

## Decision

Implement Boardwalk in **Go**.

The TUI/CLI library choices (for example Cobra and the Charm stack —
Bubble Tea, Lip Gloss, Bubbles) are dependent decisions and will be
recorded separately; this ADR settles the language only.

## Rationale

- **Beautiful TUI, fast to polish:** The Charm ecosystem is purpose-built
  for styled terminal interfaces and has proven polished results
  (lazygit, gh-dash, Glow). Go's fast compile times keep the
  tweak-and-look iteration loop tight, which is where visual quality
  actually comes from.
- **Speed where it matters:** Startup (~10–30 ms) and rendering are far
  below network latency, so Go's performance is not a user-visible
  constraint. Single static binaries cross-compiled per OS also keep
  installation and launch fast.
- **Ecosystem fit:** `gh` itself is written in Go, easing auth reuse and
  access to GitHub-oriented Go libraries (see the pending decision on
  GitHub API access).

## Alternatives considered

- **Rust (Ratatui):** Faster startup and raw CPU performance, and capable
  of equally beautiful TUIs (gitui, yazi). Rejected because that CPU edge
  is invisible in an I/O-bound tool, while lower-level rendering plumbing
  and slower compiles raise the cost of every polish iteration. This
  would flip if the maintainers' Rust fluency clearly exceeded their Go.
- **TypeScript:** Rejected because it imposes a Node runtime on users and
  its TUI tooling is weaker for a kanban-style interface.

## Consequences

- Distribution is a single static `boardwalk` binary per OS/architecture.
- Hiring/contributor expectations and all subsequent stack decisions
  (CLI framework, TUI framework, GraphQL client) are Go-ecosystem choices.
- Revisit only if a hard requirement appears that Go cannot meet (for
  example, a needed library existing solely in another ecosystem).

## Unresolved

- GitHub API access mechanism — **resolved:** Go library via `go-gh`, see
  [ADR 0002](0002-github-api-access-go-gh.md) (Final).
- All remaining PRD §13 questions are resolved (ADR 0003–0006), and the
  Go library stack is settled in
  [ADR 0007](0007-go-stack-cobra-charm.md) (Final).
