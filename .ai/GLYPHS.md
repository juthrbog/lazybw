# Glyph & Icon Design Guide

Reference for every glyph used in lazybw and conventions for adding new ones.

## Design Principles

1. **Primary icon set: nf-md- (Nerd Font Material Design)** — broadest coverage for security/credential concepts (~6,000 glyphs). All current Nerd Font icons in lazybw already use nf-md-.
2. **Plain Unicode for geometric indicators** — circles, triangles, blocks, bullets. These don't require Nerd Fonts and render consistently.
3. **Charm ecosystem alignment** — Charm apps (huh, bubbles) use plain Unicode (`✓`, `•`, `→`, `○`). We follow this for non-icon glyphs.
4. **Nerd Font Mono required** — users should use a "Nerd Font Mono" variant (not Propo) to ensure single-cell-width rendering of PUA glyphs.

## Current Glyph Inventory

### Item Type Icons (nf-md-, `ui/theme.go`)

| Variable       | Glyph | Codepoint | NF Name             | Color          |
| -------------- | ----- | --------- | -------------------- | -------------- |
| `GlyphLogin`   | 󰌾    | U+F033E   | `nf-md-lock`         | ColorHighlight |
| `GlyphCard`    | 󰁯    | U+F006F   | `nf-md-credit_card`  | ColorGreen     |
| `GlyphNote`    | 󱙒    | U+F1652   | `nf-md-note_text`    | ColorYellow    |
| `GlyphIdentity`| 󰀄    | U+F0004   | `nf-md-account`      | ColorHighlight |
| `GlyphSSHKey`  | 󰣀    | U+F08C0   | `nf-md-console`      | ColorGreen     |

### App Branding (`ui/frame.go`)

| Glyph | Codepoint | NF Name           | Usage                 |
| ----- | --------- | ------------------ | --------------------- |
| 󰊙    | U+F0299   | `nf-md-bitwarden`  | Header title icon     |

### Lock/Unlock Spinners (`ui/theme.go`)

| Glyph | Codepoint | NF Name                | Role              |
| ----- | --------- | ----------------------- | ----------------- |
| 󰌾    | U+F033E   | `nf-md-lock`            | Locked state      |
| 󰷖    | U+F1DD6   | `nf-md-lock_clock`      | Transition frame  |
| 󰌆    | U+F0306   | `nf-md-key`             | Transition frame  |
| 󰌿    | U+F033F   | `nf-md-lock_open`       | Unlocked state    |

### TOTP Countdown Circles (`ui/drawer.go`)

| Glyph | Codepoint | Condition       | Meaning            |
| ----- | --------- | --------------- | ------------------ |
| ●     | U+25CF    | secsLeft > 24   | Full (>80%)        |
| ◕     | U+25D5    | secsLeft > 18   | Three-quarters     |
| ◑     | U+25D1    | secsLeft > 12   | Half               |
| ◔     | U+25D4    | secsLeft > 6    | Quarter            |
| ○     | U+25CB    | secsLeft <= 6   | Empty (expiring)   |

### Navigation & Selection (`ui/itemrow.go`)

| Glyph | Codepoint | Usage                        |
| ----- | --------- | ---------------------------- |
| ▶     | U+25B6    | Cursor / selected item       |
| ▼     | U+25BC    | Expanded group indicator     |

### Status Indicators (`screens/vault.go`)

| Glyph | Codepoint | Usage                        |
| ----- | --------- | ---------------------------- |
| ✓     | U+2713    | Success / enabled toggle     |
| ·     | U+00B7    | Disabled / false toggle      |
| ↑↓    | U+2191-93 | Navigation direction hints   |

### Data Masking (`ui/drawer.go`)

| Glyph | Codepoint | Usage                                     |
| ----- | --------- | ----------------------------------------- |
| •     | U+2022    | Password, card number, CVV, SSN masking   |

### Visual Chrome

| Glyph | Codepoint | File             | Usage                  |
| ----- | --------- | ---------------- | ---------------------- |
| ▄     | U+2584    | `ui/gradient.go` | Gradient separator     |
| …     | U+2026    | various          | Text truncation        |
| ·     | U+00B7    | `ui/hints.go`    | Hint key separator     |

## Semantic Categories

When adding new glyphs, use these categories to pick the right style:

### Item Types — use nf-md- icons, colored per theme

For new Bitwarden item types, pick from nf-md- and assign a theme color:
- Login: 󰌾 `nf-md-lock` (ColorHighlight)
- Card: 󰁯 `nf-md-credit_card` (ColorGreen)
- Note: 󱙒 `nf-md-note_text` (ColorYellow)
- Identity: 󰀄 `nf-md-account` (ColorHighlight)
- SSH Key: 󰣀 `nf-md-console` (ColorGreen)

