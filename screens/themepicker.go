package screens

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/juthrbog/lazybw/ui"
)

// ThemePickerModel is the theme picker card of the vault screen. It only
// picks a theme. The vault screen applies it.
type ThemePickerModel struct {
	form     *huh.Form
	selected *string // the theme under the cursor
	selectH  int     // height of the select in form
	height   int     // height of the screen
	closed   bool
	picked   bool // enter closed the picker on *selected
}

// NewThemePickerModel returns a picker for a screen of the given height,
// with the cursor on the theme current, and the init command of its form.
func NewThemePickerModel(current string, height int) (ThemePickerModel, tea.Cmd) {
	p := ThemePickerModel{selected: &current, height: height}
	return p, p.build()
}

// build builds the form for the screen height and returns its init
// command. The cursor starts on *p.selected, so a rebuild keeps the theme
// that the user moved to. The form shows no help, because the footer shows
// the keys.
func (p *ThemePickerModel) build() tea.Cmd {
	options := make([]huh.Option[string], len(ui.ThemeNames))
	for i, name := range ui.ThemeNames {
		options[i] = huh.NewOption(name, name)
	}
	p.selectH = themeSelectHeight(p.height)
	p.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose theme").
				Options(options...).
				Value(p.selected).
				Height(p.selectH),
		),
	).WithWidth(40).WithShowHelp(false)
	if ui.HuhTheme != nil {
		p.form = p.form.WithTheme(ui.HuhTheme)
	}
	return p.form.Init()
}

// themeSelectHeight returns the height of the theme select for a screen of
// the given height: the title and one row for each theme, or fewer rows on
// a short screen. A select with fewer rows scrolls, so that the cursor stays
// visible. The select must fit in the overlay card. The form has no help
// line, so it adds no rows below the select.
func themeSelectHeight(height int) int {
	rows := height - ui.ChromeHeight - ui.OverlayEdgeHeight
	return max(2, min(len(ui.ThemeNames)+1, rows))
}

// Resize fits the picker to a new screen height. It rebuilds the form only
// when the height of its select changes, because Form.Init asks for the
// window size, and a rebuild on each resize message would never stop. The
// command is nil when the form stays.
func (p ThemePickerModel) Resize(height int) (ThemePickerModel, tea.Cmd) {
	p.height = height
	if themeSelectHeight(height) == p.selectH {
		return p, nil
	}
	return p, p.build()
}

// Restyle rebuilds the form, because the select copies some styles when it
// is built.
func (p ThemePickerModel) Restyle() (ThemePickerModel, tea.Cmd) {
	return p, p.build()
}

// Closed reports whether the picker closed.
func (p ThemePickerModel) Closed() bool { return p.closed }

// Picked returns the theme that enter chose. It reports false when the
// picker is open or the user cancelled it.
func (p ThemePickerModel) Picked() (string, bool) {
	if !p.picked {
		return "", false
	}
	return *p.selected, true
}

// updateThemePicker gives a message to the theme picker of the vault. When
// the picker closes on a theme, the vault applies it, restyles itself and
// tells the root model.
func (m VaultModel) updateThemePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	if !m.picker.Closed() {
		return m, cmd
	}
	m.overlay = overlayNone
	name, ok := m.picker.Picked()
	if !ok {
		return m, nil
	}
	ui.ApplyTheme(name)
	m.currentTheme = ui.CurrentTheme
	m.restyle()
	// The root model restyles its own spinner on ThemeChangedMsg.
	changed := func() tea.Msg { return ThemeChangedMsg{} }
	return m, tea.Batch(m.setToast("Theme: "+ui.CurrentTheme), changed)
}

// Update handles a message while the picker shows. The form completes over
// internal messages, so it gets all messages, not only the keys.
func (p ThemePickerModel) Update(msg tea.Msg) (ThemePickerModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "esc" {
		p.closed = true
		return p, nil
	}

	form, cmd := p.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		p.form = f
	}

	switch p.form.State {
	case huh.StateCompleted:
		p.closed = true
		p.picked = true
		return p, nil
	case huh.StateAborted:
		p.closed = true
		return p, nil
	}
	return p, cmd
}

// View renders the picker card.
func (p ThemePickerModel) View() string {
	return p.form.View()
}

// FooterHints returns the footer hints. The form shows no help, so the
// footer shows its keys.
func (p ThemePickerModel) FooterHints() []ui.HintBinding {
	return []ui.HintBinding{
		{Key: "enter", Desc: "select"},
		{Key: "esc", Desc: "cancel"},
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "/", Desc: "filter"},
	}
}
