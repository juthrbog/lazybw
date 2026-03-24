# lazybw Layout Specification

This document defines the unified frame system that all lazybw screens render
into. The goal is visual consistency across every app state — loading, locked,
vault, error, and quitting should all feel like the same application.

---

## Principles

1. **One frame, many contents.** Every state renders into the same three-zone
   structure: header, content area, footer. The zones have fixed positions; only
   the content area changes between states.

2. **No layout jumping.** The header and footer are always present and always
   occupy the same rows. Transitions between states never cause the terminal to
   flash, reflow, or shift content vertically.

3. **Chrome is constant, content is variable.** The user always sees the app
   name, always knows what state they're in, and always has contextual hints
   available. This eliminates disorientation during transitions.

4. **Single implementation.** There is one layout function (`ui.RenderFrame`)
   used by every state. No duplicated centering logic, no per-screen layout
   code.

---

## Frame Structure

```
┌─────────────────────────────────────────────────────┐
│  lazybw                             user@email.com  │  Header (1 row)
├─────────────────────────────────────────────────────┤
│                                                     │
│                                                     │
│               (state-specific content)              │  Content area
│                                                     │     (height - 2)
│                                                     │
│                                                     │
├─────────────────────────────────────────────────────┤
│  [contextual hints]                    [status info] │  Footer (1 row)
└─────────────────────────────────────────────────────┘
```

- **Header**: 1 row. Always shows "lazybw" on the left. Right side shows
  the user's email when authenticated, or nothing when unauthenticated.
- **Content area**: `height - 2` rows (total height minus header and footer).
  Each state fills this area differently (see below).
- **Footer**: 1 row. Left side shows contextual key hints for the current
  state. Right side shows sync status, toast messages, or error context.

The header and footer use the status bar style (`StyleStatusBar`) for
visual consistency — both are subtle bars that frame the content.

---

## Header

```go
ui.RenderHeader(props HeaderProps) string
```

Always rendered. Content varies by authentication state:

| State | Left | Right |
|---|---|---|
| Loading (pre-auth) | `lazybw` | |
| Login | `lazybw` | |
| Locked | `lazybw` | `user@email.com` |
| Vault | `lazybw` | `user@email.com` |
| Error | `lazybw` | `user@email.com` (if known) |
| Quitting | `lazybw` | `user@email.com` (if known) |

"lazybw" is rendered in bold with `ColorHighlight`. The email is rendered
in the default text color. The header background uses `ColorSubtle`.

---

## Footer

```go
ui.RenderFooter(props FooterProps) string
```

Always rendered. Content varies by state:

| State | Left (hints) | Right (status) |
|---|---|---|
| Loading | | |
| Login | `enter submit · q quit` | |
| Locked | `enter unlock · q quit` | |
| Vault (normal) | `j/k navigate · / search · c pwd · t totp · ? help · q quit` | `synced 2m ago` |
| Vault (filter) | `esc clear · enter confirm · ↑/↓ navigate` | `synced 2m ago` |
| Vault (toast) | `j/k navigate · / search · c pwd · t totp · ? help · q quit` | `Password copied — clears in 60s` |
| Vault (syncing) | `j/k navigate · / search · c pwd · t totp · ? help · q quit` | `⠋ Syncing…` |
| Error | `r retry · q quit` | |
| Quitting | | |

Hints are rendered in `ColorFaint`. Toast messages use `StyleToast` (green,
italic). The footer background uses `ColorSubtle` to match the header.

---

## Content Area by State

### Loading

Vertically and horizontally centered within the content area:

```


              ⠋ Loading vault…


```

Uses the `SpinnerLoad` (braille dot) animation. The text is plain, no
extra chrome — the header already provides the app identity.

### Login / Locked

Vertically and horizontally centered:

```


              Vault is locked.

            ┌──────────────────────┐
            │ Master password      │
            └──────────────────────┘

```

The login variant adds an email field above the password field.
On unlock submit, the form is replaced with the `SpinnerUnlock` animation
and "Unlocking…" text, still centered. Error messages appear below the
form on failure.

No "lazybw" title is rendered inside the content area — the header
handles that.

### Vault

Full-width content filling the entire content area:

