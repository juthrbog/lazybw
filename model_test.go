package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
	// As in main without --debug. captureLog reads the log in a test.
	log.SetOutput(io.Discard)
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

// TestInitRequestsBackgroundColor checks that Init gives the batch the
// tea.RequestBackgroundColor command itself, with no wrapper around it.
func TestInitRequestsBackgroundColor(t *testing.T) {
	batch, ok := NewRootModel(time.Minute).Init()().(tea.BatchMsg)
	if !ok {
		t.Fatal("Init must return a batch")
	}
	want := reflect.ValueOf(tea.RequestBackgroundColor).Pointer()
	for _, cmd := range batch {
		if cmd != nil && reflect.ValueOf(cmd).Pointer() == want {
			return
		}
	}
	t.Error("the batch of Init has no tea.RequestBackgroundColor command")
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
	m, cmd := update(t, m, keyPress("m"))
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

// flatView renders m without ANSI sequences and with each run of white
// space as one space, so that a check finds text that the view wraps.
func flatView(m RootModel) string {
	return strings.Join(strings.Fields(ansi.Strip(m.View().Content)), " ")
}

// errLockTimedOut is the bw lock error of the lock error tests.
var errLockTimedOut = errors.New("bw lock timed out")

func TestLockErrorShowsOnUnlockScreen(t *testing.T) {
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, screens.LockMsg{})
	if m.state != stateQuitting || m.lockFor != intentLock {
		t.Fatalf("after the lock key, state = %d and lockFor = %d, want stateQuitting and intentLock", m.state, m.lockFor)
	}
	m, _ = update(t, m, lockResult(m, errLockTimedOut))
	if m.state != stateLocked {
		t.Fatalf("state = %d, want stateLocked", m.state)
	}
	if !m.sess.IsLocked() {
		t.Error("lazybw must drop the session token when bw lock fails")
	}
	view := flatView(m)
	for _, want := range []string{"bw lock timed out", `Run "bw lock" in a terminal.`} {
		if !strings.Contains(view, want) {
			t.Errorf("unlock screen = %q, want it to contain %q", view, want)
		}
	}
}

func TestLockWithoutErrorShowsNoLockError(t *testing.T) {
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, screens.LockMsg{})
	m, _ = update(t, m, lockResult(m, nil))
	if view := flatView(m); strings.Contains(view, "bw lock failed") {
		t.Errorf("unlock screen = %q, want no lock error", view)
	}
}

func TestLockErrorAfterIdleLockKeepsTypedText(t *testing.T) {
	m := unlockedVault(t, testItems())
	m.sess.LastActive = time.Now().Add(-time.Hour)
	m, _ = update(t, m, idleCheckMsg{})
	if m.state != stateLocked {
		t.Fatalf("state = %d, want stateLocked", m.state)
	}
	for _, r := range "abc" {
		m, _ = update(t, m, keyPress(string(r)))
	}
	// The password field shows one mask character for each typed key.
	const typed = "> *** "
	if view := flatView(m); !strings.Contains(view, typed) {
		t.Fatalf("unlock screen = %q, want the typed text as %q", view, typed)
	}
	m, cmd := update(t, m, lockResult(m, errLockTimedOut))
	if m.state != stateLocked || cmd != nil {
		t.Error("the lock error must not build a new unlock screen")
	}
	view := flatView(m)
	if !strings.Contains(view, "bw lock timed out") {
		t.Errorf("unlock screen = %q, want the bw lock error", view)
	}
	if !strings.Contains(view, typed) {
		t.Errorf("unlock screen = %q, want the typed text as %q", view, typed)
	}
	if !m.sess.IsLocked() {
		t.Error("the session token must stay dropped")
	}
}

func TestLockErrorOnQuitIsKeptForMain(t *testing.T) {
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, screens.QuitMsg{})
	m, cmd := update(t, m, lockResult(m, errLockTimedOut))
	if !hasMsgType[tea.QuitMsg](collectMsgs(cmd)) {
		t.Error("a failed bw lock on quit must still quit")
	}
	if !strings.Contains(m.quitError, "bw lock timed out") {
		t.Errorf("quitError = %q, want the bw lock error", m.quitError)
	}
	if !m.sess.IsLocked() {
		t.Error("lazybw must drop the session token when bw lock fails")
	}
	var out bytes.Buffer
	printLockErrors(&out, m, nil)
	if !strings.Contains(out.String(), "bw lock timed out") {
		t.Errorf("printed %q, want the bw lock error", out.String())
	}
}

func TestQuitWithoutLockErrorPrintsNothing(t *testing.T) {
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, screens.QuitMsg{})
	m, _ = update(t, m, lockResult(m, nil))
	var out bytes.Buffer
	printLockErrors(&out, m, nil)
	printLockErrors(&out, nil, nil)
	if out.Len() != 0 {
		t.Errorf("printed %q, want nothing", out.String())
	}
}

