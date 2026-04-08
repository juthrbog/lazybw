package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// HintBinding pairs a key label with a human-readable description
// for display in the status bar footer.
type HintBinding struct {
	Key  string
	Desc string
}

// RenderHints renders hint bindings with styled keys and descriptions,
// progressively dropping hints from the right until they fit within maxWidth.
func RenderHints(hints []HintBinding, maxWidth int) string {
	if len(hints) == 0 {
		return ""
	}

	for n := len(hints); n >= 1; n-- {
		result := joinHints(hints[:n])
		if lipgloss.Width(result) <= maxWidth || n == 1 {
			return result
		}
	}
	return ""
}

func joinHints(hints []HintBinding) string {
	sep := StyleHintSep.Render(" · ")
	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = StyleHintKey.Render(h.Key) + " " + StyleHintDesc.Render(h.Desc)
	}
	return strings.Join(parts, sep)
}
