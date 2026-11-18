<h1 align="center">lazybw</h1>

<p align="center">
  A keyboard-driven terminal UI for your <a href="https://bitwarden.com">Bitwarden</a> vault.<br>
  It runs on top of the official <a href="https://bitwarden.com/help/cli/">Bitwarden CLI (<code>bw</code>)</a>.
</p>

<p align="center">
  <a href="https://github.com/juthrbog/lazybw/actions/workflows/ci.yml"><img src="https://github.com/juthrbog/lazybw/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/juthrbog/lazybw/releases/latest"><img src="https://img.shields.io/github/v/release/juthrbog/lazybw" alt="Release"></a>
  <a href="https://github.com/juthrbog/lazybw/blob/main/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/juthrbog/lazybw" alt="Go"></a>
  <a href="https://goreportcard.com/report/github.com/juthrbog/lazybw"><img src="https://goreportcard.com/badge/github.com/juthrbog/lazybw" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License"></a>
</p>

<p align="center"><img src="docs/images/lazybw.gif" alt="lazybw demo: unlock the vault, copy a password and a TOTP code, filter the list, open the new item form and the password generator" width="800"></p>

## Why lazybw?

lazybw shows your vault in one screen. You type a few letters to find an item, and one key copies the password or the TOTP code. With `bw` alone, you run `bw list items`, filter the output with `jq`, and copy the result yourself.

lazybw does not talk to the Bitwarden servers itself. It runs `bw` for you. The session token stays in memory, and lazybw clears the clipboard 60 seconds after a copy.

## Features

- Find an item fast (`/`). You type a few letters, and the list shows the items whose names have those letters in the same order.
- Copy the password (`c`), the TOTP code (`t`) or the username (`u`) with one key.
- See the current TOTP code with a countdown bar that goes from green to yellow to red. lazybw reads base32 keys, `otpauth://` URIs and Steam Guard (`steam://`) keys.
- Create (`n`), edit (`e`) and move to the trash (`d`) logins and secure notes.
- Make passwords and passphrases with the generator (`p`). The item form can put a generated password in the password field.
- Browse logins, secure notes, cards, identities and SSH keys. Each type has its own detail view.
- Group items with the same base name (`ctrl+g`). For example, "GitHub" and "GitHub (Work)" go in one group.
- Choose from 7 color themes, each with colors for a dark and a light terminal.
- Clear the clipboard 60 seconds after a copy, and lock the vault after 15 minutes with no key press. See [SECURITY.md](SECURITY.md).

lazybw does not do these things yet: restore from the trash, organizations and collections, more than one account, an SSH agent and Bitwarden Send.

## Requirements

