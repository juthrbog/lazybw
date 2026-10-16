# lazybw Design Document

lazybw is a keyboard-driven terminal UI for Bitwarden. It is written in Go
with the Charm libraries. It runs the Bitwarden CLI (`bw`) for each vault
action, and it shows each Bitwarden item type with its own fields.

This document describes the design that the code implements now. Other
documents give more detail:

- [LAYOUT.md](LAYOUT.md) describes the frame and the screen sizes.
- [KEYBINDINGS.md](KEYBINDINGS.md) lists every key.
- [GLYPHS.md](GLYPHS.md) lists the glyphs and their code points.
- [SECURITY.md](../SECURITY.md) describes the security design and the accepted risks.

## Goals

- Make the three most common actions fast: search, copy the password, copy the TOTP code.
- Use the keyboard for everything. lazybw does not capture the mouse.
- Ship one static binary. The only runtime dependency is `bw`.
- Compute TOTP codes locally, and show a live countdown in the drawer.
- Copy with OSC 52, the terminal escape sequence that sets the clipboard.
- Keep the session token in memory only, clear the clipboard after 60 seconds, and lock the vault after 15 minutes with no key press.

## Stack

| Layer | Library |
|---|---|
| TUI framework | `charm.land/bubbletea/v2` |
| Components | `charm.land/bubbles/v2` (`list`, `spinner`, `key`, `viewport`) |
| Styling | `charm.land/lipgloss/v2` |
| Forms | `charm.land/huh/v2` |
| Theme colors | `github.com/catppuccin/go` |
| ANSI text helpers | `github.com/charmbracelet/x/ansi` |
| Clipboard | OSC 52 through `tea.SetClipboard` |
| TOTP | The `totp` package of lazybw (RFC 6238, `crypto/hmac`) |

