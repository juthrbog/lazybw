package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Overlay card padding inside the 1-cell border, in cells.
const (
	cardPadX = 2
	cardPadY = 1
)

// The cells that the overlay card border and padding add to the content.
const (
	OverlayEdgeWidth  = 2*cardPadX + 2 // on the left and the right together
	OverlayEdgeHeight = 2*cardPadY + 2 // on the top and the bottom together
)

// RenderOverlay composites a foreground popup card centered over a
// dimmed background using lipgloss v2's Compositor/Layer API. The result
// is exactly width x height cells. Content that does not fit in the card is
// cut, so that the card keeps its border. In an area that is smaller than
// the border, the card itself is cut.
func RenderOverlay(bg, fg string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	// Pad background to fill the full area so the compositor
	// canvas is correctly sized.
	bg = padToArea(ansi.Strip(bg), width, height)

	// Dim the background to visually separate it from the overlay. The
	// background is dimmed as plain text: each styled segment ends with a
	// full reset, which would also end a faint style set at the line start.
	dimmed := lipgloss.NewStyle().Faint(true).Foreground(ColorFaint).Render(bg)

	// Cut the content to the space inside the border and the padding.
	// MaxWidth and MaxHeight treat 0 as no limit, so skip them then.
	if w := width - OverlayEdgeWidth; w > 0 {
		fg = lipgloss.NewStyle().MaxWidth(w).Render(fg)
	}
	if h := height - OverlayEdgeHeight; h > 0 {
		fg = lipgloss.NewStyle().MaxHeight(h).Render(fg)
	}

	// Wrap foreground in a bordered card with padding.
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorHighlight).
		Padding(cardPadY, cardPadX).
		Render(fg)
	card = lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(card)

	x := (width - lipgloss.Width(card)) / 2
	y := (height - lipgloss.Height(card)) / 2

	bgLayer := lipgloss.NewLayer(dimmed)
	fgLayer := lipgloss.NewLayer(card).X(x).Y(y).Z(1)

	// The compositor drops unstyled spaces at the end of a line, for
	// example the card padding when the card fills the width. Pad again.
	return padToArea(lipgloss.NewCompositor(bgLayer, fgLayer).Render(), width, height)
}

// padToArea ensures content fills exactly width x height. It pads short
// lines and cuts long lines to width, and it drops the lines past height
// or adds empty lines to reach it.
func padToArea(content string, width, height int) string {
	lines := strings.Split(content, "\n")

	// Remove trailing empty line from Split if content ends with \n.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var b strings.Builder
	for i := 0; i < height; i++ {
		if i < len(lines) {
			line := ansi.Truncate(lines[i], width, "")
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
