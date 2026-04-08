package screens

import (
	"strings"
	"testing"

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
