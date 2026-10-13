# VHS tapes

Scripted recordings of the five cli.md §4 human tasks plus a large-board
scroll/search tape (tui.md §7). Generate with
[charmbracelet/vhs](https://github.com/charmbracelet/vhs) once `boardwalk`
is on PATH:

```sh
for t in tapes/*.tape; do vhs "$t"; done
```

The owner reviews the GIFs before each release. The human completion gate
(owner plus 2–3 target users, no coaching) is separate and cannot be
automated; these tapes are its rehearsal and the README/demo media.

**Warning:** tapes 02–05 perform real mutations (add, move, edit) against
the selected project, and tape 05 presses `L` in the TUI. Render them
against a scratch board, or rehearse the same flows with `--dry-run` /
a throwaway item — never against a board you care about. The build
environment deliberately did not render them for this reason, in
addition to `vhs`/`ttyd` not being installed there.

Note: `vhs` was not installed in the build environment that authored these
tapes, so the GIFs have not been rendered yet; run the loop above to
produce them.
