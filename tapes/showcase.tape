# README showcase (read-only, sanitized): open the demo board,
# navigate, peek at detail, toggle density, cycle to the next board
# and back, open the board picker and help. Recorded against the
# fixture-derived demo data in tapes/demo via a demo-enabled build
# (see tapes/README.md) — no real board is touched.
Output ../docs/media/showcase.gif
Set Width 1280
Set Height 760
Set FontSize 15
Hide
Type "export BOARDWALK_DEMO_DIR=$PWD/demo"
Enter
Show
Type "boardwalk board --project acme/1"
Enter
Sleep 4s
Type "j"
Sleep 700ms
Type "j"
Sleep 700ms
Type "l"
Sleep 900ms
Enter
Sleep 2s
Escape
Sleep 800ms
Type "d"
Sleep 1200ms
Type "d"
Sleep 900ms
Type "]"
Sleep 3s
Type "["
Sleep 3s
Type "P"
Sleep 1500ms
Escape
Sleep 800ms
Type "?"
Sleep 1800ms
Escape
Sleep 600ms
Type "q"
Sleep 1s
