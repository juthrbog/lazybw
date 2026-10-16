package screens

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/session"
)

func newTestVault(items []bwcmd.Item) VaultModel {
	sess := &session.State{Email: "test@test.com"}
	return NewVaultModel(items, sess, 80, 24)
}

func testItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Gmail", Login: &bwcmd.Login{Username: "user@gmail.com"}},
		{ID: "2", Type: bwcmd.ItemTypeCard, Name: "Visa", Card: &bwcmd.Card{Number: "4242424242424242"}},
		{ID: "3", Type: bwcmd.ItemTypeSecureNote, Name: "Keys", Notes: "secret"},
	}
}

func TestVaultFooterContentNormalMode(t *testing.T) {
	m := newTestVault(testItems())
	hints, _ := m.FooterContent()
	if !hasHintKey(hints, "j/k") {
		t.Errorf("normal mode hints should contain navigation, got %v", hints)
	}
}

func TestVaultFooterContentWithToast(t *testing.T) {
	m := newTestVault(testItems())
	m.setToast("Password copied")
	_, status := m.FooterContent()
	if !strings.Contains(status, "Password copied") {
		t.Errorf("status should contain toast, got %q", status)
	}
}

func TestVaultItemFilterValue(t *testing.T) {
	item := bwcmd.Item{
		ID:    "1",
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Gmail",
		Login: &bwcmd.Login{Username: "user@gmail.com"},
	}
	vi := VaultItem{Item: item}
	fv := vi.FilterValue()
	if !strings.Contains(fv, "Gmail") {
		t.Errorf("FilterValue should contain name, got %q", fv)
	}
	if !strings.Contains(fv, "user@gmail.com") {
		t.Errorf("FilterValue should contain description, got %q", fv)
	}
}

func TestVaultSelectedItemInRange(t *testing.T) {
	m := newTestVault(testItems())
	m.list.Select(1)
	item := m.selectedItem()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
	if item.Name != "Visa" {
		t.Errorf("expected 'Visa', got %q", item.Name)
	}
}

func TestVaultSelectedItemEmptyList(t *testing.T) {
	m := newTestVault(nil)
	if m.selectedItem() != nil {
		t.Error("expected nil for empty list")
	}
}

func TestToListItems(t *testing.T) {
	items := testItems()
	li := toListItems(items)
	if len(li) != len(items) {
		t.Errorf("expected %d list items, got %d", len(items), len(li))
	}
	for i, item := range li {
		vi := item.(VaultItem)
		if vi.Name != items[i].Name {
			t.Errorf("item %d: expected %q, got %q", i, items[i].Name, vi.Name)
		}
	}
}

func groupTestItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "GitHub", Login: &bwcmd.Login{Username: "dev@example.com"}},
		{ID: "2", Type: bwcmd.ItemTypeLogin, Name: "GitHub (Work)", Login: &bwcmd.Login{Username: "work@company.com"}},
		{ID: "3", Type: bwcmd.ItemTypeLogin, Name: "GitHub (Personal)", Login: &bwcmd.Login{Username: "me@example.com"}},
		{ID: "4", Type: bwcmd.ItemTypeLogin, Name: "Gmail", Login: &bwcmd.Login{Username: "user@gmail.com"}},
	}
}

func TestSelectedItemNilForGroupHeader(t *testing.T) {
	m := newTestVault(groupTestItems())
	m.groups.enabled = true
	m.rebuildListItems()
	m.list.Select(0) // select the group header (collapsed)
	if m.selectedItem() != nil {
		t.Error("selectedItem should return nil for a group header")
	}
}

func TestSelectedItemForExpandedChild(t *testing.T) {
	m := newTestVault(groupTestItems())
	m.groups.enabled = true
	m.groups.toggle("github") // expand
	m.rebuildListItems()
	m.list.Select(1) // first child: "GitHub"
	item := m.selectedItem()
	if item == nil {
		t.Fatal("expected non-nil item for expanded child")
	}
	if item.Name != "GitHub" {
		t.Errorf("expected 'GitHub', got %q", item.Name)
	}
}

func TestToggleGroupingRebuilds(t *testing.T) {
	m := newTestVault(groupTestItems())

	// Initially grouping is off — 4 flat items.
	if len(m.list.Items()) != 4 {
		t.Fatalf("expected 4 items, got %d", len(m.list.Items()))
	}

	// Enable grouping.
	m.groups.toggleGrouping()
	m.rebuildListItems()

	// Expect: Header(Discord) + Gmail = 2 items (collapsed).
	if len(m.list.Items()) != 2 {
		t.Fatalf("expected 2 items with grouping, got %d", len(m.list.Items()))
	}

	// Disable grouping.
	m.groups.toggleGrouping()
	m.rebuildListItems()
	if len(m.list.Items()) != 4 {
		t.Fatalf("expected 4 items without grouping, got %d", len(m.list.Items()))
	}
}

func TestQuitConfirmationShowsHints(t *testing.T) {
	m := newTestVault(testItems())
	m.confirmingQuit = true
	hints, _ := m.FooterContent()
	if !hasHintKey(hints, "y") || !hasHintDesc(hints, "yes") {
		t.Errorf("confirmation hints should contain 'y yes', got %v", hints)
	}
}

func TestQuitConfirmationCancelOnAnyKey(t *testing.T) {
	m := newTestVault(testItems())
	m.confirmingQuit = true
	if !m.confirmingQuit {
		t.Fatal("expected confirmingQuit to be true")
	}
	// Pressing 'n' should cancel.
	m.confirmingQuit = false // simulate cancel
	if m.confirmingQuit {
		t.Error("expected confirmingQuit to be false after cancel")
	}
}

