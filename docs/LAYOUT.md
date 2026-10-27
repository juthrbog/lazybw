# lazybw Layout

This page describes how lazybw draws its screens. Every screen draws into one frame. The frame has three zones: a header, a content area and a footer. `RootModel.View` in `model.go` builds the frame. Each screen gives only its content and its footer.

## Principles

1. One frame for all screens. The loading, unlock, login, vault, error and quitting screens use the same three zones. Only the content area changes between screens.
2. No layout jumps. The header and the footer always show, and they always use the same rows. A change of screen does not move them.
3. Constant frame, changing content. The header always shows the app name. The footer shows the keys of the current screen, when the screen has keys.
4. One frame implementation. `RootModel.View` is the only code that builds the frame. The screens do not draw a header or a footer.

## Frame structure

```
 󰦝 lazybw  user@example.com  9 items Login           header, row 1
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄   header, row 2

              (content of the screen)                  content area

 j/k navigate · / search · c pwd       synced 2m ago   footer
```

- Header: 2 rows. Row 1 has the title and the account data. Row 2 is a gradient line.
- Content area: `m.height - ui.ChromeHeight` rows. `ui.ChromeHeight` in `ui/frame.go` is 3: the 2 header rows and the 1 footer row.
- Footer: 1 row. The key hints are on the left, and the status is on the right.

`View` gives the content exactly `m.height - ui.ChromeHeight` rows, and it cuts each line that is wider than the terminal. The renderer of Bubble Tea drops the rows below the last row of the terminal. Without this limit, a content area that is too tall pushes the footer off the screen. When the terminal has 3 rows or fewer, the frame has no content area, only the header and the footer.

The view uses the alternate screen (`AltScreen`). It turns on no mouse mode, because lazybw handles no mouse events. So you can select text with the mouse as in other programs. Some terminals send arrow keys for the mouse wheel in the alternate screen. In these terminals, the wheel moves the cursor in the list.

## Header

```go
func RenderHeader(data HeaderData) string
```

`ui.RenderHeader` in `ui/frame.go` draws the header. `ui.HeaderData` has the fields `Email`, `ItemCount`, `SelectedType` and `Width`. Both rows are exactly `Width` cells wide.

Row 1 has these parts, from left to right:

1. The title " 󰦝 lazybw". It has a color gradient from `GradientFrom` to `GradientTo`.
2. The email of the account, in a badge (`StyleHeaderBadge`: bold text on `ColorHighlight`). lazybw gets the email from `bw status`, or from the login form after a login. The badge shows on every screen when lazybw knows the email. The header cuts a long email. When less than 6 cells are free for the email, the badge does not show.
3. The item count, for example "9 items", in `ColorFaint`. It shows only on the vault screen and only when the terminal is 80 cells wide or more. The count is the number of vault items, not the number of list rows. So group headers and collapsed groups do not change it.
4. The type of the selected item (Login, Card, Note, Identity or SSH Key), in `ColorFaint`. It shows only on the vault screen, only when the terminal is 60 cells wide or more, and only when an item is selected. A group header is not an item.

The email comes first. An item count or a type that does not fit next to the email does not show.

Row 2 is a line of `▄` characters with a color gradient (`ui.RenderGradientLine`).

## Footer

```go
func RenderFooter(hints []HintBinding, status string, width int) string
```

`ui.RenderFooter` in `ui/frame.go` draws the footer. The footer is one row of exactly `width` cells, with 1 cell of padding on each side.

The key hints are on the left. `ui.RenderHints` in `ui/hints.go` draws each key in `StyleHintKey` (bold, `ColorHighlight`). It draws each description and each `·` separator in `ColorFaint`. The status is on the right.

The status gets its space first, but it leaves room for the first hint. The footer drops the hints that do not fit, from the end of the list. It cuts a status that is too long. With no hints and no status, the footer is an empty row.

