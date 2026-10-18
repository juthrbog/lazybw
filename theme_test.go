package main

import (
	"bytes"
	"image/color"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/screens"
	"github.com/juthrbog/lazybw/ui"
)

// withTheme applies the theme name for the test. It restores the theme
// and the background afterwards.
func withTheme(t *testing.T, name string) {
	t.Helper()
	keepTheme(t)
	ui.ApplyTheme(name)
}

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

func TestSetThemeWarnsOnlyAboutUnknownNames(t *testing.T) {
	keepTheme(t)
	var w bytes.Buffer
	setTheme("dracula", &w)
	if w.Len() != 0 || ui.CurrentTheme != "dracula" {
		t.Errorf("a known theme must apply with no output: output %q, theme %q", w.String(), ui.CurrentTheme)
	}
	setTheme("", &w)
	if w.Len() != 0 || ui.CurrentTheme != "dracula" {
		t.Errorf("an empty name must change nothing: output %q, theme %q", w.String(), ui.CurrentTheme)
	}

	for _, name := range []string{"nosuchtheme", "catppucin-mocha"} {
		ui.ApplyTheme("dracula")
		var w bytes.Buffer
		setTheme(name, &w)
		out := w.String()
		if !strings.Contains(out, "Warning") || !strings.Contains(out, name) {
			t.Errorf("%s: warning = %q, want a warning that names the theme", name, out)
		}
		for _, valid := range ui.ThemeNames {
			if !strings.Contains(out, valid) {
				t.Errorf("%s: warning = %q, want it to list %q", name, out, valid)
			}
		}
		if ui.CurrentTheme != "catppuccin-mocha" {
			t.Errorf("%s: CurrentTheme = %q, want catppuccin-mocha", name, ui.CurrentTheme)
		}
	}
}

func TestThemeChangeRestylesRootSpinner(t *testing.T) {
	withTheme(t, "catppuccin-mocha")
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, keyPress("T"))
	// dracula is the fifth theme.
	for range 4 {
		m, _ = update(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	m, cmd := update(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	// Feed the messages back until the vault reports the theme change.
	changed := false
	for i := 0; !changed && i < 5; i++ {
		var cmds []tea.Cmd
		for _, msg := range collectMsgsWithin(cmd, 50*time.Millisecond) {
			if _, ok := msg.(screens.ThemeChangedMsg); ok {
				changed = true
			}
			var next tea.Cmd
			m, next = update(t, m, msg)
			cmds = append(cmds, next)
		}
		cmd = tea.Batch(cmds...)
	}
	if !changed {
		t.Fatal("the vault must send ThemeChangedMsg after a theme change")
	}
	if ui.CurrentTheme != "dracula" {
		t.Fatalf("theme = %q, want dracula", ui.CurrentTheme)
	}
	if got := m.spinner.Style.GetForeground(); !sameColor(got, ui.ColorHighlight) {
		t.Errorf("root spinner color = %v, want the dracula highlight %v", got, ui.ColorHighlight)
	}
}

func TestBackgroundColorRestylesScreens(t *testing.T) {
	keepTheme(t)
	ui.IsDark = true
	ui.ApplyTheme("catppuccin-mocha")
	m := rootIn(t, stateLocked, 80, 24)
	m, _ = update(t, m, tea.BackgroundColorMsg{Color: color.White})
	if ui.IsDark {
		t.Fatal("a white background must set ui.IsDark to false")
	}
	if got := m.spinner.Style.GetForeground(); !sameColor(got, ui.ColorHighlight) {
		t.Errorf("root spinner color = %v, want the light highlight %v", got, ui.ColorHighlight)
	}
	// The forms read ui.IsDark when they draw, so the huh theme now gives
	// the light (Latte) styles.
	if got := ui.HuhTheme.Theme(true).Focused.Title.GetForeground(); !sameColor(got, ui.ColorHighlight) {
		t.Errorf("form title color = %v, want the Latte mauve %v", got, ui.ColorHighlight)
	}
}
