package screens

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/session"
	"github.com/juthrbog/lazybw/ui"
)

// The master password that the stub of repromptUnlockCmd accepts, and the
// token that it returns for it.
const (
	rightPassword = "right"
	newToken      = "new-tok"
)

// stubReprompt replaces repromptUnlockCmd for one test. rightPassword
// passes the check and returns newToken. Other passwords fail as bw unlock
// fails for a wrong master password. It returns the passwords that the
// checks got.
func stubReprompt(t *testing.T) *[]string {
	t.Helper()
	calls := &[]string{}
	orig := repromptUnlockCmd
	t.Cleanup(func() { repromptUnlockCmd = orig })
	repromptUnlockCmd = func(password string) tea.Cmd {
		*calls = append(*calls, password)
		return func() tea.Msg {
			if password == rightPassword {
				return bwcmd.UnlockResult{Token: newToken}
			}
			return bwcmd.UnlockResult{Err: &bwcmd.Error{Msg: "The provided key is not the expected type"}}
		}
	}
	return calls
}

// repromptItems has one item of each type, each with master password
// re-prompt and every protected field set.
func repromptItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "l", Type: bwcmd.ItemTypeLogin, Name: "Bank", Reprompt: bwcmd.RepromptPassword, Notes: "pin 4321",
			Login: &bwcmd.Login{Username: "me", Password: "bank-pw", Totp: alphaKey}},
		{ID: "c", Type: bwcmd.ItemTypeCard, Name: "Visa", Reprompt: bwcmd.RepromptPassword,
			Card: &bwcmd.Card{Number: "4242424242424242", Code: "123"}},
		{ID: "i", Type: bwcmd.ItemTypeIdentity, Name: "Me", Reprompt: bwcmd.RepromptPassword,
			Identity: &bwcmd.Identity{SSN: "123-45-6789", Email: "me@example.com"}},
		{ID: "k", Type: bwcmd.ItemTypeSSHKey, Name: "Server", Reprompt: bwcmd.RepromptPassword,
			SSHKey: &bwcmd.SSHKey{PrivateKey: "PRIVATE KEY", PublicKey: "ssh-ed25519 PUBLIC"}},
		{ID: "n", Type: bwcmd.ItemTypeSecureNote, Name: "Keys", Reprompt: bwcmd.RepromptPassword, Notes: "note body"},
	}
}

// repromptVault returns a vault screen in an open session with the token
// "old-tok".
func repromptVault(items []bwcmd.Item) VaultModel {
	sess := &session.State{}
	sess.SetToken("old-tok")
	return NewVaultModel(items, sess, 80, 24)
}

// selectItem moves the cursor to row i.
func selectItem(m VaultModel, i int) VaultModel {
	m.list.Select(i)
	m.syncSelection()
	return m
}

// driveVault is drive for the vault screen. It feeds the messages of cmd
// back into m until done returns true, no command is left, or no message
// arrives for 300ms.
func driveVault(t *testing.T, m VaultModel, cmd tea.Cmd, done func(VaultModel, []tea.Msg) bool) (VaultModel, []tea.Msg) {
	t.Helper()
	msgs := make(chan tea.Msg)
	stop := make(chan struct{})
	defer close(stop)
	running := 0
	start := func(c tea.Cmd) {
		if c == nil {
			return
		}
		running++
		go func() {
			select {
			case msgs <- c():
			case <-stop:
			}
		}()
	}
	start(cmd)

	var seen []tea.Msg
	deadline := time.After(5 * time.Second)
	for done == nil || !done(m, seen) {
		if running == 0 {
			return m, seen
		}
		select {
		case msg := <-msgs:
			running--
			if batch, ok := msg.(tea.BatchMsg); ok {
				for _, c := range batch {
					start(c)
				}
				continue
			}
			if msg == nil {
				continue
			}
			seen = append(seen, msg)
			if _, ok := msg.(spinner.TickMsg); ok {
				continue
			}
			updated, next := m.Update(msg)
			m = updated.(VaultModel)
			start(next)
		case <-time.After(300 * time.Millisecond):
			return m, seen
		case <-deadline:
			t.Fatal("driveVault did not end after 5s")
		}
	}
	return m, seen
}

// copied reports whether seen holds a CopiedMsg and a clipboard write. A
// copy sends both in a batch, so they can arrive in any order.
func copied(_ VaultModel, seen []tea.Msg) bool {
	var report, write bool
	for _, msg := range seen {
		if _, ok := msg.(session.CopiedMsg); ok {
			report = true
		}
		// tea.SetClipboard returns a message of an unexported type.
		if msg != nil && reflect.TypeOf(msg).Name() == "setClipboardMsg" {
			write = true
		}
	}
	return report && write
}