| Screen | Left (hints) | Right (status) |
|---|---|---|
| Loading | | |
| Login, email field | `enter submit · ctrl+q quit` | |
| Login, master password field | `enter submit · shift+tab back · ctrl+q quit` | |
| Locked | `enter unlock · ctrl+q quit` | |
| Unlocking or logging in | | |
| Two-step login, method | `enter submit · ↑/↓ navigate · esc back · ctrl+q quit` | |
| Two-step login, code | `enter submit · esc back · ctrl+q quit` | |
| Vault | `j/k navigate · / search · c pwd · t totp · ? help · q quit · ctrl+g group · d drawer · p gen · T theme` | Toast or sync age |
| Vault, filter input open | `esc clear · enter confirm · ↑/↓ navigate` | Toast or sync age |
| Vault, filter applied | `filter: <text> · esc clear · ? help · q quit · c pwd · t totp` | Toast or sync age |
| Quit confirmation | `y yes · n no` | |
| Help | `esc close · j/k scroll · / filter` | |
| Help, filter input open | `enter done · esc clear` | |
| Theme picker | `enter select · esc cancel · ↑/↓ navigate · / filter` | |
| Generator, password mode | `enter regen · +/- length · m mode · 1-4 toggles · c copy · esc close` | |
| Generator, passphrase mode | `enter regen · +/- words · m mode · 1-3 options · c copy · esc close` | |
| Master password re-prompt | `enter submit · esc cancel` | |
| Master password re-prompt, while `bw unlock` runs | `esc cancel` | |
| Error | `r retry · q quit` | |
| Quitting | | |

The vault hints put `? help` and `q quit` right after the copy keys. So a narrow terminal drops the other hints first. In the filter applied row, the footer cuts the filter text to 20 cells.

The huh forms (the unlock, login and two-step forms, the master password prompt and the theme picker) show no help line. The footer shows their keys.

While an overlay shows on the vault screen, the footer shows only the keys of the overlay and no status. The toast and the sync age come back when the overlay closes.

## Toasts and the sync age

The status of the vault footer is a toast, when one shows. Otherwise it is the sync age in `ColorFaint`: "synced just now", "synced 2m ago" or "synced 3h ago". lazybw reads the time of the last sync from `bw status` at start and on retry, and the footer shows it after the unlock. lazybw sets it again after each sync. When lazybw knows no sync time, the status is empty.

While a sync runs, the status shows the `SpinnerLoad` spinner and "Syncing…". If a toast shows during the sync, the spinner shows before the toast.

| Toast | Kind | Ends |
|---|---|---|
| `󰆏 <Field> sent to clipboard · clears in 60s` | Copy | At the clipboard clear, after 60s |
| `✓ Vault synced` | Info | After 4s, or when the new items load |
| `No username to copy` | Info | After 4s |
| `Drawer: overlay` or `Drawer: inline` | Info | After 4s |
| `Grouping enabled` or `Grouping disabled` | Info | After 4s |
| `Theme: <name>` | Info | After 4s |
| `✗ Sync failed: <error>` | Error | After 8s |
| `✗ Load failed: <error>` | Error | After 8s |
| `✗ Open URL failed: <error>` | Error | After 8s |
| `✗ Invalid TOTP secret` | Error | After 8s |

The `<Field>` of a copy toast names the field. The names are Password, Card number, Security code, SSN, SSH private key, Username, Email, Public key, TOTP and Note. The generator copies its value as a password.

Info and copy toasts use `StyleToast` (`ColorGreen`, italic). Error toasts use `StyleError` (`ColorRed`, bold). An error from bw goes into the toast as one line of plain text (`ui.SanitizeLine`). A new toast replaces the toast that shows. The timer of an older toast does not end a newer toast.

## Content area by screen

### Loading

The loading screen shows at start while `bw status` runs, after an unlock while the items load, and after a retry. `View` centers the text in the content area:

```


              ⣾  Loading vault…


```

The spinner is `ui.SpinnerLoad`, the `spinner.Dot` braille spinner of bubbles. The text is plain, because the header already shows the app name.

### Unlock and login

`LockedModel.ViewContent(width, contentHeight)` in `screens/locked.go` returns the content. It centers its content with `ui.CenterInArea`.

The unlock screen:

```


              Vault is locked.

              ┃ Master password
              ┃ >


```

The login screen has the title "Log in to Bitwarden". It has an Email field above the Master password field. The forms are 40 cells wide. No "lazybw" title shows in the content area, because the header shows it.

When you submit the form, the spinner `ui.SpinnerUnlock` and the text "Unlocking…" or "Logging in…" take the place of the form. The footer then shows no hints.

An error shows below the form. It starts with `✗` and wraps to the width of the form. For example: "Could not unlock the vault. Check your master password and try again."

If `bw lock` fails after the lock key (`l`) or an idle lock, the unlock screen also shows that error below the form. After a second failed item load, the next unlock screen shows it. The text starts with "bw lock failed:" and tells you to run `bw lock` in a terminal. It shows above the error of a failed unlock, because a short terminal cuts lines at the bottom. The error stays until lazybw leaves the unlock screen.

