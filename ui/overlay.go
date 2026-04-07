package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// RenderOverlay composites a foreground popup card centered over a
// dimmed background using lipgloss v2's Compositor/Layer API.
func RenderOverlay(bg, fg string, width, height int) string {
	// Pad background to fill the full area so the compositor
	// canvas is correctly sized.
	bg = padToArea(bg, width, height)

	// Dim the background to visually separate it from the overlay.
	dimmed := lipgloss.NewStyle().Faint(true).Render(bg)

	// Wrap foreground in a bordered card with padding.
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSubtle).
		Padding(1, 2).
		Render(fg)

	cardW := lipgloss.Width(card)
	cardH := lipgloss.Height(card)

	// Clamp card to available area.
	if cardW > width {
		cardW = width
	}
	if cardH > height {
		cardH = height
	}

	x := (width - cardW) / 2
	y := (height - cardH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	bgLayer := lipgloss.NewLayer(dimmed)
	fgLayer := lipgloss.NewLayer(card).X(x).Y(y).Z(1)

	return lipgloss.NewCompositor(bgLayer, fgLayer).Render()
}

// padToArea ensures content fills exactly width x height by padding
// lines to width and adding empty lines to reach height.
func padToArea(content string, width, height int) string {
	lines := strings.Split(content, "\n")

	// Remove trailing empty line from Split if content ends with \n.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var b strings.Builder
	for i := 0; i < height; i++ {
		if i < len(lines) {
			line := lines[i]
			lineW := lipgloss.Width(line)
			b.WriteString(line)
			if lineW < width {
				b.WriteString(strings.Repeat(" ", width-lineW))
			}
		} else {
			b.WriteString(strings.Repeat(" ", width))
		}
		if i < height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