// idleLockThenUnlock idle-locks the session "tok" of an open vault, and
// then unlocks and opens the session "tokB" before the bw lock of "tok"
// returns. It returns the model and the Seq of that first lock.
func idleLockThenUnlock(t *testing.T) (RootModel, uint64) {
	t.Helper()
	m := unlockedVault(t, testItems())
	m.sess.LastActive = time.Now().Add(-time.Hour)
	m, _ = update(t, m, idleCheckMsg{})
	first := m.lockSeq
	m, _ = update(t, m, screens.UnlockedMsg{Token: "tokB"})
	m, _ = update(t, m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch})
	if m.state != stateVault {
		t.Fatalf("state = %d, want stateVault", m.state)
	}
	return m, first
}

func TestEarlierLockResultIsDropped(t *testing.T) {
	m, first := idleLockThenUnlock(t)
	m, _ = update(t, m, screens.LockMsg{})
	m, cmd := update(t, m, bwcmd.LockResult{Err: errLockTimedOut, Seq: first})
	if m.state != stateQuitting || cmd != nil {
		t.Errorf("state = %d, want stateQuitting and no command after the result of the idle lock", m.state)
	}
	if m.pendingLock != "tokB" || m.lastCleanup.lockToken != "tokB" {
		t.Errorf("pendingLock = %q, cleanup token = %q, want tokB", m.pendingLock, m.lastCleanup.lockToken)
	}
	m, _ = update(t, m, lockResult(m, nil))
	if m.state != stateLocked {
		t.Fatalf("state = %d, want stateLocked", m.state)
	}
	if view := flatView(m); strings.Contains(view, "bw lock failed") {
		t.Errorf("unlock screen = %q, want no error of the earlier lock", view)
	}
}

func TestEarlierLockResultDoesNotEndQuit(t *testing.T) {
	m, first := idleLockThenUnlock(t)
	m, _ = update(t, m, screens.QuitMsg{})
	m, cmd := update(t, m, bwcmd.LockResult{Err: errLockTimedOut, Seq: first})
	if hasMsgType[tea.QuitMsg](collectMsgs(cmd)) {
		t.Error("the result of the idle lock must not end the quit")
	}
	if m.quitError != "" {
		t.Errorf("quitError = %q, want no error of the earlier lock", m.quitError)
	}
	if m.pendingLock != "tokB" || m.lastCleanup.lockToken != "tokB" {
		t.Errorf("pendingLock = %q, cleanup token = %q, want tokB", m.pendingLock, m.lastCleanup.lockToken)
	}
}

func TestLockErrorAfterNewUnlockGoesToLog(t *testing.T) {
	m := unlockedVault(t, testItems())
	m.sess.LastActive = time.Now().Add(-time.Hour)
	m, _ = update(t, m, idleCheckMsg{})
	m, _ = update(t, m, screens.UnlockedMsg{Token: "tokB"})
	m, _ = update(t, m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch})
	buf := captureLog(t)
	m, _ = update(t, m, lockResult(m, errLockTimedOut))
	if m.state != stateVault || m.sess.Token != "tokB" {
		t.Errorf("state = %d, token = %q, want the open vault of tokB", m.state, m.sess.Token)
	}
	if !strings.Contains(buf.String(), "bw lock failed") {
		t.Errorf("log = %q, want the bw lock failure", buf.String())
	}
}

// secondFetchFailure returns a root model on the error screen after a
// retry of the item load failed too. That failure starts bw lock.
func secondFetchFailure(t *testing.T) RootModel {
	t.Helper()
	m := errorAfterUnlock(t)
	m, _ = update(t, m, screens.RetryMsg{})
	m, _ = update(t, m, bwcmd.ItemsResult{Err: errors.New("Vault is locked."), Epoch: m.sess.Epoch})
	if m.state != stateError || m.pendingLock != "tok" {
		t.Fatalf("state = %d, pendingLock = %q, want stateError and a running bw lock", m.state, m.pendingLock)
	}
	return m
}

func TestLockErrorFromErrorScreenShowsOnNextUnlockScreen(t *testing.T) {
	m := secondFetchFailure(t)
	m, _ = update(t, m, lockResult(m, errLockTimedOut))
	if m.state != stateError {
		t.Fatalf("state = %d, want stateError", m.state)
	}
	m, _ = update(t, m, screens.RetryMsg{})
	m, _ = update(t, m, bwcmd.StatusResult{Status: bwcmd.VaultStatus{Status: "locked", UserEmail: "u@t.com"}})
	if m.state != stateLocked {
		t.Fatalf("state = %d, want stateLocked", m.state)
	}
	view := flatView(m)
	for _, want := range []string{"bw lock timed out", `Run "bw lock" in a terminal.`} {
		if !strings.Contains(view, want) {
			t.Errorf("unlock screen = %q, want it to contain %q", view, want)
		}
	}
	if m.lockErr != "" {
		t.Error("the unlock screen must take the kept lock error")
	}
}

