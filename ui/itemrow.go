package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
)

// RenderGroupRow renders a collapsible group header row.
func RenderGroupRow(baseName string, count int, expanded, selected bool, width int) string {
	cursor := "  "
	if selected {
		cursor = "▶ "
	}

	chevron := "▶"
	if expanded {
		chevron = "▼"
	}

	label := fmt.Sprintf("%s%s  %s (%d)", cursor, chevron, baseName, count)

	// Pad to full width.
	gap := width - lipgloss.Width(label)
	if gap > 0 {
		label += strings.Repeat(" ", gap)
	}

	if selected {
		label = StyleSelected.Render(label)
	} else {
		label = StyleFaint.Render(label)
	}

	return label
}

// Item row layout limits, in cells.
const (
	rowMinGap   = 2  // minimum gap between the name and the description
	rowMinNameW = 12 // a long description leaves the name this width, or 2/5 of the row if more
	rowMinDescW = 6  // a description that gets less space is dropped
)

// RenderItemRow renders a single item row in the vault list. The row is
// exactly width cells wide. The name gets its space first, so that a long
// description cannot hide it.
func RenderItemRow(item bwcmd.Item, selected bool, width int, indent bool) string {
	cursor := "  "
	if indent {
		cursor = "    "
	}
	if selected {
		if indent {
			cursor = "  ▶ "
		} else {
			cursor = "▶ "
		}
	}

	glyph := ItemGlyph(item.Type)

	name := SanitizeLine(item.Name)
	// The first line of a note with "\r\n" line ends keeps its "\r", which
	// becomes a trailing space. Trim it, so that the description stays
	// aligned to the right edge.
	desc := strings.TrimRight(SanitizeLine(item.Description()), " ")

	// cursor + glyph + 2 spaces + name + gap + desc
	prefixW := lipgloss.Width(cursor) + lipgloss.Width(glyph) + 2
	avail := width - prefixW

	nameW := lipgloss.Width(name)
	if nameW+rowMinGap+lipgloss.Width(desc) > avail {
		reserved := min(nameW, max(rowMinNameW, width*2/5))
		descMax := avail - reserved - rowMinGap
		if descMax < rowMinDescW {
			desc = ""
		} else {
			desc = Truncate(desc, descMax)
		}
	}
	descW := lipgloss.Width(desc)
	nameMax := avail
	if desc != "" {
		nameMax -= descW + rowMinGap
	}
	name = Truncate(name, nameMax)
	nameW = lipgloss.Width(name)

	gap := max(0, avail-nameW-descW)

	// Style each segment on its own. The glyph ends with a full reset,
	// which would also end a style that wraps the whole row.
	if selected {
		cursor = StyleSelected.Render(cursor)
		name = StyleSelected.Render(name)
	}
	if desc != "" {
		desc = StyleFaint.Render(desc)
	}
	row := cursor + glyph + "  " + name + strings.Repeat(" ", gap) + desc
	if avail < 0 {
		// The terminal is narrower than the cursor and the glyph.
		row = ansi.Truncate(row, max(0, width), "")
	}
	return row
}
