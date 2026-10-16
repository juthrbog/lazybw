package screens

import (
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
		_, _ = fmt.Fprint(w, ui.RenderGroupRow(v.BaseKey, v.Count, v.Expanded, selected, m.Width()))
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

// VaultModel is the main vault browsing screen.
type VaultModel struct {
	list   list.Model
	keymap ui.VaultKeyMap
	help   ui.HelpOverlay
	sess   *session.State

	rawItems []bwcmd.Item
	groups   groupState

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

	currentTheme    string
	showThemePicker bool
	themeForm       *huh.Form
	themeSelectH    int // height of the select in themeForm
	selectedTheme   *string

	confirmingQuit bool

	// Master password re-prompt. verifiedID is the ID of the selected item
	// after the user entered the master password for it. checking is true
	// while a password check runs, and checkSeq numbers the checks.
	reprompt   repromptState
	verifiedID string
	checking   bool
	checkSeq   uint64

	showGenerator bool
	genMode       string // "password" or "passphrase"
	genPassword   string // current generated output
	genErr        string // the error of the latest generate request
	genSeq        uint64 // sequence number of the latest generate request
	genLength     int
	genUppercase  bool
	genLowercase  bool
	genNumbers    bool
	genSpecial    bool
	genWords      int
	genSeparator  string
	genCapitalize bool
	genIncludeNum bool

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
	l := list.New(toListItems(items), VaultDelegate{}, width, height-drawerH)
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

	// Compute the code for the initially selected item now, so that the
	// drawer does not show "loading…" until the first tick.
	m.loadTOTP()

	return m
}

// Clear scrubs sensitive data from the vault model. It zeros []byte
// secrets in-place and drops references to sensitive string fields so
// the GC can collect the underlying memory sooner.
func (m *VaultModel) Clear() {
	secutil.ZeroItems(m.rawItems)
	m.rawItems = nil
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
	m.genPassword = ""
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
		listH := msg.Height - m.drawerHeight
		if m.drawerOverlay {
			listH = msg.Height
		}
		m.list.SetSize(msg.Width, listH)
		m.help.Resize(msg.Width, msg.Height-ui.ChromeHeight)
		if m.reprompt.open {
			m.reprompt.form = m.reprompt.form.WithWidth(repromptFormWidth(msg.Width))
		}
		// Rebuild the theme picker when the new height changes the height
		// of its select. Form.Init asks for the window size, so a rebuild
		// on each resize message would never stop.
		if m.showThemePicker && themeSelectHeight(msg.Height) != m.themeSelectH {
			return m, m.buildThemeForm()
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.reprompt.open {
			return m.updateReprompt(msg)
		}
		if m.showGenerator {
			return m.updateGenerator(msg)
		}
		if m.showThemePicker {
			return m.updateThemePicker(msg)
		}
		if m.help.Visible() {
			var cmd tea.Cmd
			m.help, cmd = m.help.Update(msg)
			return m, cmd
		}
		if m.confirmingQuit {
			switch msg.String() {
			case "y":
				m.confirmingQuit = false
				return m, func() tea.Msg { return QuitMsg{} }
			case "n", "esc":
				m.confirmingQuit = false
				return m, nil
			}
			return m, nil
		}
		// When not filtering, intercept our action keys before the list.
		if !m.list.SettingFilter() {
			if model, cmd, handled := m.handleActionKeys(msg); handled {
				return model, cmd
			}
		}
		// Forward to list for navigation, filtering, etc.
		prevFilter := m.list.FilterState()
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
		// Flatten groups while filtering, restore on filter clear.
		if fs := m.list.FilterState(); fs != prevFilter && m.groups.enabled {
			switch fs {
			case list.Filtering:
				cmds = append(cmds, m.setListItems(toListItems(m.rawItems)))
			case list.Unfiltered:
				cmds = append(cmds, m.rebuildListItems())
			}
		}
		return m, tea.Batch(cmds...)

	case session.CopiedMsg:
		// The root model schedules and runs the clipboard clear. The
		// terminal does not answer an OSC 52 write, so the toast says only
		// that lazybw sent the value.
		m.setCopyToast(fmt.Sprintf("%s %s sent to clipboard · clears in %ds",
			ui.GlyphCopy, copyLabel(msg.Field), int(session.ClipboardClearDelay.Seconds())))
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
		return m, m.replaceItems(msg.Items)

	case OpenURLResult:
		if msg.Err != nil {
			return m, m.setErrorToast("Open URL failed: " + msg.Err.Error())
		}
		return m, nil

	case bwcmd.GenerateResult:
		if msg.Seq != m.genSeq {
			return m, nil // a newer request is in flight
		}
		// The error goes in its own field, so that c cannot copy it.
		m.genPassword = msg.Password
		m.genErr = userError(msg.Err, actionOther)
		return m, nil

	case spinner.TickMsg:
		if m.syncing {
			var cmd tea.Cmd
			m.syncSpinner, cmd = m.syncSpinner.Update(msg)
			return m, cmd
		}
	}

	// The forms complete over internal messages, so they get the other
	// messages too. A sync can refresh the list while the password prompt
	// shows, and the list then needs its filter results, so the list gets
	// the messages as well.
	if m.reprompt.open {
		updated, cmd := m.updateReprompt(msg)
		if vm, ok := updated.(VaultModel); ok {
			m = vm
		}
		cmds = append(cmds, cmd)
	}
	if m.showThemePicker && m.themeForm != nil {
		return m.updateThemePicker(msg)
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
		return m, nil, true

	case key.Matches(msg, m.keymap.Generate):
		model, cmd := m.openGenerator()
		return model, cmd, true

	case key.Matches(msg, m.keymap.CycleTheme):
		model, cmd := m.openThemePicker()
		return model, cmd, true

	case key.Matches(msg, m.keymap.Quit):
		m.confirmingQuit = true
		return m, nil, true

	// Only secure notes scroll. For other items the limit is 0, so J and
	// K do nothing. A resize or a drawer change can lower the limit, so K
	// clamps first and then moves.
	case key.Matches(msg, m.keymap.ScrollDown):
		m.drawerScroll = min(m.drawerScroll+1, m.maxDrawerScroll())
		return m, nil, true

	case key.Matches(msg, m.keymap.ScrollUp):
		m.drawerScroll = max(0, min(m.drawerScroll, m.maxDrawerScroll())-1)
		return m, nil, true

	case key.Matches(msg, m.keymap.DrawerGrow):
		if !m.drawerOverlay {
			m.drawerHeight = m.clampDrawerHeight(m.drawerHeight + 1)
			m.resizeList()
		}
		return m, nil, true

	case key.Matches(msg, m.keymap.DrawerShrink):
		if !m.drawerOverlay {
			m.drawerHeight = m.clampDrawerHeight(m.drawerHeight - 1)
			m.resizeList()
		}
		return m, nil, true

	case key.Matches(msg, m.keymap.DrawerReset):
		if !m.drawerOverlay {
			m.drawerHeight = m.clampDrawerHeight(ui.DefaultDrawerHeight)
			m.resizeList()
		}
		return m, nil, true

	case key.Matches(msg, m.keymap.DrawerToggle):
		m.drawerOverlay = !m.drawerOverlay
		m.resizeList()
		if m.drawerOverlay {
			return m, m.setToast("Drawer: overlay"), true
		}
		return m, m.setToast("Drawer: inline"), true

	case key.Matches(msg, m.keymap.ToggleGrouping):
		m.groups.toggleGrouping()
		cmd := m.rebuildListItems()
		return m, tea.Batch(cmd, m.setToast(groupToastMessage(m.groups.enabled))), true

	case key.Matches(msg, m.keymap.ToggleExpand):
		if sel := m.list.SelectedItem(); sel != nil {
			if gh, ok := sel.(GroupHeaderItem); ok {
				m.groups.toggle(gh.BaseKey)
				cmd := m.rebuildListItems()
				return m, cmd, true
			}
		}
	}

	return m, nil, false
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
	if m.reprompt.open {
		m.closeReprompt()
	}
	m.loadTOTP()
}

// loadTOTP parses the TOTP key of the selected item and computes its code.
// It scrubs the params of the previous item.
func (m *VaultModel) loadTOTP() {
	if m.totpParams != nil {
		m.totpParams.Clear()
		m.totpParams = nil
	}
	m.totpCode = ""
	m.totpInvalid = false
	m.totpItemID = ""

	item := m.selectedItem()
	if item != nil {
		m.totpItemID = item.ID
	}
	if item != nil && item.Login != nil && item.Login.Totp != "" {
		p, err := totp.Parse(item.Login.Totp)
		if err == nil {
			m.totpParams = &p
		} else {
			m.totpInvalid = true
		}
	}
	m.computeTOTP()
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

// computeTOTP recomputes the TOTP code and countdown from the cached
// params. It is a no-op when no TOTP params are available.
func (m *VaultModel) computeTOTP() {
	if m.totpParams == nil {
		m.totpCode = ""
		m.totpSecsLeft = 0
		return
	}
	m.totpCode = totp.Code(*m.totpParams)
	m.totpSecsLeft = totp.SecsLeft(m.totpParams.Period)
}

// totpPeriod returns the TOTP period of the selected item, or 0 when it
// has no TOTP.
func (m VaultModel) totpPeriod() int {
	if m.totpParams == nil {
		return 0
	}
	return m.totpParams.Period
}

// HeaderInfo returns the item count and selected item type name for the header.
func (m VaultModel) HeaderInfo() (int, string) {
	count := len(m.list.Items())
	if item := m.selectedItem(); item != nil {
		return count, ui.ItemTypeName(item.Type)
	}
	return count, ""
}

func (m *VaultModel) rebuildListItems() tea.Cmd {
	return m.setListItems(buildGroupedItems(m.rawItems, m.groups))
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
		m.list, _ = m.list.Update(fm)
		return nil
	}
	return func() tea.Msg { return msg }
}

// replaceItems swaps in a fresh item list after a sync. It keeps the
// cursor on the same item when that item still exists, refreshes the
// cached TOTP params, and scrubs the old items.
func (m *VaultModel) replaceItems(items []bwcmd.Item) tea.Cmd {
	selectedID := ""
	if item := m.selectedItem(); item != nil {
		selectedID = item.ID
	}
	old := m.rawItems
	m.rawItems = items
	// While a filter is active, the list holds flat rows (see Update).
	var cmd tea.Cmd
	if m.groups.enabled && m.list.FilterState() != list.Unfiltered {
		cmd = m.setListItems(toListItems(m.rawItems))
	} else {
		cmd = m.rebuildListItems()
	}
	if selectedID == "" || !m.selectItemByID(selectedID) {
		m.clampCursor()
	}
	// The TOTP key of the same item can change in a sync, so reload it.
	// A different item also gets a fresh drawer.
	if item := m.selectedItem(); item != nil && item.ID == selectedID {
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

// selectItemByID moves the cursor to the visible row of the item with the
// given ID. It reports false when no visible row has that ID.
func (m *VaultModel) selectItemByID(id string) bool {
	for i, li := range m.list.VisibleItems() {
		if v, ok := li.(VaultItem); ok && v.ID == id {
			m.list.Select(i)
			return true
		}
	}
	return false
}

// clampCursor moves the cursor to the last visible row when the list got
// shorter than the cursor index.
func (m *VaultModel) clampCursor() {
	if n := len(m.list.VisibleItems()); n > 0 && m.list.Index() >= n {
		m.list.Select(n - 1)
	}
}

func (m *VaultModel) clampDrawerHeight(h int) int {
	return clampDrawerHeight(h, m.height)
}

// minListRows is the number of rows that the list always draws: its title
// line, where the filter input shows, and one item. The bubbles list draws
// them even when its height is smaller.
const minListRows = 2

// clampDrawerHeight limits a drawer height h to half of the screen height.
// The drawer keeps at least 3 rows. On a screen that is too short for that,
// the drawer gets the rows of the content area that the list leaves, and at
// least 1 row.
func clampDrawerHeight(h, height int) int {
	const minDrawerHeight = 3
	maxDrawerHeight := max(height/2, minDrawerHeight)
	// A taller drawer would make the list and the drawer together taller
	// than the content area, and the frame would cut the drawer.
	maxDrawerHeight = min(maxDrawerHeight, height-ui.ChromeHeight-minListRows)
	return max(1, min(max(h, minDrawerHeight), maxDrawerHeight))
}

func (m *VaultModel) resizeList() {
	listH := m.height - m.drawerHeight
	if m.drawerOverlay {
		listH = m.height
	}
	m.list.SetSize(m.width, listH)
}

// How long a toast shows. A copy toast stays until the clipboard clear.
// Tests shorten them.
var (
	toastDuration      = 4 * time.Second
	errorToastDuration = 8 * time.Second
)

// toastExpiredMsg ends toast number gen. Each toast gets a higher number,
// so the timer of an older toast does not end a newer one.
type toastExpiredMsg struct{ gen uint64 }

// setToast shows msg in the footer. It returns the command that ends the
// toast after toastDuration.
func (m *VaultModel) setToast(msg string) tea.Cmd {
	m.showToast(msg, false, false)
	return m.expireToast(toastDuration)
}

// setErrorToast shows an error in the footer. The message often holds bw
// output, so it is sanitized to one line. It returns the command that ends
// the toast after errorToastDuration.
func (m *VaultModel) setErrorToast(msg string) tea.Cmd {
	m.showToast(ui.GlyphError+" "+ui.SanitizeLine(msg), true, false)
	return m.expireToast(errorToastDuration)
}

// setCopyToast shows a copy toast. The root model clears the clipboard
// after session.ClipboardClearDelay, and the toast ends then.
func (m *VaultModel) setCopyToast(msg string) {
	m.showToast(msg, false, true)
}

// clearToast removes the toast, so that the footer shows the sync age.
func (m *VaultModel) clearToast() {
	m.showToast("", false, false)
}

func (m *VaultModel) showToast(msg string, isError, isCopy bool) {
	m.toastGen++
	m.toast = msg
	m.toastIsError = isError
	m.toastIsCopy = isCopy
}

// expireToast returns the command that ends the current toast after d.
func (m *VaultModel) expireToast(d time.Duration) tea.Cmd {
	gen := m.toastGen
	return tea.Tick(d, func(time.Time) tea.Msg { return toastExpiredMsg{gen: gen} })
}

// copyLabel returns the name of the field f for the copy toast.
func copyLabel(f session.CopyField) string {
	switch f {
	case session.CopyFieldTOTP:
		return "TOTP"
	case session.CopyFieldUsername:
		return "Username"
	case session.CopyFieldNote:
		return "Note"
	case session.CopyFieldCardNumber:
		return "Card number"
	case session.CopyFieldCVV:
		return "Security code"
	case session.CopyFieldSSN:
		return "SSN"
	case session.CopyFieldPrivateKey:
		return "SSH private key"
	case session.CopyFieldPublicKey:
		return "Public key"
	case session.CopyFieldEmail:
		return "Email"
	}
	return "Password"
}

func (m VaultModel) openThemePicker() (tea.Model, tea.Cmd) {
	selected := m.currentTheme
	m.selectedTheme = &selected
	m.showThemePicker = true
	return m, m.buildThemeForm()
}

// buildThemeForm builds the theme picker for the current screen height and
// returns its init command. The cursor starts on m.selectedTheme, so a
// rebuild keeps the theme that the user moved to.
func (m *VaultModel) buildThemeForm() tea.Cmd {
	options := make([]huh.Option[string], len(ui.ThemeNames))
	for i, name := range ui.ThemeNames {
		options[i] = huh.NewOption(name, name)
	}
	m.themeSelectH = themeSelectHeight(m.height)
	m.themeForm = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose theme").
				Options(options...).
				Value(m.selectedTheme).
				Height(m.themeSelectH),
		),
	).WithWidth(40)
	if ui.HuhTheme != nil {
		m.themeForm = m.themeForm.WithTheme(ui.HuhTheme)
	}
	return m.themeForm.Init()
}

// themeSelectHeight returns the height of the theme select for a screen of
// the given height: the title and one row for each theme, or fewer rows on
// a short screen. A select with fewer rows scrolls, so that the cursor stays
// visible. The select must fit in the overlay card with the 2 rows that the
// form adds below it, a blank line and its help line.
func themeSelectHeight(height int) int {
	const formHelpRows = 2
	rows := height - ui.ChromeHeight - ui.OverlayEdgeHeight - formHelpRows
	return max(2, min(len(ui.ThemeNames)+1, rows))
}

func (m VaultModel) updateThemePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "esc" {
		m.showThemePicker = false
		return m, nil
	}

	form, cmd := m.themeForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.themeForm = f
	}

	switch m.themeForm.State {
	case huh.StateCompleted:
		m.showThemePicker = false
		ui.ApplyTheme(*m.selectedTheme)
		m.currentTheme = *m.selectedTheme
		return m, m.setToast("Theme: " + *m.selectedTheme)
	case huh.StateAborted:
		m.showThemePicker = false
		return m, nil
	}

	return m, cmd
}

