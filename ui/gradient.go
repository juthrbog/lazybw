package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// GradientText renders text with a per-character horizontal gradient
// interpolated between from and to colors.
func GradientText(text string, from, to color.Color) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}
	gradient := lipgloss.Blend1D(len(runes), from, to)

	var b strings.Builder
	for i, r := range runes {
		idx := i
		if idx >= len(gradient) {
			idx = len(gradient) - 1
		}
		b.WriteString(lipgloss.NewStyle().Foreground(gradient[idx]).Render(string(r)))
	}
	return b.String()
}

// RenderGradientLine creates a thin horizontal gradient separator using
// half-block characters interpolated between GradientFrom and GradientTo.
func RenderGradientLine(width int) string {
	if width <= 0 {
		return ""
	}
	colors := lipgloss.Blend1D(width, GradientFrom, GradientTo)
	var b strings.Builder
	for _, c := range colors {
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render("▄"))
	}
	return b.String()
}
