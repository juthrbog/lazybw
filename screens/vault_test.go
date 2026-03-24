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
	if !strings.Contains(hints, "j/k navigate") {
		t.Errorf("normal mode hints should contain navigation, got %q", hints)
	}
}

func TestVaultFooterContentFilterMode(t *testing.T) {
	m := newTestVault(testItems())
	m.mode = modeFilter
	hints, _ := m.FooterContent()
	if !strings.Contains(hints, "esc clear") {
		t.Errorf("filter mode hints should contain 'esc clear', got %q", hints)
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

func TestVaultApplyFilterEmpty(t *testing.T) {
	m := newTestVault(testItems())
	m.filterStr = ""
	m.applyFilter()
	if len(m.filtered) != len(m.items) {
		t.Errorf("empty filter should show all items, got %d", len(m.filtered))
	}
}

func TestVaultApplyFilterMatch(t *testing.T) {
	m := newTestVault(testItems())
	m.filterStr = "gmail"
	m.applyFilter()
	if len(m.filtered) != 1 {
		t.Errorf("expected 1 match, got %d", len(m.filtered))
	}
	if m.filtered[0].Name != "Gmail" {
		t.Errorf("expected Gmail, got %q", m.filtered[0].Name)
	}
}

func TestVaultApplyFilterNoMatch(t *testing.T) {
	m := newTestVault(testItems())
	m.filterStr = "zzzzz"
	m.applyFilter()
	if len(m.filtered) != 0 {
		t.Errorf("expected 0 matches, got %d", len(m.filtered))
	}
}

func TestVaultApplyFilterByDescription(t *testing.T) {
	m := newTestVault(testItems())
	m.filterStr = "user@gmail"
	m.applyFilter()
	if len(m.filtered) != 1 {
		t.Errorf("expected 1 match by description, got %d", len(m.filtered))
	}
}

func TestVaultMoveCursorBounds(t *testing.T) {
	m := newTestVault(testItems())

	m.moveCursor(-10)
	if m.cursor != 0 {
		t.Errorf("cursor should clamp to 0, got %d", m.cursor)
	}

	m.moveCursor(100)
	if m.cursor != 2 {
		t.Errorf("cursor should clamp to len-1, got %d", m.cursor)
	}
}

func TestVaultSelectedItemInRange(t *testing.T) {
	m := newTestVault(testItems())
	m.cursor = 1
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

func TestVaultSelectedItemOutOfRange(t *testing.T) {
	m := newTestVault(testItems())
	m.cursor = 99
	if m.selectedItem() != nil {
		t.Error("expected nil for out of range cursor")
	}
}