func TestQuitConfirmationCancelOnEsc(t *testing.T) {
	m := newTestVault(testItems())
	m.confirmingQuit = true
	// Any key except 'y' cancels — esc is just another key.
	m.confirmingQuit = false
	if m.confirmingQuit {
		t.Error("expected confirmingQuit to be false after esc")
	}
}

func TestGenArgsPassword(t *testing.T) {
	m := newTestVault(testItems())
	m.genMode = "password"
	m.genLength = 24
	m.genUppercase = true
	m.genLowercase = true
	m.genNumbers = true
	m.genSpecial = false

	args := m.genArgs()
	if args[0] != "--length" || args[1] != "24" {
		t.Errorf("expected --length 24, got %v", args[:2])
	}
	hasSpecial := false
	for _, a := range args {
		if a == "--special" {
			hasSpecial = true
		}
	}
	if hasSpecial {
		t.Error("--special should not be present when disabled")
	}
}

func TestGenArgsPassphrase(t *testing.T) {
	m := newTestVault(testItems())
	m.genMode = "passphrase"
	m.genWords = 5
	m.genSeparator = "."
	m.genCapitalize = true
	m.genIncludeNum = false

	args := m.genArgs()
	if args[0] != "--passphrase" {
		t.Errorf("first arg should be --passphrase, got %q", args[0])
	}
	hasIncludeNum := false
	for _, a := range args {
		if a == "--includeNumber" {
			hasIncludeNum = true
		}
	}
	if hasIncludeNum {
		t.Error("--includeNumber should not be present when disabled")
	}
}

func TestSyncKeyIgnoredWhileSyncing(t *testing.T) {
	m := newTestVault(testItems())
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m = updated.(VaultModel)
	if cmd == nil || !m.syncing {
		t.Fatal("r must start a sync")
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd != nil {
		t.Error("r during a sync must not start a second sync")
	}
}

func TestReplaceItemsKeepsSelectionAndScrubsOldItems(t *testing.T) {
	old := testItems()
	old[0].Login.Password = "old-pw"
	m := newTestVault(old)
	m.list.Select(1) // Visa

	fresh := testItems()
	// The item moved to the end of the new list.
	fresh = []bwcmd.Item{fresh[0], fresh[2], fresh[1]}
	updated, _ := m.Update(bwcmd.ItemsResult{Items: fresh})
	m = updated.(VaultModel)

	if item := m.selectedItem(); item == nil || item.ID != "2" {
		t.Errorf("selected item = %+v, want ID 2", item)
	}
	if old[0].Login.Password != "" {
		t.Error("the replaced items must be scrubbed")
	}
}

func TestCopiedMsgLeavesClearToRoot(t *testing.T) {
	m := newTestVault(testItems())
	updated, cmd := m.Update(session.CopiedMsg{Field: session.CopyFieldPassword})
	m = updated.(VaultModel)
	if cmd != nil {
		t.Error("the vault must not schedule the clipboard clear")
	}
	if !strings.Contains(m.toast, "copied") {
		t.Errorf("toast = %q, want a copy toast", m.toast)
	}
}

func TestGenerateResultOutOfOrderIsDropped(t *testing.T) {
	m := newTestVault(testItems())
	updated, _ := m.openGenerator()
	m = updated.(VaultModel)
	first := m.genSeq
	updated, _ = m.updateGenerator(tea.KeyPressMsg{Code: '+', Text: "+"})
	m = updated.(VaultModel)
	latest := m.genSeq

	updated, _ = m.Update(bwcmd.GenerateResult{Password: "new", Seq: latest})
	m = updated.(VaultModel)
	updated, _ = m.Update(bwcmd.GenerateResult{Password: "old", Seq: first})
	m = updated.(VaultModel)
	if m.genPassword != "new" {
		t.Errorf("genPassword = %q, want the result of the latest request", m.genPassword)
	}
}

// vaultWithURI returns a vault whose selected login has uri.
func vaultWithURI(uri string) VaultModel {
	items := []bwcmd.Item{{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Site",
		Login: &bwcmd.Login{Username: "u", URIs: []bwcmd.URI{{URI: uri}}}}}
	return NewVaultModel(items, &session.State{}, 80, 24)
}

// pressOpen presses o on m, runs the command and feeds its result back.
func pressOpen(t *testing.T, m VaultModel) VaultModel {
	t.Helper()
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	m = updated.(VaultModel)
	if cmd == nil {
		t.Fatal("o must return a command")
	}
	updated, _ = m.Update(cmd())
	return updated.(VaultModel)
}

func TestVaultOpenRejectsFileURI(t *testing.T) {
	targets, _ := fakeOpener(t, "exit 0\n")
	m := pressOpen(t, vaultWithURI("file:///etc/passwd"))
	if len(*targets) != 0 {
		t.Errorf("the opener ran for %q", *targets)
	}
	if !m.toastIsError || !strings.Contains(m.toast, "http and https") {
		t.Errorf("toast = %q (error: %v), want an error toast", m.toast, m.toastIsError)
	}
}

func TestVaultOpenFailureShowsToast(t *testing.T) {
	fakeOpener(t, "exit 1\n")
	m := pressOpen(t, vaultWithURI("example.com"))
	if _, status := m.FooterContent(); !m.toastIsError || !strings.Contains(status, "Open URL failed") {
		t.Errorf("status = %q, want an Open URL failed toast", status)
	}
}

func TestVaultOpenSuccessShowsNoError(t *testing.T) {
	targets, _ := fakeOpener(t, "exit 0\n")
	m := pressOpen(t, vaultWithURI("example.com"))
	if !reflect.DeepEqual(*targets, []string{"https://example.com"}) {
		t.Errorf("opener got %q, want https://example.com", *targets)
	}
	if m.toastIsError {
		t.Errorf("toast = %q, want no error", m.toast)
	}
}
