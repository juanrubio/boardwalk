# ADR 0008 — Config format: TOML in the OS config directory

**Status:** Final
**Accepted:** 2026-10-13 — owner acceptance of the settled decision-interview contract
**Date:** 2026-10-13
**Deciders:** Project owner (accepted option A in decision interview, 2026-10-13)
**Depends on:** [ADR 0005](0005-alias-config-user-only.md) (Final)

## Context

ADR 0005 settled that aliases and the default project live only in user
config for the MVP, with resolution order: command-line reference /
`--project` → `BOARDWALK_PROJECT` → saved default → interactive selector
or actionable error. It deferred the file's name, format, and schema.
Users will hand-edit this file to fix aliases, so it must be forgiving,
comment-friendly, and located where each OS expects app config.

## Decision

The config file is **`config.toml` in TOML format**, inside a
`boardwalk` directory under the OS-native user config directory resolved
via Go's `os.UserConfigDir()`:

- Linux: `~/.config/boardwalk/config.toml` (honouring `$XDG_CONFIG_HOME`)
- macOS: `~/Library/Application Support/boardwalk/config.toml`
- Windows: `%AppData%\boardwalk\config.toml`

Schema (v1):

```toml
# Boardwalk user config. Never put GitHub tokens here —
# authentication always comes from gh (see ADR 0002).
default = "life-board"

[aliases]
life-board = "juanrubio/3"
snazzy = "juanrubio/1"
```

- `default` names an alias from `[aliases]` (or, if no such alias exists,
  is itself interpreted as an owner/number or URL reference — resolution
  must say which happened; see Consequences).
- Each alias maps a short name to a project reference in owner/number
  form (`owner/number`) or a full GitHub project URL. Alias names are
  lowercase letters, digits, and hyphens.
- Unknown keys are ignored with a warning, never an error, so newer
  Boardwalk versions can extend the file safely. Malformed TOML is an
  actionable error naming the file and line — never a silent fallback
  that could open the wrong board.
- The file stores aliases and the default only. Credentials, tokens, and
  per-repository settings are forbidden keys; if present they are ignored
  and warned about, never used.

`boardwalk project use <alias>` creates the directory and file if needed
and updates them in place, preserving user comments where the TOML
writer reasonably can.

## Rationale

- **Forgiving to hand-edit:** TOML tolerates comments, is not
  whitespace-sensitive like YAML, and is far easier to fix than strict
  JSON — matching the forgiving feel the rest of the tool aims for.
- **Native location:** `os.UserConfigDir()` puts the file where each OS
  (and tools like backup/dotfiles managers) expect it, instead of
  inventing a home-directory convention.
- **Shape fits the data:** A string plus a string→string table is all the
  MVP needs (ADR 0005); TOML expresses exactly that with nothing left
  over.

## Alternatives considered

- **YAML (option B):** Equally expressive, but indentation errors in a
  hand-edited file silently change meaning instead of failing loudly.
  Rejected.
- **JSON (option C):** Trivial to parse, but no comments and strict
  punctuation make it the least pleasant to edit by hand. Rejected.

## Consequences

- First-run flow (PRD §8 flow A) creates `boardwalk/config.toml` only
  when the user saves an alias or default; read-only commands never
  create files.
- `boardwalk project list` should print the resolved config path and how
  each alias resolves, so the file is discoverable and debuggable.
- A missing file is normal (treated as empty config); a malformed file
  blocks project resolution with a precise error rather than guessing.
- Future keys extend this file under their own sections — the first is
  the per-project UI grouping preference defined in the
  [TUI specification](../specs/tui.md) §1. Repository-local config, if
  ever added, needs its own ADR (ADR 0005) and must not reuse this
  file's trust assumptions.

## Unresolved

- JSON/error contract — **resolved:** see
  [ADR 0009](0009-json-error-contract-bare.md) (Final).
- Auth scopes/remediation — **resolved:** see
  [ADR 0010](0010-auth-scopes-upfront.md) (Final).
- TUI design — **settled** in the
  [TUI specification](../specs/tui.md) (Final), all sections.
