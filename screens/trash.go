package screens

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/secutil"
	"github.com/juthrbog/lazybw/ui"
)

// fetchTrashCmd is the bw command that lists the items in the trash. Tests
// replace it.
var fetchTrashCmd = bwcmd.FetchTrash

// ShowsTrash reports whether the list shows the items in the trash instead
// of the vault items.
func (m VaultModel) ShowsTrash() bool {
	return m.trash
}

// fetchList starts a load of the list that shows: the trash or the vault.
// Each load gets a new number, so that an older load is dropped. This is
// also true for a vault load that started before a switch to the trash
// and back.
func (m *VaultModel) fetchList() tea.Cmd {
	m.listSeq++
	if !m.trash {
		return bwcmd.FetchItems(m.sess.Token, m.sess.Epoch, m.listSeq)
	}
	return fetchTrashCmd(m.sess.Token, m.sess.Epoch, m.listSeq)
}

// toggleTrash switches the list between the vault and the trash. The rows
// of the old list go at once, so that no key acts on an item of the other
// list while the new list loads. lazybw keeps no copy of the vault list
// while the trash shows, so the switch back loads the vault list again.
func (m VaultModel) toggleTrash() (tea.Model, tea.Cmd, bool) {
	// fetchList below gives the load a new number, so a load that
	// started before this switch is dropped, also after a switch there
	// and back.
	m.trash = !m.trash

	if m.list.FilterState() != list.Unfiltered {
		m.list.ResetFilter()
		m.resizeList()
	}
	m.filterFrom = rowKey{}

	old := m.rawItems
	m.rawItems = nil
	m.grouped = nil
	cmd := m.rebuildListItems()
	m.list.Select(0)
	secutil.ZeroItems(old)

	// A pending select or edit load belongs to an item of the old list.
	m.selectID = ""
	m.loadingID = ""
	// The selected ID can be "" before and after the switch, for example
	// on a group header, so the drawer state is reset here and not left
	// to syncSelection.
	m.onCursorChange()
	// A copy toast stays, because the clipboard still holds the value
	// until the clear.
	if !m.toastIsCopy {
		m.clearToast()
	}

	m.listLoading = true
	m.listErr = false
	return m, tea.Batch(cmd, m.syncSpinner.Tick, m.fetchList()), true
}

// handleTrashResult fills the trash list with the result of bw list items
// --trash. It drops and scrubs the result when the vault list shows, or
// when a newer trash load started.
func (m VaultModel) handleTrashResult(msg bwcmd.TrashResult) (VaultModel, tea.Cmd) {
	if !m.trash || msg.Seq != m.listSeq {
		secutil.ZeroItems(msg.Items)
		return m, nil
	}
	return m.applyListResult(msg.Items, msg.Err)
}

// applyListResult fills the list with the result of a list load that is
// still current, of the vault or of the trash. A load that follows a
// successful sync also ends the sync.
func (m VaultModel) applyListResult(items []bwcmd.Item, err error) (VaultModel, tea.Cmd) {
	m.listLoading = false
	synced := m.syncReload
	if synced {
		m.syncReload = false
		m.syncing = false
	}
	if err != nil {
		m.listErr = true
		return m, m.setErrorToast("Load failed: " + userError(err, actionOther))
	}
	m.listErr = false
	cmd := m.replaceItems(items)
	if m.selectID != "" {
		m.selectKey(rowKey{id: m.selectID})
		m.selectID = ""
	}
	if synced {
		cmd = tea.Batch(cmd, m.setToast(ui.GlyphSuccess+" Vault synced"))
	}
	return m, cmd
}

// loadText is the status text while a bw command loads the list.
func (m VaultModel) loadText() string {
	switch {
	case m.syncing:
		return "Syncing" + ui.Glyphs.Ellipsis
	case m.trash:
		return "Loading trash" + ui.Glyphs.Ellipsis
	}
	return "Loading vault" + ui.Glyphs.Ellipsis
}

// emptyListText returns the text that shows in place of an empty list:
// while the list loads, after a failed load, and for an empty trash. The
// text wraps at width. It returns "" for an empty vault list that loaded,
// which shows the text of the list.
func (m VaultModel) emptyListText(width int) string {
	wrap := func(s string) string { return ansi.Wrap(s, max(1, width), "") }
	switch {
	case m.listLoading:
		return m.syncSpinner.View() + " " + ui.StyleFaint.Render(m.loadText())
	case m.listErr && m.trash:
		return ui.StyleError.Render(wrap("Could not load the trash. Press r to try again, or D to go back."))
	case m.listErr:
		return ui.StyleError.Render(wrap("Could not load the vault. Press r to try again."))
	case m.trash:
		return ui.StyleFaint.Render(wrap("The trash is empty."))
	}
	return ""
}

// helpBindings returns the entries of the help overlay. In the trash list,
// it leaves out the keys that do nothing there: new, edit and delete.
func (m VaultModel) helpBindings() []ui.HelpBinding {
	all := m.keymap.HelpBindings()
	if !m.trash {
		return all
	}
	skip := map[string]bool{
		m.keymap.New.Help().Key:    true,
		m.keymap.Edit.Help().Key:   true,
		m.keymap.Delete.Help().Key: true,
	}
	out := make([]ui.HelpBinding, 0, len(all))
	for _, b := range all {
		if !skip[b.Key] {
			out = append(out, b)
		}
	}
	return out
}
