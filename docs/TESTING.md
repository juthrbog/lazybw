# lazybw Testing Strategy

## Framework

**teatest** (`github.com/charmbracelet/x/exp/teatest`) — the official Charm
ecosystem testing tool. Built by the same team behind bubbletea, lipgloss, and
bubbles. Provides model testing and golden file support out of the box.

Combined with Go's standard `testing` package and table-driven tests for pure
functions.

No external assertion libraries. Use `t.Errorf` / `t.Fatalf` directly.

---

## Test Layers

### Layer 1: Pure Function Tests

Table-driven tests for functions with no side effects. These are the highest
ROI — fast, deterministic, and catch the most common regressions.

**Packages covered:**
- `bwcmd/` — `ParseStatus`, `ParseItems`, `Item.Description()`, `FilterValue()`, `Title()`
- `ui/` — `RenderHeader`, `RenderFooter`, `CenterInArea`, `RenderItemRow`, `RenderDrawer`
- `session/` — `State.IsLocked()`, `Lock()`, `SetToken()`, `Touch()`, `IsIdle()`
- `screens/` — `FooterContent()`, `FooterHints()`, `ViewContent()` (with known state)

### Layer 2: Model Tests via teatest

Send messages to `Update()`, assert state transitions and returned commands.
Covers the bubbletea model lifecycle without needing a real terminal.

**Pattern:**
```go
// Construct model with known state
m := screens.NewVaultModel(items, sess, 80, 24)

// Send a message
updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
m = updated.(screens.VaultModel)

// Assert state changed
if m.mode != modeFilter {
    t.Errorf("expected filter mode")
}
```

**Packages covered:**
- `screens/error.go` — key handling (r/q), retry vs quit
- `screens/vault.go` — cursor movement, filter toggle, sync flow
- `model.go` — root state machine transitions (loading → locked → vault → quitting)

### Layer 3: Golden File Tests

Snapshot rendered output to detect unintended UI changes. teatest provides
`RequireGolden()` for this.

Golden files are stored in `testdata/` directories alongside the test files.
On first run, the expected output is captured. Subsequent runs compare against
the snapshot. Use `go test -update` to regenerate after intentional changes.

**Applied to:**
- `ui/RenderDrawer` — one golden file per item type
- `ui/RenderItemRow` — one per type + selected/unselected
- `screens/ViewContent()` — rendered output per state

---

## Test Environment

### Color Profile

All `TestMain` functions set the lipgloss color profile to ASCII to strip
ANSI escape codes from rendered output:

```go
func TestMain(m *testing.M) {
    lipgloss.SetColorProfile(termenv.Ascii)
    os.Exit(m.Run())
}
```

This ensures consistent string comparisons across terminals, CI runners, and
local dev machines.

### Golden File Line Endings

`.gitattributes` marks golden files as binary to prevent Git from modifying
line endings:

```
**/testdata/** binary
```

---

## What's NOT Tested

### Subprocess Execution (`bwcmd/exec.go`)

The functions in `exec.go` (`CheckStatus`, `FetchItems`, `Unlock`, etc.) are
thin wrappers around `os/exec.Command`. Mocking `os/exec` requires either
dependency injection or test doubles, adding complexity without meaningful
coverage. The real risk is in parsing the output, which IS tested via
`parser_test.go`.

### Clipboard (`session/clipboard.go`)

`CopyToClipboard` and `ClearClipboard` call `wl-copy` or `atotto/clipboard`.
Same rationale — thin I/O wrappers. The message types (`CopiedMsg`,
`ClipboardClearedMsg`) and their handling in `Update()` ARE tested.

### Third-Party Component Internals

`huh.Form` in the locked screen is a Charm library component. We test our
integration with it (footer hints, view content per state) but not the form's
internal behavior.

---

## How to Run

```sh
# Run all tests
go test ./...

# Verbose output
go test -v ./...

# Race condition detection
go test -race ./...

# Coverage report
go test -cover ./...

# Update golden files after intentional UI changes
go test -update ./...
```

---

## Adding Tests for New Features

When adding a new feature:

1. **New pure functions** → add table-driven tests in the corresponding `_test.go`
2. **New message types** → add teatest cases asserting `Update()` handles them
3. **New rendering** → add golden file tests via `testdata/`
4. **New item types** (identity, SSH key) → add cases to existing `TestItemDescription`, `TestRenderDrawer`, `TestRenderItemRow` tables
