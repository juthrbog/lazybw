package ui

import "testing"

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
	if CurrentTheme != "nonexistent" {
		t.Errorf("CurrentTheme = %q, want %q", CurrentTheme, "nonexistent")
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
