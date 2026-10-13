# ADR 0010 — Auth scopes: require `project` + `repo` up front

**Status:** Final — amended by A1 (2026-10-13)
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Amended:** 2026-10-13 — A1, owner approval of the
[beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md)
**Date:** 2026-10-13
**Deciders:** Project owner (accepted option B in decision interview, 2026-10-13)
**Depends on:** [ADR 0002](0002-github-api-access-go-gh.md) (Final), [ADR 0009](0009-json-error-contract-bare.md) (Final)

## Context

The PRD requires Boardwalk to use existing `gh` authentication, explain
missing project/repository scopes, and name the relevant
`gh auth refresh` command (§6.5, §8 flow A), never requesting or storing
credentials directly (§8). ADR 0009 assigns auth/scope failures exit
code 3.

For GitHub Projects v2, reading projects needs `read:project`, writing
them needs the broader `project` scope, and accessing private repository
content (the issues/PRs inside a project) needs `repo`. The open choice
was when to require these: on demand per operation, all up front at
first run, or not proactively at all. The interviewer's recommendation
was on-demand (option A); the owner chose **up front (option B)**.

## Decision

Every GitHub-backed Boardwalk operation requires, before it runs:

1. `gh` installed;
2. an authenticated `gh` session;
3. the token scopes **`project`** (which includes `read:project`) and
   **`repo`**.

If any check fails, the command stops with **exit code 3** (ADR 0009)
and a single remediation message. For missing/insufficient scopes the
message is exactly the one command the user needs, covering both scopes
at once:

```text
gh auth refresh -s project,repo
```

(plus `gh auth login` guidance when `gh` is missing or unauthenticated).
Boardwalk only *prints* this command; it never runs it, and never
requests, receives, or stores credentials itself (PRD §8, ADR 0002).

*(A1)* **Performance constraint (no dedicated auth round trip):**
Boardwalk resolves the auth token once per process from `gh`'s credential
store without spawning repeated subprocesses. It does **not** make an
independent network request just to verify scopes before doing work.
Instead, scopes are inspected from the `X-OAuth-Scopes` header returned
by the first real GraphQL request (or verified via local token inspection
where supported). If scopes are insufficient, the command stops with exit
code 3 and the remediation above.

Purely local operations that touch no GitHub data (for example showing
the config path or how aliases resolve locally) are not gated.

Permission failures on a specific project after scopes pass (the project
exists but this account cannot write it) remain distinct: exit code 5
with a permission message, not a scope message (ADR 0009, PRD §12).

## Rationale

- **Simplest possible onboarding:** One check, one command, one mental
  model. A first-time user fixes auth exactly once and never meets a
  mid-workflow scope failure on their first edit — the smooth first-run
  experience (PRD §5: project selected and items listed in under five
  minutes) matters more here than scope minimality.
- **Predictable behaviour:** Every command enforces the same gate, so
  error handling and documentation describe a single flow.

## Alternatives considered

- **Least privilege, on demand (option A, interviewer's
  recommendation):** Request `read:project` for reads and add `project` /
  `repo` only when an operation needs them. Rejected by the owner in
  favour of onboarding simplicity, accepting that read-only users grant
  write-level project access.
- **No proactive guidance (option C):** Surface raw API errors with a
  generic scopes message. Rejected: it breaks the PRD's actionable-
  remediation requirement (§6.5).

## Consequences

- This deliberately trades least-privilege for simplicity. It also
  relaxes PRD §12's "smallest practical remediation" wording for scopes:
  remediation is now *one standard command*, while the §12 requirement to
  distinguish missing scopes (exit 3) from inaccessible projects
  (exit 5) is unchanged and still mandatory.
- Documentation and first-run flow must state the required scopes and
  why (`project` for Projects v2 read/write, `repo` for private issue/PR
  content) before printing the refresh command — users should never be
  asked for scopes unexplained.
- If GitHub changes Projects v2 scope requirements, this ADR's scope
  list is the single place to update; the check must read the token's
  actual scopes (via `gh`) rather than inferring them from failures.
- Fine-grained tokens / GitHub Apps are not specially handled in the MVP;
  if scope inspection is unavailable for a token type, Boardwalk attempts
  the operation and classifies a scope-style failure as exit 3 with the
  same remediation text.

## Amendment log

- **A1 (2026-10-13)** — owner approval of the
  [beauty and speed review](../reviews/2026-10-13-beauty-and-speed.md):
  Specified that scope verification must not introduce a standalone network
  round trip; scopes are inspected via `X-OAuth-Scopes` on the first real
  request or via local token inspection, preserving performance budgets
  ([performance.md](../specs/performance.md) §2).
