Output assets/konfi-usage.webp

# focused konfi usage demo across hyprland, fuzzel, and ssh.
Require bash
Require ./konfi

Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font"
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

# open hyprland through sidebar search.
Type "/"
Type "hy"
Sleep 900ms
Enter
Sleep 1800ms

# change a hyprland enum from its option list.
Down@450ms 4
Sleep 700ms
Enter
Sleep 700ms
Down
Sleep 600ms
Enter
Sleep 1200ms

# change a hyprland number using arrow-key stepping.
Up@350ms 3
Sleep 700ms
Enter
Sleep 600ms
Up@450ms 3
Sleep 700ms
Enter
Sleep 1200ms

# save the hyprland changes and run the configured reload command.
Ctrl+S
Sleep 1600ms
Enter
Sleep 1800ms

# open ghostty and change its font family through the font picker.
Escape
Sleep 300ms
Escape
Sleep 500ms
Type "/"
Type "gh"
Sleep 700ms
Enter
Sleep 1600ms
Down
Sleep 700ms
Enter
Sleep 1500ms
Type "Jet"
Sleep 900ms
Down
Sleep 500ms
Enter
Sleep 1200ms

# save the font change.
Ctrl+S
Sleep 1500ms
Enter
Sleep 1200ms

# open ssh and select a certificate file path.
Escape
Sleep 300ms
Escape
Sleep 500ms
Type "/"
Type "ss"
Sleep 700ms
Up@40ms 20
Enter
Sleep 1600ms
Type "/"
Type "cer"
Sleep 900ms
Down
Sleep 500ms
Enter
Sleep 500ms
Enter
Sleep 700ms
Type "~/.ssh/id"
Sleep 500ms
Tab
Sleep 1400ms
Enter
Sleep 700ms
Enter
Sleep 1300ms
Escape
Sleep 500ms

# add a new ssh host group and set its user.
Enter
Sleep 900ms
Type "a"
Sleep 600ms
Enter
Sleep 500ms
Type "konfi.local"
Sleep 900ms
Enter
Sleep 1600ms
Enter
Sleep 700ms
Type "a"
Sleep 500ms
Down
Sleep 400ms
Enter
Sleep 600ms
Type "deploy"
Sleep 700ms
Enter
Sleep 1000ms
Escape
Sleep 600ms
Escape
Sleep 1000ms

# save the ssh file selection and host edits.
Ctrl+S
Sleep 1800ms
Enter
Sleep 1700ms

# cycle themes as the final step.
Type "t"
Sleep 900ms
Type "t"
Sleep 900ms
Type "t"
Sleep 1200ms
Ctrl+C
Sleep 500ms
