package screens

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/session"
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
	list     list.Model
	keymap   ui.VaultKeyMap
	help     help.Model
	showHelp bool
	sess     *session.State

	rawItems []bwcmd.Item
	groups   groupState

	toast       string
	toastTime   time.Time
	syncing     bool
	syncSpinner spinner.Model

	totpCode     string
	totpSecsLeft int
	totpItemID   string

	drawerScroll int

	currentTheme    string
	showThemePicker bool
	themeForm       *huh.Form
	selectedTheme   *string

	showGenerator bool
	genMode       string // "password" or "passphrase"
	genPassword   string // current generated output
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

	l := list.New(toListItems(items), VaultDelegate{}, width, height-ui.DrawerHeight)
	l.Title = ""
	l.SetShowTitle(false)
	l.SetShowFilter(true)
	l.SetShowStatusBar(true)
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
		help:         help.New(),
		sess:         sess,
		rawItems:     items,
		groups:       newGroupState(),
		syncSpinner:  ss,
		currentTheme: ui.CurrentTheme,
		width:        width,
		height:       height,
	}
	return m
}

func (m VaultModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, tickTOTP())
	if item := m.selectedItem(); item != nil && item.Login != nil && item.Login.Totp != "" {
		cmds = append(cmds, bwcmd.GetTOTP(m.sess.Token, item.ID))
	}
	return tea.Batch(cmds...)
}

func (m VaultModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-ui.DrawerHeight)
		return m, nil

	case tea.KeyPressMsg:
		if m.showGenerator {
			return m.updateGenerator(msg)
		}
		if m.showThemePicker {
			return m.updateThemePicker(msg)
		}
		if m.showHelp {
			if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
				m.showHelp = false
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
		prevIdx := m.list.Index()
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
		// Flatten groups while filtering, restore on filter clear.
		if fs := m.list.FilterState(); fs != prevFilter && m.groups.enabled {
			switch fs {
			case list.Filtering:
				cmds = append(cmds, m.list.SetItems(toListItems(m.rawItems)))
			case list.Unfiltered:
				cmds = append(cmds, m.rebuildListItems())
			}
		}
		if m.list.Index() != prevIdx {
			cmds = append(cmds, m.onCursorChange())
		}
		return m, tea.Batch(cmds...)

	case bwcmd.PasswordResult:
		if msg.Err != nil {
			m.setToast("Error: " + msg.Err.Error())
			return m, nil
		}
		return m, session.CopyToClipboard(msg.Password, session.CopyFieldPassword)

	case bwcmd.TOTPResult:
		if msg.Err != nil {
			m.totpCode = ""
			return m, nil
		}
		m.totpCode = msg.Code
		m.totpSecsLeft = 30 - int(time.Now().Unix()%30)
		return m, nil

	case session.CopiedMsg:
		if msg.Err != nil {
			m.setToast("Copy failed: " + msg.Err.Error())
			return m, nil
		}
		label := "Password"
		switch msg.Field {
		case session.CopyFieldTOTP:
			label = "TOTP"
		case session.CopyFieldUsername:
			label = "Username"
		}
		m.setToast(label + " copied — clears in 60s")
		return m, session.ScheduleClipboardClear()

	case session.ClipboardClearedMsg:
		session.ClearClipboard()
		if time.Since(m.toastTime) >= 59*time.Second {
			m.toast = ""
		}
		return m, nil

	case TOTPTickMsg:
		m.totpSecsLeft--
		if m.totpSecsLeft <= 0 {
			if item := m.selectedItem(); item != nil && item.Login != nil && item.Login.Totp != "" {
				m.totpSecsLeft = 30
				return m, tea.Batch(tickTOTP(), bwcmd.GetTOTP(m.sess.Token, item.ID))
			}
		}
		return m, tickTOTP()

	case bwcmd.SyncResult:
		m.syncing = false
		if msg.Err != nil {
			m.setToast("Sync failed: " + msg.Err.Error())
			return m, nil
		}
		m.sess.LastSync = time.Now()
		m.setToast("Vault synced")
		return m, bwcmd.FetchItems(m.sess.Token)

	case bwcmd.ItemsResult:
		if msg.Err != nil {
			m.setToast("Load failed: " + msg.Err.Error())
			return m, nil
		}
		m.rawItems = msg.Items
		cmd := m.rebuildListItems()
		return m, cmd

	case bwcmd.GenerateResult:
		if msg.Err != nil {
			m.genPassword = "Error: " + msg.Err.Error()
		} else {
			m.genPassword = msg.Password
		}
		return m, nil

	case spinner.TickMsg:
		if m.syncing {
			var cmd tea.Cmd
			m.syncSpinner, cmd = m.syncSpinner.Update(msg)
			return m, cmd
		}
	}

	if m.showThemePicker && m.themeForm != nil {
		return m.updateThemePicker(msg)
	}

	// Forward non-key messages to the list (e.g. filter debounce timers).
	prevIdx := m.list.Index()
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	if m.list.Index() != prevIdx {
		cmds = append(cmds, m.onCursorChange())
	}
	return m, tea.Batch(cmds...)
}

