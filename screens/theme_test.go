package screens

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/ui"
)

// keepTheme restores the theme and the background after the test.
func keepTheme(t *testing.T) {
	t.Helper()
	dark, theme := ui.IsDark, ui.CurrentTheme
	t.Cleanup(func() {
		ui.IsDark = dark
		ui.ApplyTheme(theme)
	})
}

func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// pickTheme opens the theme picker, moves to the theme name and presses
// enter. It returns the messages of the commands that enter returns.
func pickTheme(t *testing.T, m VaultModel, name string) (VaultModel, []tea.Msg) {
	t.Helper()
	m = press(t, m, keyMsg("T"))
	for range len(ui.ThemeNames) {
		if *m.selectedTheme == name {
			break
		}
		m = press(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if *m.selectedTheme != name {
		t.Fatalf("picker cursor on %q, want %q", *m.selectedTheme, name)
	}
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(VaultModel)
	msgs := runMsgs(cmd)
	// huh completes the form with an internal message. Feed the messages
	// back until the picker closes.
	for i := 0; m.showThemePicker && i < 5; i++ {
		var next []tea.Msg
		for _, msg := range msgs {
			updated, cmd = m.Update(msg)
			m = updated.(VaultModel)
			next = append(next, runMsgs(cmd)...)
		}
		msgs = next
	}
	if m.showThemePicker {
		t.Fatal("the picker must close after enter")
	}
	return m, msgs
}

func TestThemePickerShowsNoHelpLine(t *testing.T) {
	m := press(t, vaultAt(testItems(), 80, 24), keyMsg("T"))
	plain := ansi.Strip(m.ViewContent(80, 24-ui.ChromeHeight))
	if !strings.Contains(plain, "Choose theme") {
		t.Fatalf("the picker must show:\n%s", plain)
	}
	for _, help := range []string{"submit", "↓ down", "/ filter"} {
		if strings.Contains(plain, help) {
			t.Errorf("the picker must show no huh help line, found %q:\n%s", help, plain)
		}
	}
	hints, _ := m.FooterContent()
	for _, k := range []string{"enter", "esc", "↑/↓", "/"} {
		if !hasHintKey(hints, k) {
			t.Errorf("picker footer = %v, want the key %q", hints, k)
		}
	}
}

func TestThemeChangeRestylesVault(t *testing.T) {
	keepTheme(t)
	ui.ApplyTheme("catppuccin-mocha")
	m := vaultAt(testItems(), 80, 24)
	m, msgs := pickTheme(t, m, "dracula")
	if ui.CurrentTheme != "dracula" {
		t.Fatalf("theme = %q, want dracula", ui.CurrentTheme)
	}
	if got := m.syncSpinner.Style.GetForeground(); !sameColor(got, ui.ColorHighlight) {
		t.Errorf("sync spinner color = %v, want the dracula highlight %v", got, ui.ColorHighlight)
	}
	if got := m.list.FilterInput.Styles().Focused.Prompt.GetForeground(); !sameColor(got, ui.ColorHighlight) {
		t.Errorf("filter prompt color = %v, want the dracula highlight %v", got, ui.ColorHighlight)
	}
	if !hasMsgOfType[ThemeChangedMsg](msgs) {
		t.Errorf("a theme change must send ThemeChangedMsg to the root model, got %#v", msgs)
	}
}

func TestBackgroundChangeRestylesList(t *testing.T) {
	keepTheme(t)
	m := vaultAt(testItems(), 80, 24)
	ui.IsDark = false
	ui.ApplyTheme("catppuccin-mocha")
	updated, _ := m.Update(ThemeChangedMsg{})
	m = updated.(VaultModel)
	// The blurred filter text is ANSI 245 on light backgrounds and ANSI 7
	// on dark ones.
	if got, want := m.list.FilterInput.Styles().Blurred.Text.GetForeground(), ui.ListStyles().Filter.Blurred.Text.GetForeground(); !sameColor(got, want) {
		t.Errorf("filter text color = %v, want the light color %v", got, want)
	}
	if got := m.syncSpinner.Style.GetForeground(); !sameColor(got, ui.ColorHighlight) {
		t.Errorf("sync spinner color = %v, want the light highlight %v", got, ui.ColorHighlight)
	}
}

func hasMsgOfType[T any](msgs []tea.Msg) bool {
	for _, msg := range msgs {
		if _, ok := msg.(T); ok {
			return true
		}
	}
	return false
}
