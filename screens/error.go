package screens

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/juthrbog/lazybw/ui"
)

// RetryMsg tells the root model to re-attempt whatever caused the error.
type RetryMsg struct{}

// ErrorModel displays a recoverable or fatal error with retry/quit options.
type ErrorModel struct {
	err   error
	fatal bool // if true, only Quit is offered
}

// NewErrorModel constructs an error screen.
func NewErrorModel(err error, fatal bool) ErrorModel {
	return ErrorModel{err: err, fatal: fatal}
}

func (m ErrorModel) Init() tea.Cmd { return nil }

func (m ErrorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			if !m.fatal {
				return m, func() tea.Msg { return RetryMsg{} }
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// ViewContent returns the error content for the root to center.
func (m ErrorModel) ViewContent() string {
	errText := "unknown error"
	if m.err != nil {
		errText = m.err.Error()
	}
	return ui.StyleError.Render("Error: " + errText)
}

// FooterHints returns the hint string for the footer.
func (m ErrorModel) FooterHints() string {
	if m.fatal {
		return "q quit"
	}
	return "r retry · q quit"
}

// View implements tea.Model (delegates to root frame).
func (m ErrorModel) View() string {
	return m.ViewContent()
}
