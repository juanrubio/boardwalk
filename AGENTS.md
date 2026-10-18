# Contributing to Boardwalk — notes for AI agents and humans

Welcome. AI coding agents are first-class contributors here: this
codebase was itself built with AI assistance, and more of it is
welcome. Contributions are judged by the change — does it respect the
specs, do the tests pass — whoever or whatever authored them. This
file is the front door: how to build, where things live, and the few
rules that aren't obvious from the code.

## Build & verify

Go version is pinned in [go.mod](go.mod) (currently 1.27.2). Three
commands cover almost everything:

```sh
make build                  # build ./boardwalk
make check                  # go vet + golangci-lint (via `go tool`) + full test suite
make run ARGS="item list"   # run without building
```

Everything needed to verify a change runs offline. Tests use captured
fixtures ([testdata/](testdata/) and `internal/fixture`), never the
live GitHub API — so `make check` is the whole bar, and it must be
green before a change is done. `make check` is also what CI runs.

## Map of the repo

- `cmd/boardwalk/` — entrypoint. Thin by design; keep it that way.
- `internal/cli/` — Cobra commands, output rendering, `--json` shapes.
- `internal/core/` — shared service logic used by both CLI and TUI.
- `internal/githubx/` — Projects v2 GraphQL, behind the `Client`
  interface so tests can run against fixtures instead of the network.
- `internal/tui/` — the Bubble Tea board (`boardwalk board`).
- `internal/cache/`, `internal/config/`, `internal/theme/`,
  `internal/exiterr/` — local snapshot cache, user config, terminal
  theming, and the exit-code error taxonomy.
- `docs/` — [specs](docs/specs/) (the behaviour contract),
  [decisions](docs/decisions/) (ADRs), [roadmap](docs/roadmap.md).
- `tapes/` — VHS demo scripts.
- [PRD.md](PRD.md) — product requirements; §3 states the principles
  everything else is measured against.

## Rules that aren't obvious from the code

- **Specs are the contract.** Read the relevant spec in
  [docs/specs/](docs/specs/) before changing behaviour. A behaviour
  change lands together with a spec amendment — mark amended passages
  and add an entry to the spec's amendment log, as the existing ones
  do. Non-trivial architecture calls get an ADR in
  [docs/decisions/](docs/decisions/), numbered after the latest one.
- **Golden frames regenerate deliberately.** TUI rendering is pinned
  by golden tests. When a rendering change is intended, run
  `go test ./internal/tui/ -run TestGoldenFrames -update-golden`,
  then eyeball the diff before committing it. Never regenerate just
  to make a red test green.
- **Boardwalk never stores tokens.** Auth comes from the user's `gh`
  session at run time ([ADR 0002](docs/decisions/0002-github-api-access-go-gh.md),
  [ADR 0010](docs/decisions/0010-auth-scopes-upfront.md)). Keep it
  that way: no credential ever lands in Boardwalk config or logs.
- **Mutations preview, confirm, and dry-run.** Every change shows a
  summary first, supports `--dry-run`, and refuses to guess in
  scripts ([ADR 0009](docs/decisions/0009-json-error-contract-bare.md)).
  New mutations follow the same shape, including the exit-code and
  `--json` error contract defined there.
- **Conventional commits.** `feat:`, `fix:`, `chore:`, `docs:`,
  `refactor:` — see `git log` for the local style.
- **Version stamp.** Release builds pass
  `-ldflags "-X github.com/juanrubio/boardwalk/internal/cli.Version=<tag>"`;
  plain builds report `dev`. Don't hardcode a version anywhere.

## Good first contributions

The [field & state gap bundle](docs/roadmap.md#6-field--state-gap-bundle)
in the roadmap is a set of small, self-contained papercuts, each
mostly extending a surface that already exists — a good place to
start. For anything bigger, file or claim an issue first so the
shape can be agreed before code is written. The full contribution
loop is in [CONTRIBUTING.md](CONTRIBUTING.md).
