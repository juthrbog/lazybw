package screens

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/secutil"
	"github.com/juthrbog/lazybw/ui"
)

// restoreKeys are the keys of the dialog that restores an item from the
// trash.
var restoreKeys = ui.DefaultConfirmKeyMap()

// restoreItemCmd is the bw command that restores an item from the trash.
// Tests replace it.
var restoreItemCmd = bwcmd.RestoreItem

// restoreState is the dialog that restores an item from the trash. It
// shows while the overlay of the vault is overlayRestore.
type restoreState struct {
	id      string
	name    string
	running bool   // bw restore item runs
	errText string // the error of the last try
}

// restoreSelected opens the restore dialog for the selected item. For an
// item with master password re-prompt, it asks for the master password
// first.
func (m VaultModel) restoreSelected() (tea.Model, tea.Cmd, bool) {
	if m.selectedItem() == nil {
		return m, nil, true
	}
	if m.needsReprompt() {
		return m, m.openReprompt(secretRestore), true
	}
	m.openRestore()
	return m, nil, true
}

// openRestore opens the restore dialog for the selected item.
func (m *VaultModel) openRestore() {
	item := m.selectedItem()
	if item == nil {
		return
	}
	m.res = restoreState{id: item.ID, name: item.Name}
	m.overlay = overlayRestore
}

// updateRestore handles a key while the restore dialog shows. y runs bw
// restore item, and n and esc close the dialog. While bw runs, the dialog
// gets no keys.
func (m VaultModel) updateRestore(msg tea.KeyPressMsg) (VaultModel, tea.Cmd) {
	if m.res.running {
		return m, nil
	}
	switch {
	case key.Matches(msg, restoreKeys.Yes):
		m.res.running = true
		m.res.errText = ""
		return m, tea.Batch(m.syncSpinner.Tick, restoreItemCmd(m.sess.Token, m.sess.Epoch, m.res.id))
	case key.Matches(msg, restoreKeys.No):
		m.closeRestore()
	}
	return m, nil
}

// closeRestore closes the restore dialog.
func (m *VaultModel) closeRestore() {
	m.res = restoreState{}
	if m.overlay == overlayRestore {
		m.overlay = overlayNone
	}
}

// handleRestoreResult handles the result of bw restore item. A failed
// restore shows the error in the dialog, and y tries again. A restore that
// worked closes the dialog, takes the item out of the list at once,
// selects the next item and loads the list again.
func (m VaultModel) handleRestoreResult(msg bwcmd.RestoreResult) (VaultModel, tea.Cmd) {
	if msg.Epoch != m.sess.Epoch || m.overlay != overlayRestore || msg.ID != m.res.id {
		return m, nil
	}
	m.res.running = false
	if msg.Err != nil {
		m.res.errText = userError(msg.Err, actionOther)
		return m, nil
	}
	name := m.res.name
	m.closeRestore()
	next := m.nextItemID(msg.ID)
	// The item leaves the list before the reload, so that U cannot open
	// the dialog for it again and run bw restore a second time.
	cmd := m.dropItem(msg.ID)
	if !m.selectKey(rowKey{id: next}) {
		m.clampCursor()
	}
	m.onCursorChange()
	m.selectID = next
	toast := m.setToast(ui.GlyphSuccess + " Restored " + ui.Truncate(ui.SanitizeLine(name), maxFilterHintW))
	return m, tea.Batch(cmd, toast, m.fetchList())
}

// dropItem takes the item id out of the list and scrubs it.
func (m *VaultModel) dropItem(id string) tea.Cmd {
	i := slices.IndexFunc(m.rawItems, func(it bwcmd.Item) bool { return it.ID == id })
	if i < 0 {
		return nil
	}
	secutil.ZeroItem(&m.rawItems[i])
	m.rawItems = slices.Delete(m.rawItems, i, i+1)
	m.grouped = nil
	return m.rebuildListItems()
}

// renderRestore renders the restore dialog for a content area of the given
// width.
func (m VaultModel) renderRestore(width int) string {
	textW := max(1, min(width-ui.OverlayEdgeWidth, genMaxWidth))
	name := ui.Truncate(ui.SanitizeLine(m.res.name), max(1, textW-len(`Restore ""?`)))
	var b strings.Builder
	b.WriteString(ui.StyleTitle.Render(`Restore "` + name + `"?`))
	b.WriteString("\n\n")
	switch {
	case m.res.running:
		b.WriteString(m.syncSpinner.View() + " " + ui.StyleFaint.Render("Restoring"+ui.Glyphs.Ellipsis))
		return b.String()
	case m.res.errText != "":
		b.WriteString(errorBlock(m.res.errText, textW+4, textW))
	default:
		b.WriteString(ansi.Wrap("The item goes back to the vault.", textW, ""))
	}
	b.WriteString("\n\n")
	b.WriteString(renderChoiceKeys(
		key.Help{Key: restoreKeys.Yes.Help().Key, Desc: "restore"},
		key.Help{Key: restoreKeys.No.Help().Key, Desc: "cancel"}))
	return b.String()
}

// restoreHints returns the footer hints while the restore dialog shows.
func (m VaultModel) restoreHints() []ui.HintBinding {
	if m.res.running {
		return nil
	}
	return []ui.HintBinding{
		ui.Hint(restoreKeys.Yes, "restore"),
		ui.Hint(restoreKeys.No, "cancel"),
	}
}
