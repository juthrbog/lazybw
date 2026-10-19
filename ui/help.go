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
// grouped by category. The scroll keys work until the user presses "/".
// Then the typed keys go into the filter until enter or esc.
type HelpOverlay struct {
	viewport  viewport.Model
	visible   bool
	filtering bool // true while the typed keys go into the filter
	filter    string
	bindings  []HelpBinding
	width     int
	height    int
}

// NewHelpOverlay creates a hidden help overlay.
func NewHelpOverlay() HelpOverlay {
	return HelpOverlay{}
}

// Show opens the help overlay with the given bindings. Width and height are
// the size of the content area that the overlay card is drawn in.
func (h *HelpOverlay) Show(bindings []HelpBinding, width, height int) {
	h.bindings = bindings
	h.width = width
	h.height = height
	h.filter = ""
	h.filtering = false
	h.visible = true
	h.viewport = viewport.New()
	h.viewport.SetWidth(h.contentWidth())
	h.viewport.SetHeight(h.contentHeight())
	h.viewport.SetContent(h.renderContent())
}

// Resize fits the overlay to a new content area. It keeps the filter and
// the scroll position.
func (h *HelpOverlay) Resize(width, height int) {
	h.width = width
	h.height = height
	if !h.visible {
		return
	}
	offset := h.viewport.YOffset()
	h.viewport.SetWidth(h.contentWidth())
	h.viewport.SetHeight(h.contentHeight())
	h.viewport.SetContent(h.renderContent())
	h.viewport.SetYOffset(offset)
}

// Hide dismisses the help overlay.
func (h *HelpOverlay) Hide() {
	h.visible = false
}

// Visible returns whether the help overlay is currently shown.
func (h HelpOverlay) Visible() bool {
	return h.visible
}

// Filtering returns whether the typed keys go into the filter.
func (h HelpOverlay) Filtering() bool {
	return h.filtering
}

// FooterHints returns the footer hints for the keys of the overlay.
func (h HelpOverlay) FooterHints() []HintBinding {
	if h.filtering {
		return []HintBinding{
			{Key: "enter", Desc: "done"},
			{Key: "esc", Desc: "clear"},
		}
	}
	return []HintBinding{
		{Key: "esc", Desc: "close"},
		{Key: "j/k", Desc: "scroll"},
		{Key: "/", Desc: "filter"},
	}
}

// Update handles input when the help overlay is visible.
func (h HelpOverlay) Update(msg tea.Msg) (HelpOverlay, tea.Cmd) {
	if !h.visible {
		return h, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if h.filtering {
			h.updateFilter(msg.String())
			return h, nil
		}
		switch msg.String() {
		case "?", "esc":
			h.visible = false
			return h, nil
		case "/":
			h.filtering = true
			return h, nil
		}

		// Forward scroll keys to viewport.
		var cmd tea.Cmd
		h.viewport, cmd = h.viewport.Update(msg)
		return h, cmd
	}

	return h, nil
}

// updateFilter handles a key while the user types a filter. Enter keeps the
// filter, and esc clears it. Both return to the scroll keys.
func (h *HelpOverlay) updateFilter(k string) {
	switch k {
	case "enter":
		h.filtering = false
	case "esc":
		h.filtering = false
		h.setFilter("")
	case "backspace":
		// The filter holds only printable ASCII, so one byte is one key.
		if len(h.filter) > 0 {
			h.setFilter(h.filter[:len(h.filter)-1])
		}
	default:
		// Single printable character → append to filter.
		if len(k) == 1 && k[0] >= 32 && k[0] < 127 {
			h.setFilter(h.filter + k)
		}
	}
}

func (h *HelpOverlay) setFilter(filter string) {
	h.filter = filter
	h.viewport.SetContent(h.renderContent())
	h.viewport.GotoTop()
}

// View renders the help overlay as a bordered popup.
func (h HelpOverlay) View() string {
	if !h.visible {
		return ""
	}

	header := StyleTitle.Render("Keybindings")
	switch {
	case h.filtering:
		header += "  " + StyleFaint.Render("/ "+h.filter+"_")
	case h.filter != "":
		header += "  " + StyleFaint.Render("/ "+h.filter)
	default:
		header += "  " + StyleFaint.Render("/ to filter")
	}

	footer := "?/esc close · ↑/↓ scroll"
	if h.filtering {
		footer = "enter done · esc clear"
	}
	if !h.viewport.AtBottom() {
		footer += " · ↓ more"
	}

	// Cut the header and the footer, so that the box does not wrap them
	// into more lines than contentHeight counts.
	w := h.contentWidth()
	header = Truncate(header, w)
	footer = StyleFaint.Render(Truncate(footer, w))

	box := lipgloss.NewStyle().
		Padding(0, 1).
		Width(h.boxWidth())

	return box.Render(header + "\n\n" + h.viewport.View() + "\n" + footer)
}

func (h HelpOverlay) boxWidth() int {
	w := h.width * 3 / 4
	if w > 70 {
		w = 70
	}
	if w < 40 {
		// The overlay card adds its border and padding.
		w = h.width - OverlayEdgeWidth
	}
	// Keep at least 1 cell for the content.
	return max(3, w)
}

func (h HelpOverlay) contentWidth() int {
	// box width minus horizontal padding (2).
	return h.boxWidth() - 2
}

func (h HelpOverlay) contentHeight() int {
	// The content area minus the overlay card border and padding (4), the
	// header (2) and the footer (1).
	return max(1, h.height-OverlayEdgeHeight-3)
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
