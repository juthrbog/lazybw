package ui

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func TestRenderHintsEmpty(t *testing.T) {
	if got := RenderHints(nil, 80); got != "" {
		t.Errorf("empty hints should return empty string, got %q", got)
	}
}

func TestRenderHintsSingleHint(t *testing.T) {
	hints := []HintBinding{{Key: "q", Desc: "quit"}}
	out := RenderHints(hints, 80)
	if out == "" {
		t.Error("single hint should produce output")
	}
	if !contains(out, "q") || !contains(out, "quit") {
		t.Errorf("output should contain key and desc, got %q", out)
	}
}

func TestRenderHintsFullWidth(t *testing.T) {
	hints := []HintBinding{
		{Key: "j/k", Desc: "navigate"},
		{Key: "q", Desc: "quit"},
	}
	out := RenderHints(hints, 200)
	if !contains(out, "navigate") || !contains(out, "quit") {
		t.Errorf("wide width should show all hints, got %q", out)
	}
}

func TestRenderHintsProgressiveDrop(t *testing.T) {
	hints := []HintBinding{
		{Key: "j/k", Desc: "navigate"},
		{Key: "/", Desc: "search"},
		{Key: "c", Desc: "pwd"},
		{Key: "q", Desc: "quit"},
	}
	// Render all to find full width, then use a narrower width.
	full := RenderHints(hints, 200)
	fullW := lipgloss.Width(full)

	// Use half the full width — should drop some hints.
	narrow := RenderHints(hints, fullW/2)
	narrowW := lipgloss.Width(narrow)
	if narrowW > fullW/2 && narrowW > lipgloss.Width(RenderHints(hints[:1], 200)) {
		t.Errorf("narrow render should drop hints to fit, got width %d (max %d)", narrowW, fullW/2)
	}
}

func TestRenderHintsSingleAlwaysRenders(t *testing.T) {
	hints := []HintBinding{{Key: "q", Desc: "quit"}}
	// Even with width=0, one hint should always render.
	out := RenderHints(hints, 0)
	if out == "" {
		t.Error("single hint should always render even at width 0")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
