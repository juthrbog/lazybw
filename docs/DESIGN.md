# lazybw Design Document

A fast, keyboard-driven TUI for Bitwarden built with Go and the Charm
ecosystem. Designed from the ground up around Bitwarden's actual item types and
the workflows developers use most.

## Goals

- Make the three most common operations instant: search, copy password, copy TOTP
- Keyboard-first; no mouse required
- Single static binary, no runtime dependencies beyond `bw`
- TOTP as a first-class citizen — live countdown visible in the drawer
- Terminal-native clipboard via OSC 52
- Security-conscious: session in memory only, auto-clear clipboard, idle lock

## Stack

| Layer | Library |
|---|---|
| TUI framework | `github.com/charmbracelet/bubbletea` |
| Components | `github.com/charmbracelet/bubbles` (list, textinput, viewport, spinner, key, help) |
| Styling | `github.com/charmbracelet/lipgloss` |
| Forms | `github.com/charmbracelet/huh` |
| Clipboard | BubbleTea OSC 52 (`tea.SetClipboard`) |
| TOTP | `totp` (local, RFC 6238 — `crypto/hmac`) |

---

## Layout Philosophy

**No vertical split.** The list panel + detail panel divided by a vertical pipe
wastes space and adds visual noise. Instead:

- The **item list** uses the full terminal width — each row shows name and
  username cleanly with room to breathe.
- A **detail drawer** appears below the list when an item is selected. It uses
  a single horizontal separator line (not a box) and renders the fields relevant
  to the item's type.
- A **status/help bar** is pinned to the very bottom.

This works at any terminal width with no split threshold. Wider terminals
simply show longer names and descriptions — no layout mode change needed.

---

## Primary Layout

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
  TOTP       843 291  ▓▓▓▓▓▓▓░░░░░  18s         [t] copy
  URL        console.aws.amazon.com             [o] open

j/k navigate  /  search  c pwd  t totp  u user  r  ?  q
```

Key observations:
- No borders on the list — the rows breathe
- The drawer separator `── Name ──── Type ─` uses the item name as a natural
  label; no redundant border box
- TOTP row shows the live 6-digit code, a depleting progress bar, and seconds
  remaining — this is ephemeral data (30s window) so displaying it inline is
  the right call
- Item type icons use coloured glyphs: `●` login (purple), `♦` card (green),
  `✎` note (yellow)
- The status/help bar is one line, always pinned to the bottom

---

## Drawer Detail

The drawer is a fixed-height area (8 rows: 1 separator + up to 6 field rows +
1 padding row) that appears below the list when an item is highlighted. It
does **not** require pressing Enter — navigating the list cursor is enough.

Field rows adapt to item type:

### Login
```
── Gmail ─────────────────────────────────────── Login ─
  Username   user@gmail.com                     [u] copy
  Password   ••••••••••                         [c] copy
  TOTP       843 291  ▓▓▓▓▓▓▓░░░░░  18s         [t] copy
  URL        mail.google.com                    [o] open
  Notes      (none)
```

### Secure Note
```
── API Keys ─────────────────────────────────────── Note ─
  ANTHROPIC_KEY=sk-ant-api03-...
  OPENAI_KEY=sk-proj-...
  (scroll with J/K)
```

### Card
```
── Visa Debit ─────────────────────────────────── Card ─
  Cardholder  John Smith
  Number      •••• •••• •••• 4242                [c] copy
  Expiry      12/27
  CVV         •••                                [v] copy
```

### When no item is selected
```
── No item selected ────────────────────────────────────
  Navigate with j/k or search with /
```

---

## Unlock Screen

Full screen, vertically and horizontally centred, minimal chrome:

```




                          lazybw

                     Vault is locked.

                  ┌──────────────────────┐
                  │ Master password      │
                  └──────────────────────┘

                    Enter to unlock · q to quit




