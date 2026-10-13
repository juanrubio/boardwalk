# Boardwalk performance specification

**Status:** Final
**Accepted:** 2026-10-13 — owner approval of the
[beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md)
(Amendment 1, "A1")
**Date:** 2026-10-13
**Scope:** Latency, request-count, rendering, and memory budgets for the
CLI ([cli.md](cli.md)) and the TUI ([tui.md](tui.md)). This spec makes
the speed priority (ADR 0001, PRD §3/§10) measurable. Caching is defined
in [ADR 0011](../decisions/0011-local-cache-and-snapshot.md).

Boardwalk is network-bound (ADR 0001). Its speed therefore comes from
**fewer round trips, a local cache, and a UI that never waits on work it
does not need**. Raw CPU speed matters much less. Every rule below
serves one of those three.

## 1. Budgets (release gate)

Budgets are measured offline against the ADR 0004 fixtures, with a
simulated round-trip latency of 400 ms per GraphQL request. Live smoke
runs on slots 1–2 are reported but not gated (network variance).

| # | Metric | Budget |
|---|---|---|
| B1 | CLI cold start to first output for local-only commands (`--help`, config/alias resolution) | < 50 ms |
| B2 | `item move` / `item edit` with a warm cache (ADR 0011) | ≤ 2 round trips: ≤ 1 lookup + 1 write request |
| B3 | `item move` / `item edit` with a cold cache | ≤ 3 round trips |
| B4 | `board` launch to first meaningful paint, warm snapshot (ADR 0011) | < 100 ms, zero round trips before paint |
| B5 | `board` launch to first live cards, cold | 1 round trip (schema + first page in one request, §2) |
| B6 | TUI keypress to rendered frame, 1,000-item fixture | < 16 ms p95 |
| B7 | TUI keypress to rendered frame, 10,000-item fixture | < 50 ms p95 |
| B8 | TUI resident memory, 10,000-item fixture | < 150 MB RSS |
| B9 | `item list` with any server-expressible filter (cli.md §2) | Downloads only matching items |

A regression past a budget fails CI the same way a broken test does.
Changing a budget requires amending this spec.

## 2. Request rules

- **Combine what the first paint needs.** The project schema (fields,
  options, iterations) and the first item page are fetched in **one**
  GraphQL document. The first page is smaller (default 50) so it returns
  sooner; later pages use 100 per page. Both sizes are tunable constants,
  set after the live spike.
- **No bodies in list queries.** Board, list, and `item list` queries
  never fetch issue/PR/draft bodies. The body is fetched when detail
  opens. The TUI prefetches it for the focused card and its immediate
  neighbours (tui.md §3).
- **Only the fields that are shown.** List queries request the field
  values that cards and rows show (tui.md §3/§6). Detail fetches the
  rest for its one item.
- **Resolve linked references from the issue side.** `owner/repo#N` and
  issue/PR URLs resolve through the issue/PR's own `projectItems`
  connection (O(1)), never by scanning the board (cli.md §1).
- **Filter on the server where possible.** Filters expressible in
  GitHub's project filter syntax are sent as the `items(query:)`
  argument (cli.md §2, tui.md §2). Client-side filtering is the fallback
  for anything the server cannot express. It must return the same items,
  and tests cover both paths against the same fixtures.
- **Mutations return what the UI needs.** Every mutation selects the
  updated item's displayed fields in its response, so a successful write
  needs no follow-up refetch. Where a payload cannot carry them, one
  targeted item query is allowed. A board refetch is never allowed
  (PRD §10).
- **One write request per save.** A multi-field save sends all of an
  item's mutations as aliased fields in a single GraphQL request (GitHub
  runs them in order). Partial results are reported per field
  (tui.md §4, cli.md §2).
- **No separate auth round trip.** Token scopes are read from the
  response headers of the first real request (ADR 0010). The token is
  resolved once per process.
- **One HTTP client per process,** with keep-alive. Independent reads
  (for example the body prefetch and the next item page) may run
  concurrently, capped at 4 requests in flight.

## 3. Rendering rules (TUI)

- **Virtualise.** Only visible cards/rows (plus one screen of overscan)
  are rendered per frame. Off-screen items cost no styling work.
- **Memoise rendered cards/rows,** keyed by (item version, width,
  focused, density, theme). A keypress that moves focus re-renders at
  most the two affected cards and the chrome.
- **Batch streamed data.** Each arriving page is merged into the model as
  one update. Merges never shift the focused card on screen (tui.md §5).
- **Heavy work off the UI loop.** Markdown rendering (Glamour), sorting
  and filtering more than 1,000 items, and snapshot (de)serialisation run
  as `tea.Cmd`s and post their results back. Rendered Markdown is cached
  per (item version, width, theme).
- **Filter as you type within budget.** Incremental `/` filtering over
  loaded items must meet B6/B7. Server-side refinement of the query is
  debounced (250 ms idle).

## 4. Fixtures and measurement

- ADR 0004 slot 3 is generated at **100, 1,000, and 10,000** items
  (amended by A1), plus the real slots 1–2.
- A benchmark harness drives the TUI headless (`teatest`) with scripted
  key sequences and records frame times. CLI benchmarks count round
  trips and wall time against a latency-injecting fixture transport.
- Results are printed in CI. Budgets B1–B9 are assertions, not
  dashboards.

## Amendment log

- **A1 (2026-10-13):** Spec created from the approved
  [beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md).
