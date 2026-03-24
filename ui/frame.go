package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderHeader renders the always-visible header bar.
func RenderHeader(email string, width int) string {
	left := lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("lazybw")

	if email == "" {
		return StyleStatusBar.Width(width).Render(left)
	}

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(email)
	gap := width - leftW - rightW - 2 // 2 for padding(0,1)
	if gap < 1 {
		gap = 1
	}

	content := left + strings.Repeat(" ", gap) + email
	return StyleStatusBar.Width(width).Render(content)
}

// RenderFooter renders the always-visible footer bar with hints and status.
func RenderFooter(hints, status string, width int) string {
	left := StyleFaint.Render(hints)
	right := status

	if left == "" && right == "" {
		return StyleStatusBar.Width(width).Render("")
	}

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := width - leftW - rightW - 2 // 2 for padding(0,1)
	if gap < 1 {
		gap = 1
	}

	content := left + strings.Repeat(" ", gap) + right
	return StyleStatusBar.Width(width).Render(content)
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
