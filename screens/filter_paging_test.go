package screens

import (
	"fmt"
	"strconv"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/ui"
)

// numberedItems returns n logins named "Item 00", "Item 01" and so on.
func numberedItems(n int) []bwcmd.Item {
	items := make([]bwcmd.Item, n)
	for i := range items {
		items[i] = bwcmd.Item{ID: strconv.Itoa(i), Type: bwcmd.ItemTypeLogin, Name: fmt.Sprintf("Item %02d", i)}
	}
	return items
}

var (
	downArrow = tea.KeyPressMsg{Code: tea.KeyDown}
	enter     = tea.KeyPressMsg{Code: tea.KeyEnter}
	esc       = tea.KeyPressMsg{Code: tea.KeyEscape}
	backspace = tea.KeyPressMsg{Code: tea.KeyBackspace}
)

// viewVault renders m as the root frame does and fails the test when the
// view panics.
func viewVault(t *testing.T, m VaultModel, width, height int) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("the view panicked: %v", r)
		}
	}()
	m.ViewContent(width, height-ui.ChromeHeight)
}

func TestTypingAfterDownSelectsFirstMatch(t *testing.T) {
	// "gi" matches GitHub, Gitea and GitLab. After two Down presses, the
	// cursor is on the third match. "gitl" matches only GitLab.
	m := typeKeys(t, vaultAt(filterItems(), 80, 24), "/gi")
	m = press(t, m, downArrow)
	m = press(t, m, downArrow)
	if m.list.Index() != 2 {
		t.Fatalf("after two Down presses, index = %d, want 2", m.list.Index())
	}
	m = typeKeys(t, m, "tl")
	if item := m.selectedItem(); item == nil || item.ID != "l" {
		t.Fatalf("while filtering gitl, selected = %+v, want GitLab (index %d of %d matches)",
			item, m.list.Index(), len(m.list.VisibleItems()))
	}
	viewVault(t, m, 80, 24)
	m = press(t, m, enter)
	if item := m.selectedItem(); item == nil || item.ID != "l" {
		t.Fatalf("after enter, selected = %+v, want GitLab", item)
	}
	if m.totpItemID != "l" {
		t.Errorf("the drawer state belongs to %q, want GitLab", m.totpItemID)
	}
}

func TestTypingMoreTextGoesToBestMatch(t *testing.T) {
	// The cursor goes to the first match when the filter text changes,
	// also when the old index is still in the matches.
	m := typeKeys(t, vaultAt(filterItems(), 80, 24), "/g")
	m = press(t, m, downArrow)
	m = press(t, m, downArrow)
	m = typeKeys(t, m, "i")
	if m.list.Index() != 0 {
		t.Errorf("after more filter text, index = %d, want 0 (the best match)", m.list.Index())
	}
}

func TestNarrowingFilterOnSecondPageDoesNotPanic(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {40, 12}} {
		width, height := size[0], size[1]
		// "1" matches Item 01, Item 10 to Item 19 and more. Down moves the
		// cursor to the second page of the matches. "15" matches one item.
		m := typeKeys(t, vaultAt(numberedItems(60), width, height), "/1")
		for i := 0; m.list.Paginator.Page == 0; i++ {
			if i > len(m.list.VisibleItems()) {
				t.Fatalf("%dx%d: Down does not reach the second page of %d matches",
					width, height, len(m.list.VisibleItems()))
			}
			m = press(t, m, downArrow)
		}
		m = typeKeys(t, m, "5")
		if n := len(m.list.VisibleItems()); n != 1 {
			t.Fatalf("%dx%d: the filter 15 has %d matches, want 1", width, height, n)
		}
		viewVault(t, m, width, height)
		if item := m.selectedItem(); item == nil || item.Name != "Item 15" {
			t.Errorf("%dx%d: selected = %+v, want Item 15", width, height, item)
		}
	}
}

func TestLateFilterMatchesKeepCursorInMatches(t *testing.T) {
	// The matches of a filter can arrive after the user moved the cursor.
	// Down runs before the matches of "15" arrive, on the matches of "1".
	m := typeKeys(t, vaultAt(numberedItems(60), 80, 24), "/1")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: '5', Text: "5"})
	m = updated.(VaultModel)
	for range 13 {
		m = press(t, m, downArrow)
	}
	for _, msg := range runMsgs(cmd) {
		if fm, ok := msg.(list.FilterMatchesMsg); ok {
			updated, _ = m.Update(fm)
			m = updated.(VaultModel)
		}
	}
	viewVault(t, m, 80, 24)
	if item := m.selectedItem(); item == nil || item.Name != "Item 15" {
		t.Errorf("selected = %+v, want Item 15 (index %d, page %d)", item, m.list.Index(), m.list.Paginator.Page)
	}
}

// lastMatchShows reports whether the page of the list shows Item 59.
func lastMatchShows(m VaultModel) bool {
	rows := m.list.VisibleItems()
	start, end := m.list.Paginator.GetSliceBounds(len(rows))
	for _, li := range rows[start:end] {
		if v, ok := li.(VaultItem); ok && v.Name == "Item 59" {
			return true
		}
	}
	return false
}

