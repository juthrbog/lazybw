<h1 align="center">lazybw</h1>

<p align="center">
  A keyboard-driven terminal UI for <a href="https://bitwarden.com">Bitwarden</a>, built with Go and the <a href="https://charm.sh">Charm</a> ecosystem.
</p>

<p align="center">
  <a href="https://github.com/juthrbog/lazybw/actions/workflows/ci.yml"><img src="https://github.com/juthrbog/lazybw/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/juthrbog/lazybw/releases/latest"><img src="https://img.shields.io/github/v/release/juthrbog/lazybw" alt="Release"></a>
  <a href="https://github.com/juthrbog/lazybw/blob/main/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/juthrbog/lazybw" alt="Go"></a>
  <a href="https://goreportcard.com/report/github.com/juthrbog/lazybw"><img src="https://goreportcard.com/badge/github.com/juthrbog/lazybw" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License"></a>
</p>

```
 󰦝 lazybw  user@example.com  Login
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄

  󰌾  Gmail                                                user@gmail.com
  󰌾  GitHub                                              dev@example.com
▶ 󰌾  AWS Console                                               aws-admin
  󰿯  Visa Debit                                                •••• 4242
  󰎞  Recovery codes                                               (note)
  󰀄  Passport                                                   Alex Doe
  󰣀  Deploy key                                          SHA256:k3Yd9QzR
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
  󰌾  AWS Console                                                  Login
  Username    aws-admin  [u] copy
  Password    ••••••••••  [c] copy
  TOTP        843 291  ██▊░ 21s  [t] copy
  URL         https://console.aws.amazon.com  [o] open
  Notes       root account  [n] copy

 j/k navigate · / search · c pwd · t totp · ? help        synced 5m ago
```

## Why lazybw?

The Bitwarden desktop app is not available on every platform. The Bitwarden CLI (`bw`) has many functions, but it was not made for daily use. To get a password, you pipe `bw list items` through `jq`, search for the right entry, and copy the result. If you do that ten times a day, you start to write wrapper scripts.

lazybw shows your vault in a terminal UI that you control with the keyboard. You do not have to leave the terminal. You search, browse, and copy passwords and TOTP codes with a few key presses.

## Features

- You navigate, search and copy with the keyboard. lazybw does not capture the mouse, so the click-and-drag text selection of your terminal works.
- One key copies a field: `c` copies the password, `t` the TOTP code and `u` the username.
- The detail panel below the list (the drawer) shows the current TOTP code and a countdown bar. The bar changes from green to yellow to red as the time runs out.
- lazybw shows logins, secure notes, cards, identities and SSH keys. Each item type has its own detail view.
- Press `/` to filter the vault with a fuzzy search while you type.
- Press `ctrl+g` to put items with similar names into groups. `enter` expands or collapses a group.
- For an item with master password re-prompt, lazybw asks for the master password before it copies a protected field.
- The session token stays in memory only. lazybw clears the clipboard 60 seconds after a copy, and also when you lock or quit. The vault locks after 15 minutes with no key press. See [SECURITY.md](SECURITY.md).
- lazybw is a single binary. At run time, it needs only `bw`.

## Requirements