func TestLockErrorFromErrorScreenIsPrintedOnQuit(t *testing.T) {
	m := secondFetchFailure(t)
	m, _ = update(t, m, lockResult(m, errLockTimedOut))
	m, cmd := update(t, m, screens.QuitMsg{})
	if !hasMsgType[tea.QuitMsg](collectMsgs(cmd)) {
		t.Fatal("quit from the error screen must quit")
	}
	var out bytes.Buffer
	printLockErrors(&out, m, nil)
	if !strings.Contains(out.String(), "bw lock timed out") {
		t.Errorf("printed %q, want the bw lock error", out.String())
	}
}

func TestLockErrorWhenNotLoggedInShowsNothing(t *testing.T) {
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, screens.LockMsg{})
	notLoggedIn := &bwcmd.Error{Msg: "You are not logged in.", Err: errors.New("exit status 1")}
	m, _ = update(t, m, lockResult(m, notLoggedIn))
	if view := flatView(m); m.state != stateLocked || strings.Contains(view, "bw lock failed") {
		t.Errorf("state = %d, view = %q, want the unlock screen with no lock error", m.state, view)
	}
}

func TestLockErrorOnLoginScreenShowsNothing(t *testing.T) {
	m := secondFetchFailure(t)
	m, _ = update(t, m, screens.RetryMsg{})
	m, _ = update(t, m, bwcmd.StatusResult{Status: bwcmd.VaultStatus{Status: "unauthenticated"}})
	m, _ = update(t, m, lockResult(m, errLockTimedOut))
	if view := flatView(m); m.state != stateLogin || strings.Contains(view, "bw lock failed") {
		t.Errorf("state = %d, view = %q, want the login screen with no lock error", m.state, view)
	}
	if m.lockErr != "" {
		t.Errorf("lockErr = %q, want no kept error when bw has no logged-in account", m.lockErr)
	}
}

// TestLockErrorStepFitsSmallTerminal fails an unlock after a failed bw lock
// on a small terminal. The step of the lock error must stay visible.
func TestLockErrorStepFitsSmallTerminal(t *testing.T) {
	timeout := fmt.Errorf("bw lock timed out after 15s: %w", context.DeadlineExceeded)
	for _, size := range [][2]int{{80, 12}, {40, 12}} {
		for _, lockErr := range []error{errLockTimedOut, timeout} {
			m := unlockedVault(t, testItems())
			m, _ = update(t, m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			m, _ = update(t, m, screens.LockMsg{})
			m, _ = update(t, m, lockResult(m, lockErr))
			m, _ = update(t, m, bwcmd.UnlockResult{Err: bwcmd.ErrInvalidPassword})
			view := flatView(m)
			for _, want := range []string{"Master password", `Run "bw lock" in a terminal.`} {
				if !strings.Contains(view, want) {
					t.Errorf("%dx%d, %v: view = %q, want it to contain %q", size[0], size[1], lockErr, view, want)
				}
			}
		}
	}
}

func TestLogReasonLeavesOutBwText(t *testing.T) {
	bwErr := &bwcmd.Error{Msg: "a line with secret-4c1d", Err: errors.New("exit status 1")}
	for _, tc := range []struct {
		err  error
		want string
	}{
		{fmt.Errorf("bw lock timed out after 15s: %w", context.DeadlineExceeded), "timed out"},
		{&bwcmd.Error{Msg: "You are not logged in. secret-4c1d"}, "not logged in"},
		{bwErr, "bw error (exit status 1)"},
		{&bwcmd.Error{Msg: "secret-4c1d"}, "bw error"},
		{errors.New("parse bw list items output: invalid character 's' in secret-4c1d"), "unexpected output from bw"},
	} {
		if got := logReason(tc.err); got != tc.want {
			t.Errorf("logReason(%q) = %q, want %q", tc.err, got, tc.want)
		}
	}

	// The root logs the reason, not the bw message.
	buf := captureLog(t)
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, bwcmd.SyncResult{Err: bwErr, Epoch: m.sess.Epoch})
	m, _ = update(t, m, screens.LockMsg{})
	_, _ = update(t, m, lockResult(m, bwErr))
	if strings.Contains(buf.String(), "secret-4c1d") {
		t.Errorf("log = %q, want no text of the bw message", buf.String())
	}
	for _, want := range []string{"bw sync failed: bw error (exit status 1)", "bw lock failed: bw error (exit status 1)"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log = %q, want it to contain %q", buf.String(), want)
		}
	}
}

