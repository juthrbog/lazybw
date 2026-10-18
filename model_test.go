package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/screens"
	"github.com/juthrbog/lazybw/session"
	"github.com/juthrbog/lazybw/ui"
)

func TestMain(m *testing.M) {
	// The theme at the start, to find a test that does not restore it.
	theme, dark, highlight := ui.CurrentTheme, ui.IsDark, ui.ColorHighlight
	dir, err := installFakeBw()
	if err != nil {
		fmt.Fprintln(os.Stderr, "install fake bw:", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	if code == 0 && (ui.CurrentTheme != theme || ui.IsDark != dark || !sameColor(ui.ColorHighlight, highlight)) {
		fmt.Fprintf(os.Stderr, "a test left the theme at %s (dark %v): use withTheme or keepTheme\n", ui.CurrentTheme, ui.IsDark)
		code = 1
	}
	os.Exit(code)
}

func TestNewRootModel(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	if m.state != stateLoading {
		t.Errorf("initial state = %d, want stateLoading (%d)", m.state, stateLoading)
	}
	if m.sess == nil {
		t.Fatal("session should not be nil")
	}
	if m.sess.IdleTimeout != 15*time.Minute {
		t.Errorf("IdleTimeout = %v, want 15m", m.sess.IdleTimeout)
	}
}

func TestRootStatusResultLocked(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	msg := bwcmd.StatusResult{
		Status: bwcmd.VaultStatus{Status: "locked", UserEmail: "u@t.com"},
	}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateLocked {
		t.Errorf("state = %d, want stateLocked (%d)", m.state, stateLocked)
	}
	if m.sess.Email != "u@t.com" {
		t.Errorf("email = %q, want %q", m.sess.Email, "u@t.com")
	}
}

func TestRootStatusResultUnauthenticated(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	msg := bwcmd.StatusResult{
		Status: bwcmd.VaultStatus{Status: "unauthenticated"},
	}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateLogin {
		t.Errorf("state = %d, want stateLogin (%d)", m.state, stateLogin)
	}
}

func TestRootStatusResultUnlockedAsksForPassword(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	msg := bwcmd.StatusResult{
		Status: bwcmd.VaultStatus{Status: "unlocked", UserEmail: "u@t.com"},
	}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	// lazybw has no session token here, so it must not fetch the items.
	if m.state != stateLocked {
		t.Errorf("state = %d, want stateLocked (%d)", m.state, stateLocked)
	}
}

func TestRootStatusResultError(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	msg := bwcmd.StatusResult{Err: errors.New("bw not found")}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateError {
		t.Errorf("state = %d, want stateError (%d)", m.state, stateError)
	}
}

func TestRootUnlockedMsg(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	msg := screens.UnlockedMsg{Token: "test-token", Email: "u@t.com"}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateLoading {
		t.Errorf("state = %d, want stateLoading (%d)", m.state, stateLoading)
	}
	if m.sess.Token != "test-token" {
		t.Errorf("token = %q, want %q", m.sess.Token, "test-token")
	}
}

func TestRootItemsResult(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	items := []bwcmd.Item{
		{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Gmail"},
	}
	msg := bwcmd.ItemsResult{Items: items}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateVault {
		t.Errorf("state = %d, want stateVault (%d)", m.state, stateVault)
	}
}

func TestRootItemsResultError(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	msg := bwcmd.ItemsResult{Err: errors.New("parse error")}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateError {
		t.Errorf("state = %d, want stateError (%d)", m.state, stateError)
	}
}

func TestRootLockMsg(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.sess.SetToken("test-token")

	msg := screens.LockMsg{}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateQuitting {
		t.Errorf("state = %d, want stateQuitting (%d)", m.state, stateQuitting)
	}
	if m.lockFor != intentLock {
		t.Errorf("lockFor = %d, want intentLock (%d)", m.lockFor, intentLock)
	}
}

func TestRootQuitMsg(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.sess.SetToken("test-token")

	msg := screens.QuitMsg{}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateQuitting {
		t.Errorf("state = %d, want stateQuitting (%d)", m.state, stateQuitting)
	}
	if m.lockFor != intentQuit {
		t.Errorf("lockFor = %d, want intentQuit (%d)", m.lockFor, intentQuit)
	}
}

func TestRootIdleLock(t *testing.T) {
	m := NewRootModel(1 * time.Millisecond)
	m.state = stateVault
	m.sess.SetToken("test-token")
	m.sess.LastActive = time.Now().Add(-time.Second)
	m.width = 80
	m.height = 24

	updated, _ := m.Update(idleCheckMsg{})
	m = updated.(RootModel)

	if m.state != stateLocked {
		t.Errorf("state = %d, want stateLocked (%d)", m.state, stateLocked)
	}
	if m.sess.Token != "" {
		t.Error("token should be cleared after idle lock")
	}
	if m.lockFor != intentLock {
		t.Errorf("lockFor = %d, want intentLock (%d)", m.lockFor, intentLock)
	}
}

// rootIn returns a root model in the given state at the given terminal size.
func rootIn(t *testing.T, state appState, width, height int) RootModel {
	t.Helper()
	m := NewRootModel(15 * time.Minute)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: width, Height: height})
	switch state {
	case stateLoading:
	case stateLogin:
		m, _ = update(t, m, bwcmd.StatusResult{Status: bwcmd.VaultStatus{Status: "unauthenticated"}})
	case stateLocked:
		m, _ = update(t, m, bwcmd.StatusResult{Status: bwcmd.VaultStatus{Status: "locked", UserEmail: "u@t.com"}})
	case stateVault:
		m, _ = update(t, m, screens.UnlockedMsg{Token: "tok", Email: "u@t.com"})
		m, _ = update(t, m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch})
	case stateError:
		err := errors.New("bw failed: " + strings.Repeat("the server did not answer in time ", 6))
		m, _ = update(t, m, bwcmd.StatusResult{Err: err})
	case stateQuitting:
		m.sess.SetToken("tok")
		m, _ = update(t, m, screens.LockMsg{})
	}
	if m.state != state {
		t.Fatalf("state = %d, want %d", m.state, state)
	}
	return m
}

