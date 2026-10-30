package screens

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/secutil"
	"github.com/juthrbog/lazybw/session"
	"github.com/juthrbog/lazybw/totp"
	"github.com/juthrbog/lazybw/ui"
)

// LockMsg tells the root model to lock the vault and show the unlock screen.
type LockMsg struct{}

// QuitMsg tells the root model to lock the vault and exit.
type QuitMsg struct{}

// TOTPTickMsg fires every second to update the TOTP countdown.
type TOTPTickMsg struct{}

// ThemeChangedMsg tells the screens that the theme or the background
// changed. Each screen restyles the parts that copied a color when they
// were built, such as a spinner. The vault screen sends it to the root
// model after a theme change in the picker.
type ThemeChangedMsg struct{}

// overlay names the card that shows over the vault list. At most one card
// shows, and it gets the keys.
type overlay int

const (
	overlayNone        overlay = iota
	overlayReprompt            // the master password re-prompt
	overlayGenerator           // the password generator
	overlayThemePicker         // the theme picker
	overlayHelp                // the key help
	overlayQuit                // the quit dialog
	overlayNewItem             // the card that picks the type of a new item
	overlayDiscard             // the dialog that asks to discard the form
	overlayDelete              // the dialog that moves an item to the trash
)

// VaultModel is the main vault browsing screen.
type VaultModel struct {
	list   list.Model
	keymap ui.VaultKeyMap
	sess   *session.State

	rawItems []bwcmd.Item
	groups   groupState
	// grouped holds the groups of rawItems. It is nil until grouping needs
	// it, and a change of rawItems drops it. See itemGroups.
	grouped *groupCache
	// filterFrom is the row that was selected when the filter input opened.
	// The list moves the cursor to the first row then. See clearedFilterKey.
	filterFrom rowKey

	toast        string
	toastIsError bool
	toastIsCopy  bool   // a copy toast stays until the clipboard clear
	toastGen     uint64 // the number of the latest toast, see toastExpiredMsg
	syncing      bool
	syncSpinner  spinner.Model

	totpCode     string
	totpSecsLeft int
	totpParams   *totp.Params // parsed TOTP params for the selected item
	totpItemID   string       // ID of the item that the TOTP state belongs to
	totpInvalid  bool         // the selected item has a TOTP key that does not parse

	drawerScroll  int
	drawerHeight  int  // current drawer height (resizable)
	drawerOverlay bool // true = overlay mode, false = inline mode

	// overlay is the card that shows, and the fields below hold the state
	// of each card.
	overlay overlay
	help    ui.HelpOverlay
	gen     GeneratorModel
	picker  ThemePickerModel

	currentTheme string

	// The item form. It shows in place of the list and the drawer while
	// formOpen is true. selectID is the item to select after the next load,
	// such as an item that the form created. loadingID is the item that
	// bw get item loads for the edit form, or "" when no load runs.
	form      itemForm
	formOpen  bool
	selectID  string
	loadingID string
	del       deleteState

	// Master password re-prompt. verifiedID is the ID of the selected item
	// after the user entered the master password for it. checking is true
	// while a password check runs, and checkSeq numbers the checks.
	reprompt   repromptState
	verifiedID string
	checking   bool
	checkSeq   uint64

	width  int
	height int
}

