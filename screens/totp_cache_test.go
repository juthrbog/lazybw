package screens

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/totp"
)

const (
	alphaKey = "JBSWY3DPEHPK3PXP"
	zetaKey  = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
)

func totpItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "a", Type: bwcmd.ItemTypeLogin, Name: "Alpha", Login: &bwcmd.Login{Totp: alphaKey}},
		{ID: "n", Type: bwcmd.ItemTypeLogin, Name: "NoCode", Login: &bwcmd.Login{Username: "x"}},
		{ID: "z", Type: bwcmd.ItemTypeLogin, Name: "Zeta", Login: &bwcmd.Login{Totp: zetaKey}},
	}
}

// runMsgs runs cmd and returns its messages. It expands batches and skips
// commands that do not return within a short time, such as timers.
func runMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		if batch, ok := msg.(tea.BatchMsg); ok {
			var out []tea.Msg
			for _, c := range batch {
				out = append(out, runMsgs(c)...)
			}
			return out
		}
		if msg == nil {
			return nil
		}
		return []tea.Msg{msg}
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

// press sends a key to m and feeds back the filter results that the list
// computes in a command.
func press(t *testing.T, m VaultModel, msg tea.KeyPressMsg) VaultModel {
	t.Helper()
	updated, cmd := m.Update(msg)
	m = updated.(VaultModel)
	for _, sub := range runMsgs(cmd) {
		if fm, ok := sub.(list.FilterMatchesMsg); ok {
			updated, _ = m.Update(fm)
			m = updated.(VaultModel)
		}
	}
	return m
}

