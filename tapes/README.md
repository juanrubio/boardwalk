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

Note: `vhs` was not installed in the build environment that authored these
tapes, so the GIFs have not been rendered yet; run the loop above to
produce them.