```

The login variant adds an email field above the password field.
Error message appears below the form on failed attempts.

---

## State Machine

```
Unauthenticated ──── bw login ────▶ Locked
                                      │
                                  bw unlock
                                      │
                                      ▼
                              ┌─── Vault ───┐
                              │  List       │◀─── r sync
                              │  Drawer     │
                              │  Filter (/) │
                              └─────────────┘
                                      │
                              l / idle timeout
                                      │
                                      ▼
                                   Locked
```

The vault is a single state with sub-modes, not separate screens:
- **Normal** — list + drawer visible; j/k moves cursor, drawer updates instantly
- **Filter** — `/` activates an inline search input above the list; live fuzzy
  filter; Esc clears and returns to Normal
- **Grouped** — `ctrl+g` toggles collapsible groups; items with similar names
  are bucketed under group headers. Enter expands/collapses a group. Grouping
  flattens automatically during filter mode.
- **Error overlay** — non-fatal errors shown in a centred box; r to retry, q to quit

There is no separate Detail screen. The drawer replaces it.

---

## TOTP Countdown

TOTP codes are computed locally using the `totp` package (RFC 6238). The seed
from `Login.Totp` — either a raw base32 secret or an `otpauth://` URI — is
parsed once when an item is selected and cached as `totp.Params`. The code is
recomputed synchronously every tick (~1s), eliminating the 1-3s latency of
shelling out to `bw get totp`.

```
  TOTP    843 291  ████ 18s          [t] copy
```

- `843 291` — current 6-digit code with a mid-space for readability
- `████` — 4-char micro-bar using Unicode block elements (U+2588–258F); 32
  states across a 30-second window for a visible change every second.
  Green >15s, yellow 10–15s, red <10s.
- `18s` — seconds remaining, updated every second via `time.Tick`
- When no TOTP key is set on the item, the row is omitted entirely
- Supports SHA1, SHA256, SHA512 algorithms and custom period/digit counts
  via `otpauth://` URI parameters

---

## Item Type Support

| Type | v1 | Fields shown |
|---|---|---|
| Login | ✅ | Username, Password, TOTP, URLs, Notes |
| Secure Note | ✅ | Note content (scrollable in drawer) |
| Card | ✅ | Cardholder, Number (masked), Expiry, CVV (masked) |
| Identity | v2 | — |
| SSH Key | v2 | — |

---

## Item List Rows

```
▶ ●  AWS Console                     aws-admin
  ●  GitHub                          dev@example.com
  ♦  Visa Debit                      •••• 4242
  ✎  API Keys
```

- `▶` cursor on selected row only
- Glyph colours: login = purple, card = green, note = yellow
- Description is right-aligned, faint: username for Login, masked last 4 for
  Card, first line of content for Note

---

## Keybindings

### Vault — Normal mode

| Key | Action |
|---|---|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `g` / `Home` | Jump to top |
| `G` / `End` | Jump to bottom |
| `c` | Copy password of selected item |
| `t` | Copy current TOTP code |
| `u` | Copy username |
| `o` | Open first URL in browser (`xdg-open`) |
| `/` | Enter filter mode |
| `r` | Sync vault (`bw sync`) |
| `l` | Lock vault immediately |
| `ctrl+g` | Toggle item grouping |
| `Enter` | Expand / collapse group header |
| `?` | Toggle full help |
| `q` / `ctrl+c` | Lock + quit |

### Vault — Filter mode

| Key | Action |
|---|---|
| `[text]` | Narrow list in real time |
| `↑` / `↓` | Move cursor in filtered list |
| `Esc` | Clear filter, return to Normal |
| `Enter` | Confirm selection, return to Normal |

### Drawer — Note scrolling

| Key | Action |
|---|---|
| `J` / `Shift+↓` | Scroll drawer down |
| `K` / `Shift+↑` | Scroll drawer up |

---

## Status / Help Bar

One line, always at the bottom. Left = context or toast; right = account info.

