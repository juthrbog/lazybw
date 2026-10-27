# Glyphs

This page lists each glyph that lazybw draws, and the rules for a new glyph. A glyph is one character on the screen, for example an icon or a block shape.

## Font

lazybw takes its icons from Nerd Fonts. Use a Nerd Fonts v3 font in its Mono variant, for example "JetBrainsMono Nerd Font Mono". Without a Nerd Font, the icons, the header logo and the unlock spinner show as empty boxes.

The icons are Material Design icons, the nf-md- set of Nerd Fonts. The code points on this page are the Nerd Fonts v3 code points.

The names and the code points on this page come from the font files of Nerd Fonts 3.5.1. In the font file, a glyph name has no prefix, for example `md-lock`. The Nerd Fonts cheat sheet adds the `nf-` prefix: `nf-md-lock`.

## Design principles

1. Use nf-md- icons (Material Design) for icons. This set has the most icons for security and credentials. All Nerd Font icons in lazybw are from this set.
2. Use plain Unicode for shapes: triangles, blocks and bullets. They need no Nerd Font.
3. Follow the Charm apps. huh and bubbles use plain Unicode, such as `✓`, `•` and `→`. lazybw does the same for each glyph that is not an icon.
4. Use a Nerd Font Mono font, not a Propo font. A Mono font draws each icon in one cell.

## Glyph sets

`ui/glyphs.go` holds each glyph in a `GlyphSet`. `ui.Glyphs` is the active set. There are two sets:

