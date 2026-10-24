package main

import (
	"context"
	"errors"
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
	// yet. If the program stops first, the exit cleanup locks again (see
	// cleanupFor and exitCleanup.run).
	pendingLock string
	// lockSeq numbers the `bw lock` runs. A LockResult with another Seq
	// belongs to an earlier lock, and Update drops it.
	lockSeq uint64
	// lockErr is the text of a failed `bw lock` that no screen shows yet,
	// for example after the lock from the error screen. The next unlock
	// screen shows it, and a quit gives it to main.
	lockErr string
	// retried is true after a retry of a failed item fetch. A second
	// failure locks the session, so that the next retry asks for the
	// password again.
	retried bool
	// lastSync is the time of the last vault sync that lazybw knows of:
	// from bw status, or from the session before a lock. A lock clears
	// the time in the session, and an unlock puts it back.
	lastSync time.Time

	// lastCleanup holds the exit cleanup of the model after the last
	// Update. All copies of the model share it. See cleanupFor.
	lastCleanup *exitCleanup
	// quitError is the text of a `bw lock` error on the way out. main
	// prints it after the terminal is restored.
	quitError string
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
		lastCleanup: &exitCleanup{},
	}
}

func (m RootModel) Init() tea.Cmd {
	return tea.Batch(bwcmd.CheckStatus(), tickIdleCheck(), m.spinner.Tick, tea.RequestBackgroundColor)
}

// Update handles msg and records the exit cleanup of the new model in
// lastCleanup.
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	if r, ok := next.(RootModel); ok && r.lastCleanup != nil {
		*r.lastCleanup = r.pendingCleanup()
	}
	return next, cmd
}

func (m RootModel) update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		// The huh forms do not get this message. ui.HuhTheme reads
		// ui.IsDark, so the forms follow the background anyway.
		ui.IsDark = msg.IsDark()
		ui.ApplyTheme(ui.CurrentTheme)
		return m.Update(screens.ThemeChangedMsg{})

	case screens.ThemeChangedMsg:
		// The theme or the background changed. Restyle the spinner here,
		// and let the active screen restyle its own parts below.
		m.spinner.Style = m.spinner.Style.Foreground(ui.ColorHighlight)

	case bwcmd.StatusResult:
		if msg.Err != nil {
			log.Printf("bw status failed: %s", logReason(msg.Err))
			m.state = stateError
			m.err = screens.NewErrorModel(msg.Err, false)
			return m, nil
		}
		// Log the status only. The other fields hold the email.
		log.Printf("bw status: %s", msg.Status.Status)
		m.sess.Email = msg.Status.UserEmail
		m.lastSync = parseLastSync(msg.Status.LastSync)
		switch msg.Status.Status {
		case "unauthenticated":
			// bw has no logged-in account, so an old session key opens
			// nothing. A kept bw lock error does not matter now.
			m.lockErr = ""
			m.state = stateLogin
			m.locked = screens.NewLockedModel(true)
			return m, m.locked.Init()
		default:
			// "locked" or "unlocked". ParseStatus rejects other values.
			// lazybw never passes the BW_SESSION of the parent shell to
			// bw, so "unlocked" cannot mean a session that lazybw can use.
			// Both ask for the master password.
			m.state = stateLocked
			m.locked = m.newUnlockScreen()
			return m, m.locked.Init()
		}

	case screens.UnlockedMsg:
		log.Print("vault unlocked, loading the items")
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
			log.Print("dropped the items of an ended session")
			secutil.ZeroItems(msg.Items)
			return m, nil
		}
		if msg.Err != nil {
			log.Printf("loading the items failed: %s", logReason(msg.Err))
		} else {
			log.Printf("loaded %d items", len(msg.Items))
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
					log.Print("locking the vault: the retry failed too")
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
			log.Print("dropped the sync result of an ended session")
			return m, nil
		}
		if msg.Err != nil {
			log.Printf("bw sync failed: %s", logReason(msg.Err))
		} else {
			log.Print("bw sync done")
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
		if msg.Err != nil {
			log.Printf("bw lock failed: %s", logReason(msg.Err))
		} else {
			log.Print("bw lock done")
		}
		if msg.Seq != m.lockSeq {
			// A newer lock runs, for example the lock key after an idle
			// lock and an unlock. That lock reports its own result, and
			// pendingLock holds its token.
			log.Print("dropped the result of an earlier bw lock")
			return m, nil
		}
		m.pendingLock = ""
		// The token is already gone from memory, so the vault stays locked
		// for lazybw. But bw can still accept the token, so the user must
		// learn of a failure.
		lockErr := screens.LockErrorText(msg.Err)
		if m.state != stateQuitting {
			return m.showLockError(lockErr), nil
		}
		if m.lockFor == intentQuit {
			m.quitError = lockErr
			cmd := m.exit()
			return m, cmd
		}
		m.state = stateLocked
		m.locked = screens.NewLockedModel(false).WithLockError(lockErr)
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
			log.Print("retry: loading the items again")
			m.retried = true
			return m, tea.Batch(bwcmd.FetchItems(m.sess.Token, m.sess.Epoch), m.spinner.Tick)
		}
		log.Print("retry: running bw status again")
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
	m.lockSeq++
	return bwcmd.Lock(token, m.lockSeq)
}

