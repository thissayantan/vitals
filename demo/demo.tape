# VHS script for the README demo gif. https://github.com/charmbracelet/vhs
#
# Generate:  vhs demo/demo.tape   →   docs/demo.gif
# Requires:  vhs, ttyd, ffmpeg, and `vitals` on PATH (go build -o vitals ./cmd/vitals).
#
# The demo shows, in order:
#   1. piping the sample session JSON into `vitals` (the rendered two lines)
#   2. launching `vitals config` and reordering / toggling a segment with live preview
#   3. `vitals doctor` confirming the environment

Output docs/demo.gif

Set Shell "bash"
Set FontSize 18
Set Width 1200
Set Height 400
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 60ms
Set PlaybackSpeed 1.0

# ── 1. render the status line from the sample session ────────────────────
Type "vitals < testdata/session.sample.json"
Sleep 600ms
Enter
Sleep 2.5s

# ── 2. the visual configurator with live preview ─────────────────────────
Type "vitals config"
Sleep 400ms
Enter
Sleep 1.5s

# move the cursor down a few segments
Down@500ms 3
Sleep 800ms

# reorder the highlighted segment (watch the live preview update)
Type "J"
Sleep 700ms
Type "J"
Sleep 900ms

# cycle the theme, then the charset
Type "t"
Sleep 900ms
Type "t"
Sleep 900ms
Type "c"
Sleep 900ms

# toggle a segment off
Down@400ms 2
Type " "
Sleep 1s

# quit the configurator
Type "q"
Sleep 800ms

# ── 3. doctor ────────────────────────────────────────────────────────────
Type "vitals doctor"
Sleep 400ms
Enter
Sleep 2.5s
