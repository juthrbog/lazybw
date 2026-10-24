package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/screens"
	"github.com/juthrbog/lazybw/session"
	"github.com/juthrbog/lazybw/ui"
)

// useASCII turns on the ASCII glyphs for one test. Call it before the test
// builds a model, because the models copy the spinners.
func useASCII(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { ui.UseASCII(false) })
	ui.UseASCII(true)
}

// asciiItems returns vault items of each type with only ASCII text, so that
// any other character in the view comes from lazybw.
func asciiItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "GitHub (Work)", Login: &bwcmd.Login{
			Username: "octocat", Password: "pw1", Totp: "JBSWY3DPEHPK3PXP",
			URIs: []bwcmd.URI{{URI: "https://github.com"}},
		}, Notes: "first line\nsecond line"},
		{ID: "2", Type: bwcmd.ItemTypeLogin, Name: "GitHub (Home)", Login: &bwcmd.Login{Username: "me", Password: "pw2"}},
		{ID: "3", Type: bwcmd.ItemTypeCard, Name: "Visa", Card: &bwcmd.Card{
			CardholderName: "A Person", Number: "4242424242424242", ExpMonth: "12", ExpYear: "2030", Code: "123",
		}},
		{ID: "4", Type: bwcmd.ItemTypeSecureNote, Name: "Wifi", Notes: "network\npassword"},
		{ID: "5", Type: bwcmd.ItemTypeIdentity, Name: "Passport", Identity: &bwcmd.Identity{
			FirstName: "A", LastName: "Person", SSN: "123-45-6789", PassportNumber: "X1", Email: "a@b.c",
		}},
		{ID: "6", Type: bwcmd.ItemTypeSSHKey, Name: "Server", SSHKey: &bwcmd.SSHKey{
			PrivateKey: "priv", PublicKey: "ssh-ed25519 AAAA", KeyFingerprint: "SHA256:abc",
		}},
		{ID: "7", Type: bwcmd.ItemTypeLogin, Name: "Bank", Reprompt: bwcmd.RepromptPassword,
			Login: &bwcmd.Login{Username: "b", Password: "bank-pw"}},
	}
}

// assertASCII fails when the view of m has a character that is not ASCII,
// or when it does not show each text in want.
func assertASCII(t *testing.T, name string, m RootModel, want ...string) {
	t.Helper()
	plain := ansi.Strip(m.View().Content)
	for _, w := range want {
		if !strings.Contains(plain, w) {
			t.Errorf("%s: the view does not show %q:\n%s", name, w, plain)
		}
	}
	for i, line := range strings.Split(plain, "\n") {
		for _, r := range line {
			if r >= utf8.RuneSelf {
				t.Errorf("%s: line %d has %q (U+%04X):\n%s", name, i, r, r, plain)
				return
			}
		}
	}
}

// TestASCIIModeDrawsOnlyASCII renders each screen, card and footer of
// lazybw with the ASCII glyphs. No line may have a character that is not
// ASCII, because a terminal can draw such a character 2 cells wide.
func TestASCIIModeDrawsOnlyASCII(t *testing.T) {
	useASCII(t)
	esc := tea.KeyPressMsg{Code: tea.KeyEscape}

	// The screens before the vault.
	m := NewRootModel(15 * time.Minute)
	m.width, m.height = 80, 24
	assertASCII(t, "loading", m)
	locked, _ := update(t, m, bwcmd.StatusResult{Status: bwcmd.VaultStatus{Status: "locked", UserEmail: "u@t.com"}})
	assertASCII(t, "unlock screen", locked, "Master password", "| ")
	login, _ := update(t, m, bwcmd.StatusResult{Status: bwcmd.VaultStatus{Status: "unauthenticated"}})
	assertASCII(t, "login screen", login, "Email")
	failed, _ := update(t, m, bwcmd.StatusResult{Err: errors.New("bw failed")})
	assertASCII(t, "error screen", failed)

	m = unlockedVault(t, asciiItems())
	assertASCII(t, "vault with TOTP", m, " # lazybw", "> L  GitHub (Work)", "[t] copy", "----")
	for _, k := range []string{"j", "j", "j", "j", "j"} {
		m, _ = update(t, m, keyPress(k))
		assertASCII(t, "vault after j", m)
	}

	// The cards over the vault.
	for _, tc := range []struct {
		name string
		key  tea.KeyPressMsg
		want string
	}{
		{"help", keyPress("?"), "Keybindings"},
		{"generator", keyPress("p"), "Generating..."},
		{"theme picker", keyPress("T"), "Choose theme"},
		{"quit dialog", keyPress("q"), "Quit lazybw?"},
		{"drawer overlay", keyPress("d"), "+----"},
		{"filter input", keyPress("/"), "^/v navigate"},
	} {
		open, _ := update(t, m, tc.key)
		assertASCII(t, tc.name, open, tc.want)
		closed, _ := update(t, open, esc)
		assertASCII(t, tc.name+" after esc", closed)
	}
	generator, _ := update(t, m, keyPress("p"))
	generator, _ = update(t, generator, keyPress("m"))
	assertASCII(t, "passphrase generator", generator, "Capitalize  x")

	// Groups, toasts and the sync spinner.
	grouped, _ := update(t, m, tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	assertASCII(t, "grouped", grouped, "GitHub (2)")
	copied, _ := update(t, m, session.CopiedMsg{Field: session.CopyFieldPassword})
	assertASCII(t, "copy toast", copied, "* Password sent to clipboard - clears in")
	syncing, _ := update(t, m, keyPress("r"))
	assertASCII(t, "sync", syncing, "Syncing...")

	// The master password re-prompt of the Bank item, the last row.
	bank := unlockedVault(t, asciiItems())
	bank, _ = update(t, bank, tea.KeyPressMsg{Code: tea.KeyEnd})
	bank, _ = update(t, bank, keyPress("c"))
	assertASCII(t, "re-prompt", bank, "Master password re-prompt")

	// A narrow terminal cuts texts with the ellipsis.
	narrow, _ := update(t, unlockedVault(t, asciiItems()), tea.WindowSizeMsg{Width: 40, Height: 14})
	assertASCII(t, "narrow", narrow)

	// The lock and the quit show the lock spinner.
	locking, _ := update(t, unlockedVault(t, asciiItems()), screens.LockMsg{})
	assertASCII(t, "locking", locking, "Locking vault...")
}

func TestASCIIFromEnv(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
		warn  bool
	}{
		{"", false, false},
		{"1", true, false},
		{"true", true, false},
		{"TRUE", true, false},
		{"0", false, false},
		{"false", false, false},
		{"yes", false, true},
	} {
		var w bytes.Buffer
		if got := asciiFromEnv(tc.value, &w); got != tc.want {
			t.Errorf("asciiFromEnv(%q) = %v, want %v", tc.value, got, tc.want)
		}
		if warned := strings.Contains(w.String(), "LAZYBW_ASCII"); warned != tc.warn {
			t.Errorf("asciiFromEnv(%q) warning = %q, want a warning: %v", tc.value, w.String(), tc.warn)
		}
	}
}
