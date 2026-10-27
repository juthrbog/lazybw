package screens

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/ui"
)

// newItemKeys are the keys of the card that picks the type of a new item.
var newItemKeys = ui.DefaultNewItemKeyMap()

// discardKeys are the keys of the dialog that asks to discard changes.
var discardKeys = ui.DefaultConfirmKeyMap()

// The bw commands of the item form. Tests replace them.
var (
	createItemCmd = bwcmd.CreateItem
	getItemCmd    = bwcmd.GetItem
	editItemCmd   = bwcmd.EditItem
)

// editSelected opens the edit form for the selected item. The edit form
// shows each field, so for an item with master password re-prompt it asks
// for the master password first. It edits logins and secure notes only.
func (m VaultModel) editSelected() (tea.Model, tea.Cmd, bool) {
	item := m.selectedItem()
	if item == nil || m.loadingID != "" {
		return m, nil, true
	}
	if item.Type != bwcmd.ItemTypeLogin && item.Type != bwcmd.ItemTypeSecureNote {
		name := strings.ToLower(ui.ItemTypeName(item.Type))
		return m, m.setToast("Editing " + name + " items is not supported yet"), true
	}
	if m.needsReprompt() {
		return m, m.openReprompt(secretEdit), true
	}
	return m, m.startEdit(), true
}

// startEdit runs bw get item for the selected item. The edit form opens
// when the result comes back.
func (m *VaultModel) startEdit() tea.Cmd {
	item := m.selectedItem()
	if item == nil {
		return nil
	}
	m.loadingID = item.ID
	m.showToast("Loading "+ui.Truncate(ui.SanitizeLine(item.Name), maxFilterHintW)+ui.Glyphs.Ellipsis, false, false)
	return getItemCmd(m.sess.Token, m.sess.Epoch, item.ID)
}

// handleGetItemResult opens the edit form with the item that bw get item
// returned. It drops the result when the session ended, or when the user
// selected another item or opened something else while bw ran.
func (m VaultModel) handleGetItemResult(msg bwcmd.GetItemResult) (VaultModel, tea.Cmd) {
	if msg.Epoch != m.sess.Epoch || msg.ID != m.loadingID {
		return m, nil
	}
	m.loadingID = ""
	m.clearToast()
	item := m.selectedItem()
	if item == nil || item.ID != msg.ID || m.overlay != overlayNone || m.formOpen {
		return m, nil
	}
	if msg.Err != nil {
		return m, m.setErrorToast("Edit failed: " + userError(msg.Err, actionOther))
	}
	var cmd tea.Cmd
	m.form, cmd = newEditForm(item.Type, msg.ID, msg.Item, m.width, m.height-ui.ChromeHeight)
	m.formOpen = true
	return m, cmd
}

// updateNewItem handles a key while the new item card shows. l and s open
// the form for that type, and esc closes the card.
func (m VaultModel) updateNewItem(msg tea.KeyPressMsg) (VaultModel, tea.Cmd) {
	var t bwcmd.ItemType
	switch {
	case key.Matches(msg, newItemKeys.Login):
		t = bwcmd.ItemTypeLogin
	case key.Matches(msg, newItemKeys.Note):
		t = bwcmd.ItemTypeSecureNote
	case key.Matches(msg, newItemKeys.Cancel):
		m.overlay = overlayNone
		return m, nil
	default:
		return m, nil
	}
	m.overlay = overlayNone
	var cmd tea.Cmd
	m.form, cmd = newItemForm(t, m.width, m.height-ui.ChromeHeight)
	m.formOpen = true
	return m, cmd
}

// updateForm handles a key while the item form shows. While a save runs,
// the form gets no keys.
func (m VaultModel) updateForm(msg tea.KeyPressMsg) (VaultModel, tea.Cmd) {
	if m.form.saving {
		return m, nil
	}
	k := m.form.keys
	switch {
	case key.Matches(msg, k.Cancel):
		if m.form.dirty {
			m.overlay = overlayDiscard
			return m, nil
		}
		m.closeForm()
		return m, nil
	case key.Matches(msg, k.Save):
		return m.saveForm()
	case key.Matches(msg, k.Generate) && m.form.onPassword():
		var cmd tea.Cmd
		m.gen, cmd = m.gen.OpenForField()
		m.overlay = overlayGenerator
		return m, cmd
	}
	var cmd tea.Cmd
	m.form, cmd = m.form.update(msg)
	return m, cmd
}

