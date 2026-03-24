package ui

import "testing"

func TestApplyThemeCatppuccinMocha(t *testing.T) {
	ApplyTheme("catppuccin-mocha")
	if CurrentTheme != "catppuccin-mocha" {
		t.Errorf("CurrentTheme = %q, want %q", CurrentTheme, "catppuccin-mocha")
	}
	// Mocha mauve is #cba6f7.
	if ColorHighlight.Dark != "#cba6f7" {
		t.Errorf("Mocha highlight dark = %q, want #cba6f7", ColorHighlight.Dark)
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
	if ColorHighlight.Dark != "#bd93f9" {
		t.Errorf("Dracula highlight dark = %q, want #bd93f9", ColorHighlight.Dark)
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
	// Should fall through to catppuccin-mocha colors.
	if ColorHighlight.Dark != "#cba6f7" {
		t.Errorf("fallback highlight dark = %q, want #cba6f7", ColorHighlight.Dark)
	}
}

func TestThemeNamesNotEmpty(t *testing.T) {
	if len(ThemeNames) == 0 {
		t.Error("ThemeNames should not be empty")
	}
}
