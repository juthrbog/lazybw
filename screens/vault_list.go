package screens

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/secutil"
	"github.com/juthrbog/lazybw/ui"
)

// VaultItem wraps bwcmd.Item to implement list.Item.
type VaultItem struct {
	bwcmd.Item
	Indent bool // true when displayed as child of an expanded group
}

// FilterValue returns the string used for fuzzy filtering.
func (v VaultItem) FilterValue() string { return v.Name + " " + v.Description() }

// VaultDelegate renders vault items using the existing row renderer.
type VaultDelegate struct{}

func (d VaultDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	selected := index == m.Index()
	switch v := item.(type) {
	case GroupHeaderItem:
		_, _ = fmt.Fprint(w, ui.RenderGroupRow(v.Name, v.Count, v.Expanded, selected, m.Width()))
	case VaultItem:
		_, _ = fmt.Fprint(w, ui.RenderItemRow(v.Item, selected, m.Width(), v.Indent))
	}
}

func (d VaultDelegate) Height() int                               { return 1 }
func (d VaultDelegate) Spacing() int                              { return 0 }
func (d VaultDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func toListItems(items []bwcmd.Item) []list.Item {
	li := make([]list.Item, len(items))
	for i, item := range items {
		li[i] = VaultItem{Item: item}
	}
	return li
}

// updateListKey handles a key that no overlay and no action key took. Up
// and down move the cursor in the matches while the filter input is open.
// The list gets all other keys, for navigation and the filter input.
func (m VaultModel) updateListKey(msg tea.KeyPressMsg) (VaultModel, tea.Cmd) {
	if m.list.SettingFilter() {
		// The filter input stays open.
		switch msg.String() {
		case "up":
			m.list.CursorUp()
			return m, nil
		case "down":
			m.list.CursorDown()
			return m, nil
		}
	}
	prevKey := m.selectedKey()
	prevFilter := m.list.FilterState()
	prevText := m.list.FilterValue()
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	fs := m.list.FilterState()
	if fs == list.Filtering && m.list.FilterValue() != prevText {
		// The filter text changed. The list sends the new matches in a
		// later message, best match first. Move the cursor to the first
		// row now, so that it is on the best match when they arrive.
		m.list.Select(0)
	}
	if fs != prevFilter {
		// The filter input changes the height of the list title. The
		// list does not count its pages again for this, so the next
		// frame would draw one row too many.
		m.resizeList()
		if prevFilter == list.Unfiltered {
			m.filterFrom = prevKey
		}
		// Flatten groups while filtering, restore on filter clear.
		if m.groups.enabled && (prevFilter == list.Unfiltered || fs == list.Unfiltered) {
			cmds = append(cmds, m.rebuildListItems())
		}
		// The list keeps the cursor index when it clears a filter, but
		// the full list often has another item at that index. So
		// select the item again. See clearedFilterKey.
		if fs == list.Unfiltered && !m.selectKey(m.clearedFilterKey(prevKey, prevText)) {
			m.clampCursor()
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *VaultModel) selectedItem() *bwcmd.Item {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}
	switch v := item.(type) {
	case VaultItem:
		return &v.Item
	case GroupHeaderItem:
		return nil
	default:
		return nil
	}
}

// rowKey names a row of the list, so that the selection can stay on it
// when the rows change: an item by its ID, or a group header by its key.
type rowKey struct {
	id    string // the item ID, or "" for a group header
	group string // the group key of a group header
}

// selectedKey returns the key of the selected row, or the zero rowKey when
// no row is selected.
func (m *VaultModel) selectedKey() rowKey {
	switch v := m.list.SelectedItem().(type) {
	case VaultItem:
		return rowKey{id: v.ID}
	case GroupHeaderItem:
		return rowKey{group: v.BaseKey}
	}
	return rowKey{}
}

// selectKey moves the cursor to the visible row of k. When the list does
// not show that row, it selects the row that holds it: the header of the
// group of an item, or the first item of a group when the list shows no
// groups. It reports false when no visible row matches, and the cursor then
// stays where it is.
func (m *VaultModel) selectKey(k rowKey) bool {
	i := -1
	switch {
	case k.id != "":
		if i = m.rowOfItem(k.id); i < 0 && m.showsGroups() {
			i = m.rowOfGroup(m.itemGroups().keyOf[k.id])
		}
	case k.group != "":
		if i = m.rowOfGroup(k.group); i < 0 && !m.showsGroups() {
			i = m.rowOfItem(m.itemGroups().firstID(k.group))
		}
	}
	if i < 0 {
		return false
	}
	m.list.Select(i)
	return true
}

// clearedFilterKey returns the row to select after a key cleared the
// filter. selected is the row that was selected before the key, and text
// is the filter text before the key. The selected match stays selected.
// When the filter had no text or no match was selected, the row that was
// selected before the filter input opened gets the selection back.
func (m *VaultModel) clearedFilterKey(selected rowKey, text string) rowKey {
	k := selected
	if text == "" || k == (rowKey{}) {
		k = m.filterFrom
	}
	m.filterFrom = rowKey{}
	return k
}

// rowOfItem returns the index of the visible row of the item with the
// given ID, or -1 when no visible row has that ID.
func (m *VaultModel) rowOfItem(id string) int {
	if id == "" {
		return -1
	}
	for i, li := range m.list.VisibleItems() {
		if v, ok := li.(VaultItem); ok && v.ID == id {
			return i
		}
	}
	return -1
}

// rowOfGroup returns the index of the visible header of the group with the
// given key, or -1 when no visible header has that key.
func (m *VaultModel) rowOfGroup(key string) int {
	if key == "" {
		return -1
	}
	for i, li := range m.list.VisibleItems() {
		if v, ok := li.(GroupHeaderItem); ok && v.BaseKey == key {
			return i
		}
	}
	return -1
}

// itemGroups returns the groups of the vault items. It groups the items
// on the first call after the items change.
func (m *VaultModel) itemGroups() *groupCache {
	if m.grouped == nil {
		m.grouped = newGroupCache(m.rawItems)
	}
	return m.grouped
}

// showsGroups reports whether the list rows are grouped: grouping is on
// and no filter is active. While a filter is active, the rows stay flat,
// so that the filter also finds the items of collapsed groups.
func (m *VaultModel) showsGroups() bool {
	return m.groups.enabled && m.list.FilterState() == list.Unfiltered
}

// rebuildListItems sets the list rows for the grouping and filter state.
// See showsGroups.
func (m *VaultModel) rebuildListItems() tea.Cmd {
	if !m.showsGroups() {
		return m.setListItems(toListItems(m.rawItems))
	}
	return m.setListItems(flattenGroups(m.itemGroups().groups, m.groups))
}

// regroup rebuilds the list rows and selects the row of k again: the same
// item or group, or the row that holds it. See selectKey.
func (m *VaultModel) regroup(k rowKey) tea.Cmd {
	cmd := m.rebuildListItems()
	if !m.selectKey(k) {
		m.clampCursor()
	}
	return cmd
}

// setListItems replaces the list rows. With a filter active, SetItems
// clears the visible rows and returns a command that filters again. The
// result is applied at once, so that the selection never points into an
// empty list in between. Otherwise the drawer would reset for an item
// that stays selected.
func (m *VaultModel) setListItems(items []list.Item) tea.Cmd {
	cmd := m.list.SetItems(items)
	if cmd == nil || m.list.FilterState() == list.Unfiltered {
		return cmd
	}
	msg := cmd()
	if fm, ok := msg.(list.FilterMatchesMsg); ok {
		m.applyFilterMatches(fm)
		return nil
	}
	return func() tea.Msg { return msg }
}

// applyFilterMatches gives the list the matches of its filter. The list
// stores the matches, but it does not count their pages or keep the cursor
// in them. Then PgDn, G and j can stop at the last page of the old matches,
// the cursor can point past the last match, and the view of the list
// panics on a page past the last match. So the list counts its pages again
// here, and a cursor past the last match moves to the last match.
func (m *VaultModel) applyFilterMatches(fm list.FilterMatchesMsg) {
	m.list, _ = m.list.Update(fm)
	// SetSize with the same size makes the list count its pages again.
	m.list.SetSize(m.list.Width(), m.list.Height())
	m.clampCursor()
}

// replaceItems swaps in a fresh item list after a sync. It keeps the
// cursor on the same item or group when it still exists, refreshes the
// cached TOTP params, and scrubs the old items.
func (m *VaultModel) replaceItems(items []bwcmd.Item) tea.Cmd {
	k := m.selectedKey()
	old := m.rawItems
	m.rawItems = items
	m.grouped = nil
	cmd := m.regroup(k)
	// The TOTP key of the same item can change in a sync, so reload it.
	// A different item also gets a fresh drawer.
	if item := m.selectedItem(); item != nil && k.id != "" && item.ID == k.id {
		m.loadTOTP()
	} else {
		m.onCursorChange()
	}
	secutil.ZeroItems(old)
	// The sync is complete. Drop its toast, so that the footer shows the
	// time of the last sync again.
	if strings.HasSuffix(m.toast, " Vault synced") {
		m.clearToast()
	}
	return cmd
}

// clampCursor moves the cursor to the last visible row when the list got
// shorter than the cursor index.
func (m *VaultModel) clampCursor() {
	if n := len(m.list.VisibleItems()); n > 0 && m.list.Index() >= n {
		m.list.Select(n - 1)
	}
}