func typeKeys(t *testing.T, m VaultModel, s string) VaultModel {
	t.Helper()
	for _, r := range s {
		m = press(t, m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

// codesFor returns the code of key for the current and the previous time
// step. A comparison with both is not flaky at a step change.
func codesFor(t *testing.T, key string) []string {
	t.Helper()
	p, err := totp.Parse(key)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	return []string{totp.CodeAt(p, now), totp.CodeAt(p, now.Add(-time.Duration(p.Period)*time.Second))}
}

// assertTOTPFor fails unless the cached TOTP state belongs to key. It
// compares the secret, which does not change with time.
func assertTOTPFor(t *testing.T, m VaultModel, key string) {
	t.Helper()
	p, err := totp.Parse(key)
	if err != nil {
		t.Fatal(err)
	}
	if m.totpParams == nil || !bytes.Equal(m.totpParams.Secret, p.Secret) {
		t.Errorf("the TOTP state does not belong to key %s", key)
	}
	if m.totpCode == "" {
		t.Error("the drawer shows no code")
	}
}

// copiedTOTP presses t and returns the key whose code the command writes
// to the clipboard, or "" when it writes nothing.
func copiedTOTP(t *testing.T, m VaultModel) string {
	t.Helper()
	_, cmd := m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	for _, msg := range runMsgs(cmd) {
		for _, key := range []string{alphaKey, zetaKey} {
			for _, code := range codesFor(t, key) {
				if msg == tea.SetClipboard(code)() {
					return key
				}
			}
		}
	}
	return ""
}

func TestTOTPFollowsFilteredSelection(t *testing.T) {
	m := newTestVault(totpItems())
	assertTOTPFor(t, m, alphaKey)
	m = typeKeys(t, m, "/zeta")
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if item := m.selectedItem(); item == nil || item.ID != "z" {
		t.Fatalf("selected = %+v, want Zeta", item)
	}
	assertTOTPFor(t, m, zetaKey)
	if got := copiedTOTP(t, m); got != zetaKey {
		t.Errorf("t copied the code of %q, want Zeta", got)
	}
}

func TestTOTPShowsAfterFilterFromItemWithoutCode(t *testing.T) {
	m := newTestVault(totpItems())
	m.list.Select(1) // NoCode
	m.syncSelection()
	if m.totpCode != "" {
		t.Fatal("NoCode has no TOTP")
	}
	m = typeKeys(t, m, "/zeta")
	updated, _ := m.Update(TOTPTickMsg{})
	m = updated.(VaultModel)
	assertTOTPFor(t, m, zetaKey)
}

func TestTOTPAfterClearingFilter(t *testing.T) {
	m := newTestVault(totpItems())
	// "a" matches Alpha and Zeta. Zeta is row 1 in the filtered list, and
	// row 1 of the full list is NoCode, so Esc must change the code.
	m = typeKeys(t, m, "/a")
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = press(t, m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	if item := m.selectedItem(); item == nil || item.ID != "z" {
		t.Fatalf("selected = %+v, want Zeta", item)
	}
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	item := m.selectedItem()
	if item == nil {
		t.Fatal("no item selected after Esc")
	}
	if item.ID == "n" && (m.totpCode != "" || m.totpParams != nil) {
		t.Errorf("after Esc, NoCode is selected, but the drawer shows code %q", m.totpCode)
	}
	if item.Login != nil && item.Login.Totp != "" {
		assertTOTPFor(t, m, item.Login.Totp)
	}
}

func TestTOTPAfterRegroup(t *testing.T) {
	items := []bwcmd.Item{
		{ID: "g1", Type: bwcmd.ItemTypeLogin, Name: "GitHub (Work)", Login: &bwcmd.Login{Totp: alphaKey}},
		{ID: "g2", Type: bwcmd.ItemTypeLogin, Name: "GitHub (Home)", Login: &bwcmd.Login{Totp: alphaKey}},
		{ID: "z", Type: bwcmd.ItemTypeLogin, Name: "Zeta", Login: &bwcmd.Login{Totp: zetaKey}},
	}

	// Row 1 is GitHub (Home). After ctrl+g, row 1 is Zeta.
	m := newTestVault(items)
	m.list.Select(1)
	m.syncSelection()
	m = press(t, m, tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	if item := m.selectedItem(); item == nil || item.ID != "z" {
		t.Fatalf("selected = %+v, want Zeta", item)
	}
	assertTOTPFor(t, m, zetaKey)
	if got := copiedTOTP(t, m); got != zetaKey {
		t.Errorf("t copied the code of %q, want Zeta", got)
	}

	// Row 2 is Zeta. After ctrl+g, no row 2 is left, so no code shows.
	m = newTestVault(items)
	m.list.Select(2)
	m.syncSelection()
	m = press(t, m, tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	if m.selectedItem() == nil && m.totpCode != "" {
		t.Errorf("after ctrl+g with no selection, code = %q, want none", m.totpCode)
	}
}

func TestDrawerScrollKeptWhenFilterStarts(t *testing.T) {
	items := []bwcmd.Item{
		{ID: "n1", Type: bwcmd.ItemTypeSecureNote, Name: "ANote", Notes: "1\n2\n3\n4\n5\n6\n7\n8\n9"},
		{ID: "a", Type: bwcmd.ItemTypeLogin, Name: "Alpha", Login: &bwcmd.Login{Totp: alphaKey}},
	}
	m := newTestVault(items)
	m = press(t, m, tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	m.drawerScroll = 2
	m = press(t, m, tea.KeyPressMsg{Code: '/', Text: "/"})
	if item := m.selectedItem(); item == nil || item.ID != "n1" {
		t.Fatalf("selected = %+v, want ANote", item)
	}
	if m.drawerScroll != 2 {
		t.Errorf("drawerScroll = %d, want 2: the same item stays selected", m.drawerScroll)
	}
}

func TestTOTPNotCopiedForGroupHeader(t *testing.T) {
	items := []bwcmd.Item{
		{ID: "g1", Type: bwcmd.ItemTypeLogin, Name: "GitHub (Work)", Login: &bwcmd.Login{Totp: alphaKey}},
		{ID: "g2", Type: bwcmd.ItemTypeLogin, Name: "GitHub (Home)", Login: &bwcmd.Login{Totp: alphaKey}},
	}
	m := newTestVault(items)
	m = press(t, m, tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	if _, ok := m.list.SelectedItem().(GroupHeaderItem); !ok {
		t.Fatalf("selected = %T, want a group header", m.list.SelectedItem())
	}
	if got := copiedTOTP(t, m); got != "" {
		t.Errorf("t on a group header copied %q", got)
	}
}

func TestInvalidTOTPShowsErrorAndToast(t *testing.T) {
	items := []bwcmd.Item{
		{ID: "x", Type: bwcmd.ItemTypeLogin, Name: "Broken", Login: &bwcmd.Login{Totp: "!!!!"}},
	}
	m := newTestVault(items)
	out := m.ViewContent(80, 21)
	if !strings.Contains(out, "invalid TOTP secret") || strings.Contains(out, "loading") {
		t.Error("the drawer must say that the TOTP secret is invalid")
	}
	updated, _ := m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	m = updated.(VaultModel)
	if !m.toastIsError || !strings.Contains(m.toast, "Invalid TOTP") {
		t.Errorf("toast = %q, want an error toast", m.toast)
	}
}

func TestTOTPTickWithHugeDigitsDoesNotPanic(t *testing.T) {
	items := []bwcmd.Item{
		{ID: "d", Type: bwcmd.ItemTypeLogin, Name: "Digits", Login: &bwcmd.Login{Totp: "otpauth://totp/x?secret=" + alphaKey + "&digits=20"}},
	}
	m := newTestVault(items)
	updated, _ := m.Update(TOTPTickMsg{})
	m = updated.(VaultModel)
	if len(m.totpCode) != 10 {
		t.Errorf("code = %q, want 10 digits", m.totpCode)
	}
}
