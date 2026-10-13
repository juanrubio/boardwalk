# Boardwalk — Product Requirements Document

**Repository:** gh-boardwalk  
**Working name / executable:** Boardwalk / boardwalk  
**Status:** Initial product brief (draft)  
**Date:** 2026-10-13

## 1. Summary

Boardwalk is a terminal-first tool for managing GitHub Projects items without opening the browser. It should make it quick to find a project, file work onto it, edit item details and project fields, and understand the board at a glance. The default experience is a friendly CLI; an interactive terminal UI (TUI) provides visual kanban and list views.

The MVP targets GitHub Projects (Projects v2) and reuses the user's authenticated GitHub CLI session rather than storing another credential.

## 2. Problem

GitHub Projects is useful for tracking work, but routine actions—finding a board, adding an issue, changing status or priority, and scanning columns—often require several browser interactions. This interrupts terminal-based work and makes small updates feel heavier than they should.

Boardwalk should make these common operations quick, discoverable, and safe from the command line.

## 3. Product principles

- Terminal-native: everyday use must not require a browser.
- Fast for the common case: project aliases, sensible defaults, concise output, and keyboard navigation.
- Respect GitHub's model: operate on real Projects v2 items and fields; do not create a parallel task system.
- Safe by default: preview mutations, make destructive actions explicit, and surface API errors.
- Useful to humans and scripts: readable output by default, structured JSON on request.

## 4. Users and jobs to be done

**Primary user:** a developer or project maintainer who already uses GitHub Projects and spends much of their working day in a terminal.

They need to:
1. See a chosen project's work grouped by status.
2. Add an existing issue or pull request to a project, or file a draft item quickly.
3. Edit an item's title/body and project-specific fields such as status, priority, assignee, dates, or iteration.
4. Move an item between workflow columns without hunting through the web UI.
5. Repeat these actions across personal and organization-owned projects.

## 5. Goals and success measures

### Goals
- Make the main add, inspect, edit, and move flows possible entirely in the terminal.
- Support multiple projects with short, memorable local aliases.
- Display custom project fields accurately, including fields whose options differ by project.
- Reuse GitHub CLI authentication and permissions.

### MVP success measures
- A first-time user with gh already authenticated can select a project and list its items in under five minutes.
- A user can add an issue or PR, change its status, and verify the result without a browser.
- At least 90% of common item-management work can be done with one command or one short TUI interaction per action.
- Mutations fail clearly and safely when permissions, field options, or network access are insufficient.

## 6. Scope

### MVP — must have

1. **Project discovery and selection**
   - List accessible Projects v2 owned by a user or organization.
   - Select a project interactively or by owner, project number, or URL.
   - Save local aliases and a default project in user config.
2. **Browse and visualize**
   - List items with title, type, repository, status, assignee, and other configured fields.
   - Filter by text, status, assignee, repository, and item type.
   - Paginate transparently and keep large boards usable.
   - Provide a TUI kanban grouped by a selected single-select field (normally Status), plus a compact list/detail view.
3. **File work**
   - Add an existing issue or pull request using OWNER/REPO#NUMBER or a URL.
   - Create a draft project item with a title and optional body.
4. **Edit and move work**
   - Edit an item's title/body where GitHub permits it.
   - Set project fields through a generic field/value interface; support common typed fields such as single-select, date, number, text, iteration, and people where the API allows.
   - Move an item to another status/column from the CLI or TUI.
5. **Auth, output, and safety**
   - Use existing gh authentication; explain missing project/repository scopes and the relevant gh auth refresh command.
   - Never persist GitHub access tokens in Boardwalk config or logs.
   - Offer human-readable output and a JSON mode for scripts.
   - Provide dry-run for mutations; require explicit confirmation for archive/remove/destructive actions.
   - Return non-zero exit codes and actionable messages on failure.

### Out of scope for MVP
- Creating or deleting GitHub projects, or administering their fields and workflows.
- Replacing GitHub Issues, discussions, notifications, or repository administration.
- Supporting legacy Projects (classic) boards.
- Offline editing, conflict resolution, or a hosted service.
- Full issue triage (comments, reactions, review workflows) beyond project-item management.

## 7. Proposed command shape

The exact names are open to UX testing; these examples describe the intended capability:

~~~text
boardwalk project list --owner juanrubio
boardwalk project use life-board
boardwalk item list --status "In Progress" --assignee @me
boardwalk item add juanrubio/snazzyemail#42
boardwalk item add --draft --title "Investigate export flow"
boardwalk item edit ITEM --field "Status=In Progress" --field "Priority=High"
boardwalk board
boardwalk board --project life-board
~~~

Provide help, shell completion, and a non-interactive mode suitable for scripts. Validate exact verbs and aliases with a small usability test before implementation.

## 8. Key user flows

### A. First run
1. Verify gh is installed and authenticated.
2. Explain missing scopes; never request or store credentials directly.
3. List accessible projects and let the user choose one.
4. Optionally save an alias and make it the default.

