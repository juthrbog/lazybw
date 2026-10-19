package screens

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/juthrbog/lazybw/ui"
)

// updateQuitConfirm handles a key while the quit dialog shows. y quits, n
// and esc close the dialog, and other keys do nothing.
func (m VaultModel) updateQuitConfirm(msg tea.KeyPressMsg) (VaultModel, tea.Cmd) {
	switch msg.String() {
	case "y":
		m.overlay = overlayNone
		return m, func() tea.Msg { return QuitMsg{} }
	case "n", "esc":
		m.overlay = overlayNone
	}
	return m, nil
}

// renderQuitConfirm renders the card of the quit dialog.
func renderQuitConfirm() string {
	keyStyle := lipgloss.NewStyle().Foreground(ui.ColorHighlight).Bold(true)

	var b strings.Builder
	b.WriteString(ui.StyleTitle.Render("Quit lazybw?"))
	b.WriteString("\n\n")
	b.WriteString("This will lock your vault and exit.")
	b.WriteString("\n\n")
	b.WriteString(keyStyle.Render("y"))
	b.WriteString(ui.StyleFaint.Render(" yes"))
	b.WriteString(ui.StyleFaint.Render(" · "))
	b.WriteString(keyStyle.Render("n"))
	b.WriteString(ui.StyleFaint.Render(" no"))
	return b.String()
}

// quitHints returns the footer hints while the quit dialog shows.
func quitHints() []ui.HintBinding {
	return []ui.HintBinding{
		{Key: "y", Desc: "yes"},
		{Key: "n", Desc: "no"},
	}
}
