package screens

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/juthrbog/lazybw/ui"
)

// RetryMsg tells the root model to re-attempt whatever caused the error.
type RetryMsg struct{}

// ErrorModel displays a recoverable or fatal error with retry/quit options.
type ErrorModel struct {
	err    error
	fatal  bool // if true, only Quit is offered
	width  int
	height int
}

// NewErrorModel constructs an error screen.
func NewErrorModel(err error, fatal bool, width, height int) ErrorModel {
	return ErrorModel{err: err, fatal: fatal, width: width, height: height}
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

func (m ErrorModel) View() string {
	errText := "unknown error"
	if m.err != nil {
		errText = m.err.Error()
	}

	body := ui.StyleError.Render("Error: "+errText) + "\n\n"
	if m.fatal {
		body += ui.StyleFaint.Render("[q] quit")
	} else {
		body += ui.StyleFaint.Render("[r] retry  [q] quit")
	}
	return body
}
