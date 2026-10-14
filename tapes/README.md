# VHS tapes

Scripted recordings of the five cli.md §4 human tasks plus a large-board
scroll/search tape (tui.md §7), and `showcase.tape`, the read-only board
tour embedded in the README. Generate with
[charmbracelet/vhs](https://github.com/charmbracelet/vhs) once `boardwalk`
is on PATH (`vhs` also needs `ttyd` and `ffmpeg`):

```sh
cd tapes && vhs showcase.tape   # writes ../docs/media/showcase.gif
for t in tapes/0*.tape; do vhs "$t"; done
```

The owner reviews the GIFs before each release. The human completion
gate was completed by the owner (see
[the run-sheet outcome](../docs/reviews/human-task-gate.md)); these
tapes remain its rehearsal and the README/demo media.

**Warning:** tapes 02–05 perform real mutations (add, move, edit) against
the selected project, and tape 05 presses `L` in the TUI. Render them
against a scratch board, or rehearse the same flows with `--dry-run` /
a throwaway item — never against a board you care about.
`showcase.tape` is read-only and safe against a real board.

Note: tapes 01–06 were authored against an older VHS parser (combined
`Set` lines and `;` separators); `showcase.tape` uses the current
one-command-per-line syntax. Modernise a tape's syntax before rendering
it with a current `vhs`.
