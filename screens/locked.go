package screens

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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
	s.Spinner = spinner.Dot
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

	if isLogin {
		emailField := huh.NewInput().
			Key("email").
			Title("Email").
			Value(email)
		return huh.NewForm(huh.NewGroup(emailField, pwField))
	}
	return huh.NewForm(huh.NewGroup(pwField))
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
			// Reset form for retry.
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

func (m LockedModel) View() string {
	var body string

	header := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorHighlight).Render("lazybw")

	if m.state == lockedUnlocking {
		status := m.spinner.View() + " Unlocking…"
		body = centerBlock(m.width, m.height,
			header+"\n\n"+status,
		)
		return body
	}

	title := "Vault is locked."
	if m.isLogin {
		title = "Log in to Bitwarden"
	}

	content := header + "\n\n" + title + "\n\n" + m.form.View()

	if m.err != nil {
		content += "\n\n" + ui.StyleError.Render(m.err.Error())
	}

	content += "\n\n" + ui.StyleFaint.Render("Enter to unlock · q to quit")

	return centerBlock(m.width, m.height, content)
}

func centerBlock(width, height int, content string) string {
	lines := strings.Split(content, "\n")
	maxLineW := 0
	for _, l := range lines {
		if w := lipgloss.Width(l); w > maxLineW {
			maxLineW = w
		}
	}

	// Horizontal centering.
	padLeft := (width - maxLineW) / 2
	if padLeft < 0 {
		padLeft = 0
	}

	// Vertical centering.
	padTop := (height - len(lines)) / 2
	if padTop < 0 {
		padTop = 0
	}

	var b strings.Builder
	for i := 0; i < padTop; i++ {
		b.WriteString("\n")
	}
	for _, l := range lines {
		b.WriteString(strings.Repeat(" ", padLeft))
		b.WriteString(l)
		b.WriteString("\n")
	}
	return b.String()
}
