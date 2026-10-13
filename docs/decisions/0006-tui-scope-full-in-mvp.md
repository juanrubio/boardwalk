# ADR 0006 — TUI scope: full item management in the MVP

**Status:** Final
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Date:** 2026-10-13
**Deciders:** Project owner (accepted option B in decision interview, 2026-10-13)
**Depends on:** [ADR 0001](0001-implementation-language-go.md) (Final), [ADR 0003](0003-command-grammar-noun-first.md) (Final), [ADR 0004](0004-acceptance-fixtures-three-boards.md) (Final)

## Context

The PRD asks (§13 question 6) how much of the TUI belongs in the MVP
versus a fast table/list first. It already requires a TUI kanban grouped
by a selected single-select field plus a compact list/detail view
(§6.2), moving items from the CLI or TUI (§6.4), and defines the initial
release (§14) as covering CLI **and** TUI. Its main risk (§12) is TUI
complexity, mitigated by keeping the list/CLI commands fully capable.

The owner's top priorities are a really nice, beautiful CLI/TUI
experience and speed. Presented with CLI-first-kanban-second (option A),
full TUI in the MVP (option B), and CLI-only MVP (option C), the owner
chose **option B**.

## Decision

The MVP TUI (`boardwalk board`) is a **full item-management surface**.
Every MVP item capability must be completable inside the TUI without
dropping to the CLI:

- Browse the kanban (grouped by a selected single-select field, normally
  Status) and the compact list/detail views, with text/status/assignee/
  repository/type filtering and search (PRD §6.2, §8 flow C).
- Open an item detail panel showing title/body and current field values
  with their live, per-project options.
- Create a draft item and add an existing issue/PR from inside the TUI
  (PRD §6.3), with the same preview/confirmation behaviour as the CLI.
- Edit title/body and set project fields (single-select, date, number,
  text, iteration, people where the API allows) from the detail panel,
  with values validated before mutation (PRD §6.4, §9).
- Move items between columns, with pending changes shown and applied on
  confirmation, optimistic feedback, and a clear conflict/error report
  (PRD §8 flow C).
- Archive/remove only with explicit confirmation, as in the CLI
  (PRD §6.5).
- Keyboard-only operation throughout, usable at typical terminal widths,
  never conveying status by colour alone (PRD §9, §10).

The noun-first CLI remains **fully capable in parallel** (ADR 0003): the
TUI is never the only way to perform an MVP action. Building order may
still put the shared core and CLI first internally, but the MVP is not
done until the full TUI above passes against the ADR 0004 fixtures.

Explicitly still out of scope (PRD §6): creating/deleting projects and
administering project fields or workflows. "Full TUI" means full *item*
management, not project administration.

## Rationale

- The TUI is the product's distinctive, beautiful experience; shipping
  the MVP without its editing surface would defer the core value.
- The owner accepted the larger polish/test surface as the cost of that
  experience, with eyes open (see Consequences).

## Alternatives considered

- **CLI first, kanban second, in-TUI editing deferred (option A):**
  Lower risk and the interviewer's recommendation. Rejected by the owner
  in favour of a complete TUI in the first release.
- **CLI only for the MVP, TUI after (option C):** Fastest to a working
  tool, but contradicts PRD §14 and removes the distinctive experience.
  Rejected.

## Consequences

- The MVP release gate includes TUI creation, editing, and moving —
  tested keyboard-only against all three ADR 0004 fixture boards,
  including the synthetic 100+-item board for responsiveness.
- TUI complexity is now the top delivery risk (PRD §12). Mitigations are
  contractual, not optional: a fully capable CLI underneath, one shared
  GitHub client/core (ADR 0002) so TUI and CLI never diverge, and
  optimistic updates with explicit rollback/error states.
- Visual/keybinding design (layouts, keys, grouping-field picker,
  behaviour when a board has no Status-like field) is now MVP-critical
  design work, to be settled in the TUI specification before build.
- The PRD §14 release definition stands unchanged and is now read
  together with this ADR's capability list.

## Unresolved

- TUI design — **settled** in the
  [TUI specification](../specs/tui.md) (Final), all sections
  (§1–§6, 2026-10-13).
- Go CLI/TUI libraries — **resolved:** Cobra + Charm, see
  [ADR 0007](0007-go-stack-cobra-charm.md) (Final).
- Config format and JSON/error contract — **resolved:** see
  [ADR 0008](0008-config-format-toml.md) and
  [ADR 0009](0009-json-error-contract-bare.md) (both Final).
- Auth scopes — **resolved:** see [ADR 0010](0010-auth-scopes-upfront.md)
  (Final). All ADR-level decisions are now settled.
