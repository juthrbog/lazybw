package screens

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
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

type vaultMode int

const (
	modeNormal vaultMode = iota
	modeFilter
)

// VaultModel is the main vault browsing screen.
type VaultModel struct {
	items     []bwcmd.Item
	filtered  []bwcmd.Item
	cursor    int
	keymap    ui.VaultKeyMap
	help      help.Model
	showHelp  bool
	mode      vaultMode
	filterStr string
	sess      *session.State

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
	m := VaultModel{
		items:       items,
		filtered:    items,
		keymap:      ui.DefaultVaultKeyMap(),
		help:        help.New(),
		sess:        sess,
		syncSpinner:  ss,
		currentTheme: ui.CurrentTheme,
		width:        width,
		height:      height,
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

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
		if m.mode == modeFilter {
			return m.updateFilter(msg)
		}
		return m.updateNormal(msg)

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
			// Refetch TOTP code for new period.
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
		m.items = msg.Items
		m.applyFilter()
		return m, nil

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

	return m, nil
}

func (m VaultModel) updateNormal(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Down):
		m.moveCursor(1)
		return m, m.onCursorChange()

	case key.Matches(msg, m.keymap.Up):
		m.moveCursor(-1)
		return m, m.onCursorChange()

	case key.Matches(msg, m.keymap.Top):
		m.cursor = 0
		return m, m.onCursorChange()

	case key.Matches(msg, m.keymap.Bottom):
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		}
		return m, m.onCursorChange()

	case key.Matches(msg, m.keymap.Copy):
		if item := m.selectedItem(); item != nil {
			if item.Type == bwcmd.ItemTypeIdentity && item.Identity != nil && item.Identity.SSN != "" {
				return m, session.CopyToClipboard(item.Identity.SSN, session.CopyFieldPassword)
			}
			if item.Type == bwcmd.ItemTypeSSHKey && item.SSHKey != nil && item.SSHKey.PrivateKey != "" {
				return m, session.CopyToClipboard(item.SSHKey.PrivateKey, session.CopyFieldPassword)
			}
			return m, bwcmd.GetPassword(m.sess.Token, item.ID)
		}

	case key.Matches(msg, m.keymap.CopyTOTP):
		if m.totpCode != "" {
			return m, session.CopyToClipboard(m.totpCode, session.CopyFieldTOTP)
		}

	case key.Matches(msg, m.keymap.CopyUsername):
		if item := m.selectedItem(); item != nil {
			if item.Login != nil {
				return m, session.CopyToClipboard(item.Login.Username, session.CopyFieldUsername)
			}
			if item.Identity != nil && item.Identity.Email != "" {
				return m, session.CopyToClipboard(item.Identity.Email, session.CopyFieldUsername)
			}
			if item.SSHKey != nil && item.SSHKey.PublicKey != "" {
				return m, session.CopyToClipboard(item.SSHKey.PublicKey, session.CopyFieldUsername)
			}
		}

	case key.Matches(msg, m.keymap.OpenURL):
		if item := m.selectedItem(); item != nil && item.Login != nil && len(item.Login.URIs) > 0 {
			url := item.Login.URIs[0].URI
			cmd := exec.Command("xdg-open", url)
			_ = cmd.Start()
		}

	case key.Matches(msg, m.keymap.Filter):
		m.mode = modeFilter
		m.filterStr = ""
		return m, nil

	case key.Matches(msg, m.keymap.Sync):
		m.syncing = true
		m.setToast("Syncing…")
		return m, tea.Batch(m.syncSpinner.Tick, bwcmd.Sync(m.sess.Token))

	case key.Matches(msg, m.keymap.Lock):
		return m, func() tea.Msg { return LockMsg{} }

	case key.Matches(msg, m.keymap.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keymap.Generate):
		return m.openGenerator()

	case key.Matches(msg, m.keymap.CycleTheme):
		return m.openThemePicker()

	case key.Matches(msg, m.keymap.Quit):
		return m, func() tea.Msg { return QuitMsg{} }

	case key.Matches(msg, m.keymap.ScrollDown):
		m.drawerScroll++
		return m, nil

	case key.Matches(msg, m.keymap.ScrollUp):
		if m.drawerScroll > 0 {
			m.drawerScroll--
		}
		return m, nil
	}

	return m, nil
}