// handleActionKeys processes app-specific keybinds (copy, sync, lock, etc.).
// Returns (model, cmd, true) if handled, or (zero, nil, false) to let the list handle the key.
func (m VaultModel) handleActionKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keymap.Copy):
		if item := m.selectedItem(); item != nil {
			if item.Type == bwcmd.ItemTypeIdentity && item.Identity != nil && item.Identity.SSN != "" {
				return m, session.CopyToClipboard(item.Identity.SSN, session.CopyFieldPassword), true
			}
			if item.Type == bwcmd.ItemTypeSSHKey && item.SSHKey != nil && item.SSHKey.PrivateKey != "" {
				return m, session.CopyToClipboard(item.SSHKey.PrivateKey, session.CopyFieldPassword), true
			}
			return m, bwcmd.GetPassword(m.sess.Token, item.ID), true
		}

	case key.Matches(msg, m.keymap.CopyTOTP):
		if m.totpCode != "" {
			return m, session.CopyToClipboard(m.totpCode, session.CopyFieldTOTP), true
		}

	case key.Matches(msg, m.keymap.CopyUsername):
		if item := m.selectedItem(); item != nil {
			if item.Login != nil {
				return m, session.CopyToClipboard(item.Login.Username, session.CopyFieldUsername), true
			}
			if item.Identity != nil && item.Identity.Email != "" {
				return m, session.CopyToClipboard(item.Identity.Email, session.CopyFieldUsername), true
			}
			if item.SSHKey != nil && item.SSHKey.PublicKey != "" {
				return m, session.CopyToClipboard(item.SSHKey.PublicKey, session.CopyFieldUsername), true
			}
		}

	case key.Matches(msg, m.keymap.OpenURL):
		if item := m.selectedItem(); item != nil && item.Login != nil && len(item.Login.URIs) > 0 {
			url := item.Login.URIs[0].URI
			cmd := exec.Command("xdg-open", url)
			_ = cmd.Start()
		}
		return m, nil, true

	case key.Matches(msg, m.keymap.Sync):
		m.syncing = true
		m.setToast("Syncing…")
		return m, tea.Batch(m.syncSpinner.Tick, bwcmd.Sync(m.sess.Token)), true

	case key.Matches(msg, m.keymap.Lock):
		return m, func() tea.Msg { return LockMsg{} }, true

	case key.Matches(msg, m.keymap.Help):
		m.showHelp = !m.showHelp
		return m, nil, true

	case key.Matches(msg, m.keymap.Generate):
		model, cmd := m.openGenerator()
		return model, cmd, true

	case key.Matches(msg, m.keymap.CycleTheme):
		model, cmd := m.openThemePicker()
		return model, cmd, true

	case key.Matches(msg, m.keymap.Quit):
		return m, func() tea.Msg { return QuitMsg{} }, true

	case key.Matches(msg, m.keymap.ScrollDown):
		m.drawerScroll++
		return m, nil, true

	case key.Matches(msg, m.keymap.ScrollUp):
		if m.drawerScroll > 0 {
			m.drawerScroll--
		}
		return m, nil, true

	case key.Matches(msg, m.keymap.ToggleGrouping):
		m.groups.toggleGrouping()
		cmd := m.rebuildListItems()
		m.setToast(groupToastMessage(m.groups.enabled))
		return m, cmd, true

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

func (m *VaultModel) onCursorChange() tea.Cmd {
	m.drawerScroll = 0
	m.totpCode = ""
	m.totpSecsLeft = 0
	m.totpItemID = ""

	item := m.selectedItem()
	if item != nil && item.Login != nil && item.Login.Totp != "" {
		m.totpItemID = item.ID
		return bwcmd.GetTOTP(m.sess.Token, item.ID)
	}
	return nil
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

func (m *VaultModel) rebuildListItems() tea.Cmd {
	return m.list.SetItems(buildGroupedItems(m.rawItems, m.groups))
}

func (m *VaultModel) setToast(msg string) {
	m.toast = msg
	m.toastTime = time.Now()
}

func (m VaultModel) openThemePicker() (tea.Model, tea.Cmd) {
	selected := m.currentTheme
	m.selectedTheme = &selected
	options := make([]huh.Option[string], len(ui.ThemeNames))
	for i, name := range ui.ThemeNames {
		options[i] = huh.NewOption(name, name)
	}
	m.themeForm = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose theme").
				Options(options...).
				Value(m.selectedTheme),
		),
	).WithWidth(40)
	if ui.HuhTheme != nil {
		m.themeForm = m.themeForm.WithTheme(ui.HuhTheme)
	}
	m.showThemePicker = true
	return m, m.themeForm.Init()
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
		m.setToast("Theme: " + *m.selectedTheme)
		return m, nil
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
	m.genPassword = "Generating…"
	return m, bwcmd.Generate(m.genArgs()...)
}

