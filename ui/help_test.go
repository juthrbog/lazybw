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

	// Press / and type "cop" to filter.
	h, _ = h.Update(tea.KeyPressMsg{Text: "/"})
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

	h, _ = h.Update(tea.KeyPressMsg{Text: "/"})
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

	h, _ = h.Update(tea.KeyPressMsg{Text: "/"})
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

// helpAt opens the help overlay with all vault bindings in the content
// area of a terminal of the given size.
func helpAt(width, height int) HelpOverlay {
	h := NewHelpOverlay()
	h.Show(DefaultVaultKeyMap().HelpBindings(), width, height-ChromeHeight)
	return h
}

func TestHelpOverlayShowsMostBindingsAt80x24(t *testing.T) {
	h := helpAt(80, 24)
	if n := h.viewport.VisibleLineCount(); n < 14 {
		t.Errorf("help shows %d lines of bindings, want at least 14", n)
	}
	if !strings.Contains(h.View(), "↓ more") {
		t.Error("help must show a cue when more lines exist")
	}
	out := RenderOverlay("", h.View(), 80, 24-ChromeHeight)
	assertArea(t, out, 80, 24-ChromeHeight)
	assertCardBorders(t, out)

	// At the end of the list, the cue goes away.
	h.viewport.GotoBottom()
	if strings.Contains(h.View(), "↓ more") {
		t.Error("help at the bottom must not show the more cue")
	}
}

func TestHelpOverlayScrollKeysBeforeFilter(t *testing.T) {
	h := helpAt(80, 24)
	h, _ = h.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	h, _ = h.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if got := h.viewport.YOffset(); got != 2 {
		t.Errorf("after j j, offset = %d, want 2", got)
	}
	h, _ = h.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if got := h.viewport.YOffset(); got != 1 {
		t.Errorf("after k, offset = %d, want 1", got)
	}
	if h.filter != "" {
		t.Errorf("j and k must not change the filter, got %q", h.filter)
	}

	// After /, j and k go into the filter.
	h, _ = h.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	h, _ = h.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	h, _ = h.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if h.filter != "jk" {
		t.Errorf("after /, filter = %q, want %q", h.filter, "jk")
	}

	// Enter keeps the filter and gives the scroll keys back.
	h, _ = h.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	h, _ = h.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if h.filter != "jk" || h.Filtering() {
		t.Errorf("after enter, filter = %q and filtering = %v, want %q and false", h.filter, h.Filtering(), "jk")
	}

	// Esc while typing clears the filter and keeps the overlay open.
	h, _ = h.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	h, _ = h.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if h.filter != "" || !h.Visible() {
		t.Errorf("esc while typing: filter = %q, visible = %v", h.filter, h.Visible())
	}
}

func TestHelpOverlayResizeKeepsState(t *testing.T) {
	h := helpAt(80, 24)
	h, _ = h.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	h, _ = h.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	h.Resize(60, 20-ChromeHeight)
	if got := h.viewport.YOffset(); got != 2 {
		t.Errorf("Resize moved the offset to %d, want 2", got)
	}

	h, _ = h.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	h, _ = h.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	for _, size := range [][2]int{{50, 15}, {30, 10}, {120, 50}} {
		width, height := size[0], size[1]-ChromeHeight
		h.Resize(width, height)
		if h.filter != "o" || !h.Filtering() {
			t.Errorf("%v: Resize lost the filter, got %q", size, h.filter)
		}
		out := RenderOverlay("", h.View(), width, height)
		assertArea(t, out, width, height)
		if height >= 8 {
			// The area has room for the whole card.
			assertCardBorders(t, out)
		}
	}
}