// NewVaultModel constructs the vault screen.
func NewVaultModel(items []bwcmd.Item, sess *session.State, width, height int) VaultModel {
	ss := spinner.New()
	ss.Spinner = ui.SpinnerLoad
	ss.Style = ss.Style.Foreground(ui.ColorHighlight)

	// Clamp the drawer as a resize does, so that a short terminal shows
	// the footer on the first frame. A height of 0 means that the size is
	// not known yet, and the first resize clamps the drawer.
	drawerH := ui.DefaultDrawerHeight
	if height > 0 {
		drawerH = clampDrawerHeight(drawerH, height)
	}
	// resizeList below sets the size of the list.
	l := list.New(toListItems(items), VaultDelegate{}, 0, 0)
	l.Title = ""
	l.SetShowTitle(false)
	l.SetShowFilter(true)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetStatusBarItemName("item", "items")
	l.DisableQuitKeybindings()

	// Resolve key conflicts: list defaults bind l/u/?  which clash with our keybinds.
	l.KeyMap.NextPage.SetKeys("right", "pgdown")
	l.KeyMap.PrevPage.SetKeys("left", "pgup")
	l.KeyMap.ShowFullHelp.SetEnabled(false)
	l.KeyMap.CloseFullHelp.SetEnabled(false)
	// By default, the list also accepts the filter on the arrow keys, tab,
	// ctrl+j and ctrl+k. The keys that follow then run vault actions, such
	// as l for lock. Only enter accepts the filter. Update moves the cursor
	// on up and down while the filter input is open.
	l.KeyMap.AcceptWhileFiltering.SetKeys("enter")

	m := VaultModel{
		list:         l,
		keymap:       ui.DefaultVaultKeyMap(),
		help:         ui.NewHelpOverlay(),
		sess:         sess,
		rawItems:     items,
		groups:       newGroupState(),
		syncSpinner:  ss,
		currentTheme: ui.CurrentTheme,
		drawerHeight: drawerH,
		width:        width,
		height:       height,
	}
	m.restyle()
	m.resizeList()

	// Compute the code for the initially selected item now, so that the
	// drawer does not show "loading…" until the first tick.
	m.loadTOTP()

	return m
}

// restyle applies the current theme to the parts of the vault screen that
// keep a copy of a style: the sync spinner and the list.
func (m *VaultModel) restyle() {
	m.syncSpinner.Style = m.syncSpinner.Style.Foreground(ui.ColorHighlight)
	m.list.Styles = ui.ListStyles()
	m.list.FilterInput.SetStyles(m.list.Styles.Filter)
}

// Clear scrubs sensitive data from the vault model. It zeros []byte
// secrets in-place and drops references to sensitive string fields so
// the GC can collect the underlying memory sooner.
func (m *VaultModel) Clear() {
	secutil.ZeroItems(m.rawItems)
	m.rawItems = nil
	m.grouped = nil
	// The list rows hold their own copies of fields such as Notes. The
	// check skips the zero VaultModel, which has no list to reset.
	if len(m.list.Items()) > 0 {
		m.list.ResetFilter()
		m.list.SetItems(nil)
	}
	if m.totpParams != nil {
		m.totpParams.Clear()
		m.totpParams = nil
	}
	m.totpCode = ""
	m.gen.Clear()
	m.closeForm()
	m.closeDelete()
	m.closeReprompt()
	m.verifiedID = ""
}

func (m VaultModel) Init() tea.Cmd {
	return tickTOTP()
}

// Update handles a message and then makes sure that the cached TOTP state
// belongs to the selected item. The list can change the selection in many
// ways (filter results, regrouping, a cleared filter), and not all of them
// move the cursor index.
func (m VaultModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.update(msg)
	vm, ok := updated.(VaultModel)
	if !ok {
		return updated, cmd
	}
	vm.syncSelection()
	return vm, cmd
}