- The Unicode set, the default. It uses the glyphs on this page.
- The ASCII set, for `--ascii` or `LAZYBW_ASCII=1`. It uses only printable ASCII. See [ASCII mode](#ascii-mode).

`main` calls `ui.UseASCII` before it applies the theme, because the theme colors the glyphs and the screens copy the spinners when they are built.

## Current glyphs

### Item type icons (`ui/theme.go`)

`initStyles()` renders each glyph in its theme color. `ItemGlyph` in `ui/drawer.go` returns the glyph for an item type. The list rows and the header card of the drawer show it.

| Variable | Glyph | Code point | Nerd Font name | Color |
| --- | --- | --- | --- | --- |
| `GlyphLogin` | 󰌾 | U+F033E | `nf-md-lock` | `ColorHighlight` |
| `GlyphCard` | 󰿯 | U+F0FEF | `nf-md-credit_card` | `ColorGreen` |
| `GlyphNote` | 󰎞 | U+F039E | `nf-md-note_text` | `ColorYellow` |
| `GlyphIdentity` | 󰀄 | U+F0004 | `nf-md-account` | `ColorHighlight` |
| `GlyphSSHKey` | 󰣀 | U+F08C0 | `nf-md-ssh` | `ColorGreen` |

### Header logo (`ui/frame.go`)

| Field | Glyph | Code point | Nerd Font name | Use |
| --- | --- | --- | --- | --- |
| `Glyphs.Logo` | 󰦝 | U+F099D | `nf-md-shield_lock` | The icon before "lazybw" in the header |

The logo is part of the title text, so it gets the color gradient of the title. Nerd Fonts has no Bitwarden glyph.

### Lock and unlock spinners (`ui/theme.go`)

`SpinnerUnlock` shows these frames in this order, 3 frames per second. `SpinnerLock` shows them in the reverse order.

| Frame | Glyph | Code point | Nerd Font name |
| --- | --- | --- | --- |
| 1 | 󰌾 | U+F033E | `nf-md-lock` |
| 2 | 󰷖 | U+F0DD6 | `nf-md-key_outline` |
| 3 | 󰌆 | U+F0306 | `nf-md-key` |
| 4 | 󰌿 | U+F033F | `nf-md-lock_open` |

The unlock screen shows `SpinnerUnlock` while `bw unlock` or `bw login` runs. The "Locking vault…" screen shows `SpinnerLock`.

`SpinnerLoad` is `spinner.Dot` from bubbles. Its frames are the braille characters ⣾ ⣽ ⣻ ⢿ ⡿ ⣟ ⣯ ⣷ (U+28FE, U+28FD, U+28FB, U+28BF, U+287F, U+28DF, U+28EF and U+28F7). It shows while the vault loads and while a sync runs. It needs no Nerd Font.

### Status glyphs (`ui/theme.go`)

| Variable | Glyph | Code point | Nerd Font name | Color | Use |
| --- | --- | --- | --- | --- | --- |
| `GlyphSuccess` | ✓ | U+2713 | none (Unicode) | `ColorGreen` | The "Vault synced" toast |
| `GlyphError` | ✗ | U+2717 | none (Unicode) | `ColorRed` | Error toasts, the error screen, and the errors of the unlock screen, the re-prompt prompt and the generator |
| `GlyphCopy` | 󰆏 | U+F018F | `nf-md-content_copy` | `ColorGreen` | The copy toast |
| `GlyphSave` | 󰆓 | U+F0193 | `nf-md-content_save` | `ColorGreen` | The "Created" toast after the item form saves |
| `GlyphEdit` | 󰏫 | U+F03EB | `nf-md-pencil` | `ColorGreen` | The "Saved" toast after an edit |

### Generator options (`screens/generator.go`)

| Glyph | Code point | Use |
| --- | --- | --- |
| ✓ | U+2713 | An option that is on, in `ColorGreen` |
| · | U+00B7 | An option that is off, in `ColorFaint` |

### TOTP countdown bar (`ui/drawer.go`)

The bar is 4 cells wide. Each cell has 8 fill steps, so the bar has 32 steps for one TOTP period. For a period of 30 seconds, the bar changes about once a second.

| Glyph | Code point | Use |
| --- | --- | --- |
| █ | U+2588 | A full cell |
| ▉ ▊ ▋ ▌ ▍ ▎ ▏ | U+2589 to U+258F | A cell that is part full |
| ░ | U+2591 | The empty track, in `ColorFaint` |

The color of the bar depends on the time that is left. The bar is green while more than half of the period is left. It is yellow while more than a third is left, and red for the rest. For a period of 30 seconds, the bar is green above 15 seconds and yellow from 11 to 15 seconds. It is red at 10 seconds and less.

### List rows (`ui/itemrow.go`)

| Glyph | Code point | Use |
| --- | --- | --- |
| ▶ | U+25B6 | The cursor on the selected row. lazybw uses it only for the cursor. |
| ▸ | U+25B8 | A collapsed group |
| ▾ | U+25BE | An expanded group |

The group glyphs are not the cursor glyph. So a selected collapsed group does not show the same glyph two times.

### Masks (`ui/drawer.go`, `bwcmd/types.go`)

| Glyph | Code point | Use |
| --- | --- | --- |
| • | U+2022 | Hides the password, the card number, the CVV, the SSN, the passport number, the license number and the SSH private key in the drawer. The list row of a card shows •••• and the last 4 digits. |

### Frame and text

| Glyph | Code point | File | Use |
| --- | --- | --- | --- |
| ▄ | U+2584 | `ui/gradient.go` | The gradient line below the header and above the inline drawer |
| … | U+2026 | `ui/text.go` and others | The end of a cut text (`Truncate`), and texts such as "Loading vault…" |
| · | U+00B7 | `ui/hints.go` | The separator between the footer hints. The copy toast also uses it. |
| ↑ ↓ | U+2191, U+2193 | `screens/`, `ui/drawer.go`, `ui/help.go` | The hint "↑/↓ navigate", the "↓ N more" row of the drawer, and "↓ more" in the help |
| ╭ ╮ ╰ ╯ ─ │ | U+256D, U+256E, U+2570, U+256F, U+2500, U+2502 | `ui/overlay.go` | The rounded border of the overlay cards (`lipgloss.RoundedBorder`) |

The huh forms draw their own glyphs, for example the field border ┃ (U+2503). The huh theme sets them. The huh forms are the unlock, login and two-step forms, the re-prompt prompt and the theme picker.

## ASCII mode

With `--ascii` or `LAZYBW_ASCII=1`, lazybw draws each glyph of the table below in its ASCII form. Every terminal draws these characters 1 cell wide, and they need no Nerd Font. Vault text, such as item names, stays as it is.

| Glyph in the Unicode set | ASCII form | Use |
| --- | --- | --- |
| 󰦝 | `#` | The header logo |
| Item icons 󰌾 󰿯 󰎞 󰀄 󰣀 | `L` `C` `N` `I` `K` | Login, card, note, identity and SSH key, in their theme colors |
| 󰆏 ✓ ✗ | `*` `+` `!` | The copy toast, success and error |
| 󰆓 | `S` | The "Created" toast after the item form saves |
| 󰏫 | `E` | The "Saved" toast after an edit |
| ▶ | `>` | The cursor of the selected row |
| ▸ ▾ | `+` `-` | A collapsed and an expanded group |
| • | `*` | A mask |
| … | `...` | The end of a cut text and of a busy text |
| · between hints | ` \| ` | The footer separator |
| · in a text | ` - ` | For example in the copy toast |
| ✓ · in the generator | `x` `-` | An option that is on or off |
| ↑ ↓ | `^` `v` | The arrow keys and the scroll cues |
| — | `-` | The dash in the text of an empty drawer |
| ▄ | `-` | The line under the header and above the drawer |
| █ ▉ to ▏ ░ | `#`, `=` or `-`, `.` | The TOTP bar: a full cell, a partial cell, and the track |
| Spinners | `spinner.Line` (`\|` `/` `-` `\`) | Loading, locking and unlocking |
| ╭ ─ ╮ │ ╰ ╯ | `+` `-` `\|` | The border of an overlay card (`lipgloss.ASCIIBorder`) |
| ┃ | `\|` | The left border of a focused form field |

## Rules for a new glyph

### Item types: nf-md- icons in a theme color

For a new Bitwarden item type, pick an nf-md- icon and a theme color. The item type table above shows the current icons.

### Status: plain Unicode

| Meaning | Glyph | Code point |
| --- | --- | --- |
| Success | ✓ | U+2713 |
| Error | ✗ | U+2717 |
| Warning | ▲ | U+25B2 |
| Copy | 󰆏 | U+F018F (`nf-md-content_copy`, the one icon in this table) |
| Loading | a spinner | none |
| On | ✓ | U+2713 |
| Off | · | U+00B7 |

### Progress: Unicode block shapes

| Meaning | Glyphs | Code points |
| --- | --- | --- |
| TOTP countdown | █▉▊▋▌▍▎▏ and the track ░ | U+2588 to U+258F, U+2591 |
| Progress bar | █ ▉ ▊ ▋ ▌ ▍ ▎ ▏ | U+2588 to U+258F |
| Shade fill | ░ ▒ ▓ | U+2591 to U+2593 |

### Navigation: Unicode triangles and arrows

| Meaning | Glyph | Code point |
| --- | --- | --- |
| Cursor | ▶ | U+25B6 |
| Collapsed group | ▸ | U+25B8 |
| Expanded group | ▾ | U+25BE |
| Up and down hints | ↑ ↓ | U+2191, U+2193 |
| Left and right hints | ← → | U+2190, U+2192 |

## Candidate glyphs for future work

### TOTP countdown (#38, done)

The 4-cell bar replaced the indicator with 5 circle steps. It has 32 steps, so it changes about once a second. A possible next step is a timer icon before the bar.

| Concept | Glyph | Code point | Nerd Font name |
| --- | --- | --- | --- |
| Timer | 󱎫 | U+F13AB | `nf-md-timer` |
| Timer outline | 󰔛 | U+F051B | `nf-md-timer_outline` |

### Vault editing (#6)

| Concept | Glyph | Code point | Nerd Font name |
| --- | --- | --- | --- |
| Delete | 󰆴 | U+F01B4 | `nf-md-delete` |
| Undo | 󰕌 | U+F054C | `nf-md-undo` |

### Folders and collections (#7)

| Concept | Glyph | Code point | Nerd Font name |
| --- | --- | --- | --- |
| Folder, closed | 󰉋 | U+F024B | `nf-md-folder` |
| Folder, open | 󰝰 | U+F0770 | `nf-md-folder_open` |
| Collection | 󰚝 | U+F069D | `nf-md-folder_star` |

### Password strength

| Concept | Glyph | Code point | Nerd Font name |
| --- | --- | --- | --- |
| Shield | 󰒘 | U+F0498 | `nf-md-shield` |
| Shield, OK | 󰕥 | U+F0565 | `nf-md-shield_check` |
| Shield, warning | 󰻌 | U+F0ECC | `nf-md-shield_alert` |

### Copy actions

| Concept | Glyph | Code point | Nerd Font name |
| --- | --- | --- | --- |
| Copy | 󰆏 | U+F018F | `nf-md-content_copy` |
| Copied | 󰄬 | U+F012C | `nf-md-check` |

### Show and hide

| Concept | Glyph | Code point | Nerd Font name |
| --- | --- | --- | --- |
| Show | 󰈈 | U+F0208 | `nf-md-eye` |
| Hide | 󰈉 | U+F0209 | `nf-md-eye_off` |

## Glyphs to avoid

- Emoji (U+1F000 and higher). Their width is not the same in all terminals, and they break the alignment of the rows.
- CJK characters. The Unicode standard makes them 2 cells wide, so they break the columns.
- Characters with the East Asian Width "Ambiguous" (A), if the alignment depends on them. A terminal with a CJK locale, or with the setting that draws these characters wide, draws them 2 cells wide. lipgloss counts them as 1 cell. Many glyphs on this page are ambiguous: ▶ ▸ ▾ • … · ↑ ↓ — ▄ █ ░, the box drawing characters of the borders, and the Nerd Font icons in the Private Use Area. lazybw keeps them in the Unicode set, and the ASCII set replaces each one. A new glyph needs an ASCII form too.
- The nf-weather- set. Its glyphs are large, and their cell width is not the same in all fonts.
- Nerd Font glyphs with many paths. They can go past the cell in a font that is not a Mono font.
- Mixed icon sets. Use nf-md- icons. Do not use nf-fa- icons and nf-md- icons for one meaning.

## Steps to add a glyph

1. Find an nf-md- icon in the [Nerd Fonts cheat sheet](https://www.nerdfonts.com/cheat-sheet).
2. Use the rules above to pick an icon or a plain Unicode glyph.
3. Add a field to `GlyphSet` in `ui/glyphs.go`. Give it a value in `unicodeGlyphs` and a printable ASCII value in `asciiGlyphs`. Draw it through `ui.Glyphs`. For a glyph with a theme color, also add a `Glyph*` variable in `ui/theme.go`, and set it in `initStyles()` from `Glyphs`.
4. If the glyph is an item glyph or the header glyph, add its rune to `TestGlyphRunes` in `ui/theme_test.go`. That test fails when the rune of an item glyph or of the header glyph changes.
5. Make sure that the glyph shows correctly in at least two terminals, for example kitty, WezTerm or Ghostty. `TestASCIIModeDrawsOnlyASCII` in `ascii_test.go` fails when the ASCII mode draws a character that is not ASCII.
6. Add the glyph to this page.
