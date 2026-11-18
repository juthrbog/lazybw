package screens

import (
	"errors"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/session"
	"github.com/juthrbog/lazybw/ui"
)

// trashLoad is one call of the fake fetchTrashCmd.
type trashLoad struct {
	epoch, seq uint64
}

// fakeFetchTrash replaces fetchTrashCmd for one test. The returned slice
// collects the calls. The command returns no message, so the test sends
// the TrashResult itself.
func fakeFetchTrash(t *testing.T) *[]trashLoad {
	t.Helper()
	var loads []trashLoad
	old := fetchTrashCmd
	t.Cleanup(func() { fetchTrashCmd = old })
	fetchTrashCmd = func(_ string, epoch, seq uint64) tea.Cmd {
		loads = append(loads, trashLoad{epoch: epoch, seq: seq})
		return nil
	}
	return &loads
}

// trashItems are the items of the trash in the tests.
func trashItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "t1", Type: bwcmd.ItemTypeLogin, Name: "Old Forum", DeletedDate: "2026-11-17T12:00:00Z",
			Login: &bwcmd.Login{Username: "me", Password: "forum-pw"}},
		{ID: "t2", Type: bwcmd.ItemTypeCard, Name: "Visa", DeletedDate: "2026-11-16T12:00:00Z",
			Card: &bwcmd.Card{Number: "4242424242424242"}},
		{ID: "t3", Type: bwcmd.ItemTypeSecureNote, Name: "Old Note", Notes: "old secret", DeletedDate: "2026-11-15T12:00:00Z"},
	}
}

// pressD sends D to m. It does not run the command, which loads a list
// with bw.
func pressD(t *testing.T, m VaultModel) (VaultModel, tea.Cmd) {
	t.Helper()
	return send(m, keyMsg("D"))
}

// trashVault returns a vault screen that shows items as the trash list.
func trashVault(t *testing.T, items []bwcmd.Item) VaultModel {
	t.Helper()
	m, _ := pressD(t, newTestVault(testItems()))
	m, _ = send(m, bwcmd.TrashResult{Items: items, Epoch: m.sess.Epoch, Seq: m.listSeq})
	if !m.ShowsTrash() || len(m.rawItems) != len(items) {
		t.Fatalf("trash list: ShowsTrash = %v, %d items, want %d", m.ShowsTrash(), len(m.rawItems), len(items))
	}
	return m
}

// contentText returns the content area of m at 80x24 without styles.
func contentText(m VaultModel) string {
	return ansi.Strip(m.ViewContent(80, 24-ui.ChromeHeight))
}

