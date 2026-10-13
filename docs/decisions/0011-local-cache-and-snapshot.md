# ADR 0011 — Local cache: schema/ID cache and board snapshot

**Status:** Final
**Accepted:** 2026-10-13 — owner approval of the
[beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md)
(Amendment 1)
**Date:** 2026-10-13
**Deciders:** Project owner (approved review recommendations P2/P3, 2026-10-13)
**Depends on:** [ADR 0002](0002-github-api-access-go-gh.md) (Final), [ADR 0005](0005-alias-config-user-only.md) (Final), [ADR 0008](0008-config-format-toml.md) (Final), [ADR 0010](0010-auth-scopes-upfront.md) (Final)

## Context

Every Boardwalk operation is dominated by GitHub GraphQL round trips
(ADR 0001). Without a cache, `boardwalk item move owner/repo#42 --to
Done` needs four or five sequential requests: scope check, project ID,
field schema, item lookup, and the write. That is roughly 1.5–3 s for the
most common command. Every `boardwalk board` launch also shows nothing
until the network answers.

The [performance specification](../specs/performance.md) sets budgets
(B2–B5) that cannot be met without local state. The original TUI spec
(§5) ruled out "background magic". That rule was aimed at polling and
silent overwrites, not at caching. This ADR defines caching in a way that
keeps the rule's intent.

## Decision

Boardwalk keeps a **local, disposable cache** under the OS user cache
directory (`os.UserCacheDir()/boardwalk/`, for example
`~/.cache/boardwalk/` on Linux, honouring `$XDG_CACHE_HOME`). It has two
parts.

### 1. Schema/ID cache (CLI and TUI)

Per project: project node ID, title, field IDs/names/types,
single-select option IDs/names/colours/descriptions, and iterations.
Also a map from item `databaseId` to node ID, used for `draft:` short
references (cli.md §1).

- Used to validate and build mutations without fetching the schema
  first.
- **Self-healing:** if validation against the cache fails (unknown
  field/option), or GitHub rejects an ID as stale, Boardwalk refetches
  the schema **once** and validates again before reporting an error.
  This happens before any write, so it never risks a duplicate mutation
  (PRD §10).
- Refreshed whenever any live schema is fetched (for example on TUI
  launch or `r`).

### 2. Board snapshot (TUI)

The last loaded board for each project: the items' displayed fields, in
project order, plus the time it was fetched. Bodies are not included.

- `boardwalk board` **paints the snapshot immediately**, and the header
  shows its age (for example `updated 3h ago · refreshing…`). It then
  **revalidates once**, using the normal progressive load (tui.md §5),
  and replaces snapshot data as live pages arrive.
- **No polling.** Revalidation happens only at launch and on `r`.
- **Writes wait for live data.** An item shown only from the snapshot is
  marked stale and cannot be moved or edited until its live data has
  arrived in this session. Pressing a mutation key on a stale card says
  so in the footer. This stops a blind write (`H`/`L`) from overwriting a
  change made elsewhere since the snapshot was taken.
- Items missing from the live result are removed when revalidation
  finishes. Focus follows tui.md §5's vanished-item rule.

### Rules for both parts

- **Never credentials.** The cache holds only project data the user can
  already read. It never stores tokens, headers, or raw responses
  (PRD §6.5, ADR 0002).
- **Disposable.** Deleting the directory is always safe. A corrupt or
  incompatible-version entry is discarded silently and refetched. It is
  never an error.
- **Private.** Files are created with user-only permissions (0600, and
  0700 for directories), because private project content is cached.
- **Escape hatches.** The global `--no-cache` flag bypasses both parts
  for one run (cli.md §3). `BOARDWALK_NO_CACHE=1` does the same.
- **Bounded.** Snapshots that have not been used for 30 days are pruned
  when Boardwalk starts.

## Rationale

- **Instant first paint** (performance B4) is the biggest perceived-speed
  win available. It is how the fastest terminal tools feel instant.
- **Fewer round trips per command** (B2/B3) without weakening
  validation. Stale cache entries can only cause an extra read, never a
  wrong write.
- **The intent of "no background magic" is kept:** data age is always
  visible, nothing polls, and stale data cannot be written through.

## Alternatives considered

- **No cache (status quo):** Simple and always fresh, but it fails the
  speed priority and budgets B2–B5. Rejected.
- **Schema cache only, no snapshot:** Fixes CLI latency, but the TUI
  still opens blank. Rejected as insufficient for the TUI-first goal.
- **Live polling board:** Rejected again, for the reasons in tui.md §5
  (it spends rate budget while idle and creates more conflict cases).

## Consequences

- An internal `cache` package sits behind the shared client (ADR 0002).
  CLI and TUI code never read cache files directly.
- Tests cover cold, warm, stale-schema (self-heal), and corrupt-cache
  paths for every fixture.
- Documentation states the cache location, what it contains, and how to
  clear or bypass it.