func (m VaultModel) openGenerator() (tea.Model, tea.Cmd) {
	m.showGenerator = true
	m.genMode = "password"
	m.genLength = 20
	m.genUppercase = true
	m.genLowercase = true
	m.genNumbers = true
	m.genSpecial = true
	m.genWords = 4
	m.genSeparator = "-"
	m.genCapitalize = true
	m.genIncludeNum = true
	return m, m.generate()
}

func (m VaultModel) updateGenerator(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showGenerator = false
		return m, nil
	case "enter", "r":
		return m, m.generate()
	case "c":
		if m.genErr == "" && m.genPassword != "" && m.genPassword != genPlaceholder {
			return m, session.CopyToClipboard(m.genPassword, session.CopyFieldPassword)
		}
	case "m", "tab":
		if m.genMode == "password" {
			m.genMode = "passphrase"
		} else {
			m.genMode = "password"
		}
		return m, m.generate()
	case "+", "=":
		if m.genMode == "password" {
			if m.genLength < 128 {
				m.genLength++
			}
		} else {
			if m.genWords < 20 {
				m.genWords++
			}
		}
		return m, m.generate()
	case "-":
		if m.genMode == "password" {
			if m.genLength > 5 {
				m.genLength--
			}
		} else {
			if m.genWords > genMinWords {
				m.genWords--
			}
		}
		return m, m.generate()
	case "1", "2", "3", "4":
		if m.genMode == "passphrase" {
			return m.togglePassphraseOption(msg.String())
		}
		switch msg.String() {
		case "1":
			m.genUppercase = !m.genUppercase
		case "2":
			m.genLowercase = !m.genLowercase
		case "3":
			m.genNumbers = !m.genNumbers
		case "4":
			m.genSpecial = !m.genSpecial
		}
		return m, m.generate()
	}
	return m, nil
}