func TestTrashKeyShowsTrash(t *testing.T) {
	loads := fakeFetchTrash(t)
	items := []bwcmd.Item{{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Gmail", Login: &bwcmd.Login{Password: "gmail-pw"}}}
	m := newTestVault(items)
	m, cmd := pressD(t, m)
	if !m.ShowsTrash() || !m.listLoading || cmd == nil {
		t.Fatalf("D: ShowsTrash = %v, listLoading = %v, cmd = %v", m.ShowsTrash(), m.listLoading, cmd)
	}
	if len(m.rawItems) != 0 || len(m.list.Items()) != 0 {
		t.Errorf("D must drop the vault rows at once, got %d items and %d rows", len(m.rawItems), len(m.list.Items()))
	}
	if items[0].Login.Password != "" {
		t.Error("D must scrub the vault items")
	}
	if len(*loads) != 1 || (*loads)[0].seq != m.listSeq {
		t.Fatalf("trash loads = %v, want one with seq %d", *loads, m.listSeq)
	}

	m, _ = send(m, bwcmd.TrashResult{Items: trashItems(), Epoch: m.sess.Epoch, Seq: m.listSeq})
	if m.listLoading || len(m.rawItems) != 3 {
		t.Errorf("TrashResult: listLoading = %v, %d items, want false and 3", m.listLoading, len(m.rawItems))
	}
	if item := m.selectedItem(); item == nil || item.ID != "t1" {
		t.Errorf("selected = %v, want the first trash item", item)
	}
}

func TestTrashKeyGoesBackToVault(t *testing.T) {
	fakeFetchTrash(t)
	trash := trashItems()
	m := trashVault(t, trash)
	m, cmd := pressD(t, m)
	if m.ShowsTrash() || cmd == nil || !m.listLoading {
		t.Fatalf("D in the trash: ShowsTrash = %v, cmd = %v, listLoading = %v", m.ShowsTrash(), cmd, m.listLoading)
	}
	if trash[0].Login.Password != "" || trash[2].Notes != "" || len(m.rawItems) != 0 {
		t.Error("D must scrub and drop the trash items")
	}
	m, _ = send(m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch, Seq: m.listSeq})
	if len(m.rawItems) != 3 || m.listLoading {
		t.Errorf("ItemsResult: %d items, listLoading = %v", len(m.rawItems), m.listLoading)
	}
}

func TestTrashSwitchClearsFilter(t *testing.T) {
	fakeFetchTrash(t)
	m := newTestVault(testItems())
	for _, msg := range []tea.KeyPressMsg{keyMsg("/"), keyMsg("g"), keyMsg("m"), {Code: tea.KeyEnter}} {
		m, _ = send(m, msg)
	}
	if m.list.FilterState() != list.FilterApplied {
		t.Fatalf("filter state = %v, want FilterApplied", m.list.FilterState())
	}
	m.filterFrom = rowKey{id: "2"}
	m, _ = pressD(t, m)
	if m.list.FilterState() != list.Unfiltered || m.filterFrom != (rowKey{}) {
		t.Errorf("D: filter state = %v, filterFrom = %+v, want no filter", m.list.FilterState(), m.filterFrom)
	}
}

func TestTrashSwitchDropsPendingSelect(t *testing.T) {
	fakeFetchTrash(t)
	m := newTestVault(testItems())
	m.selectID, m.loadingID = "2", "3"
	m, _ = pressD(t, m)
	if m.selectID != "" || m.loadingID != "" {
		t.Errorf("D: selectID = %q, loadingID = %q, want both empty", m.selectID, m.loadingID)
	}
}

func TestVaultItemsInTrashAreDropped(t *testing.T) {
	fakeFetchTrash(t)
	m := trashVault(t, trashItems())
	vault := testItems()
	vault[0].Login.Password = "gmail-pw"
	m, cmd := send(m, bwcmd.ItemsResult{Items: vault, Epoch: m.sess.Epoch, Seq: m.listSeq})
	if cmd != nil || len(m.rawItems) != 3 || m.rawItems[0].ID != "t1" {
		t.Errorf("a vault list in the trash must change nothing, rows = %d, cmd = %v", len(m.rawItems), cmd)
	}
	if vault[0].Login.Password != "" {
		t.Error("a dropped vault list must be scrubbed")
	}
}

func TestTrashItemsInVaultAreDropped(t *testing.T) {
	m := newTestVault(testItems())
	trash := trashItems()
	m, cmd := send(m, bwcmd.TrashResult{Items: trash, Epoch: m.sess.Epoch, Seq: m.listSeq})
	if cmd != nil || len(m.rawItems) != 3 || m.rawItems[0].ID != "1" {
		t.Errorf("a trash list in the vault must change nothing, cmd = %v", cmd)
	}
	if trash[0].Login.Password != "" || trash[2].Notes != "" {
		t.Error("a dropped trash list must be scrubbed")
	}
}

func TestTrashLoadFailureShowsToast(t *testing.T) {
	fakeFetchTrash(t)
	m, _ := pressD(t, newTestVault(testItems()))
	m, cmd := send(m, bwcmd.TrashResult{Err: errors.New("bw broke"), Epoch: m.sess.Epoch, Seq: m.listSeq})
	if cmd == nil || m.listLoading || !m.listErr {
		t.Fatalf("a failed load: cmd = %v, listLoading = %v, listErr = %v", cmd, m.listLoading, m.listErr)
	}
	if !strings.Contains(m.toast, "Load failed") {
		t.Errorf("toast = %q, want Load failed", m.toast)
	}
	if v := contentText(m); !strings.Contains(v, "Could not load the trash") {
		t.Errorf("the content must say that the load failed:\n%s", v)
	}
	m, _ = send(m, toastExpiredMsg{gen: m.toastGen})
	if m.toast != "" {
		t.Fatal("the toast did not expire")
	}
	if v := contentText(m); !strings.Contains(v, "Could not load the trash") {
		t.Errorf("the text must stay after the toast:\n%s", v)
	}
}

func TestVaultLoadFailureAfterTrash(t *testing.T) {
	fakeFetchTrash(t)
	m := trashVault(t, trashItems())
	m, _ = pressD(t, m)
	m, _ = send(m, bwcmd.ItemsResult{Err: errors.New("bw broke"), Epoch: m.sess.Epoch, Seq: m.listSeq})
	v := contentText(m)
	if !strings.Contains(v, "Could not load the vault. Press r to try again.") || strings.Contains(v, "No items.") {
		t.Errorf("the content must say that the vault load failed:\n%s", v)
	}
	if m.listLoading {
		t.Error("listLoading must end")
	}
	m, _ = send(m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch, Seq: m.listSeq})
	if v := contentText(m); strings.Contains(v, "Could not load") || !strings.Contains(v, "Gmail") {
		t.Errorf("a good load must show the rows:\n%s", v)
	}
}