func (m VaultModel) updateGenerator(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showGenerator = false
		return m, nil
	case "enter", "r":
		m.genPassword = "Generating…"
		return m, bwcmd.Generate(m.genArgs()...)
	case "c":
		if m.genPassword != "" && m.genPassword != "Generating…" {
			return m, session.CopyToClipboard(m.genPassword, session.CopyFieldPassword)
		}
	case "m", "tab":
		if m.genMode == "password" {
			m.genMode = "passphrase"
		} else {
			m.genMode = "password"
		}
		m.genPassword = "Generating…"
		return m, bwcmd.Generate(m.genArgs()...)
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
		m.genPassword = "Generating…"
		return m, bwcmd.Generate(m.genArgs()...)
	case "-":
		if m.genMode == "password" {
			if m.genLength > 5 {
				m.genLength--
			}
		} else {
			if m.genWords > 2 {
				m.genWords--
			}
		}
		m.genPassword = "Generating…"
		return m, bwcmd.Generate(m.genArgs()...)
	case "1":
		m.genUppercase = !m.genUppercase
		m.genPassword = "Generating…"
		return m, bwcmd.Generate(m.genArgs()...)
	case "2":
		m.genLowercase = !m.genLowercase
		m.genPassword = "Generating…"
		return m, bwcmd.Generate(m.genArgs()...)
	case "3":
		m.genNumbers = !m.genNumbers
		m.genPassword = "Generating…"
		return m, bwcmd.Generate(m.genArgs()...)
	case "4":
		m.genSpecial = !m.genSpecial
		m.genPassword = "Generating…"
		return m, bwcmd.Generate(m.genArgs()...)
	}
	return m, nil
}

func (m *VaultModel) genArgs() []string {
	if m.genMode == "passphrase" {
		args := []string{"--passphrase", "--words", fmt.Sprintf("%d", m.genWords), "--separator", m.genSeparator}
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

func (m VaultModel) renderGeneratorCard() string {
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
		fmt.Fprintf(&b, "Words       %d         Separator  %s\n", m.genWords, m.genSeparator)
		fmt.Fprintf(&b, "Capitalize  %s         Include #  %s\n",
			checkMark(m.genCapitalize), checkMark(m.genIncludeNum))
	}

	b.WriteString("\n")

	// Generated output.
	b.WriteString(ui.StyleTitle.Render(m.genPassword))

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

	if m.showGenerator {
		return ui.RenderOverlay(bg, m.renderGeneratorCard(), width, contentHeight)
	}
	if m.showThemePicker && m.themeForm != nil {
		return ui.RenderOverlay(bg, m.themeForm.View(), width, contentHeight)
	}
	if m.showHelp {
		return ui.RenderOverlay(bg, m.help.View(m.keymap), width, contentHeight)
	}

	return bg
}

// renderVaultContent renders the normal vault layout (list + drawer).
func (m VaultModel) renderVaultContent(width, contentHeight int) string {
	drawer := ui.RenderDrawer(ui.DrawerProps{
		Item:         m.selectedItem(),
		TOTPCode:     m.totpCode,
		TOTPSecsLeft: m.totpSecsLeft,
		Width:        width,
		ScrollOffset: m.drawerScroll,
	})

	m.list.SetSize(width, contentHeight-ui.DrawerHeight)
	listView := m.list.View()

	return lipgloss.JoinVertical(lipgloss.Left, listView, drawer)
}

// FooterContent returns hints and status for the footer bar.
func (m VaultModel) FooterContent() (hints, status string) {
	if m.showGenerator {
		hints = "enter regen · +/- length · m mode · 1-4 toggles · c copy · esc close"
		return hints, ""
	}
	if m.showThemePicker {
		hints = "enter select · esc cancel"
		return hints, ""
	}
	if m.showHelp {
		hints = "esc close"
		return hints, ""
	}
	if m.list.SettingFilter() {
		hints = "esc clear · enter confirm · ↑/↓ navigate"
	} else {
		hints = "j/k navigate · / search · c pwd · t totp · ctrl+g group · p gen · T theme · ? help · q quit"
	}

	toast := m.toast
	if m.syncing {
		toast = m.syncSpinner.View() + " " + toast
	}
	if toast != "" {
		status = ui.StyleToast.Render(toast)
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
