# ADR 0009 — JSON/error contract: bare JSON, documented exit codes

**Status:** Final
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Date:** 2026-10-13
**Deciders:** Project owner (accepted option A in decision interview, 2026-10-13)
**Depends on:** [ADR 0002](0002-github-api-access-go-gh.md) (Final), [ADR 0003](0003-command-grammar-noun-first.md) (Final), [ADR 0007](0007-go-stack-cobra-charm.md) (Final)

## Context

The PRD promises structured output with a stable schema and no terminal
decoration when piped (§9), human-readable output by default with a JSON
mode for scripts (§6.5), and non-zero exit codes with actionable messages
on failure (§6.5). It defines none of these concretely. Scripts will be
written against this surface on day one, making it the hardest contract
to change later.

## Decision

**Success output.** With `--json`, a command prints the bare result —
a JSON object or array, with no envelope — to stdout, so it pipes
directly into tools like `jq`. Each command's shape is documented in its
help/reference. Shapes are stable within a major version: fields may be
added; renaming or removing a field is a breaking change. Without
`--json`, output is human-readable, and all styling/decoration is
suppressed when stdout is not a terminal.

**Dry-run.** `--dry-run` prints the planned mutation and applies nothing.
Under `--json` the plan uses the same shape family as the real result
(marked as a plan), so scripts can diff intent vs. outcome.

**Errors.** Errors always go to stderr, never stdout. Normally they are
short human text naming what failed and the smallest useful remediation
(for example the `gh auth refresh …` command for scope problems). Under
`--json`, stderr carries one JSON error object:

```json
{"error": {"code": "auth", "message": "…", "remediation": "…"}}
```

**Exit codes** (the complete taxonomy; every failure maps to exactly one):

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Unclassified failure (bug or unexpected API/network outcome) |
| 2 | Usage error (bad flags, invalid field value, malformed reference) |
| 3 | Authentication/scope problem (gh missing, unauthenticated, missing scopes) |
| 4 | Not found (project, item, field, or option does not exist) |
| 5 | Permission denied (authenticated, but not allowed on this resource) |

Rate limiting and transient network failures are retried with bounded
backoff for reads (PRD §10); if they still fail, they exit 1 with the
rate-limit/retry information in the message. Mutations are never silently
retried after an uncertain outcome (PRD §10, §12).

The TUI (`boardwalk board`) reports the same error categories in its UI
with the same remediation text; exit-code mapping applies when the TUI
exits due to a startup failure (for example auth, ADR 0002).

## Rationale

- **Pipe-friendly by default:** Bare JSON respects the Unix convention
  users expect from `gh`/`jq` workflows; an envelope would tax every
  consumer forever for a versioning benefit that per-command stability
  rules already provide.
- **Scripts can branch:** Six codes are few enough to document on one
  screen and enough to distinguish "fix your auth" (3) from "that item
  does not exist" (4) — the distinction the PRD's actionable-error
  requirement (§6.5, §12) actually needs.
- **One error vocabulary:** CLI, `--json`, and TUI share categories and
  remediation strings, so behaviour never diverges between surfaces
  (ADR 0006, ADR 0007).

## Alternatives considered

- **Versioned envelope `{data, error, meta}` (option B):** Formal
  evolvability, but every consumer unwraps `.data` forever. Rejected.
- **JSON shapes only, single exit code 1 for all failures (option C):**
  Less to document, but scripts cannot react differently to auth vs.
  not-found failures, undermining PRD §6.5. Rejected.

## Consequences

- Command help must document each command's JSON shape and applicable
  exit codes; shape changes follow the add-only rule above.
- Tests assert exit codes and the JSON error object, not just stdout,
  against the ADR 0004 fixtures (including forced auth/not-found cases
  synthesised for this purpose).
- New failure kinds must fit the six codes or amend this ADR; inventing
  ad-hoc codes per command is forbidden.

## Unresolved

- Auth scopes and remediation — **resolved:** see
  [ADR 0010](0010-auth-scopes-upfront.md) (Final).
- TUI design — **settled** in the
  [TUI specification](../specs/tui.md) (Final), all sections.
