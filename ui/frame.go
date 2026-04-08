package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// HeaderData holds the values needed to render the header bar.
type HeaderData struct {
	Email        string
	ItemCount    int
	SelectedType string
	Width        int
}

// RenderHeader renders the 2-line header: title bar + gradient separator.
func RenderHeader(data HeaderData) string {
	title := GradientText(" 󰊙 lazybw", GradientFrom, GradientTo)

	// Build components left-to-right.
	parts := []string{title}
	if data.Email != "" {
		parts = append(parts, " "+StyleHeaderBadge.Render(data.Email))
	}
	if data.Width >= 80 && data.ItemCount > 0 {
		parts = append(parts, " "+StyleFaint.Render(fmt.Sprintf("%d items", data.ItemCount)))
	}
	if data.Width >= 60 && data.SelectedType != "" {
		parts = append(parts, " "+StyleFaint.Render(data.SelectedType))
	}

	bar := strings.Join(parts, "")
	barW := lipgloss.Width(bar)
	if pad := data.Width - barW; pad > 0 {
		bar += strings.Repeat(" ", pad)
	}

	return bar + "\n" + RenderGradientLine(data.Width)
}

// RenderFooter renders the always-visible footer bar with hints and status.
func RenderFooter(hints []HintBinding, status string, width int) string {
	right := status

	if len(hints) == 0 && right == "" {
		return strings.Repeat(" ", width)
	}

	rightW := lipgloss.Width(right)
	// Reserve space for right side + 1-char padding each side + minimum gap.
	availHints := width - rightW - 3
	if availHints < 0 {
		availHints = 0
	}

	left := RenderHints(hints, availHints)
	leftW := lipgloss.Width(left)
	gap := width - leftW - rightW - 2 // 1-char padding each side
	if gap < 1 {
		gap = 1
	}

	return " " + left + strings.Repeat(" ", gap) + right + " "
}

// CenterInArea centers content both vertically and horizontally
// within the given dimensions.
func CenterInArea(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	maxLineW := 0
	for _, l := range lines {
		if w := lipgloss.Width(l); w > maxLineW {
			maxLineW = w
		}
	}

	padLeft := (width - maxLineW) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	padTop := (height - len(lines)) / 2
	if padTop < 0 {
		padTop = 0
	}

	var b strings.Builder
	for i := 0; i < padTop; i++ {
		b.WriteString("\n")
	}
	for _, l := range lines {
		b.WriteString(strings.Repeat(" ", padLeft))
		b.WriteString(l)
		b.WriteString("\n")
	}
	// Pad bottom to fill the content area height.
	totalLines := padTop + len(lines)
	for i := totalLines; i < height; i++ {
		b.WriteString("\n")
	}
	return b.String()
}