- The [Bitwarden CLI (`bw`)](https://bitwarden.com/help/cli/#download-and-install), installed and on your `$PATH`.
- A terminal with Unicode support, and a [Nerd Font](https://www.nerdfonts.com) v3 as the terminal font. Use a Mono variant of the font, so that each icon is one cell wide. lazybw uses Nerd Font icons for the item types, the header logo, the copy message and the lock and unlock spinners. Without a Nerd Font, these icons show as empty boxes. [docs/GLYPHS.md](docs/GLYPHS.md) lists each icon.
- A terminal that supports OSC 52, the escape sequence that sets the clipboard. lazybw copies only through OSC 52. The terminal sends no reply, so lazybw cannot tell if a copy worked. If a copy does not reach your clipboard, turn on OSC 52 in the configuration of your terminal.
- In tmux, add `set -g set-clipboard on` to your tmux configuration. With the default value, tmux does not pass the copy to your terminal. tmux also keeps each copy in a paste buffer, and lazybw does not delete these buffers.
- Go 1.26.0 or later, to build from source. This is the `go` line in `go.mod`. With an older Go and the default `GOTOOLCHAIN=auto`, the `go` command downloads a newer Go first. A build in a clone of the repository also follows the `toolchain go1.27.2` line in `go.mod`, so it uses Go 1.27.2 or later.

## Installation

### Download a binary

Download the archive for your system from the [releases page](https://github.com/juthrbog/lazybw/releases/latest). Releases have binaries for Linux and macOS, on amd64 and arm64. Extract the archive, and put `lazybw` in a directory on your `$PATH`, for example `/usr/local/bin`. Each release has a `checksums.txt` file with the SHA-256 checksum of each archive.

### From source

```sh
go install github.com/juthrbog/lazybw@latest
```

### Build locally

```sh
git clone https://github.com/juthrbog/lazybw.git
cd lazybw
go build -o lazybw .
```

## Usage

```sh
lazybw
```

At start, lazybw runs `bw status`. If you are not logged in, lazybw shows the login form. Otherwise, it shows the unlock form, which asks for your master password.

### Two-step login and new-device verification

If your account uses two-step login, lazybw asks for the method and then for the code after you enter your email and master password. lazybw supports an authenticator app, email and YubiKey OTP. If you use email, Bitwarden sends the code to your email address.

Bitwarden can also ask for a new-device code. It does this when an account without two-step login logs in from a new device, for example a new install of `bw`. Bitwarden sends this code by email, but `bw` has no option that takes it, so lazybw cannot pass it to `bw`. To finish the new-device verification, run `bw login` once in a terminal and enter the code from the email. Then start lazybw again.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--idle-timeout` | `15m` | Lock the vault after this time with no key press. The value is a Go duration, for example `10m` or `1h30m`. The minimum is `30s`. For a smaller value, lazybw prints a warning and uses `30s`. |
| `--theme` | `catppuccin-mocha` | The color theme. See [Themes](#themes). |
| `--debug` | `false` | Write a debug log. See [Debug log](#debug-log). |
| `--version` | | Print the version and exit. |

You can also set the theme with the `LAZYBW_THEME` environment variable. The `--theme` flag overrides it. For a name that is not a theme, lazybw prints a warning with the list of themes and uses `catppuccin-mocha`.

A release binary prints its release version for `--version`. A `go install` build prints its module version, for example `lazybw v0.3.0`. A build with no version information prints `lazybw dev`.

### Debug log

With `--debug`, lazybw adds log lines to `$XDG_CACHE_HOME/lazybw/debug.log`. If `XDG_CACHE_HOME` is not set, the file is `~/.cache/lazybw/debug.log`. lazybw creates the file with mode 0600.

The log has a line for each state change. Examples are the `bw status` result, an unlock, the item count after a load, each lock, the `bw lock` result, sync results and errors. These lines do not contain the session token, the master password, the email, item names or field values. For a failed `bw` command, the log also has the stderr of `bw`, with the token, the password, the two-step code and the email hidden. See [SECURITY.md](SECURITY.md#debug-log).

### Themes

Available themes: `catppuccin-mocha` (default), `catppuccin-frappe`, `catppuccin-macchiato`, `catppuccin-latte`, `dracula`, `charm`, `base16`.

Each theme has colors for a dark and for a light terminal background. lazybw asks the terminal for its background color and uses the matching colors. In the vault, press `T` to open the theme picker. The theme that you choose stays until lazybw exits.

### Keybindings

| Key | Action |
|---|---|
| `j` / `k` | Move down / up |
| `/` | Filter the list. `enter` accepts the filter, and `esc` clears it. |
| `c` | Copy the password. For a card, copy the number. For an identity, copy the SSN. For an SSH key, copy the private key. |
| `t` | Copy the TOTP code |
| `u` | Copy the username. For an identity, copy the email. For an SSH key, copy the public key. |
| `n` | Copy the note |
| `v` | Copy the card security code |
| `o` | Open the URL in the browser |
| `p` | Open the password generator |
| `r` | Sync the vault |
| `l` | Lock the vault |
| `?` | Show the help overlay |
| `q` | Quit. lazybw asks for confirmation first: `y` locks the vault and quits, and `n` or `esc` cancels. |
| `ctrl+c` | Lock the vault and quit at once, with no confirmation |

On the unlock screen, `ctrl+q` quits, because `q` types into the password field.

For an item with master password re-prompt, `c`, `t`, `v` and `n` ask for the master password first. `enter` submits the password, and `esc` cancels. After the correct password, lazybw does the copy. It does not ask again while the item stays selected.

The help overlay (`?`) shows the vault keys in four groups: Copy, Navigation, Vault and UI. Press `/` in the overlay to filter it. The overlay does not show the keys of the password generator, the filter input or the theme picker. The footer shows these keys while each one is open. See [docs/KEYBINDINGS.md](docs/KEYBINDINGS.md) for the full reference.

## Contributing

lazybw is in early development. It does not accept contributions yet. To report a security problem, see [SECURITY.md](SECURITY.md#reporting-a-vulnerability).

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