// saveForm checks the form and starts bw create item.
func (m VaultModel) saveForm() (VaultModel, tea.Cmd) {
	ok, cmd := m.form.validate()
	if !ok {
		return m, cmd
	}
	m.form.saving = true
	m.form.bwErr = ""
	var save tea.Cmd
	if m.form.id != "" {
		save = editItemCmd(m.sess.Token, m.sess.Epoch, m.form.id, m.form.itemJSON(), m.form.private())
	} else {
		save = createItemCmd(m.sess.Token, m.sess.Epoch, m.form.itemJSON(), m.form.private())
	}
	return m, tea.Batch(m.syncSpinner.Tick, save)
}

// handleSaveResult handles the result of bw create item or bw edit item
// for the item id. A failed save keeps the form open with its values and
// shows the error. A save that worked closes the form, loads the items
// again and selects the item.
func (m VaultModel) handleSaveResult(id string, err error, epoch uint64) (VaultModel, tea.Cmd) {
	if epoch != m.sess.Epoch || !m.formOpen || !m.form.saving {
		return m, nil
	}
	m.form.saving = false
	if err != nil {
		m.form.bwErr = userError(err, actionOther)
		return m, nil
	}
	glyph, verb := ui.GlyphSave, " Created "
	if m.form.id != "" {
		glyph, verb = ui.GlyphEdit, " Saved "
	}
	name := strings.TrimSpace(m.form.lines[0].input.Value())
	m.closeForm()
	m.selectID = id
	toast := m.setToast(glyph + verb + ui.Truncate(ui.SanitizeLine(name), maxFilterHintW))
	return m, tea.Batch(toast, bwcmd.FetchItems(m.sess.Token, m.sess.Epoch))
}

// closeForm closes the item form and drops its values.
func (m *VaultModel) closeForm() {
	m.form.clear()
	m.formOpen = false
	if m.overlay == overlayDiscard || m.overlay == overlayGenerator {
		m.overlay = overlayNone
	}
}

// updateDiscard handles a key while the dialog that asks to discard the
// changes of the form shows.
func (m VaultModel) updateDiscard(msg tea.KeyPressMsg) (VaultModel, tea.Cmd) {
	switch {
	case key.Matches(msg, discardKeys.Yes):
		m.closeForm()
	case key.Matches(msg, discardKeys.No):
		m.overlay = overlayNone
	}
	return m, nil
}

// renderNewItem renders the card that picks the type of a new item.
func renderNewItem() string {
	return renderChoiceCard("New item", "Choose the type of the item.",
		newItemKeys.Login.Help(), newItemKeys.Note.Help(), newItemKeys.Cancel.Help())
}

// renderDiscard renders the dialog that asks to discard the changes.
func renderDiscard() string {
	return renderChoiceCard("Discard changes?", "The changes in the form will be lost.",
		key.Help{Key: discardKeys.Yes.Help().Key, Desc: "discard"},
		key.Help{Key: discardKeys.No.Help().Key, Desc: "keep editing"})
}

// renderChoiceCard renders a card with a title, a text and a row of keys.
func renderChoiceCard(title, text string, keys ...key.Help) string {
	keyStyle := lipgloss.NewStyle().Foreground(ui.ColorHighlight).Bold(true)
	var b strings.Builder
	b.WriteString(ui.StyleTitle.Render(title))
	b.WriteString("\n\n")
	b.WriteString(text)
	b.WriteString("\n\n")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(ui.StyleFaint.Render(ui.Glyphs.Dot))
		}
		b.WriteString(keyStyle.Render(k.Key))
		b.WriteString(ui.StyleFaint.Render(" " + k.Desc))
	}
	return b.String()
}

// newItemHints returns the footer hints while the new item card shows.
func newItemHints() []ui.HintBinding {
	return []ui.HintBinding{
		ui.Hint(newItemKeys.Login, "login"),
		ui.Hint(newItemKeys.Note, "secure note"),
		ui.Hint(newItemKeys.Cancel, "cancel"),
	}
}

// discardHints returns the footer hints while the discard dialog shows.
func discardHints() []ui.HintBinding {
	return []ui.HintBinding{
		ui.Hint(discardKeys.Yes, "discard"),
		ui.Hint(discardKeys.No, "keep editing"),
	}
}