// genSeparators are the word separators that key 3 cycles through in
// passphrase mode.
var genSeparators = []string{"-", "_", ".", " "}

// togglePassphraseOption changes the passphrase option of key k: 1 toggles
// the capitals, 2 toggles the number, and 3 moves to the next separator.
// Key 4 has no passphrase option.
func (m VaultModel) togglePassphraseOption(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "1":
		m.genCapitalize = !m.genCapitalize
	case "2":
		m.genIncludeNum = !m.genIncludeNum
	case "3":
		next := 0
		for i, sep := range genSeparators {
			if sep == m.genSeparator {
				next = (i + 1) % len(genSeparators)
			}
		}
		m.genSeparator = genSeparators[next]
	default:
		return m, nil
	}
	return m, m.generate()
}

// separatorName returns the separator as the generator card shows it.
func separatorName(sep string) string {
	if sep == " " {
		return "space"
	}
	return sep
}

// genPlaceholder is the generator output while bw generate runs.
const genPlaceholder = "Generating…"

// generate starts a new `bw generate` request. Each request gets a new
// sequence number, so that a slow, older result cannot replace a newer one.
func (m *VaultModel) generate() tea.Cmd {
	m.genSeq++
	m.genPassword = genPlaceholder
	m.genErr = ""
	return bwcmd.Generate(m.genSeq, m.genArgs()...)
}

