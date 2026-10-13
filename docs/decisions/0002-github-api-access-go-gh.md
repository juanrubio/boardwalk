# ADR 0002 — GitHub API access: Go library via go-gh

**Status:** Final
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Date:** 2026-10-13
**Deciders:** Project owner (accepted option A in decision interview, 2026-10-13)
**Depends on:** [ADR 0001 — Implementation language: Go](0001-implementation-language-go.md) (Final)

## Context

The PRD ([PRD.md](../../PRD.md) §11, §13 question 2) requires Boardwalk to
access GitHub Projects v2 through the GitHub GraphQL API, reusing the
user's authenticated `gh` session, and never persisting GitHub access
tokens in Boardwalk config or logs (§6.5). The owner's top priorities are
a beautiful CLI/TUI experience and speed, which depend on precise error
handling, caching, optimistic updates, and not re-fetching whole boards
for single-field changes (§10).

The open choice was: call `gh api graphql` as a subprocess, use a
language-level GitHub client authenticated through `gh`, or prototype on
the subprocess and migrate later.

## Decision

Use a **Go library in-process**: `cli/go-gh` to resolve configuration and
credentials from `gh`'s own store, with a Go GraphQL client (for example
`shurcooL/githubv4`) for Projects v2 queries and mutations.

Boardwalk must still verify at startup that `gh` is installed and
authenticated, and must explain missing project/repository scopes with the
relevant `gh auth refresh` remediation (PRD §6.5, §8 flow A). The exact
scope list is a dependent detail to pin down in the auth specification.

Boardwalk never reads, stores, or logs access tokens itself; credential
resolution stays inside `gh`/`go-gh`.

## Rationale

- **Beautiful and fast feel:** Typed, in-process calls give structured
  errors (permission vs. rate-limit vs. not-found), clean pagination, and
  the control needed for optimistic status moves and targeted refetches —
  without spawning a process and parsing text output per call.
- **Testability:** The client can be mocked, so CLI/TUI behaviour is
  testable offline against recorded Projects v2 fixtures.
- **No planned rewrite:** Starting here avoids a subprocess-to-library
  migration landing in the middle of TUI polish work.

## Alternatives considered

- **`gh api graphql` subprocess (option B):** Simplest prototype and
  maximal reuse of the user's `gh`. Rejected as the long-term mechanism
  because string-built queries plus parsing subprocess output make
  pagination, rate-limit, and error handling brittle as features grow.
  Remains acceptable for throwaway spikes only.
- **Hybrid — prototype on subprocess, migrate to library (option C):**
  Rejected because it schedules the migration cost exactly when TUI work
  should be the focus.

## Consequences

- Boardwalk takes a maintained dependency on `go-gh` (and a GraphQL
  client) and must track their upstream releases.
- All GitHub access goes through one internal client package; CLI and TUI
  code never construct GraphQL or handle tokens directly.
- Failure modes must distinguish: `gh` missing, `gh` unauthenticated,
  insufficient scopes, project not found/inaccessible, rate limited, and
  network failure — each with an actionable message and non-zero exit.
- Revisit only if `go-gh` cannot resolve credentials for a standalone
  (non-`gh`-extension) `boardwalk` binary; the fallback would be a
  client authenticated via `gh`'s credential store by another supported
  route, still without Boardwalk persisting tokens.

## Unresolved

- Required scopes and remediation — **resolved:** `project` + `repo`
  required up front, see [ADR 0010](0010-auth-scopes-upfront.md) (Final).
- CLI/TUI library choices within Go — **resolved:** Cobra + Charm, see
  [ADR 0007](0007-go-stack-cobra-charm.md) (Final).
- Command grammar — **resolved:** noun-first, see
  [ADR 0003](0003-command-grammar-noun-first.md) (Final).
- Remaining PRD §13 questions 4–6 — since resolved by ADR 0004,
  ADR 0005, and ADR 0006 respectively.