**Normal:**
```
j/k navigate  /  search  c pwd  t totp  u user  r sync  ?  q quit
```

**With toast:**
```
Password copied — clears in 60s              user@example.com · synced 3m ago
```

---

## bw CLI Integration

All commands run via `exec.CommandContext` with a 10-second timeout.
`BW_SESSION` is injected into each subprocess environment and never written
to disk.

### Startup flow

```
bw status
  → "unauthenticated" → show login form  → bw login [email] [pw] --raw
  → "locked"          → show unlock form → bw unlock [pw] --raw → session token
  → "unlocked"        → check $BW_SESSION env var
       present  → use it, skip unlock
       absent   → show unlock form
```

### Key commands

```bash
bw status                        # {status, userEmail, lastSync}
bw login [email] [pw] --raw      # returns session token
bw unlock [pw] --raw             # returns session token
bw lock                          # on quit / l / idle timeout
bw sync                          # r
bw list items                    # full vault JSON (all fields cached in memory)
bw generate [flags]              # password generator
```

Passwords, usernames, TOTP seeds, and card numbers are all available in the
`bw list items` response — no per-field CLI calls needed. TOTP codes are
computed locally from the cached seed (see `totp` package).

---

## Clipboard

Clipboard access uses BubbleTea's built-in OSC 52 support (`tea.SetClipboard`),
which works across terminal emulators without external tools.

After copy: toast `"Password copied — clears in 60s"`.
After 60s: overwrite clipboard with `""`.

---

## Security Considerations

- Session token in-process memory only; cleared on lock/quit
- Passwords never rendered to screen (shown as `••••••••`)
- TOTP codes are shown — they expire in 30s; this matches Bitwarden's own apps
- Clipboard cleared after 60s
- Idle lock after 15 min of inactivity (configurable via `--idle-timeout` flag)
- On quit: `bw lock` + clear session token
- Logs to file only; no sensitive data in logs
- Master password input uses `huh.EchoModePassword`

---

## Colour Palette

```go
ColorHighlight = lipgloss.AdaptiveColor{Dark: "#7D56F4", Light: "#5A3ECC"} // login glyph, selected
ColorSubtle    = lipgloss.AdaptiveColor{Dark: "#383838", Light: "#D9DCCF"} // separator lines, bg
ColorGreen     = lipgloss.AdaptiveColor{Dark: "#04B575", Light: "#027A4F"} // card glyph, copy toast, TOTP ok
ColorYellow    = lipgloss.AdaptiveColor{Dark: "#F5A623", Light: "#C47D10"} // note glyph, TOTP warning
ColorRed       = lipgloss.AdaptiveColor{Dark: "#FF4F4F", Light: "#CC2222"} // errors, TOTP urgent
ColorFaint     = lipgloss.AdaptiveColor{Dark: "#626262", Light: "#9A9A9A"} // descriptions, secondary text
```

---

## Project Layout

```
lazybw/
├── main.go
├── model.go                # Root model; state machine
├── screens/
│   ├── locked.go           # Unlock / login form (huh)
│   ├── vault.go            # List + drawer composite view
│   ├── grouping.go         # Collapsible group logic + item grouping algorithm
│   └── error.go            # Error overlay
├── ui/
│   ├── theme.go            # Colour palette + styles
│   ├── keymap.go           # KeyMap structs
│   ├── drawer.go           # Drawer renderer (pure function)
│   ├── itemrow.go          # Single item row renderer
│   └── statusbar.go        # Status / help bar renderer
├── bwcmd/
│   ├── exec.go
│   ├── types.go
│   └── parser.go
├── session/
│   ├── manager.go
│   └── clipboard.go
├── totp/
│   └── totp.go             # RFC 6238 local TOTP computation
└── go.mod
```

---

## Not in Scope (v1)

- Vault editing (create / update / delete items)
- Organisation / collection navigation
- Multiple accounts
- SSH key agent integration
- Identity item type
- Password generator UI
- Bitwarden Send
