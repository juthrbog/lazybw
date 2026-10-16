package main

import (
	"io"
	"log"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/screens"
	"github.com/juthrbog/lazybw/secutil"
	"github.com/juthrbog/lazybw/session"
	"github.com/juthrbog/lazybw/ui"
)

type appState int

const (
	stateLoading appState = iota
	stateLogin
	stateLocked
	stateVault
	stateError
	stateQuitting
)

type lockIntent int

const (
	intentLock lockIntent = iota // return to unlock screen
	intentQuit                   // exit the application
)

// idleCheckMsg fires periodically to check for idle timeout.
type idleCheckMsg struct{}

// RootModel is the top-level BubbleTea model.
type RootModel struct {
	state   appState
	lockFor lockIntent
	sess    *session.State
	locked  screens.LockedModel
	vault   screens.VaultModel
	err     screens.ErrorModel
	spinner spinner.Model
	width   int
	height  int

	// clipGen counts copies to the clipboard. clipDirty is true while a
	// copied value can still be on the clipboard.
	clipGen   uint64
	clipDirty bool

	// pendingLock holds the token of a `bw lock` that has not returned
	// yet. If the program stops first, cleanupAfterExit locks again.
	pendingLock string
	// retried is true after a retry of a failed item fetch. A second
	// failure locks the session, so that the next retry asks for the
	// password again.
	retried bool
	// lastSync is the time of the last vault sync that lazybw knows of:
	// from bw status, or from the session before a lock. A lock clears
	// the time in the session, and an unlock puts it back.
	lastSync time.Time
}

func NewRootModel(idleTimeout time.Duration) RootModel {
	s := spinner.New()
	s.Spinner = ui.SpinnerLoad
	s.Style = s.Style.Foreground(ui.ColorHighlight)
	return RootModel{
		state:   stateLoading,
		spinner: s,
		sess: &session.State{
			IdleTimeout: idleTimeout,
			LastActive:  time.Now(),
		},
	}
}

