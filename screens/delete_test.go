package screens

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/ui"
)

// fakeDelete replaces deleteItemCmd for one test. The returned slice
// collects the IDs. The command returns no message, so the test sends the
// DeleteResult itself.
func fakeDelete(t *testing.T) *[]string {
	t.Helper()
	var ids []string
	old := deleteItemCmd
	t.Cleanup(func() { deleteItemCmd = old })
	deleteItemCmd = func(_ string, _ uint64, id string) tea.Cmd {
		ids = append(ids, id)
		return nil
	}
	return &ids
}

// selectID selects the row of the item id.
func selectID(t *testing.T, m VaultModel, id string) VaultModel {
	t.Helper()
	if !m.selectKey(rowKey{id: id}) {
		t.Fatalf("no row for %s", id)
	}
	m.syncSelection()
	return m
}

func TestDeleteAsksFirst(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{keyMsg("d"), {Code: tea.KeyDelete}} {
		ids := fakeDelete(t)
		m := selectID(t, newTestVault(editItems()), "gh")
		m, _ = send(m, k)
		if m.overlay != overlayDelete {
			t.Fatalf("%s: the trash dialog did not open", k)
		}
		if v := formView(m); !strings.Contains(v, `Move "GitHub" to trash?`) {
			t.Errorf("%s: the dialog must name the item:\n%s", k, v)
		}
		if len(*ids) != 0 {
			t.Errorf("%s: the dialog must not delete before y", k)
		}
	}
}

func TestDeleteCancel(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{keyMsg("n"), esc} {
		ids := fakeDelete(t)
		m := selectID(t, newTestVault(editItems()), "gh")
		m, _ = send(m, keyMsg("d"))
		m, _ = send(m, k)
		if m.overlay != overlayNone || len(*ids) != 0 {
			t.Errorf("%s must close the dialog and delete nothing", k)
		}
	}
}

func TestDeleteMovesToTrash(t *testing.T) {
	ids := fakeDelete(t)
	m := selectID(t, newTestVault(editItems()), "visa")
	m, _ = send(m, keyMsg("d"))
	m, _ = send(m, keyMsg("y"))
	if len(*ids) != 1 || (*ids)[0] != "visa" || !m.del.running {
		t.Fatalf("y: deletes = %q, want visa", *ids)
	}
	// Keys do nothing while bw runs.
	m, _ = send(m, esc)
	if m.overlay != overlayDelete {
		t.Error("esc closed the dialog while bw runs")
	}

	m, cmd := send(m, bwcmd.DeleteResult{ID: "visa", Epoch: m.sess.Epoch})
	if m.overlay != overlayNone || cmd == nil {
		t.Fatal("a delete that worked must close the dialog and load the items")
	}
	if !strings.Contains(m.toast, "Moved Visa to trash") {
		t.Errorf("toast = %q", m.toast)
	}
	var left []bwcmd.Item
	for _, it := range editItems() {
		if it.ID != "visa" {
			left = append(left, it)
		}
	}
	m, _ = send(m, bwcmd.ItemsResult{Items: left, Epoch: m.sess.Epoch})
	if item := m.selectedItem(); item == nil || item.ID != "note" {
		t.Errorf("selected = %v, want the next item, note", item)
	}
}

func TestDeleteLastItemSelectsPrevious(t *testing.T) {
	fakeDelete(t)
	m := selectID(t, newTestVault(editItems()), "bank")
	m.verifiedID = "bank" // the re-prompt passed
	m, _ = send(m, keyMsg("d"))
	m, _ = send(m, keyMsg("y"))
	m, _ = send(m, bwcmd.DeleteResult{ID: "bank", Epoch: m.sess.Epoch})
	m, _ = send(m, bwcmd.ItemsResult{Items: editItems()[:3], Epoch: m.sess.Epoch})
	if item := m.selectedItem(); item == nil || item.ID != "note" {
		t.Errorf("selected = %v, want the previous item, note", item)
	}
}

func TestDeleteFailureShowsError(t *testing.T) {
	ids := fakeDelete(t)
	m := selectID(t, newTestVault(editItems()), "gh")
	m, _ = send(m, keyMsg("d"))
	m, _ = send(m, keyMsg("y"))
	m, _ = send(m, bwcmd.DeleteResult{ID: "gh", Err: errors.New("Not found."), Epoch: m.sess.Epoch})
	if m.overlay != overlayDelete || m.del.running || !strings.Contains(formView(m), "Not found.") {
		t.Fatalf("a failed delete must keep the dialog and show the error:\n%s", formView(m))
	}
	m, _ = send(m, keyMsg("y"))
	if len(*ids) != 2 {
		t.Errorf("y after an error ran %d deletes in all, want 2", len(*ids))
	}
}

func TestDeleteWorksForEachType(t *testing.T) {
	for _, it := range testItems() {
		ids := fakeDelete(t)
		m := selectID(t, newTestVault(testItems()), it.ID)
		m, _ = send(m, keyMsg("d"))
		send(m, keyMsg("y"))
		if len(*ids) != 1 || (*ids)[0] != it.ID {
			t.Errorf("type %d: deletes = %q, want %s", it.Type, *ids, it.ID)
		}
	}
}

func TestDeleteAsksForMasterPassword(t *testing.T) {
	ids := fakeDelete(t)
	stubReprompt(t)
	m := selectID(t, newTestVault(editItems()), "bank")
	m, _ = send(m, keyMsg("d"))
	if m.overlay != overlayReprompt {
		t.Fatal("d on an item with re-prompt must ask for the master password first")
	}
	m, _ = submitPassword(t, m, rightPassword, func(m VaultModel, _ []tea.Msg) bool { return m.overlay == overlayDelete })
	if m.overlay != overlayDelete || len(*ids) != 0 {
		t.Fatal("the right password must open the dialog, and delete nothing yet")
	}
}

func TestDeleteResultOfEndedSessionIsDropped(t *testing.T) {
	fakeDelete(t)
	m := selectID(t, newTestVault(editItems()), "gh")
	m, _ = send(m, keyMsg("d"))
	m, _ = send(m, keyMsg("y"))
	m, cmd := send(m, bwcmd.DeleteResult{ID: "gh", Epoch: m.sess.Epoch + 1})
	if m.overlay != overlayDelete || cmd != nil {
		t.Error("a result of an ended session must change nothing")
	}
}

func TestDeleteDialogFitsSmallTerminals(t *testing.T) {
	long := bwcmd.Item{ID: "x", Type: bwcmd.ItemTypeLogin, Name: strings.Repeat("Very long name ", 10)}
	for _, size := range [][2]int{{80, 24}, {50, 15}, {30, 10}} {
		m := vaultAt([]bwcmd.Item{long}, size[0], size[1])
		m, _ = send(m, keyMsg("d"))
		out := m.ViewContent(size[0], size[1]-ui.ChromeHeight)
		assertArea(t, out, size[0], size[1]-ui.ChromeHeight)
	}
}