An idle lock shows the unlock screen at once, so the error comes later. lazybw adds it to the unlock screen that shows, and it keeps the text that you typed. No text shows when bw reports that no account is logged in. On a short terminal, the screen leaves out the blank lines between the title, the form and the errors.

For two-step login, the title is "Two-step login". The method form is a select with the options Authenticator app, Email and YubiKey OTP. The code form asks for the code, and its description says where the code comes from.

### Vault

`VaultModel.ViewContent(width, contentHeight)` in `screens/vault.go` returns the content. The list is at the top, and the drawer is below it.

```
 󰦝 lazybw  user@example.com  9 items Login
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄

  󰌾  Gmail                                                        user@gmail.com
  󰌾  GitHub                                                      dev@example.com
  󰌾  GitHub (Work)                                              work@company.com
▶ 󰌾  AWS Console                                                       aws-admin
  󰿯  Visa Debit                                                        •••• 4242
  󰎞  Recovery codes                                                       (note)
  󰀄  Passport                                                           Ann User
  󰣀  Deploy key                                                       SHA256:abc
  󰌾  GitHub (Personal)                                            me@example.com



▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
  󰌾  AWS Console                                                          Login
  Username    aws-admin  [u] copy
  Password    ••••••••••  [c] copy
  TOTP        313 985  ██▊░ 21s  [t] copy
  URL         https://console.aws.amazon.com  [o] open
  Notes       root account  [n] copy

 j/k navigate · / search · c pwd · t totp · ? help · q quit       synced 2m ago
```

The list is a list from bubbles. Its first row is its title row. The title row is empty, except while the filter input is open. Then it shows "Filter: " and the filter text, and an empty row below it. So the open filter input takes one more row.

Each list row has the cursor (`▶` on the selected row), the glyph of the item type, the name, and a description on the right. The name gets its space first, so that a long description cannot hide it.

The description depends on the item type. A login shows its username. A card shows `••••` and its last 4 digits. A secure note shows "(note)". An identity shows its name or its email. An SSH key shows its fingerprint.

The list gets the rows of the content area that the drawer does not use. That is `height - ui.ChromeHeight - drawerHeight` rows in inline mode, and `height - ui.ChromeHeight` rows in overlay mode. The view draws the list at this size. So a page of the list is the page that shows, and `pgdown` moves by exactly the rows on the screen.

#### Grouping

`ctrl+g` turns grouping on. Two or more items with the same base name then go under a group header. Case does not matter. The base name is the name without the " (…)" and " - …" parts at its end, for example "GitHub" for "GitHub (Work)". An item with a name of its own also joins a group when its name has the group key as a whole word.

```
  󰌾  Gmail                                        user@gmail.com
  ▸  GitHub (3)
  󰌾  AWS Console                                       aws-admin
```

`enter` on a group header expands the group. Its items show below the header, with an indent of 2 cells:

```
  󰌾  Gmail                                        user@gmail.com
▶ ▾  GitHub (3)
    󰌾  GitHub                                    dev@example.com
    󰌾  GitHub (Work)                            work@company.com
    󰌾  GitHub (Personal)                          me@example.com
  󰌾  AWS Console                                       aws-admin
```

A collapsed group shows `▸` (U+25B8), and an expanded group shows `▾` (U+25BE). `▶` (U+25B6) is only the cursor. The label is the base name of the first item of the group, in the case of that item ("GitHub (3)", not "github (3)"). The number is the count of items in the group.

When a group header is selected, the drawer shows the group name and a line such as "3 items · enter to expand". For an expanded group, the line ends with "enter to collapse".

When the rows change, the selection stays on the same item, or on the header of its group. This is true after `ctrl+g`, after `enter` on a group, after `esc` clears a filter, and after a sync. While a filter is active, the list shows no groups.

### Error

`ErrorModel.ViewContent(width)` in `screens/error.go` returns the error text. `View` centers it in the content area:

```


              ✗ bw did not answer in time. Try again.


```

The text uses `StyleError` (`ColorRed`, bold). It wraps to at most 72 cells, and to the terminal width minus a margin of 2 cells on each side. A known error gets a short text, as in the example. For another error, the screen shows the error text. For a bw error, that is the last line that bw printed.

### Quitting

The quitting screen shows while `bw lock` runs, after the lock key (`l`) and after a quit. `View` centers the text:

```


              󰌿 Locking vault…


```