func (m VaultModel) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.drawerHeight = m.clampDrawerHeight(m.drawerHeight)
		m.resizeList()
		m.help.Resize(msg.Width, msg.Height-ui.ChromeHeight)
		if m.formOpen {
			m.form.resize(msg.Width, msg.Height-ui.ChromeHeight)
		}
		switch m.overlay {
		case overlayReprompt:
			// The password form keeps the width of the card, so that the
			// card does not cut the cursor off.
			m.reprompt.form = m.reprompt.form.WithWidth(repromptFormWidth(msg.Width))
		case overlayThemePicker:
			var cmd tea.Cmd
			m.picker, cmd = m.picker.Resize(msg.Height)
			return m, cmd
		}
		return m, nil

	case tea.KeyPressMsg:
		switch m.overlay {
		case overlayReprompt:
			return m.updateReprompt(msg)
		case overlayGenerator:
			return m.updateGenerator(msg)
		case overlayThemePicker:
			return m.updateThemePicker(msg)
		case overlayHelp:
			var cmd tea.Cmd
			m.help, cmd = m.help.Update(msg)
			if !m.help.Visible() {
				m.overlay = overlayNone
			}
			return m, cmd
		case overlayQuit:
			return m.updateQuitConfirm(msg)
		case overlayNewItem:
			return m.updateNewItem(msg)
		case overlayDiscard:
			return m.updateDiscard(msg)
		case overlayDelete:
			return m.updateDelete(msg)
		}
		if m.formOpen {
			return m.updateForm(msg)
		}
		if !m.list.SettingFilter() {
			// When not filtering, intercept our action keys before the list.
			if model, cmd, handled := m.handleActionKeys(msg); handled {
				return model, cmd
			}
		}
		return m.updateListKey(msg)

	case list.FilterMatchesMsg:
		m.applyFilterMatches(msg)
		return m, nil

	case ThemeChangedMsg:
		m.restyle()
		if m.overlay == overlayThemePicker {
			var cmd tea.Cmd
			m.picker, cmd = m.picker.Restyle()
			return m, cmd
		}
		return m, nil

	case session.CopiedMsg:
		// The root model schedules and runs the clipboard clear. The
		// terminal does not answer an OSC 52 write, so the toast says only
		// that lazybw sent the value.
		m.setCopyToast(fmt.Sprintf("%s %s sent to clipboard%sclears in %ds",
			ui.GlyphCopy, copyLabel(msg.Field), ui.Glyphs.Dot, int(session.ClipboardClearDelay.Seconds())))
		return m, nil

	case session.ClipboardClearedMsg:
		// The root model cleared the clipboard. Remove the copy toast,
		// unless a newer toast replaced it.
		if m.toastIsCopy {
			m.clearToast()
		}
		return m, nil

	case toastExpiredMsg:
		if msg.gen == m.toastGen {
			m.clearToast()
		}
		return m, nil

	case repromptResultMsg:
		return m.handleRepromptResult(msg)

	case TOTPTickMsg:
		m.computeTOTP()
		return m, tickTOTP()

	case bwcmd.SyncResult:
		m.syncing = false
		if msg.Err != nil {
			return m, m.setErrorToast("Sync failed: " + userError(msg.Err, actionOther))
		}
		m.sess.LastSync = time.Now()
		toastCmd := m.setToast(ui.GlyphSuccess + " Vault synced")
		return m, tea.Batch(toastCmd, bwcmd.FetchItems(m.sess.Token, m.sess.Epoch))

	case bwcmd.ItemsResult:
		if msg.Err != nil {
			return m, m.setErrorToast("Load failed: " + userError(msg.Err, actionOther))
		}
		cmd := m.replaceItems(msg.Items)
		if m.selectID != "" {
			m.selectKey(rowKey{id: m.selectID})
			m.selectID = ""
		}
		return m, cmd

	case bwcmd.CreateResult:
		return m.handleSaveResult(msg.ID, msg.Err, msg.Epoch)

	case bwcmd.EditResult:
		return m.handleSaveResult(msg.ID, msg.Err, msg.Epoch)

	case bwcmd.GetItemResult:
		return m.handleGetItemResult(msg)

	case bwcmd.DeleteResult:
		return m.handleDeleteResult(msg)

	case OpenURLResult:
		if msg.Err != nil {
			return m, m.setErrorToast("Open URL failed: " + msg.Err.Error())
		}
		return m, nil

	case bwcmd.GenerateResult:
		m.gen, _ = m.gen.Update(msg)
		return m, nil

	case spinner.TickMsg:
		if m.syncing || m.form.saving || m.del.running {
			var cmd tea.Cmd
			m.syncSpinner, cmd = m.syncSpinner.Update(msg)
			return m, cmd
		}
	}

	// The forms complete over internal messages, so they get the other
	// messages too. The list gets them as well while the password prompt
	// shows, but not while the theme picker shows.
	switch m.overlay {
	case overlayReprompt:
		updated, cmd := m.updateReprompt(msg)
		if vm, ok := updated.(VaultModel); ok {
			m = vm
		}
		cmds = append(cmds, cmd)
	case overlayThemePicker:
		return m.updateThemePicker(msg)
	}
	if m.formOpen {
		// The fields get the cursor blink of the focused input.
		var cmd tea.Cmd
		m.form, cmd = m.form.update(msg)
		cmds = append(cmds, cmd)
	}

	// Forward non-key messages to the list (e.g. filter debounce timers).
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// handleActionKeys processes app-specific keybinds (copy, sync, lock, etc.).
// Returns (model, cmd, true) if handled, or (zero, nil, false) to let the list handle the key.
func (m VaultModel) handleActionKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	// The secret fields go through copySecret, which asks for the master
	// password first for an item with master password re-prompt.
	case key.Matches(msg, m.keymap.Copy):
		return m.copySecret(secretMain)

	case key.Matches(msg, m.keymap.CopyTOTP):
		return m.copySecret(secretTOTP)

	case key.Matches(msg, m.keymap.CopyCVV):
		return m.copySecret(secretCVV)

	case key.Matches(msg, m.keymap.CopyNote):
		return m.copySecret(secretNote)

	case key.Matches(msg, m.keymap.CopyUsername):
		if item := m.selectedItem(); item != nil {
			switch {
			case item.Login != nil && item.Login.Username == "":
				// An empty write can clear the clipboard in some terminals.
				return m, m.setToast("No username to copy"), true
			case item.Login != nil:
				return m, session.CopyToClipboard(item.Login.Username, session.CopyFieldUsername), true
			case item.Identity != nil && item.Identity.Email != "":
				return m, session.CopyToClipboard(item.Identity.Email, session.CopyFieldEmail), true
			case item.SSHKey != nil && item.SSHKey.PublicKey != "":
				return m, session.CopyToClipboard(item.SSHKey.PublicKey, session.CopyFieldPublicKey), true
			}
		}

	case key.Matches(msg, m.keymap.OpenURL):
		if item := m.selectedItem(); item != nil && item.Login != nil && len(item.Login.URIs) > 0 {
			return m, openURL(item.Login.URIs[0].URI), true
		}
		return m, nil, true

	case key.Matches(msg, m.keymap.Sync):
		if m.syncing {
			return m, nil, true
		}
		m.syncing = true
		// The footer shows "Syncing…" until the sync ends.
		m.clearToast()
		return m, tea.Batch(m.syncSpinner.Tick, bwcmd.Sync(m.sess.Token, m.sess.Epoch)), true

	case key.Matches(msg, m.keymap.Lock):
		return m, func() tea.Msg { return LockMsg{} }, true

	case key.Matches(msg, m.keymap.Help):
		m.help.Show(m.keymap.HelpBindings(), m.width, m.height-ui.ChromeHeight)
		m.overlay = overlayHelp
		return m, nil, true

	case key.Matches(msg, m.keymap.New):
		m.overlay = overlayNewItem
		return m, nil, true

	case key.Matches(msg, m.keymap.Edit):
		return m.editSelected()

	case key.Matches(msg, m.keymap.Delete):
		return m.deleteSelected()

	case key.Matches(msg, m.keymap.Generate):
		var cmd tea.Cmd
		m.gen, cmd = m.gen.Open()
		m.overlay = overlayGenerator
		return m, cmd, true

	case key.Matches(msg, m.keymap.ThemePicker):
		var cmd tea.Cmd
		m.picker, cmd = NewThemePickerModel(m.currentTheme, m.height)
		m.overlay = overlayThemePicker
		return m, cmd, true

	case key.Matches(msg, m.keymap.Quit):
		m.overlay = overlayQuit
		return m, nil, true

	case key.Matches(msg, m.keymap.ToggleGrouping):
		k := m.selectedKey()
		m.groups.toggleGrouping()
		cmd := m.regroup(k)
		return m, tea.Batch(cmd, m.setToast(groupToastMessage(m.groups.enabled))), true

	case key.Matches(msg, m.keymap.ToggleExpand):
		if gh, ok := m.list.SelectedItem().(GroupHeaderItem); ok {
			m.groups.toggle(gh.BaseKey)
			return m, m.regroup(rowKey{group: gh.BaseKey}), true
		}
	}

	return m.handleDrawerKey(msg)
}

