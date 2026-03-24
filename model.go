package main

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/screens"
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

// idleCheckMsg fires periodically to check for idle timeout.
type idleCheckMsg struct{}

// RootModel is the top-level BubbleTea model.
type RootModel struct {
	state   appState
	sess    *session.State
	locked  screens.LockedModel
	vault   screens.VaultModel
	err     screens.ErrorModel
	spinner spinner.Model
	width   int
	height  int
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
	return tea.Batch(bwcmd.CheckStatus(), tickIdleCheck(), m.spinner.Tick)
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

	case bwcmd.StatusResult:
		if msg.Err != nil {
			m.state = stateError
			m.err = screens.NewErrorModel(msg.Err, false)
			return m, nil
		}
		m.sess.Email = msg.Status.UserEmail
		switch msg.Status.Status {
		case "unauthenticated":
			m.state = stateLogin
			m.locked = screens.NewLockedModel(true)
			return m, m.locked.Init()
		case "locked":
			m.state = stateLocked
			m.locked = screens.NewLockedModel(false)
			return m, m.locked.Init()
		case "unlocked":
			m.state = stateLoading
			return m, bwcmd.FetchItems(m.sess.Token)
		default:
			m.state = stateLocked
			m.locked = screens.NewLockedModel(false)
			return m, m.locked.Init()
		}

	case screens.UnlockedMsg:
		m.sess.SetToken(msg.Token)
		if msg.Email != "" {
			m.sess.Email = msg.Email
		}
		m.sess.Touch()
		m.state = stateLoading
		m.spinner.Spinner = ui.SpinnerLoad
		return m, tea.Batch(bwcmd.FetchItems(m.sess.Token), m.spinner.Tick)

	case bwcmd.ItemsResult:
		if msg.Err != nil {
			m.state = stateError
			m.err = screens.NewErrorModel(msg.Err, false)
			return m, nil
		}
		m.state = stateVault
		m.vault = screens.NewVaultModel(msg.Items, m.sess, m.width, m.height)
		return m, m.vault.Init()

	case screens.LockMsg:
		m.state = stateQuitting
		m.spinner.Spinner = ui.SpinnerLock
		if m.sess.Token != "" {
			token := m.sess.Token
			m.sess.Lock()
			return m, tea.Batch(m.spinner.Tick, bwcmd.Lock(token))
		}
		return m, tea.Quit

	case bwcmd.LockResult:
		return m, tea.Quit

	case screens.RetryMsg:
		m.state = stateLoading
		return m, bwcmd.CheckStatus()

	case idleCheckMsg:
		if m.state == stateVault && m.sess.IsIdle() {
			m.sess.Lock()
			m.state = stateLocked
			m.locked = screens.NewLockedModel(false)
			return m, tea.Batch(bwcmd.Lock(m.sess.Token), m.locked.Init(), tickIdleCheck())
		}
		return m, tickIdleCheck()

	case tea.KeyMsg:
		m.sess.Touch()
		if msg.String() == "ctrl+c" {
			m.state = stateQuitting
			m.spinner.Spinner = ui.SpinnerLock
			if m.sess.Token != "" {
				token := m.sess.Token
				m.sess.Lock()
				return m, tea.Batch(m.spinner.Tick, bwcmd.Lock(token))
			}
			return m, tea.Quit
		}
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

func (m RootModel) View() string {
	header := ui.RenderHeader(m.sess.Email, m.width)
	contentHeight := m.height - 2 // header + footer

	var content, hints, status string

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
		content = ui.CenterInArea(m.err.ViewContent(), m.width, contentHeight)
		hints = m.err.FooterHints()
	case stateQuitting:
		content = ui.CenterInArea(m.spinner.View()+" Locking vault…", m.width, contentHeight)
	}

	footer := ui.RenderFooter(hints, status, m.width)
	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

func tickIdleCheck() tea.Cmd {
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg {
		return idleCheckMsg{}
	})
}
