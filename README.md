# lazybw

[![CI](https://github.com/juthrbog/lazybw/actions/workflows/ci.yml/badge.svg)](https://github.com/juthrbog/lazybw/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/juthrbog/lazybw)](https://github.com/juthrbog/lazybw/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/juthrbog/lazybw)](https://github.com/juthrbog/lazybw/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/juthrbog/lazybw)](https://goreportcard.com/report/github.com/juthrbog/lazybw)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A fast, keyboard-driven TUI for [Bitwarden](https://bitwarden.com) built with Go and the [Charm](https://charm.sh) ecosystem.

```
  ●  Gmail                           user@gmail.com
  ●  GitHub                          dev@example.com
▶ ●  AWS Console                     aws-admin
  ♦  Visa Debit                      •••• 4242
  ✎  SSH Keys
  ✎  Anthropic API                   (note)
  ●  Slack                           work@company.com
                                                8 / 156

── AWS Console ───────────────────────────────── Login ─
  Username   aws-admin                          [u] copy
  Password   ••••••••••                         [c] copy
  TOTP       843 291  ● 18s                     [t] copy
  URL        console.aws.amazon.com             [o] open

j/k navigate  /  search  c pwd  t totp  u user  r  ?  q
```

## Features

- **Keyboard-first** -- navigate, search, and copy without touching the mouse
- **Instant copy** -- `c` for password, `t` for TOTP, `u` for username
- **Live TOTP countdown** -- current code with a depleting countdown indicator, color-coded by urgency
- **Full item type support** -- logins, cards, and secure notes with type-specific detail views
- **Fuzzy search** -- press `/` to filter your vault in real time
- **Item grouping** -- similar items collapse into expandable groups (`Ctrl+G` to toggle)
- **Security-conscious** -- session token in memory only, clipboard auto-clears after 60s, idle lock after 15 minutes
- **Single binary** -- no runtime dependencies beyond `bw`

## Requirements

- **Go 1.22+** (to build from source)
- **[Bitwarden CLI (`bw`)](https://bitwarden.com/help/cli/#download-and-install)** -- must be installed and available in your `$PATH`
- A terminal emulator with Unicode support

### Clipboard support

lazybw detects your display server and uses the appropriate clipboard tool:

| Environment | Tool |
|---|---|
| Wayland (`$WAYLAND_DISPLAY` set) | `wl-copy` (install via `wl-clipboard`) |
| X11 (`$DISPLAY` set) | `xclip` or `xsel` |
| macOS | `pbcopy` (built-in) |

## Installation

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

On first launch, lazybw checks `bw status` and prompts you to log in or unlock as needed.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--idle-timeout` | `15m` | Lock the vault after this duration of inactivity |
| `--theme` | `catppuccin-mocha` | Color theme (see below) |
| `--debug` | `false` | Write debug log to `$XDG_CACHE_HOME/lazybw/debug.log` |
| `--version` | | Print version and exit |

You can also set the theme via the `LAZYBW_THEME` environment variable (flag takes precedence).

### Themes

Available themes: `catppuccin-mocha` (default), `catppuccin-frappe`, `catppuccin-macchiato`, `catppuccin-latte`, `dracula`, `charm`, `base16`.

Press `T` in the vault to open the theme picker and switch themes on the fly.

### Keybindings

#### Vault

| Key | Action |
|---|---|
| `j` / `k` | Move cursor down / up |
| `g` / `G` | Jump to top / bottom |
| `/` | Search / filter |
| `c` | Copy password |
| `t` | Copy TOTP code |
| `u` | Copy username |
| `o` | Open URL in browser |
| `J` / `K` | Scroll note content in drawer |
| `p` | Open password generator |
| `r` | Sync vault |
| `l` | Lock vault |
| `Ctrl+G` | Toggle item grouping |
| `Enter` | Expand / collapse group |
| `T` | Open theme picker |
| `?` | Toggle full help |
| `q` | Lock and quit |

#### Password generator

| Key | Action |
|---|---|
| `Enter` / `r` | Regenerate |
| `+` / `-` | Increase / decrease length (or word count) |
| `m` / `Tab` | Toggle password / passphrase mode |
| `1`-`4` | Toggle uppercase / lowercase / numbers / special |
| `c` | Copy to clipboard |
| `Esc` | Close generator |

#### Filter mode

| Key | Action |
|---|---|
| Text | Narrow list in real time |
| `Up` / `Down` | Move cursor in filtered list |
| `Enter` | Confirm selection |
| `Esc` | Clear filter |

## License

Apache License 2.0 -- see [LICENSE](LICENSE) for details.
