package screens

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
)

// rowNames returns the visible rows of the list: the name of each item,
// and "group:" and the key of each group header.
func rowNames(m VaultModel) []string {
	var names []string
	for _, li := range m.list.VisibleItems() {
		switch v := li.(type) {
		case VaultItem:
			names = append(names, v.Name)
		case GroupHeaderItem:
			names = append(names, "group:"+v.BaseKey)
		}
	}
	return names
}

// selectedRow returns the name of the selected row, in the form of
// rowNames.
func selectedRow(m VaultModel) string {
	switch v := m.list.SelectedItem().(type) {
	case VaultItem:
		return v.Name
	case GroupHeaderItem:
		return "group:" + v.BaseKey
	}
	return ""
}

// selectRow moves the cursor to the row with the given name, in the form
// of rowNames.
func selectRow(t *testing.T, m VaultModel, name string) VaultModel {
	t.Helper()
	for i, n := range rowNames(m) {
		if n == name {
			m.list.Select(i)
			m.syncSelection()
			return m
		}
	}
	t.Fatalf("no row %q in %q", name, rowNames(m))
	return m
}

var ctrlG = tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}

func TestToggleGroupingKeepsSelectedItem(t *testing.T) {
	// Gmail is row 3. With grouping on, the list has 2 rows.
	m := selectRow(t, newTestVault(groupTestItems()), "Gmail")
	m = press(t, m, ctrlG)
	if got := selectedRow(m); got != "Gmail" {
		t.Errorf("after ctrl+g, selected = %q, want Gmail (rows %q)", got, rowNames(m))
	}

	// GitHub (Work) goes into the collapsed github group.
	m = selectRow(t, newTestVault(groupTestItems()), "GitHub (Work)")
	m = press(t, m, ctrlG)
	if got := selectedRow(m); got != "group:github" {
		t.Errorf("after ctrl+g, selected = %q, want the github group header", got)
	}

	// An expanded group shows the item itself.
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = selectRow(t, m, "GitHub (Personal)")
	m = press(t, m, ctrlG) // off
	m = press(t, m, ctrlG) // on, all groups collapsed
	if got := selectedRow(m); got != "group:github" {
		t.Errorf("after ctrl+g twice, selected = %q, want the github group header", got)
	}
}

func TestToggleGroupingOffSelectsFirstItemOfGroup(t *testing.T) {
	// Grouped, the alpha header is row 1. Without groups, row 1 is Beta (2).
	items := []bwcmd.Item{
		{ID: "b1", Type: bwcmd.ItemTypeLogin, Name: "Beta (1)"},
		{ID: "b2", Type: bwcmd.ItemTypeLogin, Name: "Beta (2)"},
		{ID: "a1", Type: bwcmd.ItemTypeLogin, Name: "Alpha (1)"},
		{ID: "a2", Type: bwcmd.ItemTypeLogin, Name: "Alpha (2)"},
	}
	m := press(t, newTestVault(items), ctrlG)
	m = selectRow(t, m, "group:alpha")
	m = press(t, m, ctrlG)
	if got := selectedRow(m); got != "Alpha (1)" {
		t.Errorf("after ctrl+g on the alpha header, selected = %q, want Alpha (1)", got)
	}
}

func TestEscKeepsFilteredSelection(t *testing.T) {
	items := []bwcmd.Item{
		{ID: "a", Type: bwcmd.ItemTypeLogin, Name: "Alpha"},
		{ID: "o", Type: bwcmd.ItemTypeLogin, Name: "Omega"},
		{ID: "m", Type: bwcmd.ItemTypeLogin, Name: "Omicron"},
		{ID: "z", Type: bwcmd.ItemTypeLogin, Name: "Zeta"},
	}
	m := typeKeys(t, newTestVault(items), "/om")
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = selectRow(t, m, "Omicron")
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := rowNames(m); !reflect.DeepEqual(got, []string{"Alpha", "Omega", "Omicron", "Zeta"}) {
		t.Fatalf("after esc, rows = %q, want the full list", got)
	}
	if got := selectedRow(m); got != "Omicron" {
		t.Errorf("after esc, selected = %q, want Omicron", got)
	}

	// Esc while the filter input is open does the same.
	m = typeKeys(t, newTestVault(items), "/zet")
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := selectedRow(m); got != "Zeta" {
		t.Errorf("after esc in the input, selected = %q, want Zeta", got)
	}
}

func TestEscSelectsGroupOfFilteredItem(t *testing.T) {
	// The match is row 0 of the filtered list. The aws header is row 1 of
	// the grouped list.
	items := []bwcmd.Item{
		{ID: "g", Type: bwcmd.ItemTypeLogin, Name: "Gmail"},
		{ID: "d", Type: bwcmd.ItemTypeLogin, Name: "AWS (Dev)"},
		{ID: "p", Type: bwcmd.ItemTypeLogin, Name: "AWS (Prod)"},
	}
	m := press(t, vaultAt(items, 80, 24), ctrlG)
	m = typeKeys(t, m, "/prod")
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := selectedRow(m); got != "AWS (Prod)" {
		t.Fatalf("selected = %q, want AWS (Prod)", got)
	}
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := selectedRow(m); got != "group:aws" {
		t.Errorf("after esc, selected = %q, want the aws group header (rows %q)", got, rowNames(m))
	}
}

func TestRefreshKeepsSelectedGroupHeader(t *testing.T) {
	m := press(t, newTestVault(refreshItems()), ctrlG)
	m = selectRow(t, m, "group:charlie")
	fresh := append([]bwcmd.Item{{ID: "y", Type: bwcmd.ItemTypeLogin, Name: "Aardvark"}}, refreshItems()...)
	updated, _ := m.Update(bwcmd.ItemsResult{Items: fresh})
	m = updated.(VaultModel)
	if got := selectedRow(m); got != "group:charlie" {
		t.Errorf("after a refresh, selected = %q, want the charlie group header (rows %q)", got, rowNames(m))
	}
}
