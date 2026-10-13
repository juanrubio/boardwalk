# ADR 0004 — Acceptance fixtures: three-board set, captured + synthetic

**Status:** Final — amended by A1 (2026-10-13)
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Amended:** 2026-10-13 — A1, owner approval of the
[beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md)
**Date:** 2026-10-13
**Deciders:** Project owner (accepted option A in decision interview, 2026-10-13)
**Depends on:** [ADR 0001](0001-implementation-language-go.md) (Final), [ADR 0002](0002-github-api-access-go-gh.md) (Final), [ADR 0003](0003-command-grammar-noun-first.md) (Final)

## Context

The PRD ([PRD.md](../../PRD.md) §12, §13 question 4) requires typed field
adapters tested against representative Projects v2 schemas, because custom
fields and their options differ per project (PRD §9). Without named
fixtures, "display custom fields accurately" (§5) has no testable meaning.

The owner chose a three-board fixture strategy over a single real board or
synthetic-only data. On 2026-10-13 the accessible boards were inventoried
live through the authenticated `gh` account (`juanrubio`):

- `juanrubio/1` **SnazzyEmail** — 14 items, all issues. Status
  (Backlog/Ready/In progress/In review/Done), Priority (P0/P1/P2), Size
  (XS–XL), Estimate (number), Start/Target dates. The richest field set.
- `juanrubio/3` **Life Board** — 16 items (6 issues, 10 draft issues).
  Status (Todo/In Progress/Done). The only draft-heavy board.
- Five further personal boards (18 items or fewer) all share the default
  Todo/In Progress/Done status set.
- The only organisation account, `tizonia`, has **no** Projects v2 boards.
- No accessible board has 100+ items, linked pull-request items, or
  custom iteration/text fields.

## Decision

Adopt a **three-slot fixture set**, captured as sanitised GraphQL
responses for offline tests, with live runs as smoke tests only:

1. **Rich-fields board** — a personal project exercising single-select,
   number, and date fields plus linked issues. Source (owner-confirmed
   2026-10-13): `juanrubio/1` SnazzyEmail.
2. **Differing-schema board** — a project whose Status labels/options
   differ from slot 1, including draft items, proving nothing is
   hard-coded. Source (owner-confirmed 2026-10-13): `juanrubio/3`
   Life Board.
3. **Large board** — 100+ items proving pagination stays usable and fast
   (PRD §6.2). *(A1)* To satisfy the performance budgets
   ([performance.md](../specs/performance.md) B6–B8), this slot is generated
   at **100, 1,000, and 10,000 items** as captured GraphQL fixtures.
   No live board qualifies, so this slot is **synthetic**:
   generated items in a captured-response fixture, not a live project.

Missing shapes are covered synthetically rather than by creating live
boards: organisation-owned project responses, pull-request items, and
iteration/text/people field values are synthesised in the same captured
format, modelled on the real responses from slots 1–2.

Tests must run offline from the captured fixtures; credentials or network
access must never be required for the fixture suite.

## Rationale

- Slots 1–2 anchor the adapters in real GitHub responses, including real
  field/option IDs, while slot 3 and the synthetic supplements cover
  scale and shapes no live board currently provides.
- Differing Status vocabularies between slots
  (Backlog/Ready/In progress/In review/Done vs. Todo/In Progress/Done)
  are the cheapest possible proof that field handling is dynamic.
- Capturing responses keeps tests deterministic even when the live
  boards change, directly mitigating the PRD §12 custom-field risk.

## Alternatives considered

- **One real board (option B):** Cannot prove dynamic fields and couples
  the test suite to one board's lifetime. Rejected.
- **Synthetic data only (option C):** Deterministic, but real API quirks
  (option IDs, draft-item permissions, iteration behaviour) would surface
  only after release. Rejected as the sole source; retained as the
  supplement for missing shapes.

## Consequences

- Implementation cannot be called done until the CLI/TUI flows pass
  against all three slots offline, plus a live smoke pass on slots 1–2.
- Fixture capture/sanitisation tooling (even if manual at first) is part
  of the first implementation slice; private data must never be committed
  unsanitised.
- If a real organisation board or 100+-item board becomes available, its
  captured responses should replace the synthetic fixture for that slot.

## Amendment log

- **A1 (2026-10-13)** — owner approval of the
  [beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md):
  Slot 3 expanded to generate 100, 1,000, and 10,000 item synthetic fixtures
  to validate responsiveness budgets under [performance.md](../specs/performance.md).
