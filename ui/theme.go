package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// Colour palette — AdaptiveColor adapts to dark/light terminal backgrounds.
var (
	ColorHighlight = lipgloss.AdaptiveColor{Dark: "#7D56F4", Light: "#5A3ECC"} // login glyph, selected
	ColorSubtle    = lipgloss.AdaptiveColor{Dark: "#383838", Light: "#D9DCCF"} // separator lines, bg
	ColorGreen     = lipgloss.AdaptiveColor{Dark: "#04B575", Light: "#027A4F"} // card glyph, copy toast, TOTP ok
	ColorYellow    = lipgloss.AdaptiveColor{Dark: "#F5A623", Light: "#C47D10"} // note glyph, TOTP warning
	ColorRed       = lipgloss.AdaptiveColor{Dark: "#FF4F4F", Light: "#CC2222"} // errors, TOTP urgent
	ColorFaint     = lipgloss.AdaptiveColor{Dark: "#626262", Light: "#9A9A9A"} // descriptions, secondary text
)

// Spinner definitions for loading states.
var (
	SpinnerUnlock = spinner.Spinner{
		Frames: []string{"󰌾", "󰷖", "󰌆", "󰌿"},
		FPS:    time.Second / 3,
	}
	SpinnerLock = spinner.Spinner{
		Frames: []string{"󰌿", "󰌆", "󰷖", "󰌾"},
		FPS:    time.Second / 3,
	}
	SpinnerLoad = spinner.Dot
)

// Pre-built styles initialised once at package load.
var (
	StyleTitle     lipgloss.Style
	StyleSelected  lipgloss.Style
	StyleFaint     lipgloss.Style
	StyleBorder    lipgloss.Style
	StyleStatusBar lipgloss.Style
	StyleToast     lipgloss.Style
	StyleError     lipgloss.Style
)

func init() {
	StyleTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorHighlight)

	StyleSelected = lipgloss.NewStyle().
		Bold(true)

	StyleFaint = lipgloss.NewStyle().
		Foreground(ColorFaint)

	StyleBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSubtle)

	StyleStatusBar = lipgloss.NewStyle().
		Background(ColorSubtle).
		Padding(0, 1)

	StyleToast = lipgloss.NewStyle().
		Foreground(ColorGreen).
		Italic(true)

	StyleError = lipgloss.NewStyle().
		Foreground(ColorRed).
		Bold(true)
}