func (m VaultModel) updateFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.filterStr = ""
		m.applyFilter()
		return m, nil
	case "enter":
		m.mode = modeNormal
		return m, nil
	case "backspace":
		if len(m.filterStr) > 0 {
			m.filterStr = m.filterStr[:len(m.filterStr)-1]
			m.applyFilter()
		}
		return m, nil
	case "up":
		m.moveCursor(-1)
		return m, m.onCursorChange()
	case "down":
		m.moveCursor(1)
		return m, m.onCursorChange()
	default:
		if len(msg.String()) == 1 {
			m.filterStr += msg.String()
			m.applyFilter()
		}
		return m, nil
	}
}

func (m *VaultModel) applyFilter() {
	if m.filterStr == "" {
		m.filtered = m.items
	} else {
		var filtered []bwcmd.Item
		query := strings.ToLower(m.filterStr)
		for _, item := range m.items {
			if strings.Contains(strings.ToLower(item.Name), query) ||
				strings.Contains(strings.ToLower(item.Description()), query) {
				filtered = append(filtered, item)
			}
		}
		m.filtered = filtered
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m *VaultModel) moveCursor(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
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
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	return &m.filtered[m.cursor]
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

// renderVaultContent renders the normal vault layout (filter + list + drawer).
func (m VaultModel) renderVaultContent(width, contentHeight int) string {
	drawer := ui.RenderDrawer(ui.DrawerProps{
		Item:         m.selectedItem(),
		TOTPCode:     m.totpCode,
		TOTPSecsLeft: m.totpSecsLeft,
		Width:        width,
		ScrollOffset: m.drawerScroll,
	})

	listHeight := contentHeight - ui.DrawerHeight
	if m.mode == modeFilter {
		listHeight--
	}
	if listHeight < 1 {
		listHeight = 1
	}

	listView := m.renderList(listHeight)

	var sections []string
	if m.mode == modeFilter {
		filterBar := ui.StyleTitle.Render("/") + m.filterStr + "█"
		sections = append(sections, filterBar)
	}
	sections = append(sections, listView, drawer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
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
	if m.mode == modeFilter {
		hints = "esc clear · enter confirm · ↑/↓ navigate"
	} else {
		hints = "j/k navigate · / search · c pwd · t totp · p gen · T theme · ? help · q quit"
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

func (m VaultModel) renderList(height int) string {
	if len(m.filtered) == 0 {
		empty := ui.StyleFaint.Render("  No items found")
		lines := []string{empty}
		for len(lines) < height {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	// Compute scroll window.
	start := 0
	if m.cursor >= height {
		start = m.cursor - height + 1
	}
	end := start + height
	if end > len(m.filtered) {
		end = len(m.filtered)
		start = max(0, end-height)
	}

	var lines []string
	for i := start; i < end; i++ {
		lines = append(lines, ui.RenderItemRow(m.filtered[i], i == m.cursor, m.width))
	}

	// Pad remaining lines.
	for len(lines) < height {
		lines = append(lines, "")
	}

	// Show count in bottom-right of list area.
	countStr := ui.StyleFaint.Render(fmt.Sprintf("%d / %d", m.cursor+1, len(m.filtered)))
	if len(lines) > 0 {
		last := lines[len(lines)-1]
		lastW := lipgloss.Width(last)
		countW := lipgloss.Width(countStr)
		gap := m.width - lastW - countW
		if gap > 0 {
			lines[len(lines)-1] = last + strings.Repeat(" ", gap) + countStr
		}
	}

	return strings.Join(lines, "\n")
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