// showLockError puts text, the error of a `bw lock` that returned outside
// stateQuitting, where the user can see it. An empty text does nothing.
func (m RootModel) showLockError(text string) RootModel {
	switch {
	case text == "":
	case m.state == stateLogin:
		// bw reported no logged-in account, so the old session key opens
		// nothing.
	case m.state == stateLocked:
		// An idle lock already shows the unlock screen. Add the error to
		// that screen, because a new screen wipes what the user typed in
		// the meantime.
		m.locked = m.locked.WithLockError(text)
	case m.sess.IsLocked():
		// For example the lock after a second failed item load, while the
		// error screen shows. Keep the error for the next unlock screen.
		m.lockErr = text
	default:
		// The user unlocked again before the result arrived. The error
		// goes to the log only.
		log.Print("the bw lock error came after a new unlock")
	}
	return m
}

// newUnlockScreen returns a new unlock screen with the kept error of a
// failed `bw lock`, and forgets that error.
func (m *RootModel) newUnlockScreen() screens.LockedModel {
	s := screens.NewLockedModel(false).WithLockError(m.lockErr)
	m.lockErr = ""
	return s
}

// lock ends the session and shows the unlock screen when `bw lock` returns.
func (m RootModel) lock() (RootModel, tea.Cmd) {
	m.lockFor = intentLock
	m.spinner.Spinner = ui.SpinnerLock
	clearCmd := m.clearClipboard()
	m.dropVault()
	if m.sess.IsLocked() {
		log.Print("lock key: the vault is already locked")
		m.state = stateLocked
		m.locked = m.newUnlockScreen()
		return m, tea.Batch(clearCmd, m.locked.Init())
	}
	log.Print("locking the vault: lock key")
	lockCmd := m.endSession()
	m.state = stateQuitting
	return m, tea.Batch(clearCmd, m.spinner.Tick, lockCmd)
}

// idleLock ends the session and shows the unlock screen at once. The
// LockResult that follows only adds a bw lock error to that screen.
func (m RootModel) idleLock() (RootModel, tea.Cmd) {
	log.Print("locking the vault: idle timeout")
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
		log.Print("quit: the vault is locked")
		// main prints a bw lock error that no screen showed.
		m.quitError, m.lockErr = m.lockErr, ""
		cmd := m.exit()
		return m, cmd
	}
	log.Print("locking the vault: quit")
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

// exitCleanup is the work that is left when the program stops.
type exitCleanup struct {
	// clipDirty is true when a copied value can still be on the clipboard.
	clipDirty bool
	// lockToken is the token of a session that is not locked yet.
	lockToken string
}

// run clears the clipboard by writing OSC 52 to w, and locks the session.
// It returns the error of bw lock.
func (c exitCleanup) run(w io.Writer) error {
	if c.clipDirty {
		_, _ = io.WriteString(w, ansi.SetSystemClipboard(session.ClearPlaceholder))
	}
	if c.lockToken == "" {
		return nil
	}
	// No model gets this result, so the sequence number does not matter.
	res, _ := bwcmd.Lock(c.lockToken, 0)().(bwcmd.LockResult)
	if res.Err != nil {
		log.Printf("bw lock in the exit cleanup failed: %s", logReason(res.Err))
	}
	return res.Err
}

// pendingCleanup returns the exit cleanup for m. The token to lock is the
// token of the open session, or the token of a `bw lock` that has not
// returned yet, because a closed terminal can kill that child process.
func (m RootModel) pendingCleanup() exitCleanup {
	c := exitCleanup{clipDirty: m.clipDirty, lockToken: m.pendingLock}
	if m.sess != nil && !m.sess.IsLocked() {
		c.lockToken = m.sess.Token
	}
	return c
}

// cleanupFor returns the exit cleanup for the model that tea.Program.Run
// returned. Run returns no model after a panic in Update or View, so then
// it returns the cleanup that the last Update recorded in last.
func cleanupFor(final tea.Model, last *exitCleanup) exitCleanup {
	if m, ok := final.(RootModel); ok {
		return m.pendingCleanup()
	}
	if last != nil {
		return *last
	}
	return exitCleanup{}
}

// logReason returns the text that the debug log shows for the error of a
// bw command. The text never holds the message or the output of bw. The
// message of a bw error is the last line that bw wrote to stderr, and a
// parse error can quote a part of the output. bwcmd logs the stderr of a
// failed bw command itself, with the private values taken out.
func logReason(err error) string {
	var bwErr *bwcmd.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timed out"
	case errors.Is(err, bwcmd.ErrNotLoggedIn):
		return "not logged in"
	case errors.As(err, &bwErr) && bwErr.Err != nil:
		// The error of os/exec, for example "exit status 1".
		return "bw error (" + bwErr.Err.Error() + ")"
	case errors.As(err, &bwErr):
		return "bw error"
	}
	return "unexpected output from bw"
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
		content = ui.CenterInArea(m.spinner.View()+" Loading vault"+ui.Glyphs.Ellipsis, m.width, contentHeight)
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
		content = ui.CenterInArea(m.spinner.View()+" Locking vault"+ui.Glyphs.Ellipsis, m.width, contentHeight)
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
	// No mouse mode: lazybw handles no mouse events. Mouse capture would
	// stop click and drag from selecting text, and it would stop the
	// terminals that send arrow keys for the wheel in the alternate screen
	// from doing so.
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
