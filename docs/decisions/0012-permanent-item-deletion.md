# ADR 0012 — Permanent item deletion (`item delete`)

**Status:** Final
**Accepted:** 2026-10-13 — owner request to implement permanent deletion
**Date:** 2026-10-13
**Deciders:** Project owner
**Depends on:** [ADR 0003](0003-command-grammar-noun-first.md) (Final), [ADR 0006](0006-tui-scope-full-in-mvp.md) (Final, A1), [ADR 0009](0009-json-error-contract-bare.md) (Final)
**Amends:** cli.md §2 (`item archive` row), tui.md §2. ADR 0006 A1 recorded
item deletion as out of MVP scope; this ADR supersedes that single clause.

## Context

The MVP shipped archive as the only destructive verb (cli.md §2): the item
leaves the board, the issue stays open, and GitHub can restore archived
items. The owner asked for permanent deletion. GitHub's Projects v2 API
exposes exactly one such operation, `deleteProjectV2Item`: it removes the
project item itself. A linked issue or PR is untouched on GitHub, but a
draft's content exists only inside the item, so deleting a draft is
permanent data loss. There is no undo.

## Decision

Boardwalk adds permanent deletion beside archive, with a stronger guard
because it cannot be undone:

- **CLI:** `boardwalk item delete <ref>` (the owner chose `delete` over
  `remove`, whose softness could be mistaken for archive).
- **TUI:** `D` on the focused card/row opens the delete modal. The footer
  and `?` help list it next to `x` archive.
- **Double confirmation (owner choice):** interactive use requires two
  successive affirmative confirmations, both naming the item by reference,
  kind, and title, and stating plainly that it cannot be undone and what
  happens to the underlying issue/PR (nothing) or draft (permanently
  gone). In the TUI both steps are `y` at successive stages of the modal;
  any other key cancels. In the CLI both steps are successive prompts.
- **Scripts:** `--yes` remains the single explicit opt-out (cli.md §3);
  piped runs without `--yes` refuse with exit 2, and `--dry-run` prints
  the planned deletion and sends nothing, exactly like other mutations.
- **`--json`:** success reports `{"ref", "deleted": true}`; errors follow
  ADR 0009.

## Alternatives considered

- **Typing the reference to confirm:** Stronger, but slower for the common
  case and inconsistent with every other Boardwalk confirmation. The
  owner chose double confirmation instead.
- **`item remove`:** Rejected by the owner; it reads as reversible.
- **CLI-only:** Rejected; ADR 0006 keeps CLI and TUI at parity.

## Consequences

- Archive stays the recommended default for "get this off my board";
  delete exists for drafts and items that should cease to exist on the
  project. Help text for both verbs cross-references the other.
- Tests cover: fixture deletion actually removes the item, piped-without-
  `--yes` refusal (exit 2), `--dry-run` sending nothing, and the TUI's
  two-stage modal (one `y` never deletes).
