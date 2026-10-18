package ui

import (
	"image/color"
	"math"

	"charm.land/lipgloss/v2"
)

// The text colors that textOn chooses from.
var (
	textDark  = lipgloss.Color("#1e1e2e")
	textLight = lipgloss.Color("#ffffff")
)

// textOn returns the text color for a background of color bg: textDark or
// textLight, whichever has the higher contrast with bg.
func textOn(bg color.Color) color.Color {
	if contrastRatio(textDark, bg) >= contrastRatio(textLight, bg) {
		return textDark
	}
	return textLight
}

// contrastRatio returns the WCAG 2 contrast ratio of two colors, from 1
// (no contrast) to 21 (black on white).
func contrastRatio(a, b color.Color) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// relativeLuminance returns the WCAG 2 relative luminance of c, from 0 for
// black to 1 for white.
func relativeLuminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	linear := func(v uint32) float64 {
		s := float64(v) / 0xffff
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
}