// promptError reports whether the password prompt shows an error.
func promptError(m VaultModel, _ []tea.Msg) bool {
	return m.overlay == overlayReprompt && m.reprompt.errText != ""
}

// typeVault types s into m, one key press per character.
func typeVault(m VaultModel, s string) VaultModel {
	for _, r := range s {
		updated, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = updated.(VaultModel)
	}
	return m
}

// submitPassword types password into the open prompt, presses enter and
// drives m until done returns true.
func submitPassword(t *testing.T, m VaultModel, password string, done func(VaultModel, []tea.Msg) bool) (VaultModel, []tea.Msg) {
	t.Helper()
	if m.overlay != overlayReprompt {
		t.Fatal("the password prompt is not open")
	}
	m = typeVault(m, password)
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return driveVault(t, updated.(VaultModel), cmd, done)
}

// hasClipboardWrite reports whether seen holds a clipboard write of value.
func hasClipboardWrite(seen []tea.Msg, value string) bool {
	for _, msg := range seen {
		if msg == tea.SetClipboard(value)() {
			return true
		}
	}
	return false
}

func TestRepromptCorrectPasswordRunsCopy(t *testing.T) {
	calls := stubReprompt(t)
	m := repromptVault(repromptItems())
	epoch := m.sess.Epoch
	m = press(t, m, keyMsg("c"))
	m, seen := submitPassword(t, m, rightPassword, copied)

	if !hasClipboardWrite(seen, "bank-pw") {
		t.Fatalf("after the correct password, c must copy the password, saw %v", seen)
	}
	if len(*calls) != 1 || (*calls)[0] != rightPassword {
		t.Errorf("password checks = %q, want one with the typed password", *calls)
	}
	if m.overlay == overlayReprompt {
		t.Error("the prompt must close after the check passes")
	}
	if m.sess.Token != newToken || m.sess.Epoch != epoch {
		t.Errorf("token = %q, epoch = %d, want %q and the same epoch %d", m.sess.Token, m.sess.Epoch, newToken, epoch)
	}
	// The item stays verified while it stays selected: the next copy needs
	// one key press.
	_, cmd := m.Update(keyMsg("m"))
	if values, _ := clipboardWrites(t, cmd, "pin 4321"); len(values) != 1 {
		t.Error("after the check, m on the same item must copy at once")
	}
	if len(*calls) != 1 {
		t.Errorf("password checks = %d, want no new check", len(*calls))
	}
}

func TestRepromptWrongPasswordCopiesNothing(t *testing.T) {
	stubReprompt(t)
	m := repromptVault(repromptItems())
	m = press(t, m, keyMsg("c"))
	m, seen := submitPassword(t, m, "wrong", promptError)

	if hasClipboardWrite(seen, "bank-pw") || copied(m, seen) {
		t.Fatal("a wrong password must not copy")
	}
	if m.reprompt.errText != wrongPasswordText {
		t.Errorf("prompt error = %q, want %q", m.reprompt.errText, wrongPasswordText)
	}
	if plain := strings.Join(strings.Fields(ansi.Strip(m.ViewContent(80, 24-ui.ChromeHeight))), " "); !strings.Contains(plain, "The master password is not correct.") {
		t.Errorf("the prompt must show the error:\n%s", plain)
	}
	if m.sess.Token != "old-tok" {
		t.Errorf("token = %q, want the old token after a failed check", m.sess.Token)
	}
	// The prompt stays open, so the user can try again.
	m, seen = submitPassword(t, m, rightPassword, copied)
	if !hasClipboardWrite(seen, "bank-pw") {
		t.Error("a second try with the correct password must copy")
	}
	if m.overlay == overlayReprompt {
		t.Error("the prompt must close after the check passes")
	}
}

func TestRepromptEscCancels(t *testing.T) {
	calls := stubReprompt(t)
	m := repromptVault(repromptItems())
	m = press(t, m, keyMsg("c"))
	m = typeVault(m, "partial")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(VaultModel)
	if m.overlay == overlayReprompt {
		t.Error("esc must close the password prompt")
	}
	if cmd != nil || len(*calls) != 0 {
		t.Error("esc must not copy or check the password")
	}
}

