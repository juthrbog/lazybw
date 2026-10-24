package ui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// HintBinding pairs a key label with a human-readable description
// for display in the status bar footer.
type HintBinding struct {
	Key  string
	Desc string
}

// Hint returns the footer hint for b: the key label of the help of b, and
// desc. desc is a short text, because the footer has little room.
func Hint(b key.Binding, desc string) HintBinding {
	return HintBinding{Key: b.Help().Key, Desc: desc}
}

// HintPair returns one footer hint for two bindings, such as "+/-" for a
// pair of opposite keys or "1-4" for a range. sep goes between the two key
// labels.
func HintPair(a, b key.Binding, sep, desc string) HintBinding {
	return HintBinding{Key: a.Help().Key + sep + b.Help().Key, Desc: desc}
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
	sep := StyleHintSep.Render(Glyphs.Sep)
	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = StyleHintKey.Render(h.Key) + " " + StyleHintDesc.Render(h.Desc)
	}
	return strings.Join(parts, sep)
}
