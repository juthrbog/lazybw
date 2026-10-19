package screens

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/juthrbog/lazybw/ui"
)

// quitKeys are the keys of the quit dialog.
var quitKeys = ui.DefaultConfirmKeyMap()

// updateQuitConfirm handles a key while the quit dialog shows. y quits, n
// and esc close the dialog, and other keys do nothing.
func (m VaultModel) updateQuitConfirm(msg tea.KeyPressMsg) (VaultModel, tea.Cmd) {
	switch {
	case key.Matches(msg, quitKeys.Yes):
		m.overlay = overlayNone
		return m, func() tea.Msg { return QuitMsg{} }
	case key.Matches(msg, quitKeys.No):
		m.overlay = overlayNone
	}
	return m, nil
}

// renderQuitConfirm renders the card of the quit dialog.
func renderQuitConfirm() string {
	keyStyle := lipgloss.NewStyle().Foreground(ui.ColorHighlight).Bold(true)
	yes, no := quitKeys.Yes.Help(), quitKeys.No.Help()

	var b strings.Builder
	b.WriteString(ui.StyleTitle.Render("Quit lazybw?"))
	b.WriteString("\n\n")
	b.WriteString("This will lock your vault and exit.")
	b.WriteString("\n\n")
	b.WriteString(keyStyle.Render(yes.Key))
	b.WriteString(ui.StyleFaint.Render(" " + yes.Desc))
	b.WriteString(ui.StyleFaint.Render(" · "))
	b.WriteString(keyStyle.Render(no.Key))
	b.WriteString(ui.StyleFaint.Render(" " + no.Desc))
	return b.String()
}

// quitHints returns the footer hints while the quit dialog shows.
func quitHints() []ui.HintBinding {
	return []ui.HintBinding{
		ui.Hint(quitKeys.Yes, quitKeys.Yes.Help().Desc),
		ui.Hint(quitKeys.No, quitKeys.No.Help().Desc),
	}
}