func (m RootModel) Init() tea.Cmd {
	return tea.Batch(bwcmd.CheckStatus(), tickIdleCheck(), m.spinner.Tick, func() tea.Msg { return tea.RequestBackgroundColor() })
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propagate to active child.
		switch m.state {
		case stateLocked, stateLogin:
			updated, cmd := m.locked.Update(msg)
			m.locked = updated.(screens.LockedModel)
			return m, cmd
		case stateVault:
			updated, cmd := m.vault.Update(msg)
			m.vault = updated.(screens.VaultModel)
			return m, cmd
		}
		return m, nil

	case tea.BackgroundColorMsg:
		ui.IsDark = msg.IsDark()
		ui.ApplyTheme(ui.CurrentTheme)
		m.spinner.Style = m.spinner.Style.Foreground(ui.ColorHighlight)
		return m, nil

	case bwcmd.StatusResult:
		if msg.Err != nil {
			m.state = stateError
			m.err = screens.NewErrorModel(msg.Err, false)
			return m, nil
		}
		m.sess.Email = msg.Status.UserEmail
		m.lastSync = parseLastSync(msg.Status.LastSync)
		switch msg.Status.Status {
		case "unauthenticated":
			m.state = stateLogin
			m.locked = screens.NewLockedModel(true)
			return m, m.locked.Init()
		default:
			// "locked" or "unlocked". ParseStatus rejects other values.
			// lazybw never passes the BW_SESSION of the parent shell to
			// bw, so "unlocked" cannot mean a session that lazybw can use.
			// Both ask for the master password.
			m.state = stateLocked
			m.locked = screens.NewLockedModel(false)
			return m, m.locked.Init()
		}

	case screens.UnlockedMsg:
		m.sess.SetToken(msg.Token)
		m.sess.LastSync = m.lastSync
		if msg.Email != "" {
			m.sess.Email = msg.Email
		}
		m.sess.Touch()
		// Drop the unlock screen and its form. Every way back to the
		// unlock screen builds a new one.
		m.locked = screens.LockedModel{}
		m.state = stateLoading
		m.spinner.Spinner = ui.SpinnerLoad
		return m, tea.Batch(bwcmd.FetchItems(m.sess.Token, m.sess.Epoch), m.spinner.Tick)

	case bwcmd.ItemsResult:
		// A fetch that started before a lock (or before an earlier unlock)
		// belongs to an ended session. Drop it, so that it cannot open the
		// vault again.
		if msg.Epoch != m.sess.Epoch {
			secutil.ZeroItems(msg.Items)
			return m, nil
		}
		switch m.state {
		case stateLoading:
			if msg.Err != nil {
				m.state = stateError
				m.err = screens.NewErrorModel(msg.Err, false)
				if m.retried {
					// The token can be invalid, for example after a lock
					// in another shell. Lock, so that the next retry
					// shows the unlock screen.
					m.retried = false
					return m, m.endSession()
				}
				return m, nil
			}
			m.retried = false
			m.state = stateVault
			m.vault = screens.NewVaultModel(msg.Items, m.sess, m.width, m.height)
			return m, m.vault.Init()
		case stateVault:
			// A refresh after sync. The vault keeps its UI state.
			updated, cmd := m.vault.Update(msg)
			m.vault = updated.(screens.VaultModel)
			return m, cmd
		}
		secutil.ZeroItems(msg.Items)
		return m, nil

	case bwcmd.SyncResult:
		if msg.Epoch != m.sess.Epoch || m.state != stateVault {
			return m, nil
		}

	case session.CopiedMsg:
		m.clipGen++
		m.clipDirty = true
		clearCmd := session.ScheduleClipboardClear(m.clipGen)
		if m.state == stateVault {
			updated, cmd := m.vault.Update(msg)
			m.vault = updated.(screens.VaultModel)
			return m, tea.Batch(clearCmd, cmd)
		}
		return m, clearCmd

	case session.ClipboardClearedMsg:
		// Only the timer of the latest copy clears the clipboard. An older
		// timer must not clear a newer value early.
		if msg.Gen != m.clipGen || !m.clipDirty {
			return m, nil
		}
		m.clipDirty = false
		if m.state == stateVault {
			updated, cmd := m.vault.Update(msg)
			m.vault = updated.(screens.VaultModel)
			return m, tea.Batch(session.ClearClipboard(), cmd)
		}
		return m, session.ClearClipboard()

	case screens.LockMsg:
		return m.lock()

	case screens.QuitMsg:
		return m.quit()

	case bwcmd.LockResult:
		m.pendingLock = ""
		if msg.Err != nil {
			// The token is already gone from memory, so the vault stays
			// locked for lazybw. Record the failure for --debug.
			log.Printf("bw lock failed: %v", msg.Err)
		}
		// Only a lock that the quitting screen waits for needs an action. An
		// idle lock already shows the unlock screen, and rebuilding it here
		// would wipe what the user typed in the meantime.
		if m.state != stateQuitting {
			return m, nil
		}
		if m.lockFor == intentQuit {
			cmd := m.exit()
			return m, cmd
		}
		m.state = stateLocked
		m.locked = screens.NewLockedModel(false)
		return m, m.locked.Init()

	case screens.RetryMsg:
		// Only the error screen sends RetryMsg. In any other state it is
		// stale, for example after r and ctrl+c in one input read.
		if m.state != stateError {
			return m, nil
		}
		m.state = stateLoading
		m.spinner.Spinner = ui.SpinnerLoad
		if !m.sess.IsLocked() {
			// The vault is unlocked, but loading the items failed.
			m.retried = true
			return m, tea.Batch(bwcmd.FetchItems(m.sess.Token, m.sess.Epoch), m.spinner.Tick)
		}
		return m, tea.Batch(bwcmd.CheckStatus(), m.spinner.Tick)

	case idleCheckMsg:
		if m.idle() {
			m, cmd := m.idleLock()
			return m, tea.Batch(cmd, tickIdleCheck())
		}
		return m, tickIdleCheck()

	case tea.KeyPressMsg:
		// quit also locks the vault, so ctrl+c needs no idle check.
		if msg.String() == "ctrl+c" {
			return m.quit()
		}
		// After a suspend, the first key press can arrive before the next
		// idle check. It must lock the vault, not count as activity.
		if m.idle() {
			return m.idleLock()
		}
		m.sess.Touch()
	}

	// Delegate to active child.
	var cmd tea.Cmd
	switch m.state {
	case stateLoading, stateQuitting:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case stateLocked, stateLogin:
		var updated tea.Model
		updated, cmd = m.locked.Update(msg)
		m.locked = updated.(screens.LockedModel)
	case stateVault:
		var updated tea.Model
		updated, cmd = m.vault.Update(msg)
		m.vault = updated.(screens.VaultModel)
	case stateError:
		var updated tea.Model
		updated, cmd = m.err.Update(msg)
		m.err = updated.(screens.ErrorModel)
	}

	return m, cmd
}

// idle reports whether an unlocked session, or a vault screen, passed its
// idle timeout.
func (m RootModel) idle() bool {
	open := m.state == stateVault || !m.sess.IsLocked()
	return open && m.state != stateQuitting && m.sess.IsIdle()
}

// dropVault scrubs the vault model and drops it, so that no list row,
// filter text or cached TOTP state stays reachable after a lock.
func (m *RootModel) dropVault() {
	m.vault.Clear()
	m.vault = screens.VaultModel{}
}

// endSession scrubs the vault, ends the session and returns the `bw lock`
// command. The caller sets the next state.
func (m *RootModel) endSession() tea.Cmd {
	token := m.sess.Token
	if !m.sess.LastSync.IsZero() {
		m.lastSync = m.sess.LastSync
	}
	m.dropVault()
	m.sess.Lock()
	m.pendingLock = token
	return bwcmd.Lock(token)
}

