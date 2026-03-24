package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/juthrbog/lazybw/bwcmd"
)

var (
	glyphLogin = lipgloss.NewStyle().Foreground(ColorHighlight).Render("●")
	glyphCard  = lipgloss.NewStyle().Foreground(ColorGreen).Render("♦")
	glyphNote  = lipgloss.NewStyle().Foreground(ColorYellow).Render("✎")
)

// RenderItemRow renders a single item row in the vault list.
func RenderItemRow(item bwcmd.Item, selected bool, width int) string {
	cursor := "  "
	if selected {
		cursor = "▶ "
	}

	var glyph string
	switch item.Type {
	case bwcmd.ItemTypeLogin:
		glyph = glyphLogin
	case bwcmd.ItemTypeCard:
		glyph = glyphCard
	case bwcmd.ItemTypeSecureNote:
		glyph = glyphNote
	default:
		glyph = " "
	}

	name := item.Name
	desc := item.Description()

	// cursor(2) + glyph(1) + space(2) + name + gap(2+) + desc
	fixedWidth := 2 + 1 + 2 + 2 // cursor + glyph + spacing + min gap
	nameMaxWidth := width - fixedWidth - lipgloss.Width(desc)
	if nameMaxWidth < 4 {
		nameMaxWidth = 4
	}

	// Truncate name if needed.
	if lipgloss.Width(name) > nameMaxWidth {
		name = name[:nameMaxWidth-1] + "…"
	}

	// Build the row with right-aligned description.
	left := fmt.Sprintf("%s%s  %s", cursor, glyph, name)
	leftW := lipgloss.Width(left)
	gap := width - leftW - lipgloss.Width(desc)
	if gap < 2 {
		gap = 2
	}

	row := left + strings.Repeat(" ", gap) + StyleFaint.Render(desc)

	if selected {
		row = StyleSelected.Render(row)
	}

	return row
}