func (m *VaultModel) genArgs() []string {
	if m.genMode == "passphrase" {
		// bw reads the word "space" as a space separator.
		args := []string{"--passphrase", "--words", fmt.Sprintf("%d", m.genWords), "--separator", separatorName(m.genSeparator)}
		if m.genCapitalize {
			args = append(args, "--capitalize")
		}
		if m.genIncludeNum {
			args = append(args, "--includeNumber")
		}
		return args
	}
	args := []string{"--length", fmt.Sprintf("%d", m.genLength)}
	if m.genUppercase {
		args = append(args, "--uppercase")
	}
	if m.genLowercase {
		args = append(args, "--lowercase")
	}
	if m.genNumbers {
		args = append(args, "--number")
	}
	if m.genSpecial {
		args = append(args, "--special")
	}
	return args
}

// genMaxWidth is the widest that a line of the generated value gets.
const genMaxWidth = 72

// genMinWords is the smallest word count for a passphrase. bw generate
// raises a smaller count to 3, so the card would show the wrong count.
const genMinWords = 3

// renderGeneratorCard renders the generator card for a content area of the
// given width. The generated value wraps to fit inside the card.
func (m VaultModel) renderGeneratorCard(width int) string {
	var b strings.Builder

	b.WriteString(ui.StyleTitle.Render("Password Generator"))
	b.WriteString("\n\n")

	// Mode indicator.
	if m.genMode == "password" {
		b.WriteString("Mode        ")
		b.WriteString(ui.StyleTitle.Render("[Password]"))
		b.WriteString("  ")
		b.WriteString(ui.StyleFaint.Render("Passphrase"))
		b.WriteString("\n")
	} else {
		b.WriteString("Mode        ")
		b.WriteString(ui.StyleFaint.Render("Password"))
		b.WriteString("  ")
		b.WriteString(ui.StyleTitle.Render("[Passphrase]"))
		b.WriteString("\n")
	}

	// Options.
	if m.genMode == "password" {
		fmt.Fprintf(&b, "Length      %d\n", m.genLength)
		fmt.Fprintf(&b, "Uppercase   %s   Lowercase  %s   Numbers  %s   Special  %s\n",
			checkMark(m.genUppercase), checkMark(m.genLowercase),
			checkMark(m.genNumbers), checkMark(m.genSpecial))
	} else {
		// The options show in the order of their keys 1 to 3, on rows
		// that stay narrow.
		fmt.Fprintf(&b, "Words       %d\n", m.genWords)
		fmt.Fprintf(&b, "Capitalize  %s   Include #  %s\n",
			checkMark(m.genCapitalize), checkMark(m.genIncludeNum))
		fmt.Fprintf(&b, "Separator   %s\n", separatorName(m.genSeparator))
	}

	b.WriteString("\n")

	// Generated output. A long password or passphrase wraps, so that all
	// of it shows inside the card border.
	contentW := max(1, min(width-ui.OverlayEdgeWidth, genMaxWidth))
	if m.genErr != "" {
		// errorBlock keeps a 2-cell margin on each side.
		b.WriteString(errorBlock(m.genErr, contentW+4, contentW))
	} else {
		b.WriteString(ui.StyleTitle.Render(ansi.Hardwrap(ui.SanitizeLine(m.genPassword), contentW, true)))
	}

	return b.String()
}