### Status Indicators — use plain Unicode

| Semantic       | Glyph | Codepoint |
| -------------- | ----- | --------- |
| Success        | ✓     | U+2713    |
| Error          | ✗     | U+2717    |
| Warning        | ▲     | U+25B2    |
| Loading        | (use spinner) | —   |
| Enabled        | ✓     | U+2713    |
| Disabled       | ·     | U+00B7    |

### Progress — use Unicode geometric shapes

| Semantic         | Glyphs              | Codepoints         |
| ---------------- | -------------------- | ------------------- |
| Countdown (TOTP) | ● ◕ ◑ ◔ ○           | U+25CF-U+25CB      |
| Progress bar     | █ ▉ ▊ ▋ ▌ ▍ ▎ ▏     | U+2588-U+258F      |
| Shade fill       | ░ ▒ ▓               | U+2591-U+2593      |

### Navigation — use Unicode triangles/arrows

| Semantic         | Glyph | Codepoint |
| ---------------- | ----- | --------- |
| Cursor/selected  | ▶     | U+25B6    |
| Expanded         | ▼     | U+25BC    |
| Up/Down hints    | ↑ ↓   | U+2191-93 |
| Left/Right hints | ← →   | U+2190-92 |

## Candidate Glyphs for Future Work

### TOTP Countdown Enhancement (#38)

Current circle-based countdown is good. Potential refinements:
- nf-md- timer: 󰔛 `nf-md-timer` (U+F051B) — prefix before countdown circles
- nf-md- clock alert: 󰥔 `nf-md-clock_alert` (U+F0954) — for expiring codes

### Vault Editing (#6)

| Concept    | Glyph | Codepoint | NF Name              |
| ---------- | ----- | --------- | --------------------- |
| Edit/Pencil| 󰏫    | U+F03EB   | `nf-md-pencil`        |
| Save       | 󰆓    | U+F0193   | `nf-md-content_save`  |
| Delete     | 󰆴    | U+F01B4   | `nf-md-delete`        |
| Undo       | 󰕌    | U+F054C   | `nf-md-undo`          |

### Folders & Collections (#7)

| Concept         | Glyph | Codepoint | NF Name              |
| --------------- | ----- | --------- | --------------------- |
| Folder closed   | 󰉋    | U+F024B   | `nf-md-folder`        |
| Folder open     | 󰝰    | U+F0770   | `nf-md-folder_open`   |
| Collection      | 󱉟    | U+F125F   | `nf-md-folder_star`   |

### Password Strength

| Concept    | Glyph | Codepoint | NF Name                |
| ---------- | ----- | --------- | ----------------------- |
| Shield     | 󰒃    | U+F0483   | `nf-md-shield`          |
| Shield OK  | 󰕥    | U+F0565   | `nf-md-shield_check`    |
| Shield warn| 󱍝    | U+F135D   | `nf-md-shield_alert`    |

### Copy/Clipboard Actions

| Concept   | Glyph | Codepoint | NF Name                 |
| --------- | ----- | --------- | ------------------------ |
| Copy      | 󰆏    | U+F018F   | `nf-md-content_copy`     |
| Copied OK | 󰄬    | U+F012C   | `nf-md-check`            |

### Visibility Toggle

| Concept  | Glyph | Codepoint | NF Name          |
| -------- | ----- | --------- | ----------------- |
| Show     | 󰈈    | U+F0208   | `nf-md-eye`       |
| Hide     | 󰈉    | U+F0209   | `nf-md-eye_off`   |

## Glyphs to Avoid

- **Emoji** (U+1F000+) — inconsistent double-width rendering, breaks TUI alignment
- **CJK characters** — double-width by spec, breaks column layouts
- **Em-dash** (U+2014) — some terminals render at ambiguous width
- **nf-weather-** set — large glyphs, inconsistent cell width
- **Complex multi-path Nerd Font glyphs** — can overflow cell boundaries in non-Mono font variants
- **Mixing icon sets** — stick to nf-md- for consistency; don't mix nf-fa- and nf-md- for the same semantic category

## Adding New Glyphs Checklist

1. Check the [Nerd Fonts cheat sheet](https://www.nerdfonts.com/cheat-sheet) for nf-md- candidates
2. Pick from the semantic category tables above, or find an nf-md- icon that fits
3. Define as a themed variable in `ui/theme.go` (inside `initStyles()`) following the `Glyph*` naming pattern
4. Verify rendering in at least two terminals (one of: kitty, wezterm, ghostty)
5. Update this file with the new glyph entry
