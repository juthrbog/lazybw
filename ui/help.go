package ui

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// HelpBinding pairs a key label with a description and a category
// for display in the grouped help overlay.
type HelpBinding struct {
	Key      string
	Desc     string
	Category string
}

// categoryOrder defines the display order for help binding categories.
var categoryOrder = []string{"Copy", "Navigation", "Vault", "UI"}

// HelpOverlay is a scrollable, filterable overlay that displays keybindings
// grouped by category.
type HelpOverlay struct {
	viewport viewport.Model
	visible  bool
	filter   string
	bindings []HelpBinding
	width    int
	height   int
}

// NewHelpOverlay creates a hidden help overlay.
func NewHelpOverlay() HelpOverlay {
	return HelpOverlay{}
}

// Show opens the help overlay with the given bindings.
func (h *HelpOverlay) Show(bindings []HelpBinding, width, height int) {
	h.bindings = bindings
	h.width = width
	h.height = height
	h.filter = ""
	h.visible = true
	h.viewport = viewport.New()
	h.viewport.SetWidth(h.contentWidth())
	h.viewport.SetHeight(h.contentHeight())
	h.viewport.SetContent(h.renderContent())
}

// Hide dismisses the help overlay.
func (h *HelpOverlay) Hide() {
	h.visible = false
}

// Visible returns whether the help overlay is currently shown.
func (h HelpOverlay) Visible() bool {
	return h.visible
}

// Update handles input when the help overlay is visible.
func (h HelpOverlay) Update(msg tea.Msg) (HelpOverlay, tea.Cmd) {
	if !h.visible {
		return h, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		k := msg.String()
		switch k {
		case "?", "esc":
			h.visible = false
			return h, nil
		case "backspace":
			if len(h.filter) > 0 {
				h.filter = h.filter[:len(h.filter)-1]
				h.viewport.SetContent(h.renderContent())
				h.viewport.GotoTop()
			}
			return h, nil
		default:
			// Single printable character → append to filter.
			if len(k) == 1 && k[0] >= 32 && k[0] < 127 {
				h.filter += k
				h.viewport.SetContent(h.renderContent())
				h.viewport.GotoTop()
				return h, nil
			}
		}

		// Forward scroll keys to viewport.
		var cmd tea.Cmd
		h.viewport, cmd = h.viewport.Update(msg)
		return h, cmd
	}

	return h, nil
}

// View renders the help overlay as a bordered popup.
func (h HelpOverlay) View() string {
	if !h.visible {
		return ""
	}

	var header strings.Builder
	header.WriteString(StyleTitle.Render("Keybindings"))
	if h.filter != "" {
		header.WriteString("  " + StyleFaint.Render("/ "+h.filter))
	} else {
		header.WriteString("  " + StyleFaint.Render("type to filter"))
	}
	header.WriteString("\n\n")

	footer := "\n" + StyleFaint.Render("?/esc close")

	box := lipgloss.NewStyle().
		Padding(0, 1).
		Width(h.boxWidth())

	return box.Render(header.String() + h.viewport.View() + footer)
}

func (h HelpOverlay) boxWidth() int {
	w := h.width * 3 / 4
	if w > 70 {
		w = 70
	}
	if w < 40 {
		w = h.width - 4
	}
	return w
}

func (h HelpOverlay) contentWidth() int {
	// box width minus horizontal padding (2).
	return h.boxWidth() - 2
}

func (h HelpOverlay) contentHeight() int {
	// screen height minus RenderOverlay border+padding (6) minus internal padding (0)
	// minus header (2) minus footer (2) minus visual margin (4).
	ch := h.height - 14
	if ch < 5 {
		ch = 5
	}
	return ch
}

// renderContent builds the grouped, filtered binding content.
func (h HelpOverlay) renderContent() string {
	type group struct {
		name     string
		bindings []HelpBinding
	}

	// Build groups in defined order.
	grouped := make(map[string][]HelpBinding)
	for _, b := range h.bindings {
		if h.filter != "" {
			lower := strings.ToLower(h.filter)
			if !strings.Contains(strings.ToLower(b.Key), lower) &&
				!strings.Contains(strings.ToLower(b.Desc), lower) &&
				!strings.Contains(strings.ToLower(b.Category), lower) {
				continue
			}
		}
		grouped[b.Category] = append(grouped[b.Category], b)
	}

	var ordered []group
	for _, cat := range categoryOrder {
		if bindings, ok := grouped[cat]; ok {
			ordered = append(ordered, group{name: cat, bindings: bindings})
			delete(grouped, cat)
		}
	}
	// Any remaining unknown categories.
	for cat, bindings := range grouped {
		ordered = append(ordered, group{name: cat, bindings: bindings})
	}

	if len(ordered) == 0 {
		return StyleFaint.Render("  no matches")
	}

	var b strings.Builder
	for i, g := range ordered {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(StyleHelpGroup.Render(g.name) + "\n")
		for _, binding := range g.bindings {
			line := StyleHelpKey.Render(binding.Key) + StyleFaint.Render(binding.Desc)
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}