func checkMark(v bool) string {
	if v {
		return ui.StyleToast.Render("✓")
	}
	return ui.StyleFaint.Render("·")
}

// ViewContent renders the vault content for the root frame.
func (m VaultModel) ViewContent(width, contentHeight int) string {
	bg := m.renderVaultContent(width, contentHeight)

	if m.reprompt.open {
		return ui.RenderOverlay(bg, m.renderReprompt(width, contentHeight), width, contentHeight)
	}
	if m.showGenerator {
		return ui.RenderOverlay(bg, m.renderGeneratorCard(width), width, contentHeight)
	}
	if m.showThemePicker && m.themeForm != nil {
		return ui.RenderOverlay(bg, m.themeForm.View(), width, contentHeight)
	}
	if m.help.Visible() {
		return ui.RenderOverlay(bg, m.help.View(), width, contentHeight)
	}
	if m.confirmingQuit {
		return ui.RenderOverlay(bg, m.renderQuitConfirm(), width, contentHeight)
	}

	return bg
}

func (m VaultModel) renderQuitConfirm() string {
	keyStyle := lipgloss.NewStyle().Foreground(ui.ColorHighlight).Bold(true)

	var b strings.Builder
	b.WriteString(ui.StyleTitle.Render("Quit lazybw?"))
	b.WriteString("\n\n")
	b.WriteString("This will lock your vault and exit.")
	b.WriteString("\n\n")
	b.WriteString(keyStyle.Render("y"))
	b.WriteString(ui.StyleFaint.Render(" yes"))
	b.WriteString(ui.StyleFaint.Render(" · "))
	b.WriteString(keyStyle.Render("n"))
	b.WriteString(ui.StyleFaint.Render(" no"))
	return b.String()
}

