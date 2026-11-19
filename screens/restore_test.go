package screens

import (
	"errors"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/ui"
)

// fakeRestore replaces restoreItemCmd for one test. The returned slice
// collects the IDs. The command returns no message, so the test sends the
// RestoreResult itself.
func fakeRestore(t *testing.T) *[]string {
	t.Helper()
	var ids []string
	old := restoreItemCmd
	t.Cleanup(func() { restoreItemCmd = old })
	restoreItemCmd = func(_ string, _ uint64, id string) tea.Cmd {
		ids = append(ids, id)
		return nil
	}
	return &ids
}

// without returns items without the item id.
func without(items []bwcmd.Item, id string) []bwcmd.Item {
	var out []bwcmd.Item
	for _, it := range items {
		if it.ID != id {
			out = append(out, it)
		}
	}
	return out
}

func TestRestoreAsksFirst(t *testing.T) {
	fakeFetchTrash(t)
	ids := fakeRestore(t)
	m := trashVault(t, trashItems())
	m, _ = send(m, keyMsg("U"))
	if m.overlay != overlayRestore {
		t.Fatal("U did not open the restore dialog")
	}
	if v := formView(m); !strings.Contains(v, `Restore "Old Forum"?`) {
		t.Errorf("the dialog must name the item:\n%s", v)
	}
	if len(*ids) != 0 {
		t.Error("the dialog must not restore before y")
	}
}

func TestRestoreCancel(t *testing.T) {
	fakeFetchTrash(t)
	for _, k := range []tea.KeyPressMsg{keyMsg("n"), esc} {
		ids := fakeRestore(t)
		m := trashVault(t, trashItems())
		m, _ = send(m, keyMsg("U"))
		m, _ = send(m, k)
		if m.overlay != overlayNone || len(*ids) != 0 {
			t.Errorf("%s must close the dialog and restore nothing", k)
		}
	}
}

func TestRestoreRestoresItem(t *testing.T) {
	loads := fakeFetchTrash(t)
	ids := fakeRestore(t)
	trash := trashItems()
	card := trash[1].Card
	m := trashVault(t, trash)
	m = selectID(t, m, "t2")
	m.verifiedID = "t2"
	m, _ = send(m, keyMsg("U"))
	m, _ = send(m, keyMsg("y"))
	if len(*ids) != 1 || (*ids)[0] != "t2" || !m.res.running {
		t.Fatalf("y: restores = %q, want t2", *ids)
	}
	// Keys do nothing while bw runs.
	m, _ = send(m, esc)
	if m.overlay != overlayRestore {
		t.Error("esc closed the dialog while bw runs")
	}

	before := len(*loads)
	m, cmd := send(m, bwcmd.RestoreResult{ID: "t2", Epoch: m.sess.Epoch})
	if m.overlay != overlayNone || cmd == nil || len(*loads) != before+1 {
		t.Fatal("a restore that worked must close the dialog and load the trash")
	}
	if !strings.Contains(m.toast, "Restored Visa") {
		t.Errorf("toast = %q", m.toast)
	}
	if m.rowOfItem("t2") >= 0 || len(m.rawItems) != 2 {
		t.Errorf("the restored item must leave the list at once, %d items", len(m.rawItems))
	}
	if card.Number != "" {
		t.Error("the restored item must be scrubbed")
	}
	if item := m.selectedItem(); item == nil || item.ID != "t3" {
		t.Errorf("selected = %v, want the next item, t3", item)
	}
	if m.verifiedID != "" {
		t.Errorf("verifiedID = %q, want empty", m.verifiedID)
	}
	m, _ = send(m, bwcmd.TrashResult{Items: without(trashItems(), "t2"), Epoch: m.sess.Epoch, Seq: m.listSeq})
	if item := m.selectedItem(); item == nil || item.ID != "t3" {
		t.Errorf("after the reload, selected = %v, want t3", item)
	}
}

func TestRestoreAgainBeforeReload(t *testing.T) {
	fakeFetchTrash(t)
	fakeRestore(t)
	m := trashVault(t, trashItems())
	m, _ = send(m, keyMsg("U"))
	m, _ = send(m, keyMsg("y"))
	m, _ = send(m, bwcmd.RestoreResult{ID: "t1", Epoch: m.sess.Epoch})
	m, _ = send(m, keyMsg("U"))
	if m.overlay != overlayRestore || m.res.id != "t2" {
		t.Errorf("U before the reload opened the dialog for %q, want t2", m.res.id)
	}
}

func TestRestoreWithFilterSelectsNextVisible(t *testing.T) {
	fakeFetchTrash(t)
	fakeRestore(t)
	m := trashVault(t, trashItems())
	m = typeKeys(t, m, "/old")
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.list.FilterState() != list.FilterApplied || len(m.list.VisibleItems()) != 2 {
		t.Fatalf("filter: state = %v, %d matches, want 2", m.list.FilterState(), len(m.list.VisibleItems()))
	}
	first := m.selectedItem().ID
	second := ""
	for _, r := range m.list.VisibleItems() {
		if v, ok := r.(VaultItem); ok && v.ID != first {
			second = v.ID
		}
	}
	m, _ = send(m, keyMsg("U"))
	m, _ = send(m, keyMsg("y"))
	m, _ = send(m, bwcmd.RestoreResult{ID: first, Epoch: m.sess.Epoch})
	if item := m.selectedItem(); item == nil || item.ID != second {
		t.Errorf("selected = %v, want the other match, %s", item, second)
	}
}

