package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// ChromeHeight is the number of rows that the frame uses around the content
// area: the header (2 rows) and the footer (1 row).
const ChromeHeight = 3

// HeaderData holds the values needed to render the header bar.
type HeaderData struct {
	Email        string
	ItemCount    int
	SelectedType string
	Width        int
}

// minHeaderEmailW is the narrowest email that the header shows. A narrower
// space drops the email badge.
const minHeaderEmailW = 6

// GlyphLogo is the glyph in front of the name in the header,
// nf-md-shield_lock.
const GlyphLogo = "\U000F099D"

// RenderHeader renders the 2-line header: title bar + gradient separator.
// Both lines are exactly data.Width cells wide.
func RenderHeader(data HeaderData) string {
	title := GradientText(" "+GlyphLogo+" lazybw", GradientFrom, GradientTo)

	// Build components left-to-right.
	bar := title
	if data.Email != "" {
		// The badge adds 1 cell of padding on each side, and a space
		// separates it from the title.
		const badgeChrome = 3
		emailW := data.Width - lipgloss.Width(title) - badgeChrome
		if emailW >= minHeaderEmailW {
			email := Truncate(SanitizeLine(data.Email), emailW)
			bar += " " + StyleHeaderBadge.Render(email)
		}
	}
	var extras []string
	if data.Width >= 80 && data.ItemCount > 0 {
		extras = append(extras, " "+StyleFaint.Render(fmt.Sprintf("%d items", data.ItemCount)))
	}
	if data.Width >= 60 && data.SelectedType != "" {
		extras = append(extras, " "+StyleFaint.Render(data.SelectedType))
	}
	// The email comes first. An extra that does not fit next to it is
	// dropped.
	for _, extra := range extras {
		if lipgloss.Width(bar)+lipgloss.Width(extra) <= data.Width {
			bar += extra
		}
	}

	return fitLine(bar, data.Width) + "\n" + RenderGradientLine(data.Width)
}

// RenderFooter renders the always-visible footer bar with hints and status.
// The status comes first, but it leaves room for the first hint. The hints
// that do not fit are dropped from the end. A status that is too long is
// cut, so that the footer is one line of exactly width cells.
func RenderFooter(hints []HintBinding, status string, width int) string {
	if len(hints) == 0 && status == "" {
		return strings.Repeat(" ", max(0, width))
	}

	// 1 cell of padding on each side.
	inner := width - 2
	maxRight := inner
	if len(hints) > 0 {
		// Leave room for the first hint and a 1-cell gap.
		maxRight -= lipgloss.Width(RenderHints(hints[:1], inner)) + 1
	}
	right := Truncate(status, maxRight)
	rightW := lipgloss.Width(right)

	// Reserve space for the status and a minimum gap.
	availHints := max(0, inner-rightW-1)
	left := RenderHints(hints, availHints)
	leftW := lipgloss.Width(left)
	gap := max(1, inner-leftW-rightW)

	return fitLine(" "+left+strings.Repeat(" ", gap)+right+" ", width)
}

// fitLine pads a line with spaces to width cells, or cuts it when it is
// wider.
func fitLine(line string, width int) string {
	width = max(0, width)
	w := lipgloss.Width(line)
	if w > width {
		return ansi.Truncate(line, width, "")
	}
	return line + strings.Repeat(" ", width-w)
}

// CenterInArea centers content both vertically and horizontally within the
// given dimensions. The result has exactly height lines. Content that is
// taller than the area is cut at the bottom.
func CenterInArea(content string, width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	maxLineW := 0
	for _, l := range lines {
		if w := lipgloss.Width(l); w > maxLineW {
			maxLineW = w
		}
	}

	padLeft := strings.Repeat(" ", max(0, (width-maxLineW)/2))
	padTop := max(0, (height-len(lines))/2)

	out := make([]string, 0, height)
	for range padTop {
		out = append(out, "")
	}
	for _, l := range lines {
		if len(out) == height {
			break
		}
		out = append(out, padLeft+l)
	}
	// Pad the bottom to fill the content area height.
	for len(out) < height {
		out = append(out, "")
	}
	return strings.Join(out, "\n")
}