// drawerProps returns the drawer props for a content area of the given
// size. The view and the scroll limit both use it, so that they count the
// same note lines.
func (m VaultModel) drawerProps(width, contentHeight int) ui.DrawerProps {
	props := ui.DrawerProps{
		Item:         m.selectedItem(),
		TOTPCode:     m.totpCode,
		TOTPSecsLeft: m.totpSecsLeft,
		TOTPPeriod:   m.totpPeriod(),
		TOTPInvalid:  m.totpInvalid,
		ScrollOffset: m.drawerScroll,
	}
	props.RepromptVerified = m.repromptVerified(props.Item)
	if m.drawerOverlay {
		props.Width = overlayDrawerWidth(width)
		props.AutoHeight = true
		// RenderOverlay cuts the card content to the rows inside the border
		// and the padding. Size the note page to those rows, so that J can
		// scroll to the last line. Height counts the header, and the page
		// keeps at least one note row.
		props.Height = max(2, min(ui.DefaultDrawerHeight, contentHeight-ui.OverlayEdgeHeight))
	} else {
		props.Width = width
		props.Height = m.drawerHeight
	}
	return props
}

// overlayDrawerWidth returns the drawer width in overlay mode.
func overlayDrawerWidth(width int) int {
	// Constrain overlay drawer width so it doesn't span the full terminal.
	w := width * 3 / 4
	if w > 80 {
		w = 80
	}
	if w < 40 {
		// Leave room for the card border and padding.
		w = width - ui.OverlayEdgeWidth
	}
	return w
}