The spinner is `ui.SpinnerLock`. After the lock key, the unlock screen follows. After a quit, lazybw exits. If `bw lock` fails on a quit, lazybw prints the error to stderr after it restores the terminal.

An idle lock does not show this screen. It shows the unlock screen at once.

## Drawer

`ui.RenderDrawer(ui.DrawerProps)` in `ui/drawer.go` draws the drawer for the selected row.

### Inline mode

Inline mode is the default. The drawer is below the list, and it has these rows:

1. A gradient line.
2. The header card: the glyph of the item type, the item name, and on the right the type badge. A secure note with text adds the hint `[n] copy` before the badge. An item with master password re-prompt adds the marker "re-prompt" in `ColorYellow`.
3. The field rows.

The drawer is 8 rows high by default (`ui.DefaultDrawerHeight`). `+` and `-` change the height by one row, and `=` sets it back to 8. The height stays between 3 rows and half of the terminal height. It also leaves the list at least 2 rows: the title row and one item. On a very short terminal, the drawer gets the rows that the list leaves, and at least 1 row.

A field row has a label of 12 cells, the value, and a copy hint such as `[c] copy`. When the hint leaves less than 4 cells for the value, the hint does not show. When an item has more fields than the drawer has rows, the last row shows "↓ N more".

| Item type | Rows |
|---|---|
| Login | Username `[u] copy`, Password `[c] copy`, TOTP `[t] copy` (when the item has a TOTP key), URL `[o] open` (when the item has a URL), Notes `[n] copy` (the first line) |
| Card | Cardholder, Number `•••• •••• •••• 4242` `[c] copy`, Expiry, CVV `•••` `[v] copy` |
| Secure note | The text of the note. `J` and `K` scroll it. |
| Identity | Name, Email `[u] copy`, Phone, Company, SSN `[c] copy`, Passport, License, Address. Only the fields with a value show. |
| SSH key | Fingerprint, Public Key `[u] copy`, Private Key `[c] copy` |

An empty username, password or note of a login shows "(none)" and no copy hint. A card with no CVV shows "(none)" and no `[v]` hint.

The TOTP row shows the code, a 4-cell countdown bar and the seconds left, for example `313 985  ██▊░ 21s`. A 6-digit code gets a space in the middle. The bar is green while more than half of the period is left. It is yellow while more than a third is left, and red for the rest. In a narrow drawer, lazybw first drops the copy hint, and then the seconds. A TOTP key that does not parse shows "invalid TOTP secret" in `StyleError`.

For an item with master password re-prompt, the drawer hides the TOTP code and the note until you enter the master password for the item. The TOTP row and the Notes row show "(hidden)". The text of a secure note shows "(hidden: master password re-prompt)".

