<h1 align="center">lazybw</h1>

<p align="center">
  A fast, keyboard-driven TUI for <a href="https://bitwarden.com">Bitwarden</a> built with Go and the <a href="https://charm.sh">Charm</a> ecosystem.
</p>

<p align="center">
  <a href="https://github.com/juthrbog/lazybw/actions/workflows/ci.yml"><img src="https://github.com/juthrbog/lazybw/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/juthrbog/lazybw/releases/latest"><img src="https://img.shields.io/github/v/release/juthrbog/lazybw" alt="Release"></a>
  <a href="https://github.com/juthrbog/lazybw/blob/main/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/juthrbog/lazybw" alt="Go"></a>
  <a href="https://goreportcard.com/report/github.com/juthrbog/lazybw"><img src="https://goreportcard.com/badge/github.com/juthrbog/lazybw" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License"></a>
</p>

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

## Why lazybw?

The Bitwarden desktop app isn't available on every platform, and the CLI -- while powerful -- wasn't built for daily driving. Grabbing a password means piping `bw list items` through `jq`, grepping for the right entry, and copying the result. Do that ten times a day and you'll end up writing wrapper scripts just to stay sane.

lazybw puts your entire vault in a keyboard-driven TUI so you never have to leave the terminal. Search, browse, copy passwords and TOTP codes -- all in a few keystrokes.

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

## Installation

### Download a binary

Download a pre-built binary from the [releases page](https://github.com/juthrbog/lazybw/releases/latest), extract it, and place it somewhere on your `$PATH` (e.g. `/usr/local/bin`). Binaries are available for Linux and macOS on both amd64 and arm64. Each release includes a `checksums.txt` for verification.

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

| Key | Action |
|---|---|
| `j` / `k` | Navigate |
| `/` | Search / filter |
| `c` | Copy password |
| `t` | Copy TOTP code |
| `u` | Copy username |
| `o` | Open URL in browser |
| `p` | Open password generator |
| `r` | Sync vault |
| `?` | Help overlay |
| `q` | Lock and quit |

Press `?` in-app for the complete list including password generator and filter keybindings. See [docs/KEYBINDINGS.md](docs/KEYBINDINGS.md) for the full reference.

## Contributing

lazybw is in early development and not yet accepting contributions. This may change in the future -- check back later.

## License

Apache License 2.0 -- see [LICENSE](LICENSE) for details.
