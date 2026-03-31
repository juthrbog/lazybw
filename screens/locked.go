package screens

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/ui"
)

// UnlockedMsg is sent to the root model when the unlock succeeds.
type UnlockedMsg struct {
	Token string
	Email string
}

type lockedState int

const (
	lockedInput lockedState = iota
	lockedUnlocking
)

// LockedModel is the unlock/login prompt screen.
type LockedModel struct {
	form    *huh.Form
	isLogin bool
	state   lockedState
	spinner spinner.Model
	err     error
	width   int
	height  int

	email    *string
	password *string
}

// NewLockedModel constructs the unlock model.
func NewLockedModel(isLogin bool) LockedModel {
	email := ""
	password := ""
	s := spinner.New()
	s.Spinner = ui.SpinnerUnlock
	s.Style = s.Style.Foreground(ui.ColorHighlight)
	m := LockedModel{
		isLogin:  isLogin,
		email:    &email,
		password: &password,
		spinner:  s,
	}
	m.form = buildLockedForm(isLogin, m.email, m.password)
	return m
}

func buildLockedForm(isLogin bool, email, password *string) *huh.Form {
	pwField := huh.NewInput().
		Key("password").
		Title("Master password").
		EchoMode(huh.EchoModePassword).
		Value(password)

	var form *huh.Form
	if isLogin {
		emailField := huh.NewInput().
			Key("email").
			Title("Email").
			Value(email)
		form = huh.NewForm(huh.NewGroup(emailField, pwField))
	} else {
		form = huh.NewForm(huh.NewGroup(pwField))
	}
	form = form.WithWidth(40)
	if ui.HuhTheme != nil {
		form = form.WithTheme(ui.HuhTheme)
	}
	return form
}

func (m LockedModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m LockedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case bwcmd.UnlockResult:
		m.state = lockedInput
		if msg.Err != nil {
			m.err = msg.Err
			pw := ""
			m.password = &pw
			m.form = buildLockedForm(m.isLogin, m.email, m.password)
			return m, m.form.Init()
		}
		email := *m.email
		return m, func() tea.Msg {
			return UnlockedMsg{Token: msg.Token, Email: email}
		}
	}

	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "ctrl+q" {
		return m, tea.Quit
	}

	if m.state == lockedUnlocking {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	switch m.form.State {
	case huh.StateCompleted:
		m.state = lockedUnlocking
		m.err = nil
		pw := *m.password
		email := *m.email
		if m.isLogin {
			return m, tea.Batch(m.spinner.Tick, bwcmd.LoginUser(email, pw))
		}
		return m, tea.Batch(m.spinner.Tick, bwcmd.Unlock(pw))
	case huh.StateAborted:
		return m, tea.Quit
	}

	return m, cmd
}

// ViewContent returns the screen content for the root frame to center.
func (m LockedModel) ViewContent(width, contentHeight int) string {
	if m.state == lockedUnlocking {
		status := m.spinner.View() + " Unlocking…"
		return ui.CenterInArea(status, width, contentHeight)
	}

	title := "Vault is locked."
	if m.isLogin {
		title = "Log in to Bitwarden"
	}

	content := title + "\n\n" + m.form.View()

	if m.err != nil {
		content += "\n\n" + ui.StyleError.Render(m.err.Error())
	}

	return ui.CenterInArea(content, width, contentHeight)
}

// FooterContent returns hints and status for the footer bar.
func (m LockedModel) FooterContent() (hints, status string) {
	if m.state == lockedUnlocking {
		return "", ""
	}
	if m.isLogin {
		return "enter submit · ctrl+q quit", ""
	}
	return "enter unlock · ctrl+q quit", ""
}

// View implements tea.Model (kept for interface compliance).
func (m LockedModel) View() tea.View {
	return tea.NewView(m.ViewContent(m.width, m.height))
}
