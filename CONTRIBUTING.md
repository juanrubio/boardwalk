# Contributing to Boardwalk

Thanks for wanting to help. Boardwalk itself was built with AI
assistance, and contributions made with AI tools are welcome here —
encouraged, even. What matters is the change: does it respect the
specs, and is `make check` green. That bar is the same for everyone,
and it is the whole bar.

## Quickstart

You need Go (the version pinned in [go.mod](go.mod)) — and, to run
Boardwalk against real boards, the GitHub CLI set up as described in
the [README](README.md#install). The test suite needs neither `gh`
nor network access; it runs entirely against captured fixtures.

```sh
git clone https://github.com/juanrubio/boardwalk.git
cd boardwalk
make check   # vet + lint + full offline test suite
make build   # builds ./boardwalk
```

## Where things live

[AGENTS.md](AGENTS.md) has the map of the repo and the rules that
aren't obvious from the code. It is written for AI coding tools and
new humans alike — start there, or point your agent at it.

## The contribution loop

1. **Issue first for bigger things.** Small fixes can go straight to
   a PR; anything larger deserves a short issue so the shape can be
   agreed before code is written.
2. **Specs before behaviour.** [docs/specs/](docs/specs/) describe
   the exact behaviour. If your change alters behaviour, amend the
   relevant spec in the same PR (see the amendment logs at the end
   of each spec). Non-trivial architecture calls get an ADR in
   [docs/decisions/](docs/decisions/).
3. **Code and tests together.** New behaviour comes with tests;
   rendering changes update the golden frames deliberately (how is
   in [AGENTS.md](AGENTS.md)).
4. **`make check` green**, then open the PR. Commit messages follow
   the conventional style already in `git log` (`feat:`, `fix:`,
   `docs:`, …).

Looking for a place to start? The
[roadmap](docs/roadmap.md#6-field--state-gap-bundle) lists a bundle
of small, self-contained gaps — each one a five-second trip to the
browser today, each one a tidy first PR.