- The [Bitwarden CLI (`bw`)](https://bitwarden.com/help/cli/#download-and-install) on your `$PATH`. lazybw is one binary, and at run time it needs only `bw`.
- A [Nerd Font](https://www.nerdfonts.com) v3 Mono in your terminal, for the icons. Without one, run lazybw with `--ascii`.
- A terminal that supports OSC 52, the escape sequence that sets the clipboard. Most current terminals support it, but some turn it off by default. In tmux, add `set -g set-clipboard on` to your tmux configuration.

## Installation

### Homebrew

On macOS or Linux, this command installs `bw` and lazybw:

```sh
brew install bitwarden-cli juthrbog/tap/lazybw
```

### Download a binary

Download the archive for your system from the [releases page](https://github.com/juthrbog/lazybw/releases/latest). Releases have binaries for Linux, macOS and Windows, on amd64 and arm64. Extract the archive, and put `lazybw` (`lazybw.exe` on Windows) in a directory on your `$PATH`. Then [install `bw`](https://bitwarden.com/help/cli/#download-and-install).

Each release is signed and has an SBOM (a list of the modules in the binary) for each archive. To make sure that a download is genuine, see [Verify a download](docs/INSTALL.md#verify-a-download).

### With Go

This needs Go 1.26 or later. It does not install `bw`.

```sh
go install github.com/juthrbog/lazybw@latest
```

For a build from a clone, see [docs/INSTALL.md](docs/INSTALL.md#build-from-a-clone).

## Quick start

1. If your account is on the EU cloud, run `bw config server https://vault.bitwarden.eu`. If your account is on a self-hosted server, run `bw config server <url>` with the URL of your server. Do this before you start lazybw, because lazybw has no server setting.
2. Run `lazybw`.
3. If you are not logged in, enter your email and master password. If your account uses two-step login, lazybw then asks for the method and the code. If Bitwarden asks for a new-device code, see [Bitwarden asks for a new-device code](#bitwarden-asks-for-a-new-device-code). If you are logged in, enter your master password to unlock.
4. Type `/` and a few letters of an item name. Press `enter` to keep the filter.
5. Press `c` to copy the password, or `t` to copy the TOTP code.
6. Press `q` and then `y` to lock the vault and quit.

lazybw supports two-step login with an authenticator app, email and YubiKey OTP. It does not support SSO or login with an API key. For those, log in with `bw login` first, then start lazybw and unlock with your master password. lazybw cannot unlock an account that has no master password.

## Keybindings

These are the keys that you use most in the vault:

| Key | Action |
|---|---|
| `j` / `k` | Move down / up |
| `/` | Filter the list. `esc` clears the filter. |
| `c` | Copy the password (or the card number, SSN or SSH private key) |
| `t` | Copy the TOTP code |
| `u` | Copy the username (or the identity email or SSH public key) |
| `m` | Copy the note |
| `v` | Copy the card security code |
| `o` | Open the first URL of the item in the browser |
| `n` | Create a login (`l`) or a secure note (`s`) |
| `e` | Edit the selected login or secure note |
| `d` | Move the selected item to the trash, after you press `y` to confirm |
| `p` | Open the password generator |
| `ctrl+g` | Turn grouping on or off. `enter` expands or collapses a group. |
| `r` | Sync the vault |
| `l` | Lock the vault |
| `T` | Open the theme picker |
| `?` | Show the vault keys |
| `q` | Lock the vault and quit, after you press `y` to confirm |
| `ctrl+c` | Lock the vault and quit at once |

In the item form, `tab` goes to the next field, `ctrl+s` saves and `esc` closes the form. If you changed a field, lazybw asks before it discards the changes. `ctrl+g` puts a generated password in the password field, and `ctrl+r` shows or hides the password.

Create and edit work only for logins and secure notes. lazybw shows cards, identities and SSH keys, but you cannot change them in lazybw.

`d` does not delete an item permanently. You can restore the item from the trash in the Bitwarden web vault or app for 30 days.

On the unlock screen, `ctrl+q` quits, because `q` types into the password field.

Press `?` in lazybw for the vault keys. The footer shows the keys of the other screens. For every key, see [docs/KEYBINDINGS.md](docs/KEYBINDINGS.md).

## Themes

The themes are `catppuccin-mocha` (default), `catppuccin-frappe`, `catppuccin-macchiato`, `catppuccin-latte`, `dracula`, `charm` and `base16`. lazybw asks the terminal for its background color and uses the dark or the light colors of the theme.

In the vault, press `T` to try the themes. lazybw forgets this choice when it exits. To keep a theme, set `LAZYBW_THEME` in your shell profile, for example `export LAZYBW_THEME=dracula`.

## Configuration

| Flag | Environment variable | Default | Description |
|---|---|---|---|
| `--idle-timeout` | | `15m` | Lock the vault after this time with no key press. The value is a Go duration, for example `10m` or `1h30m`. The minimum is `30s`. |
| `--theme` | `LAZYBW_THEME` | `catppuccin-mocha` | The color theme. The flag overrides the variable. |
| `--ascii` | `LAZYBW_ASCII` | `false` | Draw only ASCII characters. Use it if you have no Nerd Font, or if icons or rows do not line up. See [docs/GLYPHS.md](docs/GLYPHS.md#ascii-mode). |
| `--debug` | | `false` | Write a debug log. See [Troubleshooting](#troubleshooting). |
| `--version` | | | Print the version and exit. |

If the theme name does not exist, lazybw uses `catppuccin-mocha`. It also prints a warning with the list of themes.

If the idle timeout is over, the next key press locks the vault. lazybw does not do the action of that key.

## Troubleshooting

### A copy does not reach the clipboard

lazybw copies only through OSC 52, and the terminal sends no reply. So lazybw cannot tell if a copy worked.

1. Turn on OSC 52 clipboard access in the configuration of your terminal.
2. If you use tmux, add `set -g set-clipboard on` to your tmux configuration. With the default value, tmux does not pass the copy to your terminal.

### Bitwarden asks for a new-device code

Bitwarden asks for this code when an account without two-step login logs in from a new device. A new install of `bw` counts as a new device. `bw` has no option that takes the code, so lazybw cannot pass it.

1. Run `bw login` once in a terminal, and enter the code from the email.
2. Start lazybw again.

### Icons show as empty boxes

Set a Nerd Font v3 Mono as your terminal font, or run `lazybw --ascii`.

### `o` does not open the URL

lazybw opens only http and https URLs. On Linux, it uses `xdg-open`, so make sure that `xdg-open` is installed.

### Debug log

Run `lazybw --debug`. lazybw writes the log to `~/.cache/lazybw/debug.log`, or to `$XDG_CACHE_HOME/lazybw/debug.log` if that variable is set. The log holds no passwords, session tokens or field values. See [SECURITY.md](SECURITY.md#debug-log).

## Documentation

- [Keybindings](docs/KEYBINDINGS.md): every key on every screen
- [Install](docs/INSTALL.md): each install method, and how to make sure that a download is genuine
- [ASCII mode](docs/GLYPHS.md#ascii-mode): what lazybw draws when you use `--ascii`
- [Security](SECURITY.md): how lazybw handles your session, master password and clipboard

## Contributing

lazybw does not accept pull requests yet. Bug reports and feature requests are welcome in [GitHub Issues](https://github.com/juthrbog/lazybw/issues). To report a security problem, see [SECURITY.md](SECURITY.md#reporting-a-vulnerability).

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