// syncSelection reloads the TOTP state when the selected item is not the
// item that the state belongs to.
func (m *VaultModel) syncSelection() {
	id := ""
	if item := m.selectedItem(); item != nil {
		id = item.ID
	}
	if id != m.totpItemID {
		m.onCursorChange()
	}
}

// onCursorChange resets the per-item drawer state for a new selection. A
// new selection also ends the master password re-prompt: the next protected
// copy asks for the password again, and an open prompt for the item that
// was selected closes.
func (m *VaultModel) onCursorChange() {
	m.drawerScroll = 0
	m.verifiedID = ""
	if m.overlay == overlayReprompt {
		m.closeReprompt()
	}
	m.loadTOTP()
}

// HeaderInfo returns the item count and selected item type name for the
// header. The count is the number of vault items, not of list rows, so a
// group header or a collapsed group does not change it.
func (m VaultModel) HeaderInfo() (int, string) {
	count := len(m.rawItems)
	if item := m.selectedItem(); item != nil {
		return count, ui.ItemTypeName(item.Type)
	}
	return count, ""
}

// ViewContent renders the vault content for the root frame.
func (m VaultModel) ViewContent(width, contentHeight int) string {
	var bg string
	if m.formOpen {
		bg = m.form.view(width, contentHeight, m.syncSpinner.View())
	} else {
		bg = m.renderVaultContent(width, contentHeight)
	}

	var card string
	switch m.overlay {
	case overlayReprompt:
		card = m.renderReprompt(width, contentHeight)
	case overlayGenerator:
		card = m.gen.View(width)
	case overlayThemePicker:
		card = m.picker.View()
	case overlayHelp:
		card = m.help.View()
	case overlayQuit:
		card = renderQuitConfirm()
	case overlayNewItem:
		card = renderNewItem()
	case overlayDiscard:
		card = renderDiscard()
	case overlayDelete:
		card = m.renderDelete(width)
	default:
		return bg
	}
	return ui.RenderOverlay(bg, card, width, contentHeight)
}

