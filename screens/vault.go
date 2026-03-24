package screens

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/session"
	"github.com/juthrbog/lazybw/ui"
)

// LockMsg tells the root model to lock the vault.
type LockMsg struct{}

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
		syncSpinner: ss,
		width:       width,
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

	case tea.KeyMsg:
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

	case spinner.TickMsg:
		if m.syncing {
			var cmd tea.Cmd
			m.syncSpinner, cmd = m.syncSpinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m VaultModel) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			return m, bwcmd.GetPassword(m.sess.Token, item.ID)
		}

	case key.Matches(msg, m.keymap.CopyTOTP):
		if m.totpCode != "" {
			return m, session.CopyToClipboard(m.totpCode, session.CopyFieldTOTP)
		}

	case key.Matches(msg, m.keymap.CopyUsername):
		if item := m.selectedItem(); item != nil && item.Login != nil {
			return m, session.CopyToClipboard(item.Login.Username, session.CopyFieldUsername)
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

	case key.Matches(msg, m.keymap.Quit):
		return m, func() tea.Msg { return LockMsg{} }

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

func (m VaultModel) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

// ViewContent renders the vault content for the root frame.
func (m VaultModel) ViewContent(width, contentHeight int) string {
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
	if m.showHelp {
		listHeight -= 5
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
	if m.showHelp {
		sections = append(sections, m.help.View(m.keymap))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// FooterContent returns hints and status for the footer bar.
func (m VaultModel) FooterContent() (hints, status string) {
	if m.mode == modeFilter {
		hints = "esc clear · enter confirm · ↑/↓ navigate"
	} else {
		hints = "j/k navigate · / search · c pwd · t totp · ? help · q quit"
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
func (m VaultModel) View() string {
	return m.ViewContent(m.width, m.height)
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