func TestRepromptProtectsEveryField(t *testing.T) {
	tests := []struct {
		item int
		key  string
	}{
		{0, "c"}, {0, "t"}, {0, "m"}, // password, TOTP and note
		{1, "c"}, {1, "v"}, // card number and security code
		{2, "c"}, // SSN
		{3, "c"}, // SSH private key
		{4, "m"}, // note
	}
	secrets := []string{"bank-pw", "pin 4321", "4242424242424242", "123", "123-45-6789", "PRIVATE KEY", "note body"}
	secrets = append(secrets, codesFor(t, alphaKey)...)
	for _, tt := range tests {
		m := selectItem(repromptVault(repromptItems()), tt.item)
		name := tt.key + " on " + m.selectedItem().Name
		updated, cmd := m.Update(keyMsg(tt.key))
		m = updated.(VaultModel)
		if values, fields := clipboardWrites(t, cmd, secrets...); len(values) != 0 || len(fields) != 0 {
			t.Errorf("%s: copied %q before the password check", name, values)
		}
		if m.overlay != overlayReprompt {
			t.Errorf("%s: the password prompt must open", name)
		}
	}
}

func TestRepromptLeavesOtherFieldsAlone(t *testing.T) {
	// Usernames, emails and public keys are not protected by re-prompt.
	for i, want := range map[int]string{0: "me", 2: "me@example.com", 3: "ssh-ed25519 PUBLIC"} {
		m := selectItem(repromptVault(repromptItems()), i)
		updated, cmd := m.Update(keyMsg("u"))
		m = updated.(VaultModel)
		if values, _ := clipboardWrites(t, cmd, want); len(values) != 1 || m.overlay == overlayReprompt {
			t.Errorf("u on %s must copy %q with no prompt", m.selectedItem().Name, want)
		}
	}
}

func TestRepromptCopiesTheAskedField(t *testing.T) {
	stubReprompt(t)
	m := selectItem(repromptVault(repromptItems()), 1) // the card
	m = press(t, m, keyMsg("v"))
	_, seen := submitPassword(t, m, rightPassword, copied)
	if !hasClipboardWrite(seen, "123") || hasClipboardWrite(seen, "4242424242424242") {
		t.Error("after the check, v must copy the security code and nothing else")
	}
}

func TestRepromptEndsOnSelectionChange(t *testing.T) {
	calls := stubReprompt(t)
	m := repromptVault(repromptItems())
	m = press(t, m, keyMsg("c"))
	m, _ = submitPassword(t, m, rightPassword, copied)
	m = press(t, m, keyMsg("j"))
	m = press(t, m, keyMsg("k"))
	if item := m.selectedItem(); item == nil || item.ID != "l" {
		t.Fatalf("selected = %+v, want the login again", item)
	}
	updated, cmd := m.Update(keyMsg("c"))
	m = updated.(VaultModel)
	if values, _ := clipboardWrites(t, cmd, "bank-pw"); len(values) != 0 || m.overlay != overlayReprompt {
		t.Error("after a selection change, c must ask for the password again")
	}
	if len(*calls) != 1 {
		t.Errorf("password checks = %d, want 1", len(*calls))
	}
}

func TestRepromptRefreshThatRemovesItemEndsVerification(t *testing.T) {
	stubReprompt(t)
	m := repromptVault(repromptItems())
	m = press(t, m, keyMsg("c"))
	m, _ = submitPassword(t, m, rightPassword, copied)
	if m.verifiedID != "l" {
		t.Fatalf("verifiedID = %q, want the login", m.verifiedID)
	}
	// A refresh that keeps the item keeps the check.
	updated, _ := m.Update(bwcmd.ItemsResult{Items: repromptItems()})
	m = updated.(VaultModel)
	if m.verifiedID != "l" {
		t.Errorf("a refresh that keeps the item must keep the check, verifiedID = %q", m.verifiedID)
	}
	// A refresh that removes it ends the check.
	updated, _ = m.Update(bwcmd.ItemsResult{Items: repromptItems()[1:]})
	m = updated.(VaultModel)
	if m.verifiedID != "" {
		t.Errorf("a refresh that removes the item must end the check, verifiedID = %q", m.verifiedID)
	}
}

func TestRepromptDrawerHidesUntilVerified(t *testing.T) {
	stubReprompt(t)
	m := repromptVault(repromptItems())
	view := func() string { return ansi.Strip(m.ViewContent(80, 24-ui.ChromeHeight)) }
	shownCode := func(out string) bool {
		for _, code := range codesFor(t, alphaKey) {
			if strings.Contains(out, code[:3]+" "+code[3:]) {
				return true
			}
		}
		return false
	}
	out := view()
	if shownCode(out) || strings.Contains(out, "pin 4321") || !strings.Contains(out, "re-prompt") {
		t.Fatalf("before the check, the drawer must hide the TOTP code and the note:\n%s", out)
	}
	m = press(t, m, keyMsg("t"))
	m, _ = submitPassword(t, m, rightPassword, copied)
	out = view()
	if !shownCode(out) || !strings.Contains(out, "pin 4321") {
		t.Errorf("after the check, the drawer must show the TOTP code and the note:\n%s", out)
	}
}