func TestRestoreLastItemSelectsPrevious(t *testing.T) {
	fakeFetchTrash(t)
	fakeRestore(t)
	m := selectID(t, trashVault(t, trashItems()), "t3")
	m, _ = send(m, keyMsg("U"))
	m, _ = send(m, keyMsg("y"))
	m, _ = send(m, bwcmd.RestoreResult{ID: "t3", Epoch: m.sess.Epoch})
	m, _ = send(m, bwcmd.TrashResult{Items: trashItems()[:2], Epoch: m.sess.Epoch, Seq: m.listSeq})
	if item := m.selectedItem(); item == nil || item.ID != "t2" {
		t.Errorf("selected = %v, want the previous item, t2", item)
	}
}

func TestRestoreFailureShowsError(t *testing.T) {
	fakeFetchTrash(t)
	ids := fakeRestore(t)
	m := trashVault(t, trashItems())
	m, _ = send(m, keyMsg("U"))
	m, _ = send(m, keyMsg("y"))
	m, _ = send(m, bwcmd.RestoreResult{ID: "t1", Err: errors.New("Cipher is not in trash."), Epoch: m.sess.Epoch})
	if m.overlay != overlayRestore || m.res.running || !strings.Contains(formView(m), "Cipher is not in trash.") {
		t.Fatalf("a failed restore must keep the dialog and show the error:\n%s", formView(m))
	}
	if len(m.rawItems) != 3 {
		t.Error("a failed restore must keep the item in the list")
	}
	m, _ = send(m, keyMsg("y"))
	if len(*ids) != 2 {
		t.Errorf("y after an error ran %d restores in all, want 2", len(*ids))
	}
}

func TestRestoreWorksForEachType(t *testing.T) {
	fakeFetchTrash(t)
	for _, it := range trashItems() {
		ids := fakeRestore(t)
		m := selectID(t, trashVault(t, trashItems()), it.ID)
		m, _ = send(m, keyMsg("U"))
		send(m, keyMsg("y"))
		if len(*ids) != 1 || (*ids)[0] != it.ID {
			t.Errorf("type %d: restores = %q, want %s", it.Type, *ids, it.ID)
		}
	}
}

func TestRestoreAsksForMasterPassword(t *testing.T) {
	fakeFetchTrash(t)
	ids := fakeRestore(t)
	stubReprompt(t)
	m := selectID(t, trashVault(t, editItems()), "bank")
	m, _ = send(m, keyMsg("U"))
	if m.overlay != overlayReprompt {
		t.Fatal("U on an item with re-prompt must ask for the master password first")
	}
	m, _ = submitPassword(t, m, rightPassword, func(m VaultModel, _ []tea.Msg) bool { return m.overlay == overlayRestore })
	if m.overlay != overlayRestore || len(*ids) != 0 {
		t.Fatal("the right password must open the dialog, and restore nothing yet")
	}
}

func TestRestoreResultOfEndedSessionIsDropped(t *testing.T) {
	fakeFetchTrash(t)
	fakeRestore(t)
	m := trashVault(t, trashItems())
	m, _ = send(m, keyMsg("U"))
	m, _ = send(m, keyMsg("y"))
	m, cmd := send(m, bwcmd.RestoreResult{ID: "t1", Epoch: m.sess.Epoch + 1})
	if m.overlay != overlayRestore || cmd != nil || len(m.rawItems) != 3 {
		t.Error("a result of an ended session must change nothing")
	}
}

func TestRestoreDialogFitsSmallTerminals(t *testing.T) {
	fakeFetchTrash(t)
	long := bwcmd.Item{ID: "x", Type: bwcmd.ItemTypeLogin, Name: strings.Repeat("Very long name ", 10)}
	for _, size := range [][2]int{{80, 24}, {50, 15}, {30, 10}} {
		m := vaultAt(testItems(), size[0], size[1])
		m, _ = pressD(t, m)
		m, _ = send(m, bwcmd.TrashResult{Items: []bwcmd.Item{long}, Epoch: m.sess.Epoch, Seq: m.listSeq})
		m, _ = send(m, keyMsg("U"))
		if m.overlay != overlayRestore {
			t.Fatalf("%dx%d: U did not open the dialog", size[0], size[1])
		}
		out := m.ViewContent(size[0], size[1]-ui.ChromeHeight)
		assertArea(t, out, size[0], size[1]-ui.ChromeHeight)
		if size[0] >= 50 {
			// From 50x15, the whole card fits.
			cardColumns(t, out)
		}
	}
}