func TestVaultLoadFromBeforeTrashIsDropped(t *testing.T) {
	fakeFetchTrash(t)
	fakeDelete(t)
	m := selectID(t, newTestVault(editItems()), "visa")
	m, _ = send(m, keyMsg("d"))
	m, _ = send(m, keyMsg("y"))
	// The delete starts a vault load. It still runs when D shows the trash.
	m, _ = send(m, bwcmd.DeleteResult{ID: "visa", Epoch: m.sess.Epoch})
	deleteLoad := m.listSeq
	m, _ = pressD(t, m)
	m, _ = pressD(t, m)

	old := editItems()
	m, cmd := send(m, bwcmd.ItemsResult{Items: old, Epoch: m.sess.Epoch, Seq: deleteLoad})
	if cmd != nil || len(m.rawItems) != 0 || !m.listLoading {
		t.Errorf("a vault load from before D must change nothing: %d rows, listLoading = %v", len(m.rawItems), m.listLoading)
	}
	if old[1].Card.Number != "" || old[3].Login.Password != "" {
		t.Error("a dropped vault load must be scrubbed")
	}
	m, _ = send(m, bwcmd.ItemsResult{Items: editItems(), Epoch: m.sess.Epoch, Seq: m.listSeq})
	if len(m.rawItems) != len(editItems()) || m.listLoading {
		t.Errorf("the newest vault load must fill the rows: %d rows, listLoading = %v", len(m.rawItems), m.listLoading)
	}
}

func TestSyncFailureReloadsFailedList(t *testing.T) {
	loads := fakeFetchTrash(t)
	m, _ := pressD(t, newTestVault(testItems()))
	m, _ = send(m, bwcmd.TrashResult{Err: errors.New("bw broke"), Epoch: m.sess.Epoch, Seq: m.listSeq})
	before := len(*loads)
	m, _ = send(m, keyMsg("r"))
	m, cmd := send(m, bwcmd.SyncResult{Err: errors.New("offline"), Epoch: m.sess.Epoch})
	if cmd == nil || len(*loads) != before+1 || (*loads)[before].seq != m.listSeq {
		t.Fatalf("a failed sync after a failed load must load the trash again: loads = %v", *loads)
	}
	if v := contentText(m); !strings.Contains(v, "Loading trash…") {
		t.Errorf("the content must show the load in place of the error:\n%s", v)
	}
	m, _ = send(m, bwcmd.TrashResult{Items: trashItems(), Epoch: m.sess.Epoch, Seq: m.listSeq})
	if len(m.rawItems) != 3 || m.listErr {
		t.Errorf("the load must fill the rows: %d rows, listErr = %v", len(m.rawItems), m.listErr)
	}

	// Back in the vault, the same holds for a failed vault load.
	m, _ = pressD(t, m)
	m, _ = send(m, bwcmd.ItemsResult{Err: errors.New("bw broke"), Epoch: m.sess.Epoch, Seq: m.listSeq})
	seq := m.listSeq
	m, _ = send(m, keyMsg("r"))
	m, cmd = send(m, bwcmd.SyncResult{Err: errors.New("offline"), Epoch: m.sess.Epoch})
	if cmd == nil || m.listSeq != seq+1 || !m.listLoading {
		t.Errorf("a failed sync after a failed vault load must load the vault again: seq = %d, listLoading = %v", m.listSeq, m.listLoading)
	}
}

func TestSyncFailureWithoutListErrorLoadsNothing(t *testing.T) {
	loads := fakeFetchTrash(t)
	m := trashVault(t, trashItems())
	before, seq := len(*loads), m.listSeq
	m, _ = send(m, keyMsg("r"))
	m, _ = send(m, bwcmd.SyncResult{Err: errors.New("offline"), Epoch: m.sess.Epoch})
	if len(*loads) != before || m.listSeq != seq || m.listLoading {
		t.Errorf("a failed sync must not load the list again: loads = %v, listLoading = %v", *loads, m.listLoading)
	}
}

