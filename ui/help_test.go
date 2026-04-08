package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func testBindings() []HelpBinding {
	return []HelpBinding{
		{Key: "c", Desc: "copy password", Category: "Copy"},
		{Key: "t", Desc: "copy TOTP", Category: "Copy"},
		{Key: "j/k", Desc: "move up/down", Category: "Navigation"},
		{Key: "?", Desc: "help", Category: "UI"},
		{Key: "q", Desc: "quit", Category: "UI"},
	}
}

func TestHelpOverlayShowHide(t *testing.T) {
	h := NewHelpOverlay()
	if h.Visible() {
		t.Error("new overlay should not be visible")
	}

	h.Show(testBindings(), 80, 24)
	if !h.Visible() {
		t.Error("overlay should be visible after Show")
	}

	h.Hide()
	if h.Visible() {
		t.Error("overlay should not be visible after Hide")
	}
}

func TestHelpOverlayCloseOnEsc(t *testing.T) {
	h := NewHelpOverlay()
	h.Show(testBindings(), 80, 24)

	h, _ = h.Update(tea.KeyPressMsg{Text: "esc"})
	if h.Visible() {
		t.Error("overlay should close on esc")
	}
}

func TestHelpOverlayCloseOnQuestion(t *testing.T) {
	h := NewHelpOverlay()
	h.Show(testBindings(), 80, 24)

	h, _ = h.Update(tea.KeyPressMsg{Text: "?"})
	if h.Visible() {
		t.Error("overlay should close on ?")
	}
}

func TestHelpOverlayViewWhenHidden(t *testing.T) {
	h := NewHelpOverlay()
	if v := h.View(); v != "" {
		t.Errorf("hidden overlay should return empty string, got %q", v)
	}
}

func TestHelpOverlayRenderContent(t *testing.T) {
	h := NewHelpOverlay()
	h.Show(testBindings(), 80, 24)

	view := h.View()
	if !strings.Contains(view, "Keybindings") {
		t.Error("view should contain title")
	}
	if !strings.Contains(view, "Copy") {
		t.Error("view should contain Copy category")
	}
	if !strings.Contains(view, "Navigation") {
		t.Error("view should contain Navigation category")
	}
	if !strings.Contains(view, "UI") {
		t.Error("view should contain UI category")
	}
}

func TestHelpOverlayFilter(t *testing.T) {
	h := NewHelpOverlay()
	h.Show(testBindings(), 80, 24)

	// Type "cop" to filter.
	h, _ = h.Update(tea.KeyPressMsg{Text: "c"})
	h, _ = h.Update(tea.KeyPressMsg{Text: "o"})
	h, _ = h.Update(tea.KeyPressMsg{Text: "p"})

	view := h.View()
	if !strings.Contains(view, "copy") {
		t.Error("filtered view should contain matching bindings")
	}
	// "Navigation" category shouldn't match "cop".
	content := h.renderContent()
	if strings.Contains(content, "Navigation") {
		t.Error("filtered view should not contain non-matching categories")
	}
}

func TestHelpOverlayFilterNoMatches(t *testing.T) {
	h := NewHelpOverlay()
	h.Show(testBindings(), 80, 24)

	h, _ = h.Update(tea.KeyPressMsg{Text: "z"})
	h, _ = h.Update(tea.KeyPressMsg{Text: "z"})
	h, _ = h.Update(tea.KeyPressMsg{Text: "z"})

	content := h.renderContent()
	if !strings.Contains(content, "no matches") {
		t.Error("should show 'no matches' for bad filter")
	}
}

func TestHelpOverlayFilterBackspace(t *testing.T) {
	h := NewHelpOverlay()
	h.Show(testBindings(), 80, 24)

	h, _ = h.Update(tea.KeyPressMsg{Text: "z"})
	h, _ = h.Update(tea.KeyPressMsg{Text: "z"})

	// Backspace should remove last filter char.
	h, _ = h.Update(tea.KeyPressMsg{Text: "backspace"})
	if h.filter != "z" {
		t.Errorf("filter should be 'z' after backspace, got %q", h.filter)
	}

	// Backspace again to clear.
	h, _ = h.Update(tea.KeyPressMsg{Text: "backspace"})
	if h.filter != "" {
		t.Errorf("filter should be empty after backspace, got %q", h.filter)
	}
}