// syncBuffer is a buffer that is safe for use from more than one
// goroutine. A command of an earlier test can still run and log.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// captureLog sends the log output to a buffer for the test, with no time
// stamps.
func captureLog(t *testing.T) *syncBuffer {
	t.Helper()
	buf := &syncBuffer{}
	out, flags := log.Writer(), log.Flags()
	log.SetOutput(buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(out)
		log.SetFlags(flags)
	})
	return buf
}

// TestDebugLogHasNoSecrets drives an unlock, the item load and a lock
// through RootModel.Update. The log must show each step and no secret.
func TestDebugLogHasNoSecrets(t *testing.T) {
	const (
		token    = "session-token-5f1c"
		password = "master-password-9e2a"
		email    = "person-7b3d@example.org"
	)
	items := []bwcmd.Item{
		{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "item-name-login", Login: &bwcmd.Login{
			Username: "field-username", Password: "field-password", Totp: "JBSWY3DPEHPK3PXP",
			URIs: []bwcmd.URI{{URI: "https://field-uri.example"}},
		}},
		{ID: "2", Type: bwcmd.ItemTypeCard, Name: "item-name-card", Card: &bwcmd.Card{
			CardholderName: "field-holder", Number: "4111111111111111", Code: "9431",
		}},
		{ID: "3", Type: bwcmd.ItemTypeSecureNote, Name: "item-name-note", Notes: "field-note-body"},
	}
	secrets := []string{
		token, password, email,
		"item-name-login", "field-username", "field-password", "JBSWY3DPEHPK3PXP", "field-uri",
		"item-name-card", "field-holder", "4111111111111111", "9431",
		"item-name-note", "field-note-body",
	}

	buf := captureLog(t)
	m := NewRootModel(15 * time.Minute)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = update(t, m, bwcmd.StatusResult{Status: bwcmd.VaultStatus{Status: "locked", UserEmail: email}})

	// The unlock: type the master password, and let bw unlock return the
	// token. The unlock screen then sends UnlockedMsg.
	start := len(buf.String())
	for _, r := range password {
		m, _ = update(t, m, keyPress(string(r)))
	}
	m, cmd := update(t, m, bwcmd.UnlockResult{Token: token})
	unlocked, ok := waitForMsg[screens.UnlockedMsg](cmd, time.Second)
	if !ok {
		t.Fatal("the unlock screen must send UnlockedMsg after bw unlock returns a token")
	}
	m, _ = update(t, m, unlocked)
	m, _ = update(t, m, bwcmd.ItemsResult{Items: items, Epoch: m.sess.Epoch})
	if m.state != stateVault {
		t.Fatalf("state = %d, want stateVault", m.state)
	}
	unlockLog := buf.String()[start:]
	if strings.TrimSpace(unlockLog) == "" {
		t.Error("the unlock and the item load must write to the log")
	}
	if !strings.Contains(unlockLog, "loaded 3 items") {
		t.Errorf("log of the unlock = %q, want the item count", unlockLog)
	}

	// The lock: the lock key, and the result of the fake bw lock.
	start = len(buf.String())
	m, cmd = update(t, m, screens.LockMsg{})
	res, ok := waitForMsg[bwcmd.LockResult](cmd, 5*time.Second)
	if !ok {
		t.Fatal("the lock key must run bw lock")
	}
	m, _ = update(t, m, res)
	if m.state != stateLocked {
		t.Fatalf("state = %d, want stateLocked", m.state)
	}
	if lockLog := buf.String()[start:]; strings.TrimSpace(lockLog) == "" {
		t.Error("the lock must write to the log")
	}

	for line := range strings.SplitSeq(buf.String(), "\n") {
		for _, s := range secrets {
			if strings.Contains(line, s) {
				t.Errorf("log line %q contains the secret %q", line, s)
			}
		}
	}
}

// TestFailedBwLockShowsOnUnlockScreen runs the fake bw lock, which fails
// for failLockToken, and checks that the unlock screen shows its message.
func TestFailedBwLockShowsOnUnlockScreen(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width, m.height = 80, 24
	m, _ = update(t, m, screens.UnlockedMsg{Token: failLockToken, Email: "u@t.com"})
	m, _ = update(t, m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch})
	m, cmd := update(t, m, screens.LockMsg{})
	res, ok := waitForMsg[bwcmd.LockResult](cmd, 5*time.Second)
	if !ok {
		t.Fatal("the lock key must run bw lock")
	}
	if res.Err == nil {
		t.Fatal("the fake bw lock must fail for failLockToken")
	}
	m, _ = update(t, m, res)
	want := "bw lock failed: " + strings.TrimSuffix(failLockMessage, ".")
	if view := flatView(m); m.state != stateLocked || !strings.Contains(view, want) {
		t.Errorf("state = %d, view = %q, want the unlock screen with %q", m.state, view, want)
	}
}
