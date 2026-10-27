package screens

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
)

// editFixture is the bw get item output of a login with keys that
// bwcmd.Item does not model.
const editFixture = `{
  "passwordHistory": [{"lastUsedDate": "2025-01-01T00:00:00.000Z", "password": "older"}],
  "revisionDate": "2026-03-01T10:00:00.000Z",
  "creationDate": "2024-01-01T10:00:00.000Z",
  "deletedDate": null,
  "object": "item",
  "id": "gh",
  "organizationId": "org-1",
  "folderId": "folder-1",
  "type": 1,
  "reprompt": 0,
  "name": "GitHub",
  "notes": "tab\there\r\nline 2",
  "favorite": true,
  "fields": [{"name": "pin", "value": "1234", "type": 1, "linkedId": null}],
  "login": {
    "fido2Credentials": [{"credentialId": "cred", "keyValue": "key", "rpId": "github.com", "counter": "0"}],
    "uris": [{"match": 3, "uri": "https://github.com"}, {"match": null, "uri": "https://gist.github.com"}],
    "username": "octo",
    "password": "hunter2",
    "totp": "otpauth://totp/x?secret=JBSWY3DPEHPK3PXP",
    "passwordRevisionDate": "2025-06-01T00:00:00.000Z"
  },
  "collectionIds": ["col-1"]
}`

// decodeFixture decodes s as GetItem does.
func decodeFixture(t *testing.T, s string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var item map[string]any
	if err := dec.Decode(&item); err != nil {
		t.Fatal(err)
	}
	return item
}

// editCall is one call of the fake editItemCmd.
type editCall struct {
	id      string
	item    map[string]any
	private []string
}

// fakeEdit replaces getItemCmd and editItemCmd for one test. The get
// returns no message, so the test sends the GetItemResult itself.
func fakeEdit(t *testing.T) (gets *[]string, edits *[]editCall) {
	t.Helper()
	var g []string
	var e []editCall
	oldGet, oldEdit := getItemCmd, editItemCmd
	t.Cleanup(func() { getItemCmd, editItemCmd = oldGet, oldEdit })
	getItemCmd = func(_ string, _ uint64, id string) tea.Cmd {
		g = append(g, id)
		return nil
	}
	editItemCmd = func(_ string, _ uint64, id string, item map[string]any, private []string) tea.Cmd {
		e = append(e, editCall{id: id, item: item, private: private})
		return nil
	}
	return &g, &e
}

// editItems are the list items of the edit tests.
func editItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "gh", Type: bwcmd.ItemTypeLogin, Name: "GitHub", Login: &bwcmd.Login{Username: "octo"}},
		{ID: "visa", Type: bwcmd.ItemTypeCard, Name: "Visa", Card: &bwcmd.Card{Number: "4242424242424242"}},
		{ID: "note", Type: bwcmd.ItemTypeSecureNote, Name: "Keys", Notes: "secret"},
		{ID: "bank", Type: bwcmd.ItemTypeLogin, Name: "Bank", Reprompt: bwcmd.RepromptPassword, Login: &bwcmd.Login{Password: "pw"}},
	}
}

// openEdit selects the item id, presses e and sends the GetItemResult.
func openEdit(t *testing.T, m VaultModel, id, fixture string) VaultModel {
	t.Helper()
	if !m.selectKey(rowKey{id: id}) {
		t.Fatalf("no row for %s", id)
	}
	m.syncSelection()
	m, _ = send(m, keyMsg("e"))
	if m.loadingID != id {
		t.Fatalf("e: loadingID = %q, want %q", m.loadingID, id)
	}
	m, _ = send(m, bwcmd.GetItemResult{ID: id, Item: decodeFixture(t, fixture), Epoch: m.sess.Epoch})
	if !m.formOpen || m.form.id != id {
		t.Fatal("the edit form did not open")
	}
	return m
}