### B. File an item
1. User supplies an issue/PR reference or draft title/body.
2. Boardwalk resolves the project and previews the item and requested fields.
3. In interactive mode, the user confirms; dry-run prints the planned mutation without applying it.
4. Boardwalk adds it and prints a stable link and resulting item ID.

### C. Update from the board
1. TUI loads items and current field options.
2. User filters/searches, opens an item detail panel, or moves an item between columns.
3. Boardwalk shows pending changes and applies them to GitHub on confirmation.
4. The view refreshes or clearly reports a conflict/error.

## 9. Functional requirements

- Resolve a project from an alias, URL, owner/number, or interactive selector.
- Fetch project fields and allowed options dynamically; never assume every board uses the same status labels.
- Distinguish linked issues, pull requests, and draft items in output.
- Keep field editing explicit and validate values before mutation.
- Surface pagination, rate limits, permission errors, archived items, and unsupported field types clearly.
- TUI must work at typical terminal widths and remain usable with keyboard-only navigation.
- Structured output has a stable schema and omits terminal decoration when piped.

## 10. Non-functional requirements

- **Security:** rely on gh's credential store; redact tokens and sensitive request data from logs.
- **Reliability:** retry transient read failures with bounded backoff; do not silently retry a mutation that may already have succeeded.
- **Performance:** show progress for network operations; avoid repeated full-board requests for a single field update. Specific latency, round-trip limits, and rendering budgets are defined in [performance.md](docs/specs/performance.md) (Amendment 1). Local caching and snapshot rules are defined in [ADR 0011](docs/decisions/0011-local-cache-and-snapshot.md).
- **Compatibility:** document supported OSes and minimum GitHub CLI version once the implementation stack is chosen.
- **Accessibility:** use sensible contrast, never convey status by color alone, support keyboard navigation, and provide a plain-list fallback.

## 11. Technical assumptions and dependencies

- GitHub Projects v2 is accessed through the GitHub GraphQL API.
- The GitHub CLI (gh) is installed and authenticated. Boardwalk may call gh api graphql or use an equivalent supported client, but must not copy tokens into its own files.
- Project/item IDs and custom-field schemas are discovered at runtime.
- Public and private project access follows the authenticated account's permissions.

## 12. Risks and mitigations

- **Custom-field variety:** implement typed field adapters and test representative schemas; return a precise unsupported-field message.
- **Permissions:** distinguish missing scopes from inaccessible projects and explain the smallest practical remediation.
- **GraphQL rate limits/network failures:** expose retry/rate-limit information and avoid duplicate writes after uncertain outcomes.
- **TUI complexity:** keep list/CLI commands fully capable so the TUI is an enhancement, not a requirement.
- **Accidental changes:** preview edits, support dry-run, and reserve yes/no bypasses for explicit automation.

## 13. Open questions

1. ~~Which implementation language best fits the existing gh workflow and desired TUI: Go, Rust, or TypeScript?~~ **Resolved: Go** — see [ADR 0001](docs/decisions/0001-implementation-language-go.md) (Final).
2. ~~Should the MVP use gh api graphql as a subprocess, or a language-level GitHub client authenticated through gh?~~ **Resolved: language-level Go client via `cli/go-gh`** — see [ADR 0002](docs/decisions/0002-github-api-access-go-gh.md) (Final).
3. ~~What command grammar feels most natural: boardwalk item, boardwalk issue, or shorter aliases?~~ **Resolved: noun-first (`item`, `project`, `board`), no short aliases in v1** — see [ADR 0003](docs/decisions/0003-command-grammar-noun-first.md) (Final).
4. ~~Which projects and custom fields should be the first real-world acceptance fixtures?~~ **Resolved: three-board set (rich-fields, differing-schema, synthetic 100+), captured for offline tests; slots are `juanrubio/1` SnazzyEmail and `juanrubio/3` Life Board** — see [ADR 0004](docs/decisions/0004-acceptance-fixtures-three-boards.md) (Final).
5. ~~Should aliases live only in user config, or also support repository-local config?~~ **Resolved: user config only for MVP; resolution order flag → `BOARDWALK_PROJECT` → saved default** — see [ADR 0005](docs/decisions/0005-alias-config-user-only.md) (Final).
6. ~~How much of the TUI belongs in MVP versus a fast table/list first?~~ **Resolved: full item management in the TUI for the MVP (browse, detail, create, edit, move), CLI fully capable in parallel** — see [ADR 0006](docs/decisions/0006-tui-scope-full-in-mvp.md) (Final).

## 14. Initial release definition

The MVP is ready when a user can authenticate through gh, discover/select a Projects v2 board, view and filter its items in CLI and TUI, add an issue/PR or draft item, update common fields/status, and do all of that without opening a browser—while receiving clear previews, errors, and safe handling of destructive operations.

*(A1)* In addition, the MVP must pass the visual acceptance requirements ([tui.md §7](docs/specs/tui.md), golden tests and VHS verification) and adhere to the performance budgets ([performance.md](docs/specs/performance.md) B1–B9) tested offline across all three fixture tiers.