// viewLines renders m and returns the lines of the frame.
func viewLines(m RootModel) []string {
	return strings.Split(m.View().Content, "\n")
}

func TestRootViewFillsTerminal(t *testing.T) {
	states := map[string]appState{
		"loading": stateLoading, "login": stateLogin, "locked": stateLocked,
		"vault": stateVault, "error": stateError, "quitting": stateQuitting,
	}
	for name, state := range states {
		for _, size := range [][2]int{{80, 24}, {60, 20}, {40, 10}} {
			width, height := size[0], size[1]
			lines := viewLines(rootIn(t, state, width, height))
			if len(lines) != height {
				t.Errorf("%s at %dx%d: view has %d lines, want %d", name, width, height, len(lines), height)
			}
			for i, l := range lines {
				if w := lipgloss.Width(l); w > width {
					t.Errorf("%s at %dx%d: line %d is %d cells wide", name, width, height, i, w)
				}
			}
		}
	}
}

func TestRootFooterHintsAt80x24(t *testing.T) {
	tests := []struct {
		state appState
		want  []string
	}{
		{stateLogin, []string{"enter submit", "ctrl+q quit"}},
		{stateLocked, []string{"enter unlock", "ctrl+q quit"}},
		{stateError, []string{"r retry", "q quit"}},
		{stateVault, []string{"j/k navigate", "? help", "q quit"}},
	}
	for _, tt := range tests {
		lines := viewLines(rootIn(t, tt.state, 80, 24))
		footer := ansi.Strip(lines[len(lines)-1])
		for _, want := range tt.want {
			if !strings.Contains(footer, want) {
				t.Errorf("state %d: footer = %q, want it to contain %q", tt.state, footer, want)
			}
		}
	}
}

func TestRootVaultFooterOnFirstFrameAt40x12(t *testing.T) {
	// The size arrives before the vault exists, and no resize follows.
	m := NewRootModel(15 * time.Minute)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 40, Height: 12})
	m, _ = update(t, m, screens.UnlockedMsg{Token: "tok", Email: "u@t.com"})
	m, _ = update(t, m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch})
	lines := viewLines(m)
	if len(lines) != 12 {
		t.Fatalf("view has %d lines, want 12", len(lines))
	}
	if footer := ansi.Strip(lines[11]); !strings.Contains(footer, "j/k navigate") {
		t.Errorf("last line = %q, want the vault footer", footer)
	}
	// The drawer is clamped to 6 rows, so the list keeps 2 item rows.
	frame := ansi.Strip(strings.Join(lines, "\n"))
	if !strings.Contains(frame, "Gmail") || !strings.Contains(frame, "Visa") {
		t.Errorf("the list must show 2 items above the drawer:\n%s", frame)
	}
}