func TestRepromptClearsTypedPassword(t *testing.T) {
	stubReprompt(t)
	m := repromptVault(repromptItems())
	m = press(t, m, keyMsg("c"))
	m = typeVault(m, rightPassword)
	typed := m.reprompt.password
	if *typed != rightPassword {
		t.Fatalf("the prompt holds %q, want the typed password", *typed)
	}
	updated, _ := m.submitReprompt()
	m = updated.(VaultModel)
	if *typed != "" || *m.reprompt.password != "" {
		t.Error("submit must clear the typed password")
	}
	if strings.Contains(m.ViewContent(80, 21), rightPassword) {
		t.Error("the prompt must not show the password")
	}
}

func TestRepromptResultAfterLockIsDropped(t *testing.T) {
	stubReprompt(t)
	m := repromptVault(repromptItems())
	m = press(t, m, keyMsg("c"))
	m = typeVault(m, rightPassword)
	updated, check := m.submitReprompt()
	m = updated.(VaultModel)
	// The session ends while bw runs.
	m.sess.Lock()
	updated, cmd := m.Update(check())
	m = updated.(VaultModel)
	if values, fields := clipboardWrites(t, cmd, "bank-pw"); len(values) != 0 || len(fields) != 0 {
		t.Error("a result from an ended session must not copy")
	}
	if !m.sess.IsLocked() {
		t.Error("a result from an ended session must not open the session again")
	}
}

// busyPrompt reports whether the password prompt waits for a check.
func busyPrompt(m VaultModel, _ []tea.Msg) bool {
	return m.overlay == overlayReprompt && m.reprompt.busy()
}

// submitWhileFirstCheckRuns starts a check, closes its prompt, and submits
// a second password in a new prompt while the first check runs. It returns
// the vault and the first check.
func submitWhileFirstCheckRuns(t *testing.T, m VaultModel, second string) (VaultModel, tea.Cmd) {
	t.Helper()
	m = press(t, m, keyMsg("c"))
	m = typeVault(m, rightPassword)
	updated, first := m.submitReprompt()
	m = updated.(VaultModel)
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	m = press(t, m, keyMsg("c"))
	m, _ = submitPassword(t, m, second, busyPrompt)
	return m, first
}

func TestRepromptOneCheckAtATime(t *testing.T) {
	calls := stubReprompt(t)
	m, first := submitWhileFirstCheckRuns(t, repromptVault(repromptItems()), rightPassword)
	if len(*calls) != 1 {
		t.Fatalf("checks = %d: a second check must wait for the first", len(*calls))
	}
	plain := ansi.Strip(m.ViewContent(80, 24-ui.ChromeHeight))
	if m.reprompt.errText != "" || !strings.Contains(plain, "Checking the master password") {
		t.Errorf("prompt error = %q: the prompt must show that the check runs:\n%s", m.reprompt.errText, plain)
	}
	if *m.reprompt.password != "" {
		t.Error("submit must clear the typed password")
	}
	// The first check ends. Its prompt is closed, so it copies nothing.
	// The second password waited for it, and its check starts now.
	updated, cmd := m.Update(first())
	m = updated.(VaultModel)
	if m.sess.Token != newToken {
		t.Errorf("token = %q, want the token of the passed check", m.sess.Token)
	}
	if len(*calls) != 2 || (*calls)[1] != rightPassword {
		t.Fatalf("password checks = %q, want the second password checked after the first check", *calls)
	}
	m, seen := driveVault(t, m, cmd, copied)
	if !hasClipboardWrite(seen, "bank-pw") {
		t.Errorf("the second check must copy the password, saw %v", seen)
	}
	if m.overlay == overlayReprompt || m.reprompt.errText != "" {
		t.Errorf("the prompt must close with no error, open = %v, error = %q", m.overlay == overlayReprompt, m.reprompt.errText)
	}
}

func TestRepromptWaitingPasswordFails(t *testing.T) {
	calls := stubReprompt(t)
	m, first := submitWhileFirstCheckRuns(t, repromptVault(repromptItems()), "wrong")
	updated, cmd := m.Update(first())
	m, seen := driveVault(t, updated.(VaultModel), cmd, promptError)
	if len(*calls) != 2 || hasClipboardWrite(seen, "bank-pw") {
		t.Errorf("checks = %q: a wrong waiting password must be checked and copy nothing", *calls)
	}
	if m.reprompt.errText != wrongPasswordText {
		t.Errorf("prompt error = %q, want %q", m.reprompt.errText, wrongPasswordText)
	}
}