When no row is selected, the drawer says that no item is selected, and it names the `j`/`k` and `/` keys. When a group header is selected, the drawer shows the group summary (see [Grouping](#grouping)).

### Overlay mode

`z` switches the drawer to overlay mode, and `z` again switches it back. The toast "Drawer: overlay" or "Drawer: inline" shows.

In overlay mode, the list uses all of the content area, and the drawer shows as a card over the list (see [Overlays](#overlays)). The card has no gradient line. `+`, `-` and `=` do nothing in overlay mode.

The drawer in the card is 3/4 of the terminal width, and at most 80 cells. On a terminal where 3/4 of the width is less than 40 cells, the card uses the full width. The card height fits the fields. A secure note shows at most 7 lines at a time, and `J` and `K` scroll it.

## Overlays

```go
func RenderOverlay(bg, fg string, width, height int) string
```

`ui.RenderOverlay` in `ui/overlay.go` draws a card over the vault content. It dims the vault content: it draws it as plain text in `ColorFaint` with the faint attribute. It draws `fg` in a card with a rounded border in `ColorHighlight` and a padding of 1 row and 2 cells. The card is in the center of the area. The result is exactly `width` x `height` cells. It cuts the content that does not fit in the card, so the border stays.

The vault shows one of these overlays at a time. The drawer in overlay mode stays below them.

| Overlay | Opens with | Content |
|---|---|---|
| Quit confirmation | `q` | "Quit lazybw?", "This will lock your vault and exit." and the keys `y yes · n no` |
| Help | `?` | The title "Keybindings" and the filter state, then the keys in the groups Copy, Navigation, Vault and UI. The last row of the card shows `?/esc close · ↑/↓ scroll`, and `· ↓ more` when more rows are below. While you type a filter, it shows `enter done · esc clear`. |
| Theme picker | `T` | The huh select "Choose theme" with the 7 themes. On a short terminal, the select gets fewer rows and scrolls. |
| Generator | `p` | The title "Password Generator", the mode, the options and the generated value |
| Master password re-prompt | `c`, `t`, `v` or `m` on an item with re-prompt | The title "Master password re-prompt", the text "Enter your master password to continue." and the password field |

The content of the help card is 3/4 of the terminal width, and at most 70 cells. The theme picker is 40 cells wide. The password field of the re-prompt card is 40 cells wide, or less on a narrow terminal.

The generator card in password mode:

```
╭──────────────────────────────────────────────────────────╮
│                                                          │
│  Password Generator                                      │
│                                                          │
│  Mode        [Password]  Passphrase                      │
│  Length      20                                          │
│  Uppercase   ✓   Lowercase  ✓   Numbers  ✓   Special  ✓  │
│                                                          │
│  Xk2#pL9vQ!rT4mW8zN6e                                    │
│                                                          │
╰──────────────────────────────────────────────────────────╯
```

In passphrase mode, the card shows the rows Words, "Capitalize ✓   Include # ✓" and Separator. A long value wraps to at most 72 cells, so that all of it shows in the card. While `bw generate` runs, the value is "Generating…". An error shows in `StyleError` in place of the value.

In the re-prompt card, an error takes the place of the text "Enter your master password to continue.", above the password field. So a short terminal does not cut the error. While `bw unlock` runs, the card shows "Checking the master password…" in place of the field.

## Centering

```go
func CenterInArea(content string, width, height int) string
```

`ui.CenterInArea` in `ui/frame.go` centers content in an area:

1. It splits the content into lines.
2. It measures the widest line with `lipgloss.Width()`, which ignores ANSI codes.
3. It adds `(height - lineCount) / 2` empty lines above the content.
4. It adds `(width - widestLine) / 2` spaces before each line.
5. It cuts content that is taller than the area at the bottom. It adds empty lines below the content, so that the result has exactly `height` lines.

`View` uses it for the loading, error and quitting screens. `LockedModel.ViewContent` uses it for the unlock and login screens. No screen has its own centering code.

## Size propagation

When a `tea.WindowSizeMsg` comes to the root model:

1. The root model stores `m.width` and `m.height`.
2. The root model sends the same `tea.WindowSizeMsg` to the active screen: the unlock and login screen, or the vault screen. lazybw has no message type of its own for the size.
3. The vault screen subtracts `ui.ChromeHeight` itself. It limits the drawer height, and it sizes the list to the content height minus the drawer height. It also resizes the help, the theme picker and the password field. The unlock and login screen keeps the raw size and gets the content height from `View`.

In `View`, the root model gives `m.width` and `m.height - ui.ChromeHeight` to `ViewContent` of the screen. The error screen gets only the width. `NewVaultModel` gets the size from the root model when the items load, so the first frame of the vault has the correct size.

## Code map

| File | What it draws |
|---|---|
| `model.go` | `RootModel.View`: the frame, and the loading and quitting content |
| `ui/frame.go` | `ChromeHeight`, `RenderHeader`, `RenderFooter` and `CenterInArea` |
| `ui/hints.go` | `RenderHints`: the key hints of the footer |
| `ui/gradient.go` | `GradientText` and `RenderGradientLine` |
| `ui/overlay.go` | `RenderOverlay`: the overlay cards |
| `ui/drawer.go` | `RenderDrawer`: the drawer |
| `ui/itemrow.go` | `RenderItemRow` and `RenderGroupRow`: the list rows |
| `ui/help.go` | `HelpOverlay`: the content of the help card |
| `screens/locked.go` | `LockedModel.ViewContent` and `FooterContent`: the unlock, login and two-step screens |
| `screens/vault.go` | `VaultModel.ViewContent` and `FooterContent`: the vault, and the overlay card that shows |
| `screens/vault_list.go` | The list rows and the selection |
| `screens/vault_drawer.go` | The drawer size, the drawer props and the list above the drawer |
| `screens/toast.go` | The toasts and the status on the right of the footer |
| `screens/generator.go` | `GeneratorModel`: the generator card and its hints |
| `screens/themepicker.go` | `ThemePickerModel`: the theme picker card and its hints |
| `screens/quitconfirm.go` | The quit confirmation card and its hints |
| `screens/reprompt.go` | The master password re-prompt card and its hints |
| `screens/error.go` | `ErrorModel.ViewContent` and `FooterHints`: the error screen |
