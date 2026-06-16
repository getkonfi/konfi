Output assets/konfi-usage.gif

# generic konfi usage demo. keep app-specific showcases as separate blocks so
# a hyprland segment can be added later without changing the base flow.
Require bash
Require ./konfi

Set Shell "bash"
Set FontFamily "JetBrains Mono"
Set FontSize 30
Set Width 2560
Set Height 1440
Set LineHeight 1.15
Set Theme "Catppuccin Mocha"
Set Padding 24
Set WindowBar Colorful
Set Framerate 18
Set PlaybackSpeed 1.0
Set TypingSpeed 70ms
Set LoopOffset 12%

# launch in a clean temp environment, but do not record shell setup noise.
Hide
Type "clear"
Enter
Type "tools/demo/konfi-demo.sh"
Enter
Sleep 1600ms
Show

# dashboard: detected apps, versions, and supported-but-missing apps.
Sleep 1800ms

# open kitty through sidebar search.
Type "/"
Type "kitty"
Sleep 900ms
Enter
Sleep 1800ms

# inspect font family, then change it through the font picker.
Down
Sleep 900ms
Right
Sleep 1200ms
Left
Sleep 700ms
Enter
Sleep 1000ms
Type "Jet"
Sleep 1000ms
Enter
Sleep 1100ms

# change a numeric field.
Down@650ms 4
Sleep 800ms
Enter
Sleep 800ms
Backspace@180ms 4
Sleep 400ms
Type "14.0"
Sleep 900ms
Enter
Sleep 1100ms

# jump to a larger enum and change it.
Type "/"
Type "powerline"
Sleep 1000ms
Enter
Sleep 700ms
Enter
Sleep 1300ms
Down@650ms 3
Sleep 900ms
Enter
Sleep 1200ms

# show all pending changes without saving.
Ctrl+S
Sleep 2200ms
Escape
Sleep 900ms
Ctrl+C
Sleep 500ms