func TestRepromptEscDropsWaitingPassword(t *testing.T) {
	calls := stubReprompt(t)
	m, first := submitWhileFirstCheckRuns(t, repromptVault(repromptItems()), rightPassword)
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.overlay == overlayReprompt {
		t.Fatal("esc must close the prompt")
	}
	// A new prompt starts with no waiting password.
	m = press(t, m, keyMsg("c"))
	updated, cmd := m.Update(first())
	m = updated.(VaultModel)
	if values, _ := clipboardWrites(t, cmd, "bank-pw"); len(values) != 0 || len(*calls) != 1 {
		t.Errorf("checks = %d: after esc, the waiting password must not be checked", len(*calls))
	}
	if m.reprompt.busy() {
		t.Error("the new prompt must take a password, not wait for a check")
	}
}

func TestRepromptPromptFitsSmallTerminals(t *testing.T) {
	stubReprompt(t)
	for _, size := range [][2]int{{80, 24}, {50, 15}, {30, 10}} {
		width, height := size[0], size[1]
		m := vaultAt(repromptItems(), width, height)
		m.sess.SetToken("old-tok")
		m = press(t, m, keyMsg("c"))
		m, _ = submitPassword(t, m, "wrong", promptError)
		out := m.ViewContent(width, height-ui.ChromeHeight)
		assertArea(t, out, width, height-ui.ChromeHeight)
		if width >= 50 {
			cardColumns(t, out)
		}
	}
}

// promptView returns the plain text of the vault screen of m for a
// terminal of the given size.
func promptView(m VaultModel, width, height int) string {
	return ansi.Strip(m.ViewContent(width, height-ui.ChromeHeight))
}

func TestRepromptCardKeepsInputAndError(t *testing.T) {
	stubReprompt(t)
	for _, size := range [][2]int{{40, 12}, {80, 16}, {30, 10}} {
		width, height := size[0], size[1]
		name := fmt.Sprintf("%dx%d", width, height)
		m := vaultAt(repromptItems(), width, height)
		m.sess.SetToken("old-tok")
		m = press(t, m, keyMsg("c"))
		m = typeVault(m, "abc")
		if out := promptView(m, width, height); !strings.Contains(out, "> ***") {
			t.Errorf("%s: the card must show the input row:\n%s", name, out)
		}
		m, _ = submitPassword(t, m, "", promptError)
		out := promptView(m, width, height)
		if !strings.Contains(out, "✗ The master password") {
			t.Errorf("%s: the card must show the error:\n%s", name, out)
		}
		if !strings.Contains(out, "┃ >") {
			t.Errorf("%s: the card must show the input row below the error:\n%s", name, out)
		}
		// The footer shows the keys, so the card does not.
		if strings.Contains(out, "enter submit") {
			t.Errorf("%s: the card must not show the form help:\n%s", name, out)
		}
	}
}

func TestRepromptCardShowsEmptyPasswordError(t *testing.T) {
	stubReprompt(t)
	m := vaultAt(repromptItems(), 40, 12)
	m = press(t, m, keyMsg("c"))
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	out := promptView(m, 40, 12)
	if !strings.Contains(out, "enter your master password") || !strings.Contains(out, "┃ >") {
		t.Errorf("an empty password must show an error above the input:\n%s", out)
	}
}

func TestRepromptCardFollowsResize(t *testing.T) {
	m := vaultAt(repromptItems(), 80, 24)
	m = press(t, m, keyMsg("c"))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 30, Height: 24})
	m = updated.(VaultModel)
	// A form as wide as the card scrolls a long input, so that the cursor
	// after the last character stays inside the card. A wider form fills
	// the card with characters, and the card cuts the cursor off.
	m = typeVault(m, strings.Repeat("x", 30))
	out := m.ViewContent(30, 24-ui.ChromeHeight)
	cardColumns(t, out)
	for _, line := range strings.Split(ansi.Strip(out), "\n") {
		if !strings.Contains(line, "> *") {
			continue
		}
		inner := []rune(line)
		inner = inner[3 : len(inner)-3] // the border and the padding
		if inner[len(inner)-1] == '*' {
			t.Errorf("the cursor must show at the end of the input row: %q", line)
		}
		return
	}
	t.Errorf("no input row in:\n%s", ansi.Strip(out))
}
