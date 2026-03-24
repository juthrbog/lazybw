# lazybw

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

j/k navigate  /  search  c pwd  t totp  u user  ctrl+r  ?  q
```

## Features

- **Keyboard-first** -- navigate, search, and copy without touching the mouse
- **Instant copy** -- `c` for password, `t` for TOTP, `u` for username
- **Live TOTP countdown** -- current code with a depleting countdown indicator, color-coded by urgency
- **Full item type support** -- logins, cards, and secure notes with type-specific detail views
- **Fuzzy search** -- press `/` to filter your vault in real time
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
| `--debug` | `false` | Write debug log to `$XDG_CACHE_HOME/lazybw/debug.log` |
| `--version` | | Print version and exit |

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
| `Ctrl+r` | Sync vault |
| `Ctrl+l` | Lock vault |
| `?` | Toggle full help |
| `q` | Lock and quit |

#### Filter mode

| Key | Action |
|---|---|
| Text | Narrow list in real time |
| `Up` / `Down` | Move cursor in filtered list |
| `Enter` | Confirm selection |
| `Esc` | Clear filter |

## License

Apache License 2.0 -- see [LICENSE](LICENSE) for details.
