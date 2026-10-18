package screens

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/ui"
)

// RetryMsg tells the root model to try again. If the vault is unlocked,
// the root model loads the items again. Otherwise it runs bw status again.
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
	case tea.KeyPressMsg:
		switch msg.String() {
		case "r":
			if !m.fatal {
				return m, func() tea.Msg { return RetryMsg{} }
			}
		case "q", "ctrl+c":
			// The root model locks the vault and clears the clipboard.
			return m, func() tea.Msg { return QuitMsg{} }
		}
	}
	return m, nil
}

// maxErrorWidth is the widest that a line of the error gets, so that a long
// bw error stays easy to read on a wide terminal.
const maxErrorWidth = 72

// ViewContent returns the error content for the root to center. It wraps
// the error to fit a screen of the given width, with a 2-cell margin on
// each side. A width that leaves no space for text does not wrap.
func (m ErrorModel) ViewContent(width int) string {
	errText := "unknown error"
	if m.err != nil {
		errText = userError(m.err, actionOther)
	}
	return errorBlock(errText, width, maxErrorWidth)
}

// errorBlock renders an error message, which often holds bw output. It
// sanitizes the message and wraps it to fit a screen of the given width,
// with a 2-cell margin on each side, and to at most maxWidth cells. A width
// that leaves no space for text does not wrap.
func errorBlock(msg string, width, maxWidth int) string {
	errText := ui.SanitizeText(msg)
	// The first line starts with the glyph and a space. The other lines
	// start with an indent of the same width.
	const indent = "  "
	if w := min(width-4, maxWidth) - len(indent); w > 0 {
		errText = ansi.Wrap(errText, w, "")
	}
	lines := strings.Split(errText, "\n")
	for i, line := range lines {
		// Style each line on its own: a style on all lines pads them to
		// the same width.
		prefix := indent
		if i == 0 {
			prefix = ui.GlyphError + " "
		}
		lines[i] = prefix + ui.StyleError.Render(line)
	}
	return strings.Join(lines, "\n")
}

// FooterHints returns the hint bindings for the footer.
func (m ErrorModel) FooterHints() []ui.HintBinding {
	if m.fatal {
		return []ui.HintBinding{{Key: "q", Desc: "quit"}}
	}
	return []ui.HintBinding{
		{Key: "r", Desc: "retry"},
		{Key: "q", Desc: "quit"},
	}
}

// View implements tea.Model (delegates to root frame).
func (m ErrorModel) View() tea.View {
	return tea.NewView(m.ViewContent(0))
}