// lastMatchSelected reports whether Item 59 is selected.
func lastMatchSelected(m VaultModel) bool {
	item := m.selectedItem()
	return item != nil && item.Name == "Item 59"
}

// assertKeysReachLastMatch presses each key up to 70 times from m. PgDn
// must reach the page that shows Item 59, and j and G must select it.
func assertKeysReachLastMatch(t *testing.T, m VaultModel, when string) {
	t.Helper()
	keys := []struct {
		name    string
		key     tea.KeyPressMsg
		reached func(VaultModel) bool
	}{
		{"pgdown", tea.KeyPressMsg{Code: tea.KeyPgDown}, lastMatchShows},
		{"j", keyMsg("j"), lastMatchSelected},
		{"G", keyMsg("G"), lastMatchSelected},
	}
	for _, k := range keys {
		pressed := m
		for i := 0; i < 70 && !k.reached(pressed); i++ {
			pressed = press(t, pressed, k.key)
		}
		if !k.reached(pressed) {
			t.Errorf("%s, %s does not reach Item 59 (index %d, page %d of %d)", when, k.name,
				pressed.list.Index(), pressed.list.Paginator.Page, pressed.list.Paginator.TotalPages)
		}
	}
}

func TestPagingAfterRegroupWithFilterReachesLastMatch(t *testing.T) {
	m := typeKeys(t, vaultAt(numberedItems(60), 80, 24), "/item")
	m = press(t, m, enter)
	m = press(t, m, ctrlG)
	if m.list.FilterState() != list.FilterApplied || len(m.list.VisibleItems()) != 60 {
		t.Fatalf("after ctrl+g, filter state = %v with %d matches, want 60 applied matches",
			m.list.FilterState(), len(m.list.VisibleItems()))
	}
	assertKeysReachLastMatch(t, m, "after ctrl+g with a filter")
}

func TestPagingAfterSyncWithFilterReachesLastMatch(t *testing.T) {
	m := typeKeys(t, vaultAt(numberedItems(60), 80, 24), "/item")
	m = press(t, m, enter)
	updated, _ := m.Update(bwcmd.ItemsResult{Items: numberedItems(60)})
	m = updated.(VaultModel)
	assertKeysReachLastMatch(t, m, "after a sync with a filter")
}

func TestEscOnEmptyFilterKeepsSelection(t *testing.T) {
	for _, grouped := range []bool{false, true} {
		m := vaultAt(filterItems(), 80, 24)
		if grouped {
			m = press(t, m, ctrlG)
		}
		m = selectRow(t, m, "GitLab")
		m = press(t, m, keyMsg("/"))
		m = press(t, m, esc)
		if got := selectedRow(m); got != "GitLab" {
			t.Errorf("grouped %v: after / and esc, selected = %q, want GitLab", grouped, got)
		}

		// Text that the user deletes again also counts as no filter.
		m = press(t, m, keyMsg("/"))
		m = typeKeys(t, m, "al")
		m = press(t, m, backspace)
		m = press(t, m, backspace)
		m = press(t, m, esc)
		if got := selectedRow(m); got != "GitLab" {
			t.Errorf("grouped %v: after deleted filter text and esc, selected = %q, want GitLab", grouped, got)
		}
	}
}

func TestEscOnEmptyFilterKeepsGroupHeader(t *testing.T) {
	m := press(t, vaultAt(groupTestItems(), 80, 24), ctrlG)
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter}) // expand the github group
	m = selectRow(t, m, "group:github")
	m = press(t, m, keyMsg("/"))
	m = press(t, m, esc)
	if got := selectedRow(m); got != "group:github" {
		t.Errorf("after / and esc, selected = %q, want the github group header (rows %q)", got, rowNames(m))
	}
}

func TestFilterWithoutMatchesGoesBackToSelection(t *testing.T) {
	// Enter on a filter without matches clears the filter. Esc does too.
	for name, k := range map[string]tea.KeyPressMsg{"enter": enter, "esc": esc} {
		m := selectRow(t, vaultAt(filterItems(), 80, 24), "Bravo")
		m = typeKeys(t, m, "/zzz")
		if n := len(m.list.VisibleItems()); n != 0 {
			t.Fatalf("the filter zzz has %d matches, want 0", n)
		}
		m = press(t, m, k)
		if m.list.FilterState() != list.Unfiltered {
			t.Fatalf("%s: filter state = %v, want no filter", name, m.list.FilterState())
		}
		if got := selectedRow(m); got != "Bravo" {
			t.Errorf("%s: after a filter without matches, selected = %q, want Bravo", name, got)
		}
	}
}

// TestSlashFrameFitsArea renders the frame right after "/", before any
// command of the list runs. The filter input makes the title one row
// taller, so the list must count its pages again at once.
func TestSlashFrameFitsArea(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {40, 12}} {
		width, height := size[0], size[1]
		m := vaultAt(numberedItems(60), width, height)
		updated, _ := m.Update(keyMsg("/"))
		m = updated.(VaultModel)
		if m.list.FilterState() != list.Filtering {
			t.Fatalf("%dx%d: filter state = %v, want Filtering", width, height, m.list.FilterState())
		}
		assertArea(t, m.ViewContent(width, height-ui.ChromeHeight), width, height-ui.ChromeHeight)
	}
}