func TestEditFormShowsValues(t *testing.T) {
	fakeEdit(t)
	m := openEdit(t, newTestVault(editItems()), "gh", editFixture)
	v := formView(m)
	for _, want := range []string{"Edit login", "GitHub", "octo", "otpauth://totp/x", "https://github.com", "https://gist.github.com"} {
		if !strings.Contains(v, want) {
			t.Errorf("the form does not show %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "hunter2") {
		t.Error("the edit form must mask the password")
	}
	if m.form.dirty {
		t.Error("a new edit form has no changes")
	}
	// Both URIs and one empty line.
	n := 0
	for _, l := range m.form.lines {
		if l.kind == fieldURI {
			n++
		}
	}
	if n != 3 {
		t.Errorf("the form has %d URI lines, want 3", n)
	}
}

// TestEditKeepsOtherKeys changes only the name. Each other key of the item
// must go back to bw with no change, also the notes with a tab and \r\n,
// which the text area would change.
func TestEditKeepsOtherKeys(t *testing.T) {
	_, edits := fakeEdit(t)
	m := openEdit(t, newTestVault(editItems()), "gh", editFixture)
	m = typeInto(m, " Work")
	send(m, ctrlS)
	if len(*edits) != 1 || (*edits)[0].id != "gh" {
		t.Fatalf("edits = %+v, want one edit of gh", *edits)
	}

	got := (*edits)[0].item
	want := decodeFixture(t, editFixture)
	want["name"] = "GitHub Work"
	if !jsonEqual(t, got, want) {
		gotJSON, _ := json.MarshalIndent(got, "", "  ")
		t.Errorf("sent item:\n%s\nwant only the name changed", gotJSON)
	}
}

// jsonEqual reports whether a and b encode to the same JSON.
func jsonEqual(t *testing.T, a, b any) bool {
	t.Helper()
	ja, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	jb, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Equal(ja, jb)
}

func TestEditURIs(t *testing.T) {
	_, edits := fakeEdit(t)
	m := openEdit(t, newTestVault(editItems()), "gh", editFixture)
	// Go to the second URI and clear it.
	for range 5 {
		m, _ = send(m, tab)
	}
	if l := m.form.lines[m.form.focus]; l.kind != fieldURI || l.input.Value() != "https://gist.github.com" {
		t.Fatalf("focus is on %q, want the second URI", l.input.Value())
	}
	for range len("https://gist.github.com") {
		m, _ = send(m, backspace)
	}
	// Add a URI in the empty line.
	m, _ = send(m, tab)
	m = typeInto(m, "new.example")
	m, _ = send(m, ctrlS)
	if len(*edits) != 1 {
		t.Fatalf("got %d edits, want 1", len(*edits))
	}
	uris := (*edits)[0].item["login"].(map[string]any)["uris"].([]any)
	want := []any{
		map[string]any{"match": json.Number("3"), "uri": "https://github.com"},
		map[string]any{"match": nil, "uri": "new.example"},
	}
	if !reflect.DeepEqual(uris, want) {
		t.Errorf("uris = %#v, want %#v", uris, want)
	}
}

func TestEditSavesAndRefreshes(t *testing.T) {
	_, edits := fakeEdit(t)
	m := openEdit(t, newTestVault(editItems()), "gh", editFixture)
	m, _ = send(m, tab)
	m, _ = send(m, tab)
	m, _ = send(m, ctrlR)
	for range len("hunter2") {
		m, _ = send(m, backspace)
	}
	m = typeInto(m, "new-pw")
	m, _ = send(m, ctrlS)
	if len(*edits) != 1 {
		t.Fatalf("got %d edits, want 1", len(*edits))
	}
	if pw := (*edits)[0].item["login"].(map[string]any)["password"]; pw != "new-pw" {
		t.Errorf("password = %v, want new-pw", pw)
	}
	if !strings.Contains(strings.Join((*edits)[0].private, "\n"), "new-pw") {
		t.Error("the private values must hold the new password")
	}

	m, cmd := send(m, bwcmd.EditResult{ID: "gh", Epoch: m.sess.Epoch})
	if m.formOpen || cmd == nil || !strings.Contains(m.toast, "Saved GitHub") {
		t.Fatalf("a save that worked must close the form, load the items and show a toast: %q", m.toast)
	}
	items := editItems()
	items[0].Login.Password = "new-pw"
	items[0].Login.Username = "octo2"
	m, _ = send(m, bwcmd.ItemsResult{Items: items, Epoch: m.sess.Epoch})
	if item := m.selectedItem(); item == nil || item.ID != "gh" {
		t.Fatalf("selected = %v, want gh", item)
	}
	if !strings.Contains(formView(m), "octo2") {
		t.Errorf("the drawer must show the new values:\n%s", formView(m))
	}
}

func TestEditFailureKeepsForm(t *testing.T) {
	_, edits := fakeEdit(t)
	m := openEdit(t, newTestVault(editItems()), "gh", editFixture)
	m = typeInto(m, "x")
	m, _ = send(m, ctrlS)
	m, _ = send(m, bwcmd.EditResult{ID: "gh", Err: errors.New("Not found."), Epoch: m.sess.Epoch})
	if !m.formOpen || !strings.Contains(formView(m), "Not found.") || m.form.lines[0].input.Value() != "GitHubx" {
		t.Fatalf("a failed edit must keep the form and show the error:\n%s", formView(m))
	}
	m, _ = send(m, ctrlS)
	if len(*edits) != 2 {
		t.Errorf("got %d edits in all, want 2", len(*edits))
	}
}

func TestEditUnsupportedType(t *testing.T) {
	gets, _ := fakeEdit(t)
	m := newTestVault(editItems())
	m.selectKey(rowKey{id: "visa"})
	m.syncSelection()
	m, _ = send(m, keyMsg("e"))
	if m.formOpen || len(*gets) != 0 {
		t.Error("e on a card must open no form")
	}
	if m.toast != "Editing card items is not supported yet" {
		t.Errorf("toast = %q", m.toast)
	}
}

func TestEditSecureNote(t *testing.T) {
	_, edits := fakeEdit(t)
	m := openEdit(t, newTestVault(editItems()), "note",
		`{"id":"note","type":2,"name":"Keys","notes":"secret","secureNote":{"type":0},"login":null}`)
	if !strings.Contains(formView(m), "Edit secure note") {
		t.Errorf("view:\n%s", formView(m))
	}
	m, _ = send(m, tab)
	m = typeInto(m, " more")
	send(m, ctrlS)
	if len(*edits) != 1 || (*edits)[0].item["notes"] != "secret more" || (*edits)[0].item["login"] != nil {
		t.Errorf("edits = %+v", *edits)
	}
}

func TestEditAsksForMasterPassword(t *testing.T) {
	gets, _ := fakeEdit(t)
	stubReprompt(t)
	m := newTestVault(editItems())
	m.selectKey(rowKey{id: "bank"})
	m.syncSelection()
	m, _ = send(m, keyMsg("e"))
	if m.overlay != overlayReprompt || len(*gets) != 0 {
		t.Fatal("e on an item with re-prompt must ask for the master password first")
	}

	m, _ = submitPassword(t, m, "wrong", promptError)
	if m.formOpen || len(*gets) != 0 || m.loadingID != "" {
		t.Fatal("a wrong password must load no item")
	}
	m, _ = submitPassword(t, m, rightPassword, func(m VaultModel, _ []tea.Msg) bool { return m.loadingID != "" })
	if len(*gets) != 1 || (*gets)[0] != "bank" || m.loadingID != "bank" {
		t.Fatalf("the right password must load the item: gets = %q", *gets)
	}
}

func TestGetItemResultAfterSelectionChangeIsDropped(t *testing.T) {
	fakeEdit(t)
	m := newTestVault(editItems())
	m.selectKey(rowKey{id: "gh"})
	m.syncSelection()
	m, _ = send(m, keyMsg("e"))
	m, _ = send(m, downArrow)
	m, _ = send(m, bwcmd.GetItemResult{ID: "gh", Item: decodeFixture(t, editFixture), Epoch: m.sess.Epoch})
	if m.formOpen || m.loadingID != "" {
		t.Error("a load for an item that is not selected must open no form")
	}
}

func TestGetItemFailureShowsError(t *testing.T) {
	fakeEdit(t)
	m := newTestVault(editItems())
	m.selectKey(rowKey{id: "gh"})
	m.syncSelection()
	m, _ = send(m, keyMsg("e"))
	m, _ = send(m, bwcmd.GetItemResult{ID: "gh", Err: errors.New("Not found."), Epoch: m.sess.Epoch})
	if m.formOpen || !m.toastIsError || !strings.Contains(m.toast, "Edit failed: Not found.") {
		t.Errorf("toast = %q, want the error", m.toast)
	}
}