```
  󰌾  Gmail                           user@gmail.com
  󰌾  GitHub                          dev@example.com
▶ 󰌾  AWS Console                     aws-admin
  󰁯  Visa Debit                      •••• 4242
  󱙒  SSH Keys
  󱙒  Anthropic API                   (note)
  󰌾  Slack                           work@company.com
                                                8 / 156

── AWS Console ───────────────────────────────── Login ─
  Username   aws-admin                          [u] copy
  Password   ••••••••••                         [c] copy
  TOTP       843 291  ● 18s                     [t] copy
  URL        console.aws.amazon.com             [o] open
```

The list occupies `contentHeight - DrawerHeight` rows. The drawer
occupies a fixed 8 rows at the bottom of the content area. When the
full help is toggled (`?`), the help text reduces the list height.

When filter mode is active, a filter input row appears at the top of
the content area, further reducing list height by 1.

### Error

Vertically and horizontally centered within the content area:

```


              Error: bw CLI timed out

              The vault could not be reached.
              Check your network connection.


```

Error text uses `StyleError` (red, bold). Description uses default text.

### Quitting

Vertically and horizontally centered:

```


              󰌿 Locking vault…


```

Uses the `SpinnerLock` animation. Brief and to the point.

---

## Centering

For states that center their content (loading, locked, error, quitting),
a single shared helper is used:

```go
ui.CenterInArea(content string, width, height int) string
```

This function:
1. Splits content into lines
2. Measures the widest line using `lipgloss.Width()` (ANSI-safe)
3. Pads vertically: `(height - lineCount) / 2` blank lines above
4. Pads horizontally: `(width - maxLineWidth) / 2` spaces before each line

No screen implements its own centering.

---

## Implementation: `ui.RenderFrame`

```go
type FrameProps struct {
    State    string // "loading", "locked", "vault", etc.
    Email    string // empty when unauthenticated
    Content  string // pre-rendered content for the content area
    Hints    string // left side of footer
    Status   string // right side of footer (sync, toast, etc.)
    Width    int
    Height   int
}

func RenderFrame(props FrameProps) string
```

The root model calls `RenderFrame` in its `View()` method for every state.
Each screen model is responsible only for rendering its content string —
it does not manage headers, footers, or outer layout.

### How each screen provides content

- **Loading**: `m.spinner.View() + " Loading vault…"` → centered by root
- **Locked**: `m.locked.ViewContent()` → form/spinner, centered by root
- **Vault**: `m.vault.ViewContent()` → list + drawer, full-width
- **Error**: `m.err.ViewContent()` → error text, centered by root
- **Quitting**: `m.spinner.View() + " Locking vault…"` → centered by root

Screen models expose a `ViewContent()` method that returns the raw content
string without any frame chrome. The root model wraps it in `RenderFrame`.

---

## Size Propagation

When `tea.WindowSizeMsg` arrives at the root model:

1. Root stores `m.width` and `m.height`
2. Root calculates `contentWidth = m.width` and `contentHeight = m.height - 2`
3. Root propagates a custom `ContentSizeMsg{Width, Height}` to the active
   child screen
4. Each screen uses this to calculate its internal layout (list height,
   drawer position, centering, etc.)

This ensures screens never need to know about the header/footer — they
only know the dimensions of the space they're allowed to fill.

---

## File Changes

| File | Role |
|---|---|
| `ui/frame.go` (new) | `RenderFrame`, `RenderHeader`, `RenderFooter`, `CenterInArea` |
| `model.go` | Call `RenderFrame` in `View()`, propagate `ContentSizeMsg` |
| `screens/locked.go` | Expose `ViewContent()`, remove `centerBlock`, remove header rendering |
| `screens/vault.go` | Expose `ViewContent()`, remove inline status bar rendering |
| `screens/error.go` | Expose `ViewContent()`, remove raw text rendering |
| `ui/statusbar.go` | Refactor into footer-compatible form or replace with `RenderFooter` |

---

## Why This Matters

The current implementation has four separate layout approaches:
- `model.go:renderTransition()` — manual centering with "lazybw" header
- `screens/locked.go:centerBlock()` — duplicate centering with "lazybw" header
- `screens/error.go:View()` — raw text, no centering, no chrome
- `screens/vault.go:View()` — full layout with its own status bar

This causes:
- Jarring visual transitions (the frame itself changes shape between states)
- Duplicated code (two centering implementations)
- Inconsistent information hierarchy (some states show app name, others don't)
- The error screen feels disconnected from the rest of the app

The unified frame eliminates all of these. Every transition is smooth because
the outer structure never changes — only the content area updates.
