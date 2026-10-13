# ADR 0003 — Command grammar: noun-first (`item`, `project`, `board`)

**Status:** Final
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Date:** 2026-10-13
**Deciders:** Project owner (accepted option A in decision interview, 2026-10-13)
**Depends on:** [ADR 0001](0001-implementation-language-go.md) (Final), [ADR 0002](0002-github-api-access-go-gh.md) (Final)

## Context

The PRD ([PRD.md](../../PRD.md) §7, §13 question 3) proposed a command
shape but left the exact grammar open to UX testing: `boardwalk item`,
`boardwalk issue`, or shorter aliases. The grammar is the surface users
type hundreds of times, so it directly affects the owner's top priorities
— a beautiful experience and speed of everyday use.

A GitHub Projects v2 *item* can be a linked issue, a linked pull request,
or a draft item (PRD §4, §9). Any grammar must name all three honestly.

## Decision

Use **noun-first grammar**:

- `boardwalk project …` — discover, select, and alias projects.
- `boardwalk item …` — list, add, edit, and move items (issues, PRs, and
  drafts through one verb set).
- `boardwalk board` — the interactive TUI for the selected project.

The example commands in PRD §7 stand as the intended shape. Global flags
(for example `--project`, `--json`, `--dry-run`, `--yes`) and exact verb
names remain details for the CLI specification. **No short verb-first
aliases in v1**; they may be layered on later, without breaking changes,
once real usage shows which commands deserve them. Shell completion and
help quality are required instead (PRD §7).

## Rationale

- **Honest model:** `item` covers issues, PRs, and drafts uniformly;
  `issue` would misname two of the three and force a second inconsistent
  command family later.
- **Discoverable and scriptable:** Noun-first groups related verbs under
  `boardwalk item --help`, keeps scripts readable, and grows cleanly when
  new nouns appear.
- **Reversible where it matters:** Adding aliases later is free; removing
  or renaming a published grammar is not. Deferring shortcuts is the
  safe direction.

## Alternatives considered

- **`boardwalk issue` (option B):** Familiar on day one, but misnames PRs
  and drafts. Rejected.
- **Verb-first shortcuts such as `boardwalk add` / `boardwalk move`
  (option C):** Fastest to type, but ambiguous as soon as the noun is
  unclear (add an item? an alias? a field?) and collides with future
  nouns. Rejected for v1; may return as aliases over this grammar.

## Consequences

- Help, completion, documentation, and examples all organise around the
  three nouns; the TUI is launched as `boardwalk board`.
- Usability testing (PRD §7) now validates verbs, flags, and wording
  *within* this grammar rather than the grammar itself.
- Any future shortcut must be an alias for a noun-first command, never a
  divergent spelling with different behaviour.

## Unresolved

- CLI details — **settled** in the
  [CLI specification](../specs/cli.md) (Final): item references (§1),
  verb/flag inventory (§2), global flags/script behaviour (§3), and
  help/completion validation (§4).
- Acceptance fixtures — **resolved:** three-board set, see
  [ADR 0004](0004-acceptance-fixtures-three-boards.md) (Final).
- Remaining PRD §13 questions 5–6 — since resolved by ADR 0005 and
  ADR 0006 respectively.
