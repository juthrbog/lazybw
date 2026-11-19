package screens

import (
	"errors"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
)

// applyKey sends a key to m and feeds back the filter results that the
// list computes in a command.
func applyKey(t *testing.T, m VaultModel, msg tea.KeyPressMsg) VaultModel {
	t.Helper()
	updated, cmd := m.Update(msg)
	m = updated.(VaultModel)
	for _, sub := range filterMsgs(cmd) {
		updated, _ = m.Update(sub)
		m = updated.(VaultModel)
	}
	return m
}

// filterMsgs runs cmd and returns the list.FilterMatchesMsg values that it
// produces. Commands that do not return at once, such as timers, are
// skipped.
func filterMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-ch:
	case <-time.After(50 * time.Millisecond):
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, filterMsgs(c)...)
		}
		return out
	}
	if fm, ok := msg.(list.FilterMatchesMsg); ok {
		return []tea.Msg{fm}
	}
	return nil
}

func refreshItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "a", Type: bwcmd.ItemTypeLogin, Name: "Alpha", Login: &bwcmd.Login{Totp: "JBSWY3DPEHPK3PXP"}},
		{ID: "b", Type: bwcmd.ItemTypeLogin, Name: "Bravo", Login: &bwcmd.Login{Totp: "GEZDGNBVGY3TQOJQ"}},
		{ID: "c", Type: bwcmd.ItemTypeLogin, Name: "Charlie (Work)", Login: &bwcmd.Login{Username: "c"}},
		{ID: "d", Type: bwcmd.ItemTypeLogin, Name: "Charlie (Home)", Login: &bwcmd.Login{Username: "d"}},
	}
}

func TestRefreshWithFilterAndGroupingKeepsState(t *testing.T) {
	m := newTestVault(refreshItems())
	m = applyKey(t, m, tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	for _, r := range "/a" {
		m = applyKey(t, m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m = applyKey(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.list.FilterState() != list.FilterApplied {
		t.Fatalf("filter state = %v, want applied", m.list.FilterState())
	}
	for item := m.selectedItem(); item == nil || item.ID != "b"; item = m.selectedItem() {
		before := m.list.Index()
		m = applyKey(t, m, tea.KeyPressMsg{Code: 'j', Text: "j"})
		if m.list.Index() == before {
			t.Fatal("Bravo is not in the filtered list")
		}
	}
	if m.totpCode == "" {
		t.Fatal("Bravo must show a TOTP code before the refresh")
	}

	// The sync returns a new matching item in front of Bravo.
	fresh := append([]bwcmd.Item{{ID: "z", Type: bwcmd.ItemTypeLogin, Name: "Aardvark"}}, refreshItems()...)
	updated, _ := m.Update(bwcmd.ItemsResult{Items: fresh})
	m = updated.(VaultModel)

	if item := m.selectedItem(); item == nil || item.ID != "b" {
		t.Errorf("selected = %+v, want Bravo", item)
	}
	if m.totpCode == "" {
		t.Error("Bravo must still show a TOTP code after the refresh")
	}
	if m.list.FilterState() != list.FilterApplied || m.list.FilterValue() != "a" {
		t.Errorf("filter = %v %q, want the applied filter a", m.list.FilterState(), m.list.FilterValue())
	}
	if !m.groups.enabled {
		t.Error("grouping must stay on after the refresh")
	}
}

func TestRefreshShorterListClampsCursor(t *testing.T) {
	m := newTestVault(testItems())
	m.list.Select(2)
	updated, _ := m.Update(bwcmd.ItemsResult{Items: testItems()[:2]})
	m = updated.(VaultModel)
	if m.selectedItem() == nil {
		t.Error("a shorter list must still have a selected row")
	}
}

func TestSyncLastsUntilListLoads(t *testing.T) {
	m := newTestVault(testItems())
	m, _ = send(m, keyMsg("r"))
	m, _ = send(m, bwcmd.SyncResult{})
	if !m.syncing || m.toast != "" {
		t.Fatalf("after the sync: syncing = %v, toast = %q, want the sync to go on", m.syncing, m.toast)
	}
	if got := m.footerStatus(); !strings.Contains(got, "Syncing") {
		t.Errorf("footer = %q, want Syncing until the list loads", got)
	}
	m, _ = send(m, bwcmd.ItemsResult{Items: testItems(), Seq: m.listSeq})
	if m.syncing || !strings.Contains(m.toast, "Vault synced") {
		t.Errorf("after the load: syncing = %v, toast = %q, want the sync toast", m.syncing, m.toast)
	}
}

func TestSyncUsesNewRepromptSetting(t *testing.T) {
	m := newTestVault(testItems())
	id := m.selectedItem().ID
	m, _ = send(m, keyMsg("r"))
	m, _ = send(m, bwcmd.SyncResult{})
	items := testItems()
	for i := range items {
		if items[i].ID == id {
			items[i].Reprompt = bwcmd.RepromptPassword
		}
	}
	m, _ = send(m, bwcmd.ItemsResult{Items: items, Seq: m.listSeq})
	m, _ = send(m, keyMsg("d"))
	if m.overlay != overlayReprompt {
		t.Errorf("overlay = %v, want the password prompt after the synced list", m.overlay)
	}
}

func TestSyncEndsOnFailedListLoad(t *testing.T) {
	m := newTestVault(testItems())
	m, _ = send(m, keyMsg("r"))
	m, _ = send(m, bwcmd.SyncResult{})
	m, _ = send(m, bwcmd.ItemsResult{Err: errors.New("boom"), Seq: m.listSeq})
	if m.syncing || !m.toastIsError || strings.Contains(m.toast, "Vault synced") {
		t.Errorf("syncing = %v, toast = %q, want the load error", m.syncing, m.toast)
	}
}

func TestSyncIgnoresStaleListLoad(t *testing.T) {
	m := newTestVault(testItems())
	stale := m.listSeq
	m, _ = send(m, keyMsg("r"))
	m, _ = send(m, bwcmd.SyncResult{})
	m, _ = send(m, bwcmd.ItemsResult{Items: testItems(), Seq: stale})
	if !m.syncing {
		t.Error("a load from before the sync must not end the sync")
	}
}

func TestListLoadDuringSyncDoesNotEndIt(t *testing.T) {
	m := newTestVault(testItems())
	m, _ = send(m, keyMsg("r"))
	// A list load ends while bw sync still runs.
	m, _ = send(m, bwcmd.ItemsResult{Items: testItems(), Seq: m.listSeq})
	if !m.syncing || m.toast != "" {
		t.Errorf("syncing = %v, toast = %q, want the sync to go on", m.syncing, m.toast)
	}
}

func TestClearDropsListRows(t *testing.T) {
	m := newTestVault(testItems())
	m.Clear()
	// The rows hold copies of Notes that ZeroItems cannot reach.
	if n := len(m.list.Items()); n != 0 {
		t.Errorf("list has %d rows after Clear, want 0", n)
	}
}

func TestClearOnZeroVaultModel(t *testing.T) {
	var m VaultModel
	m.Clear() // must not panic
}