// lock ends the session and shows the unlock screen when `bw lock` returns.
func (m RootModel) lock() (RootModel, tea.Cmd) {
	m.lockFor = intentLock
	m.spinner.Spinner = ui.SpinnerLock
	clearCmd := m.clearClipboard()
	m.dropVault()
	if m.sess.IsLocked() {
		m.state = stateLocked
		m.locked = screens.NewLockedModel(false)
		return m, tea.Batch(clearCmd, m.locked.Init())
	}
	lockCmd := m.endSession()
	m.state = stateQuitting
	return m, tea.Batch(clearCmd, m.spinner.Tick, lockCmd)
}

// idleLock ends the session and shows the unlock screen at once. The
// LockResult that follows is ignored.
func (m RootModel) idleLock() (RootModel, tea.Cmd) {
	clearCmd := m.clearClipboard()
	lockCmd := m.endSession()
	m.lockFor = intentLock
	m.state = stateLocked
	m.locked = screens.NewLockedModel(false)
	return m, tea.Batch(clearCmd, lockCmd, m.locked.Init())
}

// quit locks the vault if it is unlocked, clears the clipboard and exits.
func (m RootModel) quit() (RootModel, tea.Cmd) {
	m.lockFor = intentQuit
	m.state = stateQuitting
	m.spinner.Spinner = ui.SpinnerLock
	m.dropVault()
	if m.sess.IsLocked() {
		cmd := m.exit()
		return m, cmd
	}
	// Clear the clipboard now, and again in exit after bw lock returns.
	var clearCmd tea.Cmd
	if m.clipDirty {
		clearCmd = session.ClearClipboard()
	}
	lockCmd := m.endSession()
	return m, tea.Batch(clearCmd, m.spinner.Tick, lockCmd)
}

// exit returns the command that ends the program. If a copied value can
// still be on the clipboard, the clipboard is cleared first. tea.Sequence
// makes sure that the clear is written before the program stops.
// tea.Batch gives no order, and the clear is often lost.
func (m *RootModel) exit() tea.Cmd {
	if !m.clipDirty {
		return tea.Quit
	}
	m.clipDirty = false
	return tea.Sequence(session.ClearClipboard(), tea.Quit)
}

// clearClipboard returns a command that clears the clipboard if a copied
// value can still be on it. Otherwise it returns nil, so that unrelated
// clipboard content is left alone.
func (m *RootModel) clearClipboard() tea.Cmd {
	if !m.clipDirty {
		return nil
	}
	m.clipDirty = false
	return session.ClearClipboard()
}

// cleanupAfterExit runs after the program stops. On the normal quit path
// there is nothing left to do. When the program stops in another way, for
// example on SIGTERM or SIGHUP, it clears the clipboard by writing OSC 52
// to w and locks the vault. It also runs `bw lock` again when a lock was
// still running, because a closed terminal can kill that child process.
func (m RootModel) cleanupAfterExit(w io.Writer) {
	if m.clipDirty {
		_, _ = io.WriteString(w, ansi.SetSystemClipboard(session.ClearPlaceholder))
	}
	token := m.pendingLock
	if m.sess != nil && !m.sess.IsLocked() {
		token = m.sess.Token
	}
	if token != "" {
		_ = bwcmd.Lock(token)()
	}
}

func (m RootModel) View() tea.View {
	hd := ui.HeaderData{
		Email: m.sess.Email,
		Width: m.width,
	}
	if m.state == stateVault {
		hd.ItemCount, hd.SelectedType = m.vault.HeaderInfo()
	}
	header := ui.RenderHeader(hd)
	contentHeight := m.height - ui.ChromeHeight

	var (
		content string
		hints   []ui.HintBinding
		status  string
	)

	switch m.state {
	case stateLoading:
		content = ui.CenterInArea(m.spinner.View()+" Loading vault…", m.width, contentHeight)
	case stateLogin, stateLocked:
		content = m.locked.ViewContent(m.width, contentHeight)
		hints, status = m.locked.FooterContent()
	case stateVault:
		content = m.vault.ViewContent(m.width, contentHeight)
		hints, status = m.vault.FooterContent()
	case stateError:
		content = ui.CenterInArea(m.err.ViewContent(m.width), m.width, contentHeight)
		hints = m.err.FooterHints()
	case stateQuitting:
		content = ui.CenterInArea(m.spinner.View()+" Locking vault…", m.width, contentHeight)
	}

	footer := ui.RenderFooter(hints, status, m.width)
	frame := []string{header}
	if contentHeight > 0 {
		// Give the content exactly contentHeight rows, and cut what is too
		// wide. The renderer drops rows below the terminal, so a content
		// area that is too tall would push the footer off the screen.
		content = lipgloss.NewStyle().
			Height(contentHeight).
			MaxHeight(contentHeight).
			MaxWidth(m.width).
			Render(content)
		frame = append(frame, content)
	}
	frame = append(frame, footer)
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, frame...))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// parseLastSync parses the lastSync value of bw status, an RFC 3339 time.
// It returns the zero time for an empty or a malformed value.
func parseLastSync(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// idleCheckInterval is the time between two idle checks. Tests shorten it.
var idleCheckInterval = 30 * time.Second

func tickIdleCheck() tea.Cmd {
	return tea.Tick(idleCheckInterval, func(time.Time) tea.Msg {
		return idleCheckMsg{}
	})
}