func TestTrashLoadingShowsInFooter(t *testing.T) {
	fakeFetchTrash(t)
	m, _ := pressD(t, newTestVault(testItems()))
	if _, status := m.FooterContent(); !strings.Contains(ansi.Strip(status), "Loading trash…") {
		t.Errorf("status = %q, want Loading trash…", ansi.Strip(status))
	}
	m, cmd := send(m, spinner.TickMsg{ID: m.syncSpinner.ID()})
	if cmd == nil {
		t.Error("the spinner must tick while the list loads")
	}
	m, _ = send(m, bwcmd.TrashResult{Epoch: m.sess.Epoch, Seq: m.listSeq})
	if _, cmd := send(m, spinner.TickMsg{ID: m.syncSpinner.ID()}); cmd != nil {
		t.Error("the spinner must stop after the load")
	}
}

func TestTrashLoadingShowsInContent(t *testing.T) {
	fakeFetchTrash(t)
	m, _ := pressD(t, newTestVault(testItems()))
	if v := contentText(m); !strings.Contains(v, "Loading trash…") || strings.Contains(v, "No items.") {
		t.Errorf("the content must say that the trash loads:\n%s", v)
	}
	m, _ = pressD(t, m)
	if v := contentText(m); !strings.Contains(v, "Loading vault…") {
		t.Errorf("the content must say that the vault loads:\n%s", v)
	}
}

func TestEmptyTrashText(t *testing.T) {
	fakeFetchTrash(t)
	m := trashVault(t, nil)
	if v := contentText(m); !strings.Contains(v, "The trash is empty.") || strings.Contains(v, "No items.") {
		t.Errorf("an empty trash must say so:\n%s", v)
	}
}

func TestStaleTrashResultIsDropped(t *testing.T) {
	loads := fakeFetchTrash(t)
	m := newTestVault(testItems())
	m, _ = pressD(t, m)
	m, _ = pressD(t, m)
	m, _ = pressD(t, m)
	if len(*loads) != 2 || (*loads)[0].seq == (*loads)[1].seq {
		t.Fatalf("trash loads = %v, want 2 with different seqs", *loads)
	}
	first, last := (*loads)[0].seq, (*loads)[1].seq

	old := trashItems()
	m, _ = send(m, bwcmd.TrashResult{Items: old, Epoch: m.sess.Epoch, Seq: first})
	if len(m.rawItems) != 0 || !m.listLoading {
		t.Errorf("an older trash load must change nothing: %d rows, listLoading = %v", len(m.rawItems), m.listLoading)
	}
	if old[0].Login.Password != "" {
		t.Error("an older trash load must be scrubbed")
	}
	m, _ = send(m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch, Seq: m.listSeq})
	if len(m.rawItems) != 0 {
		t.Error("a vault load in the trash must be dropped")
	}
	m, _ = send(m, bwcmd.TrashResult{Items: trashItems(), Epoch: m.sess.Epoch, Seq: last})
	if len(m.rawItems) != 3 || m.listLoading {
		t.Errorf("the newest trash load must fill the rows: %d rows, listLoading = %v", len(m.rawItems), m.listLoading)
	}
}

func TestTrashResultAfterRestoreIsDropped(t *testing.T) {
	loads := fakeFetchTrash(t)
	fakeRestore(t)
	m := trashVault(t, trashItems())
	// r starts a sync, and its result starts a trash load.
	m, _ = send(m, keyMsg("r"))
	m, _ = send(m, bwcmd.SyncResult{Epoch: m.sess.Epoch})
	syncLoad := (*loads)[len(*loads)-1].seq

	m, _ = send(m, keyMsg("U"))
	m, _ = send(m, keyMsg("y"))
	m, _ = send(m, bwcmd.RestoreResult{ID: "t1", Epoch: m.sess.Epoch})
	if len(m.rawItems) != 2 {
		t.Fatalf("after the restore, %d rows, want 2", len(m.rawItems))
	}
	m, _ = send(m, bwcmd.TrashResult{Items: trashItems(), Epoch: m.sess.Epoch, Seq: syncLoad})
	if len(m.rawItems) != 2 || m.rowOfItem("t1") >= 0 {
		t.Error("a trash load from before the restore must not bring the item back")
	}
}

