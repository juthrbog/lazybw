package ui

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestApplyThemeCatppuccinMocha(t *testing.T) {
	ApplyTheme("catppuccin-mocha")
	if CurrentTheme != "catppuccin-mocha" {
		t.Errorf("CurrentTheme = %q, want %q", CurrentTheme, "catppuccin-mocha")
	}
	if ColorHighlight == nil {
		t.Error("ColorHighlight should not be nil")
	}
	if HuhTheme == nil {
		t.Error("HuhTheme should not be nil")
	}
}

func TestApplyThemeDracula(t *testing.T) {
	ApplyTheme("dracula")
	if CurrentTheme != "dracula" {
		t.Errorf("CurrentTheme = %q, want %q", CurrentTheme, "dracula")
	}
	if ColorHighlight == nil {
		t.Error("ColorHighlight should not be nil")
	}
}

func TestApplyThemeUpdatesGlyphs(t *testing.T) {
	ApplyTheme("catppuccin-mocha")
	if GlyphLogin == "" {
		t.Error("GlyphLogin should not be empty after ApplyTheme")
	}
	if GlyphCard == "" {
		t.Error("GlyphCard should not be empty after ApplyTheme")
	}
}

func TestApplyThemeUnknownFallsToDefault(t *testing.T) {
	ApplyTheme("nonexistent")
	if CurrentTheme != DefaultTheme {
		t.Errorf("CurrentTheme = %q, want %q", CurrentTheme, DefaultTheme)
	}
	if ColorHighlight == nil {
		t.Error("ColorHighlight should not be nil after fallback")
	}
}

func TestThemeNamesNotEmpty(t *testing.T) {
	if len(ThemeNames) == 0 {
		t.Error("ThemeNames should not be empty")
	}
}

// The terminal backgrounds that the contrast tests use.
var (
	darkBg  = lipgloss.Color("#1e1e2e")
	lightBg = lipgloss.Color("#ffffff")
)

// withBackground sets IsDark for the test and restores IsDark and the theme
// afterwards.
func withBackground(t *testing.T, dark bool) {
	t.Helper()
	prevDark, prevTheme := IsDark, CurrentTheme
	t.Cleanup(func() {
		IsDark = prevDark
		ApplyTheme(prevTheme)
	})
	IsDark = dark
}

func TestThemeTextContrast(t *testing.T) {
	const minContrast = 3.0
	for _, dark := range []bool{true, false} {
		withBackground(t, dark)
		bg := lightBg
		if dark {
			bg = darkBg
		}
		for _, name := range ThemeNames {
			ApplyTheme(name)
			checks := []struct {
				what   string
				fg, bg color.Color
			}{
				{"faint text", ColorFaint, bg},
				{"yellow text", ColorYellow, bg},
				{"badge text", StyleHeaderBadge.GetForeground(), StyleHeaderBadge.GetBackground()},
			}
			for _, c := range checks {
				if got := contrastRatio(c.fg, c.bg); got < minContrast {
					t.Errorf("%s (dark %v): %s contrast = %.2f:1, want at least %.1f:1", name, dark, c.what, got, minContrast)
				}
			}
		}
	}
}

func TestCharmPickerOptionsReadableOnDark(t *testing.T) {
	withBackground(t, true)
	ApplyTheme("charm")
	styles := HuhTheme.Theme(false) // the theme follows IsDark, not huh
	for what, style := range map[string]lipgloss.Style{
		"option":            styles.Focused.Option,
		"unselected option": styles.Focused.UnselectedOption,
		"selected option":   styles.Focused.SelectedOption,
	} {
		if got := contrastRatio(style.GetForeground(), darkBg); got < 4.5 {
			t.Errorf("%s contrast = %.2f:1, want at least 4.5:1", what, got)
		}
	}
}

func TestHuhThemeFollowsIsDark(t *testing.T) {
	withBackground(t, true)
	ApplyTheme("catppuccin-mocha")
	theme := HuhTheme
	mauve := func() color.Color { return theme.Theme(false).Focused.Title.GetForeground() }
	if got, want := mauve(), lipgloss.Color("#cba6f7"); !sameColor(got, want) {
		t.Errorf("dark background: title color = %v, want the Mocha mauve %v", got, want)
	}
	// A form keeps the theme that it was built with. After the background
	// changes, the same theme gives the light styles.
	IsDark = false
	if got, want := mauve(), lipgloss.Color("#8839ef"); !sameColor(got, want) {
		t.Errorf("light background: title color = %v, want the Latte mauve %v", got, want)
	}
}

func TestContrastRatio(t *testing.T) {
	if got := contrastRatio(lipgloss.Color("#000000"), lipgloss.Color("#ffffff")); got < 20.99 || got > 21.01 {
		t.Errorf("black on white = %.2f, want 21", got)
	}
	if got := contrastRatio(lipgloss.Color("#767676"), lipgloss.Color("#ffffff")); got < 4.5 || got > 4.6 {
		t.Errorf("#767676 on white = %.2f, want 4.54", got)
	}
}

func sameColor(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}
