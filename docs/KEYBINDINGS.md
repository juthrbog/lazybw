# Keybindings

This page lists the keys of each lazybw screen. The key names are the names that Bubble Tea gives the keys, for example `enter`, `shift+down` or `pgdown`.

A test in `ui/keybindings_doc_test.go` reads this file. It fails when a key of `DefaultVaultKeyMap`, `DefaultGeneratorKeyMap`, `DefaultConfirmKeyMap`, `DefaultNewItemKeyMap` or `DefaultFormKeyMap` (`ui/keymap.go`) is not on this page as inline code.

## All screens

| Key | Action |
|---|---|
| `ctrl+c` | Quit at once, with no confirmation. lazybw locks the vault, clears the clipboard if it copied a value, and exits. |

When the idle timeout (`--idle-timeout`, 15 minutes by default) is over, lazybw locks the vault. A key press after the timeout only locks the vault. It does not run its action. `ctrl+c` still quits, and the quit also locks the vault.

## Vault

### Move in the list

The vault list is a list from the bubbles library, and it handles these keys. lazybw changes the page keys, and it turns off the list keys that lazybw uses for other actions.

| Key | Action |
|---|---|
| `j` / `down` | Move the cursor down one row |
| `k` / `up` | Move the cursor up one row |
| `g` / `home` | Go to the first row |
| `G` / `end` | Go to the last row |
| `pgdown` / `right` | Go one page down. A page is the list rows that show on the screen. |
| `pgup` / `left` | Go one page up |
| `/` | Open the filter input. See [Filter mode](#filter-mode). |
| `esc` | Clear the filter, when a filter is applied |

The page keys work only when the list has more than one page.

### Actions

These are the action bindings of `DefaultVaultKeyMap` in `ui/keymap.go`. The help overlay, the footer and the drawer hints take their key labels from these bindings. `DefaultVaultKeyMap` also has the bindings `Navigate`, `Jump` and `Filter` for the list keys above, so that the help overlay and the footer can name them. The list handles those keys.

| Key | Action |
|---|---|
| `c` | Copy the main secret. That is the password of a login, the card number, the SSN of an identity or the private key of an SSH key. |
| `t` | Copy the TOTP code. If the TOTP key does not parse, lazybw shows the error toast "Invalid TOTP secret". |
| `u` | Copy the username of a login, the email of an identity, or the public key of an SSH key |
| `m` | Copy the note |
| `v` | Copy the security code (CVV) of a card |
| `o` | Open the first URL of a login in the default browser. lazybw opens only http and https URLs. It adds `https://` to a URL that has no scheme. |
| `J` / `shift+down` | Scroll the note in the drawer down. Only a secure note scrolls. |
| `K` / `shift+up` | Scroll the note in the drawer up |
| `+` | Make the inline drawer one row taller |
| `-` | Make the inline drawer one row shorter |
| `=` | Set the inline drawer to its default height of 8 rows |
| `z` | Switch the drawer between inline mode and overlay mode |
| `n` | Create an item. See [New item](#new-item). |
| `e` | Edit the selected login or secure note. See [Item form](#item-form). |
| `p` | Open the password generator |
| `r` | Sync the vault with `bw sync`, and then load the items again. While a sync runs, `r` does nothing. |
| `l` | Lock the vault and show the unlock screen |
| `ctrl+g` | Turn item grouping on or off |
| `enter` | Expand or collapse the selected group |
| `T` | Open the theme picker |
| `?` | Open the help overlay |
| `q` | Ask for confirmation, and then lock the vault and quit. See [Quit confirmation](#quit-confirmation). |

`ctrl+c` is not in `DefaultVaultKeyMap`. The root model quits on it before a screen gets the key. See [All screens](#all-screens).

A copy key copies nothing when the selected item does not have that field. When the username of a login is empty, `u` shows the toast "No username to copy".

The drawer keys `+`, `-` and `=` do nothing in overlay mode. The drawer height stays between 3 rows and half of the terminal height.

When an item has master password re-prompt, `c`, `t`, `v`, `m` and `e` first ask for the master password. See [Master password re-prompt](#master-password-re-prompt).

While an overlay shows (help, generator, theme picker, re-prompt, new item or quit confirmation), the vault keys do nothing. The overlay gets the keys.

## Filter mode

The filter input opens after `/`. The list then shows only the items that match the filter text.

| Key | Action |
|---|---|
| Text | Type into the filter. Each time the filter text changes, the cursor goes to the best match (the first row). |
| `up` / `down` | Move the cursor in the matches. The filter input stays open. |
| `enter` | Accept the filter. The input closes and the list keeps only the matches. With no filter text, `enter` does nothing. With no match, `enter` clears the filter. |
| `esc` | Clear the filter and close the input |

All other keys type into the filter. For example, `j`, `k`, `q` and `l` are text in the filter, not vault keys. Only `enter` accepts the filter.

When `esc` clears the filter, the selected match stays selected. If the filter has no text, or no match is selected, lazybw selects the item that was selected before `/` again. A group header counts as an item here.

After `enter`, the vault keys work again on the matches. The footer shows the filter text. `esc` clears the filter, and `/` opens the input again with the same text.

While a filter is active, the list shows no groups. So the filter also finds the items of collapsed groups.

## Quit confirmation

The dialog opens after `q` on the vault screen. These are the bindings of `DefaultConfirmKeyMap` in `ui/keymap.go`.

| Key | Action |
|---|---|
| `y` | Lock the vault and quit |
| `n` / `esc` | Close the dialog and go back to the vault |

The dialog ignores all other keys. `ctrl+c` quits at once.

## Help overlay

The help overlay opens after `?`. It lists the vault keys in four groups: Copy, Navigation, Vault and UI.

| Key | Action |
|---|---|
| `?` / `esc` | Close the help |
| `j` / `down` | Scroll down one line |
| `k` / `up` | Scroll up one line |
| `pgdown` / `f` / `space` | Scroll down one page |
| `pgup` / `b` | Scroll up one page |
| `ctrl+d` / `d` | Scroll down half a page |
| `ctrl+u` / `u` | Scroll up half a page |
| `/` | Start to type a filter |

While you type a filter:

| Key | Action |
|---|---|
| Text | Add the character to the filter. The help shows only the keys whose key, description or group contains the filter text. Case does not matter. |
| `backspace` | Delete the last character of the filter |
| `enter` | Keep the filter and go back to the scroll keys |
| `esc` | Clear the filter and go back to the scroll keys |

## Theme picker

The theme picker opens after `T`. It is a select form of huh.

| Key | Action |
|---|---|
| `up` / `down` | Move to the previous or the next theme. `k` and `j` also work. |
| `/` | Filter the theme names |
| `enter` | Apply the selected theme. The footer shows the toast "Theme: <name>". |
| `esc` | Close the picker and keep the current theme |

`esc` always closes the picker, also while the filter is open. The theme that you select stays until lazybw exits. To set a theme at start, use `--theme` or `LAZYBW_THEME`.

## Password generator

The generator opens after `p`. It opens in password mode, with a length of 20 and all four character sets on. Each change of an option makes a new value at once. The keys are the bindings of `DefaultGeneratorKeyMap` in `ui/keymap.go`, and the footer of the generator takes its key labels from them.

Both modes:

| Key | Action |
|---|---|
| `enter` / `r` | Make a new value. When the item form opened the generator, only `r` makes a new value. |
| `m` / `tab` | Switch between password mode and passphrase mode |
| `c` | Copy the value. `c` copies nothing while `bw generate` runs or when an error shows. |
| `esc` | Close the generator |

When the item form opened the generator with `ctrl+g`, `enter` puts the value in the password field and closes the generator. See [Item form](#item-form).

While the generator shows, the footer shows only its keys. After `c`, the toast "Password sent to clipboard" shows in the footer when you close the generator.

Password mode:

| Key | Action |
|---|---|
| `+` / `=` | Make the password one character longer, up to 128 |
| `-` | Make the password one character shorter, down to 5 |
| `1` | Turn uppercase letters on or off |
| `2` | Turn lowercase letters on or off |
| `3` | Turn numbers on or off |
| `4` | Turn special characters on or off |

Passphrase mode starts with 4 words, the separator `-`, capital letters on and a number on.

| Key | Action |
|---|---|
| `+` / `=` | Add one word, up to 20 |
| `-` | Remove one word, down to 3. `bw generate` raises a smaller count to 3. |
| `1` | Turn capital letters on or off |
| `2` | Turn the number on or off |
| `3` | Go to the next separator: `-`, `_`, `.` or space |

In passphrase mode, `4` does nothing.

## New item

`n` opens a card that asks for the type of the new item.

| Key | Action |
|---|---|
| `l` | Open the item form for a login |
| `s` | Open the item form for a secure note |
| `esc` | Close the card |

## Item form

`n` opens the item form for a new item, and `e` opens it for the selected item. For an edit, the fields start with the values of the item, and lazybw saves with `bw edit item`. `e` edits logins and secure notes. For another item type, `e` shows the toast "Editing <type> items is not supported yet".

The item form shows in place of the list and the drawer. A login has the fields Name, Username, Password, TOTP, URI and Notes. A secure note has the fields Name and Notes. Letters type text, so the form commands use `tab`, `ctrl` and `esc`.

| Key | Action |
|---|---|
| `tab` | Go to the next field |
| `shift+tab` | Go to the previous field |
| `enter` | Add a new line in Notes. In the other fields, go to the next field. |
| `ctrl+s` | Save the item with `bw create item` or `bw edit item` |
| `ctrl+g` | Open the password generator. Only the Password field takes this key. |
| `ctrl+r` | Show or hide the password. The password is hidden each time the form opens. |
| `esc` | Close the form. If you changed a value, a dialog asks "Discard changes?" first: `y` discards the changes and `n` or `esc` goes back to the form. |

The URI field has one line for each URI and one empty line. When you type in the empty line, a new empty line shows below it. lazybw does not save empty lines, so to remove a URI, clear its line.

The name is required. The TOTP field takes an `otpauth://` URI or a base32 key. If a check fails, the field shows the error and lazybw does not save.

While `bw` saves, the form shows "Saving…" and ignores all keys. If `bw` fails, the form shows the error and keeps all values, and `ctrl+s` tries again. If `bw` succeeds, the form closes, lazybw loads the items again, selects the item and shows the toast "Created" or "Saved" with the name.

If the vault locks while the form shows, lazybw discards the form and its values.

## Master password re-prompt

A Bitwarden item can have master password re-prompt. For such an item, `c`, `t`, `v`, `m` and `e` open a prompt that asks for the master password. `u` and `o` do not ask.

| Key | Action |
|---|---|
| `enter` | Submit the master password. lazybw runs `bw unlock` with it. If `bw unlock` accepts the password, lazybw copies the field or opens the edit form. |
| `esc` | Close the prompt and copy nothing. `esc` also works while `bw unlock` runs. |

While `bw unlock` runs, the prompt ignores all other keys. The prompt does not accept an empty password.

After `bw unlock` accepts the password, the copy keys of the item do not ask again while the item stays selected. When you select another item, lock the vault, or a sync removes the item, the next copy asks again.

## Unlock and login screens

The unlock screen shows when the vault is locked. The login screen shows when no account is logged in. The forms show no help line. The footer shows their keys.

Unlock screen:

| Key | Action |
|---|---|
| `enter` | Unlock the vault with the master password |
| `ctrl+q` | Quit |

Login screen:

| Key | Action |
|---|---|
| `enter` | On the email field, go to the master password field. On the master password field, log in. |
| `tab` | Go from the email field to the master password field |
| `shift+tab` | Go back from the master password field to the email field |
| `ctrl+q` | Quit |

A form does not accept an empty master password or an email that is not valid. It shows an error and stays on the field.

While `bw unlock` or `bw login` runs, only `ctrl+q` and `ctrl+c` work.

### Two-step login

If the account uses two-step login, lazybw first asks for the method and then for the code.

Method form:

| Key | Action |
|---|---|
| `up` / `down` | Select the method: Authenticator app, Email or YubiKey OTP. `k` and `j` also work. |
| `enter` | Use the selected method |
| `esc` | Go back to the login form |
| `ctrl+q` | Quit |

Code form:

| Key | Action |
|---|---|
| `enter` | Log in with the code |
| `esc` | Go back to the login form |
| `ctrl+q` | Quit |

`esc` drops the master password that lazybw kept for the two-step login. You then type it again in the login form.

## Error screen

The error screen shows when `bw status` or the item load fails.

| Key | Action |
|---|---|
| `r` | Try again |
| `q` | Quit at once, with no confirmation. lazybw locks the vault if it is unlocked, and clears the clipboard if it copied a value. |

If the vault is unlocked and the item load failed, `r` loads the items again. Otherwise `r` runs `bw status` again. If the item load fails a second time, lazybw locks the vault, so that the next `r` shows the unlock screen. If that `bw lock` fails, the unlock screen shows its error.
