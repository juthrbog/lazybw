package ui

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func TestGradientTextNonEmpty(t *testing.T) {
	out := GradientText("hello", GradientFrom, GradientTo)
	if out == "" {
		t.Error("gradient text should not be empty")
	}
	// Each character is individually styled, so check visual width matches.
	if w := lipgloss.Width(out); w != 5 {
		t.Errorf("gradient text should be 5 chars wide, got %d", w)
	}
}

func TestGradientTextEmpty(t *testing.T) {
	out := GradientText("", GradientFrom, GradientTo)
	if out != "" {
		t.Errorf("empty input should return empty string, got %q", out)
	}
}

func TestRenderGradientLineWidth(t *testing.T) {
	out := RenderGradientLine(40)
	w := lipgloss.Width(out)
	if w != 40 {
		t.Errorf("gradient line should be 40 chars wide, got %d", w)
	}
}

func TestRenderGradientLineZero(t *testing.T) {
	out := RenderGradientLine(0)
	if out != "" {
		t.Errorf("zero width should return empty string, got %q", out)
	}
}