// maxDrawerScroll returns the largest drawer scroll offset for the
// selected item.
func (m VaultModel) maxDrawerScroll() int {
	return ui.MaxScrollOffset(m.drawerProps(m.width, m.height-ui.ChromeHeight))
}

// renderVaultContent renders the normal vault layout (list + drawer).
func (m VaultModel) renderVaultContent(width, contentHeight int) string {
	drawer := ui.RenderDrawer(m.drawerProps(width, contentHeight))

	if m.drawerOverlay {
		m.list.SetSize(width, contentHeight)
		listView := m.list.View()
		return ui.RenderOverlay(listView, drawer, width, contentHeight)
	}

	m.list.SetSize(width, contentHeight-m.drawerHeight)
	listView := m.list.View()

	return lipgloss.JoinVertical(lipgloss.Left, listView, drawer)
}

// maxFilterHintW is the widest filter text that the footer shows.
const maxFilterHintW = 20

// FooterContent returns hints and status for the footer bar.
func (m VaultModel) FooterContent() ([]ui.HintBinding, string) {
	if m.reprompt.open {
		return m.repromptHints(), ""
	}
	if m.confirmingQuit {
		return []ui.HintBinding{
			{Key: "y", Desc: "yes"},
			{Key: "n", Desc: "no"},
		}, ""
	}
	if m.showGenerator {
		if m.genMode == "passphrase" {
			return []ui.HintBinding{
				{Key: "enter", Desc: "regen"},
				{Key: "+/-", Desc: "words"},
				{Key: "m", Desc: "mode"},
				{Key: "1-3", Desc: "options"},
				{Key: "c", Desc: "copy"},
				{Key: "esc", Desc: "close"},
			}, ""
		}
		return []ui.HintBinding{
			{Key: "enter", Desc: "regen"},
			{Key: "+/-", Desc: "length"},
			{Key: "m", Desc: "mode"},
			{Key: "1-4", Desc: "toggles"},
			{Key: "c", Desc: "copy"},
			{Key: "esc", Desc: "close"},
		}, ""
	}
	if m.showThemePicker {
		return []ui.HintBinding{
			{Key: "enter", Desc: "select"},
			{Key: "esc", Desc: "cancel"},
		}, ""
	}
	if m.help.Visible() {
		if m.help.Filtering() {
			return []ui.HintBinding{
				{Key: "enter", Desc: "done"},
				{Key: "esc", Desc: "clear"},
			}, ""
		}
		return []ui.HintBinding{
			{Key: "esc", Desc: "close"},
			{Key: "j/k", Desc: "scroll"},
			{Key: "/", Desc: "filter"},
		}, ""
	}

	var hints []ui.HintBinding
	switch {
	case m.list.SettingFilter():
		hints = []ui.HintBinding{
			{Key: "esc", Desc: "clear"},
			{Key: "enter", Desc: "confirm"},
			{Key: "↑/↓", Desc: "navigate"},
		}
	case m.list.FilterState() == list.FilterApplied:
		// Show that the list is filtered, and how to clear the filter. The
		// filter text takes space, so this list is shorter than the normal
		// one, and help and quit come right after the filter hints.
		filter := ui.Truncate(ui.SanitizeLine(m.list.FilterValue()), maxFilterHintW)
		hints = []ui.HintBinding{
			{Key: "filter:", Desc: filter},
			{Key: "esc", Desc: "clear"},
			{Key: "?", Desc: "help"},
			{Key: "q", Desc: "quit"},
			{Key: "c", Desc: "pwd"},
			{Key: "t", Desc: "totp"},
		}
	default:
		// The footer drops hints from the end when they do not fit, so
		// help and quit come right after the copy keys.
		hints = []ui.HintBinding{
			{Key: "j/k", Desc: "navigate"},
			{Key: "/", Desc: "search"},
			{Key: "c", Desc: "pwd"},
			{Key: "t", Desc: "totp"},
			{Key: "?", Desc: "help"},
			{Key: "q", Desc: "quit"},
			{Key: "ctrl+g", Desc: "group"},
			{Key: "d", Desc: "drawer"},
			{Key: "p", Desc: "gen"},
			{Key: "T", Desc: "theme"},
		}
	}

	var status string
	toast := m.toast
	if m.syncing {
		if toast == "" {
			toast = "Syncing…"
		}
		toast = m.syncSpinner.View() + " " + toast
	}
	if toast != "" {
		if m.toastIsError {
			status = ui.StyleError.Render(toast)
		} else {
			status = ui.StyleToast.Render(toast)
		}
	} else {
		status = ui.StyleFaint.Render(formatLastSync(m.sess.LastSync))
	}
	return hints, status
}

// View implements tea.Model (kept for interface compliance).
func (m VaultModel) View() tea.View {
	return tea.NewView(m.ViewContent(m.width, m.height))
}

func formatLastSync(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "synced just now"
	case d < time.Hour:
		return fmt.Sprintf("synced %dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("synced %dh ago", int(d.Hours()))
	}
}

func tickTOTP() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return TOTPTickMsg{}
	})
}
