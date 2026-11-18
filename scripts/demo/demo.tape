# The README demo of lazybw. It runs lazybw against the fake bw in this
# directory, so it needs no Bitwarden account.
#
# Run it from the root of the repository with `task demo`.

Output docs/images/lazybw.gif

Require go

Set Shell bash
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 18
Set Width 1200
Set Height 720
Set Padding 16
Set Theme "Catppuccin Mocha"
Set TypingSpeed 80ms
Set CursorBlink false

Hide
Type "go build -o dist/lazybw . && export PATH=$PWD/scripts/demo:$PWD/dist:$PATH && clear"
Enter
Sleep 5s
Show

# Unlock. The fake bw takes any master password.
Type "lazybw"
Enter
Sleep 1.5s
Type "not-my-real-password"
Sleep 500ms
Enter
Sleep 2s

# Move through the list. The drawer shows the selected item.
Down
Sleep 1.2s
Down
Sleep 1.2s
Down
Sleep 2.5s

# Copy the password and the TOTP code.
Type "c"
Sleep 2s
Type "t"
Sleep 2s

# Search with a fuzzy filter.
Type "/"
Sleep 500ms
Type "visa"
Sleep 1.5s
Enter
Sleep 2.5s
Escape
Sleep 1.5s

# Open the form for a new login, then cancel.
Type "n"
Sleep 1.5s
Type "l"
Sleep 2.5s
Escape
Sleep 1.5s

# Open the password generator, then close it.
Type "p"
Sleep 3s
Escape
Sleep 1.5s