func TestTrashSwitchWithGroupHeader(t *testing.T) {
	fakeFetchTrash(t)
	m := newTestVault(groupTestItems())
	m = press(t, m, ctrlG)
	if _, ok := m.list.SelectedItem().(GroupHeaderItem); !ok {
		m = selectRow(t, m, "group:github")
	}
	m.drawerScroll = 3
	m.verifiedID = "x"
	m, _ = pressD(t, m)
	if m.drawerScroll != 0 || m.verifiedID != "" {
		t.Errorf("D: drawerScroll = %d, verifiedID = %q, want 0 and empty", m.drawerScroll, m.verifiedID)
	}
}

func TestTrashSwitchDropsVerified(t *testing.T) {
	fakeFetchTrash(t)
	stubReprompt(t)
	m := selectID(t, newTestVault(editItems()), "bank")
	m.verifiedID = "bank"
	m, _ = pressD(t, m)
	if m.verifiedID != "" {
		t.Fatal("D must drop the verified item")
	}
	m, _ = send(m, bwcmd.TrashResult{Items: editItems()[3:], Epoch: m.sess.Epoch, Seq: m.listSeq})
	m, _ = send(m, keyMsg("c"))
	if m.overlay != overlayReprompt {
		t.Error("c on a re-prompt item in the trash must ask for the master password again")
	}
}

func TestTrashSwitchDropsEditLoad(t *testing.T) {
	fakeFetchTrash(t)
	gets, _ := fakeEdit(t)
	m := selectID(t, newTestVault(editItems()), "gh")
	m, _ = send(m, keyMsg("e"))
	if len(*gets) != 1 || m.loadingID != "gh" {
		t.Fatalf("e: gets = %q, loadingID = %q", *gets, m.loadingID)
	}
	m, _ = pressD(t, m)
	m, _ = send(m, bwcmd.TrashResult{Items: editItems(), Epoch: m.sess.Epoch, Seq: m.listSeq})
	m = selectID(t, m, "gh")
	m, cmd := send(m, bwcmd.GetItemResult{ID: "gh", Item: map[string]any{"id": "gh", "type": 1}, Epoch: m.sess.Epoch})
	if m.formOpen || cmd != nil {
		t.Error("an edit load from before D must open no form")
	}
}

func TestTrashSwitchKeepsCopyToast(t *testing.T) {
	fakeFetchTrash(t)
	m := newTestVault(testItems())
	m, _ = send(m, session.CopiedMsg{Field: session.CopyFieldPassword})
	m, _ = pressD(t, m)
	_, status := m.FooterContent()
	if s := ansi.Strip(status); !strings.Contains(s, "sent to clipboard") {
		t.Errorf("status = %q, want the copy toast", s)
	}
	m, _ = send(m, session.ClipboardClearedMsg{})
	_, status = m.FooterContent()
	if s := ansi.Strip(status); !strings.Contains(s, "Loading trash…") {
		t.Errorf("status = %q, want Loading trash…", s)
	}
}

func TestTrashIgnoresItemKeys(t *testing.T) {
	fakeFetchTrash(t)
	ids := fakeDelete(t)
	gets, _ := fakeEdit(t)
	for _, k := range []tea.KeyPressMsg{keyMsg("n"), keyMsg("e"), keyMsg("d"), {Code: tea.KeyDelete}} {
		m := trashVault(t, trashItems())
		m, cmd := send(m, k)
		if m.overlay != overlayNone || m.formOpen || cmd != nil {
			t.Errorf("%s in the trash: overlay = %v, formOpen = %v, cmd = %v", k, m.overlay, m.formOpen, cmd)
		}
	}
	if len(*ids) != 0 || len(*gets) != 0 {
		t.Errorf("deletes = %q, gets = %q, want none", *ids, *gets)
	}
}

func TestRestoreKeyInVaultDoesNothing(t *testing.T) {
	ids := fakeRestore(t)
	m, cmd := send(newTestVault(testItems()), keyMsg("U"))
	if m.overlay != overlayNone || cmd != nil || len(*ids) != 0 {
		t.Errorf("U in the vault: overlay = %v, cmd = %v", m.overlay, cmd)
	}
}

func TestRestoreOnGroupHeaderDoesNothing(t *testing.T) {
	fakeFetchTrash(t)
	m := trashVault(t, groupTestItems())
	m = press(t, m, ctrlG)
	if _, ok := m.list.SelectedItem().(GroupHeaderItem); !ok {
		m = selectRow(t, m, "group:github")
	}
	m, cmd := send(m, keyMsg("U"))
	if m.overlay != overlayNone || cmd != nil {
		t.Errorf("U on a group header: overlay = %v, cmd = %v", m.overlay, cmd)
	}
}