`go.mod` sets `go 1.26.0` and `toolchain go1.27.2`. The huh forms are the
unlock form, the login form, the two-step login forms, the master password
prompt and the theme picker. lipgloss v2 has no `AdaptiveColor`. Each theme
picks its light or dark colors with `lipgloss.LightDark` (see
[Theme and Background](#theme-and-background)).

---

## Layout Philosophy

lazybw has no vertical split. A list panel and a detail panel side by side
waste space and add visual noise. The screen has these parts:

- A 2-row header at the top: the title bar and a gradient line.
- The item list. It uses the full width of the terminal. Each row shows the item glyph, the name and a short description.
- The detail drawer below the list. A gradient line separates it from the list. It shows the fields of the selected item.
- A 1-row footer at the bottom: key hints on the left, and a toast or the sync age on the right.

The layout is the same at every terminal width. A wider terminal shows
longer names and descriptions. `d` changes the drawer to a card over the
list (see [Drawer Detail](#drawer-detail)).

---

## Primary Layout

The mockups in this document use the Nerd Font glyphs of the code. Without
a Nerd Font, the glyphs show as empty boxes. This is the vault screen at 80
columns:

```
 󰦝 lazybw  user@example.com  8 items Login
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄

  󰌾  Gmail                                                        user@gmail.com
  󰌾  GitHub                                                      dev@example.com
▶ 󰌾  AWS Console                                                       aws-admin
  󰿯  Visa Debit                                                        •••• 4242
  󰎞  API Keys                                                             (note)
  󰀄  Passport                                                         John Smith
  󰣀  Deploy key                                                    SHA256:abcdef
  󰌾  Slack                                                      work@company.com


▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
  󰌾  AWS Console                                                          Login
  Username    aws-admin  [u] copy
  Password    ••••••••••  [c] copy
  TOTP        843 291  ██▍░ 18s  [t] copy
  URL         console.aws.amazon.com  [o] open
  Notes       (none)

 j/k navigate · / search · c pwd · t totp · ? help · q quit       synced 3m ago
```

Key observations:

- The header shows the logo glyph and `lazybw` in a color gradient. The email shows in a badge. The item count shows at 80 columns or more, and the type of the selected item shows at 60 columns or more.
- The list has no border. Its first row holds the filter input, and it is empty while no filter is open.
- `▶` marks the selected row.
- The drawer header card shows the item glyph, the item name and a type badge.
- The TOTP row shows the live code, a 4-cell countdown bar and the seconds that are left.
- The footer drops hints from the end when they do not fit. The status on the right gets space first.

Each item type has its own glyph and color:

| Type | Glyph | Code point | Color |
|---|---|---|---|
| Login | 󰌾 | U+F033E | `ColorHighlight` |
| Card | 󰿯 | U+F0FEF (`nf-md-credit_card`) | `ColorGreen` |
| Secure note | 󰎞 | U+F039E (`nf-md-note_text`) | `ColorYellow` |
| Identity | 󰀄 | U+F0004 | `ColorHighlight` |
| SSH key | 󰣀 | U+F08C0 | `ColorGreen` |

The header logo is U+F099D (`nf-md-shield_lock`). A test in
`ui/theme_test.go` makes sure that each item glyph and the header glyph
have these runes.

---

## Drawer Detail

The drawer shows the selected row. The user does not press Enter. A cursor
move is enough. The default drawer height is 8 rows: the gradient line, the
header card and 6 field rows.

- `+` and `-` change the drawer height by one row. `=` sets it back to 8 rows.
- The drawer keeps at least 3 rows and at most half of the screen height, and the list keeps at least 2 rows. On a very short screen, the drawer gets the rows that the list leaves.
- When the fields do not fit, the last row shows `↓ N more`.
- `d` toggles the overlay mode. The drawer then shows as a card over the list, at most 80 columns wide, and the list gets all of the content area. In overlay mode, `+`, `-` and `=` do nothing.

Each field row has a label, the value and a copy hint, such as `[c] copy`.
In a narrow drawer, lazybw drops the copy hint first, so that the value
stays. An empty username, an empty password and an empty CVV show `(none)`
and no copy hint, because the copy key has nothing to copy. lazybw masks
the password, the card number, the CVV, the SSN, the passport number, the
license number and the SSH private key. The card number shows its last 4
digits.

lazybw removes escape sequences and control characters from vault text
before it draws the text. A copy always uses the stored value.

A login:

```
  󰌾  Gmail                                                                Login
  Username    user@gmail.com  [u] copy
  Password    ••••••••••  [c] copy
  TOTP        843 291  ██▍░ 18s  [t] copy
  URL         mail.google.com  [o] open
  Notes       First line of the notes  [n] copy
```

The URL row shows the first URL of the login. The Notes row shows the first
line of the notes. A login with no TOTP key has no TOTP row, and a login
with no URL has no URL row.

A secure note:

```
  󰎞  API Keys                                               [n] copy   Note
  ANTHROPIC_KEY=sk-ant-api03-...
  OPENAI_KEY=sk-proj-...
```

The drawer shows the note body, wrapped to the drawer width. `J` and `K`
(or Shift+Down and Shift+Up) scroll the note. Only a secure note scrolls.
The copy hint is in the header card, because a note has no field rows.

A card:

```
  󰿯  Visa Debit                                                           Card
  Cardholder  John Smith
  Number      •••• •••• •••• 4242  [c] copy
  Expiry      12/2027
  CVV         •••  [v] copy
```

A card with no CVV shows `CVV         (none)` and no `[v]` hint.

An identity:

```
  󰀄  Passport                                                         Identity
  Name        John Smith
  Email       john@example.com  [u] copy
  SSN         •••••••••  [c] copy
```

An identity shows only the fields that have a value: Name, Email, Phone,
Company, SSN, Passport, License and Address.

An SSH key:

```
  󰣀  Deploy key                                                        SSH Key
  Fingerprint SHA256:abcdef
  Public Key  ssh-ed25519 AAAA...  [u] copy
  Private Key ••••••••••  [c] copy
```

An item with master password re-prompt (see
[Master Password Re-prompt](#master-password-re-prompt)):

```
  󰌾  Bank                                                re-prompt   Login
  Username    me  [u] copy
  Password    ••••••••••  [c] copy
  TOTP        (hidden)  [t] copy
  Notes       (hidden)  [n] copy
```

A selected group header (see [Grouping](#grouping)):

```
  AWS
  3 items · enter to expand
```

An empty list, for example a filter with no match:

```
  No item selected — navigate with j/k or search with /
```

---

## Unlock Screen

The unlock screen shows in the content area between the header and the
footer. The form and its title are centered:

```
               Vault is locked.

               ┃ Master password
               ┃ >

               ✗ Could not unlock the vault. Check your
                 master password and try again.
```

The footer shows `enter unlock · ctrl+q quit`. The form shows no help line,
because the footer shows the keys. While `bw unlock` runs, the screen shows
a spinner and `Unlocking…`.

The login variant has the title `Log in to Bitwarden`, an Email field and a
Master password field. Enter moves to the next field, and Enter on the last
field submits the form. Shift+Tab goes back from the Master password field
to the Email field. On the Master password field, the footer shows
`enter submit · shift+tab back · ctrl+q quit`. While `bw login` runs, the
screen shows a spinner and `Logging in…`. The two-step login forms are in
[Two-step login](#two-step-login).

An error shows below the form. It starts with `✗` and wraps to the width of
the form (40 cells). Known bw errors get a short text, for example
`Could not log in. Check your email and master password and try again.`
for a failed login. Other errors show the last line that bw wrote to
stderr. `screens/usererror.go` holds the texts.

When `bw lock` fails, the unlock screen can also show the error (see
[Locking and quitting](#locking-and-quitting)). For an idle lock, lazybw
adds the error to the unlock screen that already shows, and it keeps what
the user typed. The lock error shows above the error of a failed unlock,
because it is a security warning. When the content is too tall,
`ui.CenterInArea` cuts lines at the bottom, so the lower error goes first.
On a short terminal, the screen also leaves out the blank lines between the
title, the form and the errors.

---

## State Machine

```
                         bw status
                             │
      ┌──────────────────────┼─────────────────────┐
"unauthenticated"     "locked" or            error, or output
      │               "unlocked"             that does not parse
      ▼                      ▼                     ▼
    Login                 Locked                 Error ── r ──▶ bw status again
      │                      │
  bw login               bw unlock
      └──────────┬───────────┘
                 ▼
              Loading ── bw list items fails ──▶ Error ── r ──▶ bw list items again
                 │
          bw list items
                 ▼
          ┌─── Vault ───┐
          │  List       │◀─── r: bw sync, then bw list items
          │  Drawer     │
          │  Filter (/) │
          └─────────────┘
           │     │     │
           l   idle   q then y, or ctrl+c
           │   lock    │
           ▼     │     ▼
     Locking     │   Locking
     vault…      │   vault…
    (bw lock)    │  (bw lock)
           │     │     │
           ▼     ▼     ▼
        Locked Locked  exit
```

The vault is one state with modes, not a set of separate screens:

- Normal mode: the list and the drawer show. `j` and `k` move the cursor, and the drawer follows at once.
- Filter mode: `/` opens a filter input in the first list row (see [Filter](#filter)).
- Grouped mode: `ctrl+g` groups items with similar names under headers (see [Grouping](#grouping)).
- Overlays: the help (`?`), the password generator (`p`), the theme picker (`T`), the quit confirmation (`q`) and the master password prompt. Each overlay is a card over the dimmed vault. It gets all keys except `ctrl+c` until it closes.

There is no separate detail screen. The drawer takes its place.

The error screen shows an error that lazybw can recover from. `r` tries
again and `q` quits. If the session is unlocked, `r` runs `bw list items`
again. A second failure of `bw list items` locks the session, because the
token can be invalid, for example after a lock in another shell. The next
`r` then runs `bw status` and shows the unlock screen.

### Locking and quitting

- `l` drops the vault data, clears the clipboard and shows `Locking vault…` while `bw lock` runs. Then the unlock screen shows.
- An idle lock shows the unlock screen at once. `bw lock` runs in the background.
- `q` opens the card `Quit lazybw?`. `y` quits. `n` or Esc closes the card.
- `ctrl+c` quits at once from every screen. `ctrl+q` quits from the unlock and login screens, and `q` quits from the error screen.
- A quit from an unlocked vault clears the clipboard and shows `Locking vault…` while `bw lock` runs. Then lazybw clears the clipboard again and stops.

The idle timeout is 15 minutes by default. `--idle-timeout` sets another
value, and the minimum is 30 seconds. lazybw compares the idle time with
the timeout every 30 seconds. The comparison also uses the wall clock, so
the time of a system suspend counts. After a suspend, the first key press
locks the vault and does not count as activity.

When `bw lock` fails, lazybw drops the session token anyway. It shows the
error in these cases:

- After the lock key or an idle lock, the unlock screen shows it (see [Unlock Screen](#unlock-screen)).
- After a second failed item load, `RootModel` keeps the error in `lockErr`. The next unlock screen shows it. A quit from the error screen gives it to `main.go`, which prints it to stderr.
- After a quit, `main.go` prints the error to stderr after the terminal is restored.
- When the exit cleanup runs `bw lock` and that `bw lock` fails, `main.go` prints its error to stderr too (see [Exit cleanup](#exit-cleanup)).

`screens.LockErrorText` returns no text when bw reports that no account is
logged in, because the old session key then opens nothing. If the user
unlocks again before the result arrives, the error goes only to the debug
log. Each `bw lock` result carries the sequence number of its lock
(`bwcmd.LockResult.Seq`). When a newer lock runs, `RootModel` drops the
result of an earlier lock, so that it cannot end the newer lock.

---

## TOTP Countdown

lazybw computes TOTP codes locally in the `totp` package (RFC 6238). It does
not run `bw get totp`. When the selection changes, lazybw parses the TOTP
key of the selected login one time and keeps the result as `totp.Params`.

A `tea.Tick` fires every second. On each tick, lazybw computes the code and
the seconds left from the cached parameters. `t` computes the code again
at the time of the copy. At a step change, the cached code can be up to a
second old.

```
  TOTP        843 291  ██▍░ 18s  [t] copy
```

- `843 291` is the current code. A 6-digit code gets a space in the middle. Other lengths show as they are.
- `██▍░` is a 4-cell bar. The filled part uses the block elements U+2588 to U+258F, and the empty track uses the light shade U+2591. The bar has 32 steps for the period, so a 30-second period shows a change every second.
- The bar and the seconds change color with the period. They are green while more than half of the period is left, yellow while more than a third is left, and red for the rest. For a 30-second period, they are green from 30 to 16 seconds, yellow from 15 to 11 seconds and red from 10 to 1 second.
- `18s` is the number of seconds left.
- In a narrow drawer, the row drops the copy hint first and then the seconds text.

lazybw accepts the key forms that the Bitwarden clients accept:

- A raw base32 secret. lazybw ignores case and each character outside the base32 alphabet, such as spaces, hyphens and padding.
- An `otpauth://` URI. lazybw reads the `secret`, `period`, `digits` and `algorithm` parameters. The algorithm can be SHA1, SHA256 or SHA512.
- A `steam://` key. A Steam Guard code has 5 characters from the Steam alphabet, uses HMAC-SHA1 and has a 30-second period.

The code length is at most 10 digits. A larger `digits` value counts as 10,
as in Bitwarden. For 10-digit codes, Bitwarden computes the modulus in 32
bits, which wraps. Its 10-digit codes differ from the RFC value on about a
third of the time steps. lazybw keeps the RFC value.

When the key does not parse, the TOTP row shows `invalid TOTP secret` in
the error style. `t` then shows the error toast `Invalid TOTP secret`. For
an item with master password re-prompt, the TOTP row shows `(hidden)` until
the master password passes.

---

## Item Type Support

| Type | Fields in the drawer | `c` copies | `u` copies |
|---|---|---|---|
| Login | Username, Password, TOTP, first URL, first line of the notes | Password | Username |
| Secure note | The note body (scrolls with `J` and `K`) | | |
| Card | Cardholder, Number (masked), Expiry, CVV (masked) | Card number | |
| Identity | Name, Email, Phone, Company, SSN, Passport, License, Address | SSN | Email |
| SSH key | Fingerprint, Public Key, Private Key (masked) | Private key | Public key |

`t` copies the TOTP code of a login. `v` copies the card security code. `n`
copies the notes of the item. `o` opens the first URL of a login. A key
that has no value to copy on the selected item does nothing. `u` on a login
with no username shows `No username to copy`, because an empty write can
clear the clipboard in some terminals.

---

## Item List Rows

```
▶ 󰌾  AWS Console                                                       aws-admin
  󰌾  GitHub                                                      dev@example.com
  󰿯  Visa Debit                                                        •••• 4242
  󰎞  API Keys                                                             (note)
```

- `▶` marks the selected row only.
- The glyph shows the item type in its color (see [Primary Layout](#primary-layout)).
- The description is right-aligned and faint. A login shows its username. A card shows `••••` and the last 4 digits. A secure note shows `(note)`. An identity shows the first and the last name, or the email. An SSH key shows its fingerprint.
- The list never shows note text. Secure notes often start with an API key or a recovery code, so the note body shows only in the drawer.
- The name gets space first. lazybw cuts a long description, or drops it when less than 6 cells are left for it.

---

## Grouping

`ctrl+g` turns grouping on and off. The footer shows `Grouping enabled` or
`Grouping disabled`.

```
▶ ▾  AWS (3)
    󰌾  AWS (Prod)                                                          admin
    󰌾  AWS (Dev)                                                             dev
    󰌾  AWS Access Key                                                       AKIA
  ▸  GitHub (2)
  󰌾  Laws of Motion                                                       newton
```

lazybw groups the items in two steps (`screens/grouping.go`):

1. It strips each final ` (…)` and ` - …` part from the item name. For example, `AWS (Prod) - Recovery Codes` becomes `AWS`. The lowercase result is the group key. Items with the same key form a group.
2. An item with a key of its own can join a group of two or more items. It joins when its name has the group key as a whole word. The characters right before and right after the key must not be letters or digits. So `AWS Access Key` joins `aws`, but `Laws of Motion` does not. When more keys match, the longest key wins.

A group of two or more items gets a header row. The header label is the
name of the first item of the group, without the stripped parts. It keeps
the case of that item, for example `GitHub (2)`. The count is the number of
vault items in the group.

`▸` (U+25B8) marks a collapsed group, and `▾` (U+25BE) marks an expanded
group. `▶` is only the cursor. An expanded group shows
its items below the header, with an indent.

Enter on a header expands or collapses the group. A selected header shows
the group name and `N items · enter to expand` (or `collapse`) in the
drawer. While a filter is open or applied, the list is flat, so that the
filter also finds the items of collapsed groups. After `ctrl+g`, Enter on a
header, Esc on a filter or a sync, the same item stays selected. When the
item is in a collapsed group, its group header is selected. When grouping
is turned off, lazybw forgets which groups were expanded.

---

## Filter

`/` opens the filter input in the first list row. The list shows only the
items that match the typed text. The match is fuzzy and uses the item name
and the list description. Each time the filter text changes, the cursor
goes to the best match, which is the first row.

| Key | Action |
|---|---|
| Text | Narrow the list. The cursor goes to the first row. |
| `↑` / `↓` | Move the cursor in the matches. The input stays open. |
| Enter | Accept the filter and close the input. The list stays filtered. With no match, Enter clears the filter. |
| Esc | Clear the filter. The selected match stays selected. |

Only Enter accepts the filter. Tab, Shift+Tab, Ctrl+J and Ctrl+K do not
accept it. While the input is open, all other keys type into the filter,
except `ctrl+c`, which quits.

With no filter text, or with no match, Esc selects again the row that was
selected before `/`. A group header counts as a row here. While the filter
is applied, the footer shows
`filter: <text> · esc clear`. The action keys, such as `c` and `t`, work
on the filtered list.

---

## Master Password Re-prompt

A Bitwarden item can have "Master password re-prompt" turned on.
`bw list items` gives the setting as `reprompt`, and the value 1 means
that the clients ask for the master password. bw does not enforce it, so
lazybw does.

- For such an item, `c`, `t`, `v` and `n` open the card `Master password re-prompt` with a password input. `u` and `o` do not ask.
- Enter submits the password. Esc cancels and closes the card. The footer shows `enter submit · esc cancel`.
- lazybw runs `bw unlock --passwordenv BW_PASSWORD --raw` with the typed password. While bw runs, the card shows `Checking the master password…`, and only Esc works.
- If bw accepts the password, it returns a new session key, and the old key stops working. lazybw stores the new key with `session.State.ReplaceToken`. The session epoch does not change, so the bw commands that started in this session still belong to it. Then lazybw runs the copy that the user asked for.
- If bw does not accept the password, the card shows `The master password is not correct. Try again.`, and lazybw copies nothing.
- After the password passes, lazybw does not ask again while the item stays selected. A selection change, a lock, or a refresh that removes the item ends this.
- The drawer shows a yellow `re-prompt` marker in the header card. The TOTP code and the note text stay hidden until the password passes. The TOTP row and the Notes row show `(hidden)`, and a secure note shows `(hidden: master password re-prompt)`.

If the user closes the card while bw runs, the `bw unlock` goes on. lazybw
stores its new key, so that the next bw command works. A password
that the user submits in a new card waits for that run to end. Two
`bw unlock` runs at the same time can store a key that does not work.
Re-prompt is a UI control, not encryption.

---

## Password Generator

`p` opens the password generator card over the vault. lazybw runs
`bw generate` for each change. Each time the card opens, the generator
starts in password mode with length 20 and all four character sets.

```
Password Generator

Mode        [Password]  Passphrase
Length      20
Uppercase   ✓   Lowercase  ✓   Numbers  ✓   Special  ✓

k7#Qp2!vX9mLr4@Tz8&w
```

| Key | Password mode | Passphrase mode |
|---|---|---|
| Enter or `r` | Make a new value | Make a new value |
| `m` or Tab | Switch to passphrase mode | Switch to password mode |
| `+` or `=` | Length + 1 (at most 128) | Word count + 1 (at most 20) |
| `-` | Length - 1 (at least 5) | Word count - 1 (at least 3) |
| `1` | Toggle uppercase | Toggle capitalize |
| `2` | Toggle lowercase | Toggle include number |
| `3` | Toggle numbers | Cycle the separator: `-`, `_`, `.`, space |
| `4` | Toggle special | Nothing |
| `c` | Copy the value | Copy the value |
| Esc | Close the card | Close the card |

The passphrase starts with 4 words, the `-` separator, capitals and a
number. The smallest word count is 3. `bw generate` raises a smaller count
to 3, so a smaller count on the card does not match the output. The
footer shows `enter regen · +/- words · m mode · 1-3 options · c copy · esc close`
in passphrase mode.

The value wraps to fit in the card, at most 72 cells wide. While bw runs,
the card shows `Generating…`. Each request gets a sequence number, so that
a slow, older result cannot replace a newer one. If `bw generate` fails,
the card shows the error in the error style, and `c` copies nothing.

---

## Keybindings

[KEYBINDINGS.md](KEYBINDINGS.md) is the full reference, with the keys of
each overlay and of the unlock screen. These are the vault keys:

| Key | Action |
|---|---|
| `j` / `↓` | Move the cursor down |
| `k` / `↑` | Move the cursor up |
| `g` / Home | Go to the top |
| `G` / End | Go to the bottom |
| `→` / PgDn | Go one visible page down |
| `←` / PgUp | Go one visible page up |
| `c` | Copy the password, card number, SSN or SSH private key |
| `t` | Copy the current TOTP code |
| `u` | Copy the username, identity email or SSH public key |
| `v` | Copy the card security code |
| `n` | Copy the notes |
| `o` | Open the first URL in the browser |
| `/` | Open the filter |
| `r` | Sync the vault (`bw sync`) |
| `l` | Lock the vault |
| `p` | Open the password generator |
| `T` | Open the theme picker |
| `ctrl+g` | Turn grouping on or off |
| Enter | Expand or collapse a group |
| `J` / Shift+Down | Scroll the note in the drawer down |
| `K` / Shift+Up | Scroll the note in the drawer up |
| `+` / `-` | Make the drawer taller or shorter |
| `=` | Set the drawer height back to 8 rows |
| `d` | Toggle the drawer overlay |
| `?` | Show the help overlay |
| `q` | Ask to quit, then lock and quit on `y` |
| `ctrl+c` | Lock and quit at once |

---

## Footer and Toasts

The footer is one line at the bottom. Hints are on the left. A toast, or
the sync age when no toast shows, is on the right.

```
 j/k navigate · / search · c pwd · t totp · ? help · q quit       synced 3m ago
 j/k navigate · / search · c pwd   󰆏 Password sent to clipboard · clears in 60s
```

The normal hints are `j/k navigate · / search · c pwd · t totp · ? help · q quit · ctrl+g group · d drawer · p gen · T theme`.
lazybw drops hints from the end when they do not fit. Each overlay, the
filter input and the quit confirmation show their own hints.

A toast shows one message in the footer. A new toast replaces the old one.

- An info toast ends after 4 seconds, for example `Drawer: overlay`, `Grouping enabled` or `Theme: dracula`.
- An error toast ends after 8 seconds. It starts with `✗` and uses the error style, for example `Sync failed: …`, `Load failed: …`, `Open URL failed: …` or `Invalid TOTP secret`.
- A copy toast stays until the clipboard clear (see [Clipboard](#clipboard)).
- The timer of an older toast does not end a newer toast.

While a sync runs, the footer shows a spinner and `Syncing…`. After a
successful sync, it shows `✓ Vault synced` until the new items load. With
no toast, the footer shows the age of the last sync: `synced just now`,
`synced 3m ago` or `synced 2h ago`. lazybw reads the time of the last sync
from `bw status` at start, and it sets the time again after each sync in
lazybw.

---

## bw CLI Integration

lazybw runs every bw command with `exec.CommandContext` and an argument
list. No shell runs. User text goes only after `--` or in `--code=`.

### Commands

| Command | When | Time limit | Secret in the environment |
|---|---|---|---|
| `bw status` | At start, and on retry | 15 s | None |
| `bw unlock --passwordenv BW_PASSWORD --raw` | Unlock form, master password re-prompt | 2 min | `BW_PASSWORD` |
| `bw login --passwordenv BW_PASSWORD --raw [--method N] [--code=CODE] -- EMAIL` | Login form | 2 min | `BW_PASSWORD` |
| `bw list items` | After an unlock, after a sync, on retry | 2 min | `BW_SESSION` |
| `bw sync` | `r` | 2 min | `BW_SESSION` |
| `bw lock` | Lock key, idle lock, quit, a second failed item load, exit cleanup | 15 s | `BW_SESSION` |
| `bw generate [options]` | Password generator | 15 s | None |

`bw status` returns `{status, userEmail, lastSync}`. `bw unlock --raw` and
`bw login --raw` print only the session key. An exit code of 0 with no
session key counts as a failure: `bw did not return a session key`.
`bw list items` returns the whole vault as JSON. Passwords, usernames, TOTP
keys and card numbers are all in that output, so lazybw needs no command
for each field. lazybw zeros the output of `bw list items`, `bw unlock`
and `bw login` after it reads it.

### Environment of the bw process

lazybw passes the master password and the session token to bw in the
environment, not in the arguments. Other users can read the arguments with
`ps` or `/proc/PID/cmdline`. `childEnv` in `bwcmd/exec.go` builds the
environment of each bw process:

1. It copies the environment of lazybw without `BW_SESSION`, `BW_PASSWORD`, `BW_RESPONSE`, `BW_QUIET`, `BW_RAW`, `BW_PRETTY`, `BW_CLEANEXIT`, `BW_NOINTERACTION` and `BITWARDENCLI_DEBUG`. The match ignores case. The BW_ output variables change what bw prints, and `BITWARDENCLI_DEBUG` makes bw write its debug log to stdout. Variables such as `BITWARDENCLI_APPDATA_DIR`, `NODE_EXTRA_CA_CERTS` and the proxy variables stay.
2. It adds `BW_NOINTERACTION=true`, so that bw fails at once when it needs an answer that nobody can type. bw also gets no stdin.
3. It adds `BW_SESSION=<token>` when lazybw has a session token.
4. It adds `BW_PASSWORD=<password>` for `bw unlock` and `bw login`.

lazybw sets `BW_SESSION` itself. It never passes the `BW_SESSION` of the
shell to bw.

### Startup flow

```
bw status (no BW_SESSION)
  → "unauthenticated" → login form  → bw login --passwordenv BW_PASSWORD --raw -- EMAIL
  → "locked"          → unlock form → bw unlock --passwordenv BW_PASSWORD --raw
  → "unlocked"        → unlock form → bw unlock --passwordenv BW_PASSWORD --raw
  → error             → error screen
```

`bw status` runs with no session key, so bw reports `locked` for an
account that is logged in. If bw reports `unlocked`, `RootModel.Update`
treats it as `locked` and shows the unlock form. lazybw does not reuse a
session that the shell exported.

`ParseStatus` returns an error for output
that does not parse, for `null` and for any other status value. The error
starts with `parse bw status output:`. A parse error of `bw list items`
starts with `parse bw list items output:`. Both parsers remove white space
and a leading UTF-8 byte order mark first.

After a successful unlock or login, lazybw stores the token, starts a new
session epoch and runs `bw list items`. The session epoch is a counter
that changes on each unlock and each lock. Each `bw list items` and
`bw sync` result carries the epoch of the session that started it. lazybw
drops and scrubs a result that belongs to an ended session.

### Two-step login

The first `bw login` has no `--method` and no `--code`:

1. If bw needs a method (`Login failed. No provider selected.`), the method form shows: Authenticator app (`--method 0`), Email (`--method 1`) or YubiKey OTP (`--method 3`).
2. If bw asks for a code (`Code is required.`) before the user chose a method, the method form also shows first. Without `--method`, bw sends the code only in a second request to the server. For an account with only the email method, the first request makes the server send a new code. The code that the user typed can then be old. lazybw asks for the method, and then it passes `--method` with the code.
3. If bw did not ask for a code yet and the user picks Email, lazybw runs `bw login --method 1` with no code. Bitwarden then sends the code by email.
4. The code form shows. lazybw runs `bw login --method N --code=CODE`.
5. If bw asks for a code again after it got one, Bitwarden wants a new-device code, and bw has no option for it. The login form then shows a text that tells the user to run `bw login` once in a terminal.

Esc on a two-step form goes back to the login form, and lazybw drops the
master password that it kept for the next run.

### Errors and time limits

A failed bw command returns a `bwcmd.Error`. Its message is the last line
that bw wrote to stderr, without ANSI sequences. `bwcmd/errors.go` matches
the known bw messages, and the screens show plain texts for them. With
`--debug`, the full stderr goes to the debug log, with the session token,
the password, the two-step code and the email replaced by `[hidden]`.

`bw status`, `bw lock` and `bw generate` have 15 seconds. `bw list items`,
`bw sync`, `bw login` and `bw unlock` have 2 minutes. These commands wait
for the network, read the whole vault or run the key derivation. After the
time limit, lazybw kills bw. On Unix, it kills the whole process group, so
that a wrapper script cannot leave bw running with `BW_PASSWORD` in its
environment. The screens show `bw did not answer in time. Try again.`

---

## Clipboard

lazybw copies only with OSC 52, through `tea.SetClipboard`. OSC 52 works
only in a terminal that supports it and has it turned on. In tmux, add
`set -g set-clipboard on` to the tmux configuration. With the default
value, tmux does not pass the copy to the outer terminal. The terminal
sends no reply to OSC 52, so lazybw cannot tell if a copy worked.

After a copy, the footer shows `󰆏 <Field> sent to clipboard · clears in 60s`.
The field names are Password, Card number, Security code, SSN, SSH private
key, Username, Email, Public key, TOTP and Note. The text does not say that
the copy worked, because lazybw cannot know it.

60 seconds after the last copy, lazybw overwrites the clipboard with a
one-space placeholder (`session.ClearPlaceholder = " "`). tmux ignores an
OSC 52 write with an empty payload, so an empty clear can leave the secret
on the clipboard. Only the timer of the latest copy clears the clipboard.
An older timer does nothing, so that it cannot clear a newer value early.
The copy toast ends with the clear.

lazybw also clears the clipboard on a lock, an idle lock and a quit. The
exit cleanup writes the same placeholder after the program stops (see
[Security Considerations](#security-considerations)). lazybw clears the
clipboard only when a copied value can still be on it, so that it leaves
other clipboard content alone.

---

## Opening URLs

`o` opens the first URL of the selected login in the default browser.
`screens/openurl.go` picks the opener for the operating system:

| System | Command |
|---|---|
| macOS | `open URL` |
| Windows | `rundll32 url.dll,FileProtocolHandler URL` |
| Linux and other systems | `xdg-open URL` |

lazybw opens only `http` and `https` URLs. A URL with no scheme, such as
`console.aws.amazon.com` or `192.168.1.1:8080`, gets `https://` in front.
lazybw does not open a URL with another scheme, such as `file://`, or a
URL with no host. lazybw starts the opener and waits for it, so that no
zombie process stays. When the URL is not allowed or the opener fails, the
footer shows the error toast `Open URL failed: …`.

---

## Theme and Background

`--theme` picks the theme. Without the flag, lazybw reads `LAZYBW_THEME`.
Without both, lazybw uses `catppuccin-mocha`. For an unknown name, lazybw
prints a warning that lists the valid themes, and it uses
`catppuccin-mocha`.

`T` opens the theme picker. Enter applies the selected
theme and shows `Theme: <name>`, and Esc cancels. lazybw does not save the
choice.

Each theme has a dark and a light set of colors. At start, lazybw asks the
terminal for its background color. Until the reply comes, lazybw uses the
dark colors. When the reply comes, lazybw sets `ui.IsDark` and applies the
theme again, and each screen restyles its spinner and its list. The huh
forms read `ui.IsDark` each time they draw (`followBackground` in
`ui/theme.go`), so a form that was built before the reply also matches.

### Palette

`ui/theme.go` sets these colors for the active theme and background:

| Color | Use |
|---|---|
| `ColorHighlight` | Login and identity glyphs, titles, hint keys, the header badge, the overlay card border, the filter prompt and the spinners |
| `ColorGreen` | Card and SSH key glyphs, toasts, the copy and success glyphs, enabled generator options, the TOTP bar in the first half of the period |
| `ColorYellow` | Note glyph, the `re-prompt` marker, the TOTP bar while more than a third of the period is left |
| `ColorRed` | Errors, the error glyph, the TOTP bar in the last third of the period |
| `ColorFaint` | Descriptions, hint texts, empty values, group rows that are not selected, and the dimmed background of an overlay |
| `ColorSubtle` | Set by each theme. No screen uses it now. |
| `GradientFrom`, `GradientTo` | The header title and the gradient lines |

The catppuccin themes take their colors from the flavor of the theme on a
dark background, and from Latte on a light background:

| Color | Catppuccin color |
|---|---|
| `ColorHighlight` | Mauve |
| `ColorSubtle` | Surface0 |
| `ColorGreen` | Green |
| `ColorYellow` | Yellow on dark, `#a56a00` on light |
| `ColorRed` | Red |
| `ColorFaint` | Overlay1 on dark, Overlay2 on light |
| Gradient | Mauve to Blue |

The other themes use fixed colors. Each cell gives the dark value first,
then the light value:

| Theme | Highlight | Green | Yellow | Red | Faint | Gradient |
|---|---|---|---|---|---|---|
| `dracula` | `#bd93f9` / `#7c3aed` | `#50fa7b` / `#027A4F` | `#f1fa8c` / `#C47D10` | `#ff5555` / `#CC2222` | `#6272a4` / `#767676` | `#bd93f9` to `#ff79c6` / `#7c3aed` to `#db2777` |
| `charm` | `#7D56F4` / `#5A3ECC` | `#04B575` / `#027A4F` | `#F5A623` / `#C47D10` | `#FF4F4F` / `#CC2222` | `#8a8a8a` / `#767676` | `#ff70a0` to `#7571f9` / `#db2777` to `#4338ca` |
| `base16` | `#a16946` / `#a16946` | `#a1b56c` / `#027A4F` | `#f7ca88` / `#C47D10` | `#ab4642` / `#CC2222` | `#8a8a8a` / `#767676` | `#00bcd4` to `#2196f3` / `#0097a7` to `#1565c0` |

`ColorSubtle` is `#44475a` / `#D9DCCF` in `dracula`, and `#383838` /
`#D9DCCF` in `charm` and `base16`.

Some palette colors have too little contrast for text. lazybw replaces
them with colors that have a contrast of at least 4.5:1. These are
`#767676` on `#ffffff`, `#8a8a8a` on `#1e1e2e`, and the darker Latte
yellow `#a56a00` on `#ffffff`. The text of the header badge is `#1e1e2e` or `#ffffff`,
whichever has the higher WCAG 2 contrast with `ColorHighlight`
(`ui/contrast.go`). For the `charm` theme, lazybw also changes the option
text color of the huh forms. The huh default is hard to read on both
backgrounds.

### Mouse

lazybw turns on no mouse mode, because it handles no mouse events. Click
and drag in the terminal selects text as usual. In the alternate
screen, terminals with alternate scroll mode send arrow keys for the mouse
wheel, so the wheel moves the selection in the list.

---

## Security Considerations

[SECURITY.md](../SECURITY.md) has the full security design. In short:

- The session token is in process memory only. lazybw passes it to bw in `BW_SESSION`, never in an argument and never on disk.
- The master password goes to bw in `BW_PASSWORD`, never in an argument. The forms drop the password after the submit.
- The drawer masks passwords, card numbers, CVVs, SSNs, passport numbers, license numbers and SSH private keys. TOTP codes show, because they end after one period. The Bitwarden apps also show them.
- The list shows `(note)` for a secure note, and never the note text.
- lazybw removes escape sequences and control characters from vault text before it draws the text.
- The clipboard clears 60 seconds after a copy, and on a lock or a quit.
- A lock drops the vault data, zeros the decoded TOTP secret, drops the session token and runs `bw lock`.
- lazybw asks for the master password before it copies or shows a protected field of an item with master password re-prompt.

### Exit cleanup

lazybw locks the vault and clears the clipboard on every way out. The ways
out are `q`, `ctrl+c`, `q` on the error screen, `ctrl+q` on the unlock
screen, SIGTERM, SIGHUP and a panic. Bubble Tea turns SIGTERM into a quit. `main.go` turns
SIGHUP, which the system sends when the terminal closes, into a quit too.

After the program stops, `finish` in `main.go` runs the exit cleanup. It
writes the clipboard placeholder if a copied value can still be on the
clipboard. It runs `bw lock` for a session that is still open. It also runs
`bw lock` again when an earlier `bw lock` did not return, because a closed
terminal can kill that bw process. Then `finish` prints the `bw lock`
errors of the exit to stderr. These are the error of the lock on quit and
the error of the lock in the cleanup.

While the cleanup runs, lazybw ignores SIGHUP, SIGTERM and SIGINT, and the
bw process inherits that.

After a panic in `Update` or `View`, Bubble Tea restores the terminal and
returns no model from `Run`. So `RootModel.Update` records the exit
cleanup of each new model in `lastCleanup`, and `cleanupFor` in `model.go`
uses that record when `Run` returns no model.

### Trace file of Bubble Tea

When `TEA_TRACE` is set, Bubble Tea writes a trace file. A trace file
records every key press, including the master password, and the OSC 52
clipboard payloads. So before `tea.NewProgram`, if `TEA_TRACE` is set,
lazybw prints a warning to stderr and removes the variable. Bubble Tea then
writes no trace file. `newProgram` in `main.go` does both steps in this
order.

### Debug log

`--debug` writes a log to `$XDG_CACHE_HOME/lazybw/debug.log`, or to
`~/.cache/lazybw/debug.log` when `XDG_CACHE_HOME` is not set. The directory
gets mode 0700 and the file gets mode 0600. lazybw appends to the file.
Without `--debug`, lazybw drops all log output, because the default log
target is stderr, which draws over the TUI.

`RootModel` writes a line with no secrets at each state change:

- The status value from `bw status`.
- A successful unlock.
- The item count of each item load.
- The start of each lock: the lock key, an idle lock, a quit, or the lock after a second failed item load.
- The result of `bw lock`.
- The result of each sync.
- Errors. `logReason` in `model.go` gives only the kind of error, for example `timed out` or `bw error (exit status 1)`. The line does not repeat the message of bw, because that message is a line of the bw output.

A failed bw command also writes its subcommand, its error and its stderr,
with the private values replaced by `[hidden]`. The lines of `RootModel`
never hold the session token, the master password, the email, item names
or field values.

---

## Command-line Flags

| Flag | Default | Effect |
|---|---|---|
| `--idle-timeout` | `15m` | Lock the vault after this time with no key press. The minimum is `30s`. A smaller value prints a warning and uses `30s`. |
| `--theme` | `catppuccin-mocha` | The color theme. `LAZYBW_THEME` sets it when the flag is not given. |
| `--debug` | `false` | Write the debug log (see [Debug log](#debug-log)). |
| `--version` | | Print the version and stop. |

`--version` prints `lazybw <version>`. A release build gets the version
from goreleaser through `-X main.version`. A `go install` build has no
ldflags, so the version variable stays `dev`. lazybw then prints the module
version from `runtime/debug.ReadBuildInfo`, for example `lazybw v0.3.0`.

A `go build` in a git clone also has a module version: the tag of the
commit, or a pseudo-version with `+dirty` for changed files. When the build
info has no version or says `(devel)`, lazybw prints `lazybw dev`. A build
with `-buildvcs=false` or from a source archive says `(devel)`.

If `bw` is not on the `PATH`, lazybw prints where to get it and stops.

---

## Project Layout

```
lazybw/
├── main.go               # Flags, theme, debug log, SIGHUP, exit cleanup
├── model.go              # RootModel: state machine, lock, quit, idle lock, clipboard timer
├── screens/
│   ├── locked.go         # Unlock, login and two-step forms (huh)
│   ├── vault.go          # List, drawer, filter, toasts, generator, theme picker, quit card
│   ├── grouping.go       # Group keys and the grouping algorithm
│   ├── reprompt.go       # Master password re-prompt
│   ├── openurl.go        # Allowed URLs and the opener for each system
│   ├── error.go          # Error screen and error block
│   └── usererror.go      # Plain texts for known errors
├── ui/
│   ├── theme.go          # Themes, palette, styles, glyphs, huh theme
│   ├── contrast.go       # WCAG contrast and the header badge text color
│   ├── keymap.go         # VaultKeyMap and the help bindings
│   ├── drawer.go         # Drawer renderer
│   ├── itemrow.go        # Item row and group header row
│   ├── frame.go          # Header, footer and centering
│   ├── gradient.go       # Gradient text and gradient line
│   ├── help.go           # Help overlay
│   ├── hints.go          # Footer hint renderer
│   ├── overlay.go        # Overlay card over the dimmed background
│   └── text.go           # Sanitize, truncate and wrap vault text
├── bwcmd/
│   ├── exec.go           # bw commands, child environment, time limits
│   ├── errors.go         # bw error type and known bw messages
│   ├── kill_unix.go      # Kill the process group after a time limit
│   ├── kill_other.go     # Default kill on other systems
│   ├── parser.go         # Parse bw status and bw list items
│   └── types.go          # Item types
├── session/
│   ├── manager.go        # Session state, epoch, idle test
│   └── clipboard.go      # OSC 52 copy and clear
├── secutil/
│   └── zero.go           # Drop the secrets of vault items
├── totp/
│   └── totp.go           # RFC 6238 TOTP and Steam Guard codes
└── go.mod
```

---

## Not in Scope

- Vault editing (create, update or delete items)
- Organization and collection navigation
- More than one account
- SSH agent integration
- Bitwarden Send
- Reuse of a `BW_SESSION` that the shell exported
- A clipboard path other than OSC 52
