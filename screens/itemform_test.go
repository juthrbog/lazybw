package screens

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/ui"
)

var (
	tab      = tea.KeyPressMsg{Code: tea.KeyTab}
	shiftTab = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	ctrlS    = tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
	ctrlR    = tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}
	ctrlGKey = tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}
)

// createCall is one call of the fake createItemCmd.
type createCall struct {
	item    map[string]any
	private []string
}

// fakeCreate replaces createItemCmd for one test. The returned slice
// collects the calls. The command returns no message, so the test sends
// the CreateResult itself.
func fakeCreate(t *testing.T) *[]createCall {
	t.Helper()
	var calls []createCall
	old := createItemCmd
	t.Cleanup(func() { createItemCmd = old })
	createItemCmd = func(_ string, _ uint64, item map[string]any, private []string) tea.Cmd {
		calls = append(calls, createCall{item: item, private: private})
		return nil
	}
	return &calls
}

// send gives msg to m and returns the new model and command. It does not run
// the command, which can start bw.
func send(m VaultModel, msg tea.Msg) (VaultModel, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(VaultModel), cmd
}

// typeInto types s into the focused field of m, one key per rune.
func typeInto(m VaultModel, s string) VaultModel {
	for _, r := range s {
		m, _ = send(m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

// openForm opens the item form of m for the type key k ("l" or "s").
func openForm(t *testing.T, m VaultModel, k string) VaultModel {
	t.Helper()
	m, _ = send(m, keyMsg("n"))
	if m.overlay != overlayNewItem {
		t.Fatalf("n: overlay = %v, want the new item card", m.overlay)
	}
	m, _ = send(m, keyMsg(k))
	if !m.formOpen || m.overlay != overlayNone {
		t.Fatalf("%s: the form did not open", k)
	}
	return m
}

func formView(m VaultModel) string {
	return ansi.Strip(m.ViewContent(80, 24-ui.ChromeHeight))
}

func TestNewItemCardPicksType(t *testing.T) {
	m := openForm(t, newTestVault(testItems()), "l")
	if v := formView(m); !strings.Contains(v, "New login") || !strings.Contains(v, "Password") {
		t.Errorf("login form:\n%s", v)
	}
	m = openForm(t, newTestVault(testItems()), "s")
	if v := formView(m); !strings.Contains(v, "New secure note") || strings.Contains(v, "Password") {
		t.Errorf("secure note form:\n%s", v)
	}

	m, _ = send(newTestVault(testItems()), keyMsg("n"))
	m, _ = send(m, esc)
	if m.overlay != overlayNone || m.formOpen {
		t.Error("esc must close the new item card and open no form")
	}
}

func TestCreateLoginSendsFormValues(t *testing.T) {
	calls := fakeCreate(t)
	m := openForm(t, newTestVault(testItems()), "l")
	m = typeInto(m, "GitHub")
	m, _ = send(m, enter) // enter moves to the next field
	m = typeInto(m, "octo")
	m, _ = send(m, tab)
	m = typeInto(m, "hunter2")
	m, _ = send(m, tab)
	m = typeInto(m, "JBSWY3DPEHPK3PXP")
	m, _ = send(m, tab)
	m = typeInto(m, "github.com")
	m, _ = send(m, tab) // the new empty URI line
	m = typeInto(m, "gist.github.com")
	m, _ = send(m, tab) // the next empty URI line
	m, _ = send(m, tab) // notes
	m = typeInto(m, "line 1")
	m, _ = send(m, enter)
	m = typeInto(m, "line 2")

	m, _ = send(m, ctrlS)
	if len(*calls) != 1 {
		t.Fatalf("ctrl+s ran create %d times, want 1", len(*calls))
	}
	if !m.form.saving {
		t.Error("the form must show that it saves")
	}
	item := (*calls)[0].item
	if item["name"] != "GitHub" || item["notes"] != "line 1\nline 2" || item["type"] != int(bwcmd.ItemTypeLogin) {
		t.Errorf("item = %v", item)
	}
	login := item["login"].(map[string]any)
	if login["username"] != "octo" || login["password"] != "hunter2" || login["totp"] != "JBSWY3DPEHPK3PXP" {
		t.Errorf("login = %v", login)
	}
	uris := login["uris"].([]any)
	if len(uris) != 2 || uris[0].(map[string]any)["uri"] != "github.com" || uris[1].(map[string]any)["uri"] != "gist.github.com" {
		t.Errorf("uris = %v, want the two URIs and no empty one", uris)
	}
	private := strings.Join((*calls)[0].private, "\n")
	for _, v := range []string{"hunter2", "octo", "JBSWY3DPEHPK3PXP", "line 1"} {
		if !strings.Contains(private, v) {
			t.Errorf("private values %q do not hold %q", (*calls)[0].private, v)
		}
	}

	// Keys do nothing while the save runs.
	m, _ = send(m, esc)
	if !m.formOpen {
		t.Error("esc closed the form while the save runs")
	}

	m, cmd := send(m, bwcmd.CreateResult{ID: "new", Epoch: m.sess.Epoch})
	if m.formOpen || cmd == nil {
		t.Fatal("a create that worked must close the form and load the items")
	}
	if !strings.Contains(m.toast, "Created GitHub") {
		t.Errorf("toast = %q", m.toast)
	}
	items := append(testItems(), bwcmd.Item{ID: "new", Type: bwcmd.ItemTypeLogin, Name: "GitHub"})
	m, _ = send(m, bwcmd.ItemsResult{Items: items, Epoch: m.sess.Epoch})
	if item := m.selectedItem(); item == nil || item.ID != "new" {
		t.Errorf("selected = %v, want the new item", item)
	}
}

func TestCreateSecureNote(t *testing.T) {
	calls := fakeCreate(t)
	m := openForm(t, newTestVault(testItems()), "s")
	m = typeInto(m, "Recovery")
	m, _ = send(m, tab)
	m = typeInto(m, "code")
	send(m, ctrlS)
	if len(*calls) != 1 {
		t.Fatalf("ctrl+s ran create %d times, want 1", len(*calls))
	}
	item := (*calls)[0].item
	if item["name"] != "Recovery" || item["notes"] != "code" || item["login"] != nil || item["secureNote"] == nil {
		t.Errorf("item = %v", item)
	}
}

func TestCreateChecksFields(t *testing.T) {
	calls := fakeCreate(t)
	m := openForm(t, newTestVault(testItems()), "l")
	m, _ = send(m, ctrlS)
	if len(*calls) != 0 || !strings.Contains(formView(m), "Enter a name") {
		t.Errorf("an empty name must show an error and not save:\n%s", formView(m))
	}

	m = typeInto(m, "Bad TOTP")
	if strings.Contains(formView(m), "Enter a name") {
		t.Error("the name error must go away when the name changes")
	}
	m, _ = send(m, tab)
	m, _ = send(m, tab)
	m, _ = send(m, tab)
	// The base32 decoder skips each byte outside the alphabet, as the
	// Bitwarden clients do, so only a key with no base32 byte fails.
	m = typeInto(m, "1890")
	m, _ = send(m, shiftTab)
	m, _ = send(m, ctrlS)
	if len(*calls) != 0 || !strings.Contains(formView(m), "Enter an otpauth:// URI or a base32 key") {
		t.Errorf("a bad TOTP key must show an error and not save:\n%s", formView(m))
	}
	if m.form.lines[m.form.focus].kind != fieldTOTP {
		t.Error("the focus must move to the field with the error")
	}
}

func TestCreateFailureKeepsForm(t *testing.T) {
	calls := fakeCreate(t)
	m := openForm(t, newTestVault(testItems()), "l")
	m = typeInto(m, "GitHub")
	m, _ = send(m, ctrlS)
	m, _ = send(m, bwcmd.CreateResult{Err: errors.New("Something went wrong."), Epoch: m.sess.Epoch})
	if !m.formOpen || m.form.saving {
		t.Fatal("a failed create must keep the form open")
	}
	v := formView(m)
	if !strings.Contains(v, "Something went wrong.") || !strings.Contains(v, "GitHub") {
		t.Errorf("the form must show the error and keep the values:\n%s", v)
	}
	m, _ = send(m, ctrlS)
	if len(*calls) != 2 {
		t.Errorf("ctrl+s after an error ran create %d times in all, want 2", len(*calls))
	}
}

func TestCreateResultOfEndedSessionIsDropped(t *testing.T) {
	fakeCreate(t)
	m := openForm(t, newTestVault(testItems()), "l")
	m = typeInto(m, "GitHub")
	m, _ = send(m, ctrlS)
	m, cmd := send(m, bwcmd.CreateResult{ID: "new", Epoch: m.sess.Epoch + 1})
	if !m.formOpen || cmd != nil {
		t.Error("a result of an ended session must change nothing")
	}
}

func TestFormCancel(t *testing.T) {
	calls := fakeCreate(t)
	m := openForm(t, newTestVault(testItems()), "l")
	m, _ = send(m, esc)
	if m.formOpen || m.overlay != overlayNone {
		t.Error("esc on a form with no changes must close it")
	}

	m = openForm(t, m, "l")
	m = typeInto(m, "x")
	m, _ = send(m, esc)
	if !m.formOpen || m.overlay != overlayDiscard {
		t.Fatal("esc on a changed form must ask to discard")
	}
	if !strings.Contains(formView(m), "Discard changes?") {
		t.Errorf("view:\n%s", formView(m))
	}
	m, _ = send(m, keyMsg("n"))
	if !m.formOpen || m.overlay != overlayNone || m.form.lines[0].input.Value() != "x" {
		t.Fatal("n must go back to the form with its values")
	}
	m, _ = send(m, esc)
	m, _ = send(m, keyMsg("y"))
	if m.formOpen || m.overlay != overlayNone {
		t.Error("y must close the form")
	}
	if len(*calls) != 0 {
		t.Error("a cancel must not create an item")
	}
}

func TestFormMasksPassword(t *testing.T) {
	m := openForm(t, newTestVault(testItems()), "l")
	m, _ = send(m, tab)
	m, _ = send(m, tab)
	m = typeInto(m, "hunter2")
	if strings.Contains(formView(m), "hunter2") {
		t.Error("the password must be masked")
	}
	m, _ = send(m, ctrlR)
	if !strings.Contains(formView(m), "hunter2") {
		t.Error("ctrl+r must show the password")
	}
	m, _ = send(m, ctrlR)
	if strings.Contains(formView(m), "hunter2") {
		t.Error("ctrl+r again must mask the password")
	}

	// A new form masks the password again.
	m, _ = send(m, ctrlR)
	m, _ = send(m, esc)
	m, _ = send(m, keyMsg("y"))
	m = openForm(t, m, "l")
	if m.form.reveal {
		t.Error("a new form must mask the password")
	}
}

func TestFormGeneratorFillsPassword(t *testing.T) {
	m := openForm(t, newTestVault(testItems()), "l")
	m, _ = send(m, ctrlGKey)
	if m.overlay == overlayGenerator {
		t.Fatal("ctrl+g outside the password field must not open the generator")
	}
	m, _ = send(m, tab)
	m, _ = send(m, tab)
	m, _ = send(m, ctrlGKey)
	if m.overlay != overlayGenerator {
		t.Fatal("ctrl+g on the password field must open the generator")
	}
	m, _ = send(m, bwcmd.GenerateResult{Password: "gen-pw", Seq: m.gen.seq})
	m, _ = send(m, enter)
	if m.overlay != overlayNone || !m.formOpen {
		t.Fatal("enter must close the generator and go back to the form")
	}
	if got := m.form.lines[m.form.focus].input.Value(); got != "gen-pw" {
		t.Errorf("password = %q, want the generated value", got)
	}
	if !m.form.dirty {
		t.Error("a generated password is a change")
	}
}

func TestGeneratorEnterRegeneratesOutsideForm(t *testing.T) {
	m := newTestVault(testItems())
	m, _ = send(m, keyMsg("p"))
	seq := m.gen.seq
	m, _ = send(m, enter)
	if m.overlay != overlayGenerator || m.gen.seq == seq {
		t.Error("enter in the generator of the vault must generate a new value")
	}
}

func TestClearDropsForm(t *testing.T) {
	m := openForm(t, newTestVault(testItems()), "l")
	m = typeInto(m, "GitHub")
	m.Clear()
	if m.formOpen || len(m.form.lines) != 0 {
		t.Error("Clear must drop the form and its values")
	}
}

func TestFormFitsSmallTerminals(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {40, 12}, {30, 8}} {
		width, height := size[0], size[1]
		m := openForm(t, vaultAt(testItems(), width, height), "l")
		for range 8 {
			m, _ = send(m, tab)
			out := m.ViewContent(width, height-ui.ChromeHeight)
			assertArea(t, out, width, height-ui.ChromeHeight)
			if !strings.Contains(ansi.Strip(out), strings.TrimSpace(ui.Glyphs.Cursor)) {
				t.Errorf("%dx%d: the focused field is not in view:\n%s", width, height, ansi.Strip(out))
			}
		}
	}
}