func TestSyncInTrashLoadsTrash(t *testing.T) {
	loads := fakeFetchTrash(t)
	m := trashVault(t, trashItems())
	before := len(*loads)
	m, _ = send(m, keyMsg("r"))
	m, cmd := send(m, bwcmd.SyncResult{Epoch: m.sess.Epoch})
	if cmd == nil || len(*loads) != before+1 || (*loads)[before].seq != m.listSeq {
		t.Errorf("a sync in the trash must load the trash with a new seq: loads = %v", *loads)
	}
}

func TestCopyWorksInTrash(t *testing.T) {
	fakeFetchTrash(t)
	m := trashVault(t, trashItems())
	_, cmd := send(m, keyMsg("c"))
	values, _ := clipboardWrites(t, cmd, "forum-pw")
	if len(values) != 1 {
		t.Errorf("c in the trash wrote %q, want the password", values)
	}
}

func TestFooterShowsTrashHints(t *testing.T) {
	fakeFetchTrash(t)
	m := trashVault(t, trashItems())
	hints, _ := m.FooterContent()
	for _, want := range [][2]string{{"U", "restore"}, {"D", "vault"}} {
		if !hasHint(hints, want[0], want[1]) {
			t.Errorf("trash hints %v have no %s %s", hints, want[0], want[1])
		}
	}
	for _, bad := range [][2]string{{"n", "new"}, {"e", "edit"}, {"d", "trash"}} {
		if hasHint(hints, bad[0], bad[1]) {
			t.Errorf("trash hints %v must not have %s %s", hints, bad[0], bad[1])
		}
	}
	for _, status := range []string{"⠋ Loading trash…", "synced 5m ago"} {
		for _, width := range []int{50, 80} {
			out := ansi.Strip(ui.RenderFooter(hints, status, width))
			if !strings.Contains(out, "U restore") || !strings.Contains(out, "D vault") {
				t.Errorf("footer at %d with %q = %q, want U restore and D vault", width, status, out)
			}
		}
	}

	vaultHints, _ := newTestVault(testItems()).FooterContent()
	if !hasHint(vaultHints, "D", "show trash") {
		t.Errorf("vault hints %v have no D show trash", vaultHints)
	}

	for _, msg := range []tea.KeyPressMsg{keyMsg("/"), keyMsg("o"), {Code: tea.KeyEnter}} {
		m, _ = send(m, msg)
	}
	if m.list.FilterState() != list.FilterApplied {
		t.Fatalf("filter state = %v, want FilterApplied", m.list.FilterState())
	}
	hints, _ = m.FooterContent()
	if !hasHint(hints, "U", "restore") || !hasHint(hints, "D", "vault") {
		t.Errorf("filtered trash hints %v have no U restore and D vault", hints)
	}
}

func TestTrashHelpLeavesOutItemKeys(t *testing.T) {
	fakeFetchTrash(t)
	keys := func(m VaultModel) map[string]bool {
		got := make(map[string]bool)
		for _, b := range m.helpBindings() {
			got[b.Key] = true
		}
		return got
	}
	m := trashVault(t, trashItems())
	m, _ = send(m, keyMsg("?"))
	if m.overlay != overlayHelp {
		t.Fatal("? did not open the help")
	}
	got := keys(m)
	for _, k := range []string{"n", "e", "d"} {
		if got[k] {
			t.Errorf("the trash help must not list %s", k)
		}
	}
	for _, k := range []string{"D", "U", "c"} {
		if !got[k] {
			t.Errorf("the trash help must list %s", k)
		}
	}
	vault := keys(newTestVault(testItems()))
	for _, k := range []string{"n", "e", "d", "D", "U"} {
		if !vault[k] {
			t.Errorf("the vault help must list %s", k)
		}
	}
}

func TestClearDropsTrashState(t *testing.T) {
	fakeFetchTrash(t)
	fakeRestore(t)
	trash := trashItems()
	m := trashVault(t, trash)
	m, _ = send(m, keyMsg("U"))
	if m.overlay != overlayRestore {
		t.Fatal("U did not open the restore dialog")
	}
	m.listLoading = true
	m.Clear()
	if trash[0].Login.Password != "" || trash[2].Notes != "" || m.rawItems != nil {
		t.Error("Clear must scrub the trash items")
	}
	if m.overlay != overlayNone || m.res != (restoreState{}) || m.trash || m.listLoading || m.listErr {
		t.Errorf("Clear: overlay = %v, res = %+v, trash = %v, listLoading = %v", m.overlay, m.res, m.trash, m.listLoading)
	}
}