// maxFilterHintW is the widest filter text that the footer shows.
const maxFilterHintW = 20

// FooterContent returns hints and status for the footer bar.
func (m VaultModel) FooterContent() ([]ui.HintBinding, string) {
	switch m.overlay {
	case overlayReprompt:
		return m.repromptHints(), ""
	case overlayGenerator:
		return m.gen.FooterHints(), ""
	case overlayThemePicker:
		return m.picker.FooterHints(), ""
	case overlayHelp:
		return m.help.FooterHints(), ""
	case overlayQuit:
		return quitHints(), ""
	case overlayNewItem:
		return newItemHints(), ""
	case overlayDiscard:
		return discardHints(), ""
	case overlayDelete:
		return m.deleteHints(), ""
	}
	if m.formOpen {
		return m.form.hints(), ""
	}

	// The key labels come from the bindings. The footer has little room,
	// so it has its own short descriptions.
	k, lk := m.keymap, m.list.KeyMap
	var hints []ui.HintBinding
	switch {
	case m.list.SettingFilter():
		// updateListKey moves the cursor on up and down.
		hints = []ui.HintBinding{
			ui.Hint(lk.CancelWhileFiltering, "clear"),
			ui.Hint(lk.AcceptWhileFiltering, "confirm"),
			{Key: ui.Glyphs.UpDown(), Desc: "navigate"},
		}
	case m.list.FilterState() == list.FilterApplied:
		// Show that the list is filtered, and how to clear the filter. The
		// filter text takes space, so this list is shorter than the normal
		// one, and help and quit come right after the filter hints.
		filter := ui.Truncate(ui.SanitizeLine(m.list.FilterValue()), maxFilterHintW)
		hints = []ui.HintBinding{
			{Key: "filter:", Desc: filter},
			ui.Hint(lk.ClearFilter, "clear"),
			ui.Hint(k.Help, "help"),
			ui.Hint(k.Quit, "quit"),
			ui.Hint(k.Copy, "pwd"),
			ui.Hint(k.CopyTOTP, "totp"),
		}
	default:
		// The footer drops hints from the end when they do not fit, so
		// help and quit come right after the copy keys.
		hints = []ui.HintBinding{
			ui.Hint(k.Navigate, "navigate"),
			ui.Hint(k.Filter, "search"),
			ui.Hint(k.Copy, "pwd"),
			ui.Hint(k.CopyTOTP, "totp"),
			ui.Hint(k.Help, "help"),
			ui.Hint(k.Quit, "quit"),
			ui.Hint(k.New, "new"),
			ui.Hint(k.Edit, "edit"),
			ui.Hint(k.Delete, "trash"),
			ui.Hint(k.ToggleGrouping, "group"),
			ui.Hint(k.DrawerToggle, "drawer"),
			ui.Hint(k.Generate, "gen"),
			ui.Hint(k.ThemePicker, "theme"),
		}
	}

	return hints, m.footerStatus()
}

// View implements tea.Model (kept for interface compliance).
func (m VaultModel) View() tea.View {
	return tea.NewView(m.ViewContent(m.width, m.height))
}

func tickTOTP() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return TOTPTickMsg{}
	})
}