func TestRootVaultFooterShowsAppliedFilter(t *testing.T) {
	m := rootIn(t, stateVault, 80, 24)
	for _, k := range []string{"/", "g", "m"} {
		m, _ = update(t, m, keyPress(k))
	}
	m, _ = update(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	lines := viewLines(m)
	footer := ansi.Strip(lines[len(lines)-1])
	if !strings.Contains(footer, "filter: gm") || !strings.Contains(footer, "esc clear") {
		t.Errorf("footer = %q, want the filter text and esc clear", footer)
	}
}

func TestCopyNoteIsClearedLikeOtherSecrets(t *testing.T) {
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, keyPress("j"))
	m, _ = update(t, m, keyPress("j")) // the secure note "Keys"
	m, cmd := update(t, m, keyPress("n"))
	msgs := collectMsgs(cmd)
	if !hasMsg(msgs, tea.SetClipboard("secret")()) {
		t.Fatal("n must copy the note")
	}
	if !hasMsg(msgs, session.CopiedMsg{Field: session.CopyFieldNote}) {
		t.Fatal("n must report a note copy")
	}
	m, _ = update(t, m, session.CopiedMsg{Field: session.CopyFieldNote})
	if !m.clipDirty {
		t.Fatal("a note copy must start the clipboard clear")
	}
	m, cmd = update(t, m, session.ClipboardClearedMsg{Gen: m.clipGen})
	if !hasMsg(collectMsgs(cmd), clipboardClear) {
		t.Error("the clear timer of a note copy must clear the clipboard")
	}
}

func TestRootRetryMsg(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.state = stateError

	msg := screens.RetryMsg{}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateLoading {
		t.Errorf("state = %d, want stateLoading (%d)", m.state, stateLoading)
	}
}

func TestRootViewHasNoMouseMode(t *testing.T) {
	states := map[string]appState{
		"loading": stateLoading, "login": stateLogin, "locked": stateLocked,
		"vault": stateVault, "error": stateError, "quitting": stateQuitting,
	}
	for name, state := range states {
		if mode := rootIn(t, state, 80, 24).View().MouseMode; mode != tea.MouseModeNone {
			t.Errorf("%s: MouseMode = %v, want MouseModeNone", name, mode)
		}
	}
}

// numberedItems returns n logins named "Item 00", "Item 01" and so on.
func numberedItems(n int) []bwcmd.Item {
	items := make([]bwcmd.Item, n)
	for i := range items {
		items[i] = bwcmd.Item{ID: strconv.Itoa(i), Type: bwcmd.ItemTypeLogin, Name: fmt.Sprintf("Item %02d", i)}
	}
	return items
}

var itemNumber = regexp.MustCompile(`Item (\d\d)`)

// listRows returns the numbers of the items that the list of the frame
// shows, from the top. The drawer below the list starts with a line that
// names no item.
func listRows(t *testing.T, m RootModel) []int {
	t.Helper()
	var rows []int
	for _, line := range viewLines(m)[2:] {
		match := itemNumber.FindStringSubmatch(ansi.Strip(line))
		if match == nil {
			if len(rows) > 0 {
				break
			}
			continue
		}
		n, _ := strconv.Atoi(match[1])
		rows = append(rows, n)
	}
	return rows
}

// cursorRow returns the list row of the frame that shows the cursor.
func cursorRow(m RootModel) string {
	for _, line := range viewLines(m) {
		if plain := ansi.Strip(line); strings.HasPrefix(plain, "▶ ") {
			return plain
		}
	}
	return ""
}

func TestRootPgDnMovesOnePage(t *testing.T) {
	items := numberedItems(60)
	for _, resizeFirst := range []bool{true, false} {
		m := NewRootModel(15 * time.Minute)
		if resizeFirst {
			m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
		}
		m, _ = update(t, m, screens.UnlockedMsg{Token: "tok", Email: "u@t.com"})
		m, _ = update(t, m, bwcmd.ItemsResult{Items: items, Epoch: m.sess.Epoch})
		if !resizeFirst {
			m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
		}
		rows := listRows(t, m)
		page := len(rows)
		if page < 2 || rows[0] != 0 {
			t.Fatalf("resize first %v: rows = %v, want a page from item 0", resizeFirst, rows)
		}
		for first := page; first < len(items); first += page {
			m, _ = update(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
			rows = listRows(t, m)
			if len(rows) == 0 || rows[0] != first {
				t.Fatalf("resize first %v: after PgDn, rows = %v, want a page from item %d", resizeFirst, rows, first)
			}
			// The cursor stays on the first row of the page.
			if got := cursorRow(m); !strings.Contains(got, fmt.Sprintf("Item %02d", first)) {
				t.Errorf("resize first %v: after PgDn, cursor row = %q, want item %d", resizeFirst, got, first)
			}
		}
		if last := rows[len(rows)-1]; last != len(items)-1 {
			t.Errorf("resize first %v: PgDn stops at a page that ends with item %d, want item %d", resizeFirst, last, len(items)-1)
		}
	}
}
