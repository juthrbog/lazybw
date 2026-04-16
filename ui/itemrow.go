package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
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

// RenderItemRow renders a single item row in the vault list.
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

	name := item.Name

	// cursor(2-4) + glyph(1) + space(2) + name
	maxName := width - lipgloss.Width(cursor) - 1 - 2
	if maxName < 4 {
		maxName = 4
	}
	if lipgloss.Width(name) > maxName {
		name = name[:maxName-1] + "…"
	}

	row := fmt.Sprintf("%s%s  %s", cursor, glyph, name)

	if selected {
		row = StyleSelected.Render(row)
	}

	return row
}
