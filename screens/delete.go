package screens

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/ui"
)

// deleteKeys are the keys of the dialog that moves an item to the trash.
var deleteKeys = ui.DefaultConfirmKeyMap()

// deleteItemCmd is the bw command that moves an item to the trash. Tests
// replace it.
var deleteItemCmd = bwcmd.DeleteItem

// deleteState is the dialog that moves an item to the trash. It shows
// while the overlay of the vault is overlayDelete.
type deleteState struct {
	id      string
	name    string
	running bool   // bw delete item runs
	errText string // the error of the last try
}

// deleteSelected opens the trash dialog for the selected item. For an item
// with master password re-prompt, it asks for the master password first.
func (m VaultModel) deleteSelected() (tea.Model, tea.Cmd, bool) {
	if m.selectedItem() == nil {
		return m, nil, true
	}
	if m.needsReprompt() {
		return m, m.openReprompt(secretDelete), true
	}
	m.openDelete()
	return m, nil, true
}

// openDelete opens the trash dialog for the selected item.
func (m *VaultModel) openDelete() {
	item := m.selectedItem()
	if item == nil {
		return
	}
	m.del = deleteState{id: item.ID, name: item.Name}
	m.overlay = overlayDelete
}

// updateDelete handles a key while the trash dialog shows. y runs bw delete
// item, and n and esc close the dialog. While bw runs, the dialog gets no
// keys.
func (m VaultModel) updateDelete(msg tea.KeyPressMsg) (VaultModel, tea.Cmd) {
	if m.del.running {
		return m, nil
	}
	switch {
	case key.Matches(msg, deleteKeys.Yes):
		m.del.running = true
		m.del.errText = ""
		return m, tea.Batch(m.syncSpinner.Tick, deleteItemCmd(m.sess.Token, m.sess.Epoch, m.del.id))
	case key.Matches(msg, deleteKeys.No):
		m.closeDelete()
	}
	return m, nil
}

// closeDelete closes the trash dialog.
func (m *VaultModel) closeDelete() {
	m.del = deleteState{}
	if m.overlay == overlayDelete {
		m.overlay = overlayNone
	}
}

// handleDeleteResult handles the result of bw delete item. A failed delete
// shows the error in the dialog, and y tries again. A delete that worked
// closes the dialog, loads the items again and selects the next item.
func (m VaultModel) handleDeleteResult(msg bwcmd.DeleteResult) (VaultModel, tea.Cmd) {
	if msg.Epoch != m.sess.Epoch || m.overlay != overlayDelete || msg.ID != m.del.id {
		return m, nil
	}
	m.del.running = false
	if msg.Err != nil {
		m.del.errText = userError(msg.Err, actionOther)
		return m, nil
	}
	name := m.del.name
	m.closeDelete()
	m.selectID = m.nextItemID(msg.ID)
	toast := m.setToast(ui.GlyphDelete + " Moved " + ui.Truncate(ui.SanitizeLine(name), maxFilterHintW) + " to trash")
	return m, tea.Batch(toast, bwcmd.FetchItems(m.sess.Token, m.sess.Epoch))
}

// nextItemID returns the ID of the item in the row after the item id, or
// in the row before it when it is the last row. It returns "" when no other
// row holds an item.
func (m *VaultModel) nextItemID(id string) string {
	rows := m.list.VisibleItems()
	i := m.rowOfItem(id)
	if i < 0 {
		return ""
	}
	itemAt := func(j int) string {
		if v, ok := rows[j].(VaultItem); ok {
			return v.ID
		}
		return ""
	}
	for j := i + 1; j < len(rows); j++ {
		if next := itemAt(j); next != "" {
			return next
		}
	}
	for j := i - 1; j >= 0; j-- {
		if prev := itemAt(j); prev != "" {
			return prev
		}
	}
	return ""
}

// renderDelete renders the trash dialog for a content area of the given
// width.
func (m VaultModel) renderDelete(width int) string {
	textW := max(1, min(width-ui.OverlayEdgeWidth, genMaxWidth))
	name := ui.Truncate(ui.SanitizeLine(m.del.name), max(1, textW-len(`Move "" to trash?`)))
	var b strings.Builder
	b.WriteString(ui.StyleTitle.Render(`Move "` + name + `" to trash?`))
	b.WriteString("\n\n")
	switch {
	case m.del.running:
		b.WriteString(m.syncSpinner.View() + " " + ui.StyleFaint.Render("Moving to trash"+ui.Glyphs.Ellipsis))
		return b.String()
	case m.del.errText != "":
		b.WriteString(errorBlock(m.del.errText, textW+4, textW))
	default:
		b.WriteString(ansi.Wrap("You can restore the item from the trash for 30 days.", textW, ""))
	}
	b.WriteString("\n\n")
	b.WriteString(renderChoiceKeys(
		key.Help{Key: deleteKeys.Yes.Help().Key, Desc: "move to trash"},
		key.Help{Key: deleteKeys.No.Help().Key, Desc: "cancel"}))
	return b.String()
}

// deleteHints returns the footer hints while the trash dialog shows.
func (m VaultModel) deleteHints() []ui.HintBinding {
	if m.del.running {
		return nil
	}
	return []ui.HintBinding{
		ui.Hint(deleteKeys.Yes, "move to trash"),
		ui.Hint(deleteKeys.No, "cancel"),
	}
}

// rowsOf is list.Model.VisibleItems, named for the tests.
var _ = list.Item(VaultItem{})
