package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	catppuccin "github.com/catppuccin/go"
)

// ThemeNames lists all available themes.
var ThemeNames = []string{
	"catppuccin-mocha",
	"catppuccin-frappe",
	"catppuccin-macchiato",
	"catppuccin-latte",
	"dracula",
	"charm",
	"base16",
}

// CurrentTheme holds the name of the active theme.
var CurrentTheme = "catppuccin-mocha"

// Colour palette — set by ApplyTheme.
var (
	ColorHighlight lipgloss.AdaptiveColor
	ColorSubtle    lipgloss.AdaptiveColor
	ColorGreen     lipgloss.AdaptiveColor
	ColorYellow    lipgloss.AdaptiveColor
	ColorRed       lipgloss.AdaptiveColor
	ColorFaint     lipgloss.AdaptiveColor
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

// Pre-built styles — set by initStyles, called from ApplyTheme.
var (
	StyleTitle     lipgloss.Style
	StyleSelected  lipgloss.Style
	StyleFaint     lipgloss.Style
	StyleBorder    lipgloss.Style
	StyleStatusBar lipgloss.Style
	StyleToast     lipgloss.Style
	StyleError     lipgloss.Style
)

// HuhTheme is applied to huh forms (unlock/login screen).
var HuhTheme *huh.Theme

// Glyph variables — re-rendered by initStyles after theme change.
var (
	GlyphLogin    string
	GlyphCard     string
	GlyphNote     string
	GlyphIdentity string
	GlyphSSHKey   string
)

func init() {
	ApplyTheme("catppuccin-mocha")
}

// ApplyTheme sets the color palette, styles, and glyphs for the given theme.
func ApplyTheme(name string) {
	CurrentTheme = name

	switch name {
	case "catppuccin-mocha":
		applyCatppuccin(catppuccin.Mocha, catppuccin.Latte)
		HuhTheme = huh.ThemeCatppuccin()
	case "catppuccin-frappe":
		applyCatppuccin(catppuccin.Frappe, catppuccin.Latte)
		HuhTheme = huh.ThemeCatppuccin()
	case "catppuccin-macchiato":
		applyCatppuccin(catppuccin.Macchiato, catppuccin.Latte)
		HuhTheme = huh.ThemeCatppuccin()
	case "catppuccin-latte":
		applyCatppuccin(catppuccin.Latte, catppuccin.Latte)
		HuhTheme = huh.ThemeCatppuccin()
	case "dracula":
		applyDracula()
		HuhTheme = huh.ThemeDracula()
	case "charm":
		applyCharm()
		HuhTheme = huh.ThemeCharm()
	case "base16":
		applyBase16()
		HuhTheme = huh.ThemeBase16()
	default:
		applyCatppuccin(catppuccin.Mocha, catppuccin.Latte)
		HuhTheme = huh.ThemeCatppuccin()
	}

	initStyles()
}

func applyCatppuccin(dark, light catppuccin.Flavor) {
	ColorHighlight = lipgloss.AdaptiveColor{Dark: dark.Mauve().Hex, Light: light.Mauve().Hex}
	ColorSubtle = lipgloss.AdaptiveColor{Dark: dark.Surface0().Hex, Light: light.Surface0().Hex}
	ColorGreen = lipgloss.AdaptiveColor{Dark: dark.Green().Hex, Light: light.Green().Hex}
	ColorYellow = lipgloss.AdaptiveColor{Dark: dark.Yellow().Hex, Light: light.Yellow().Hex}
	ColorRed = lipgloss.AdaptiveColor{Dark: dark.Red().Hex, Light: light.Red().Hex}
	ColorFaint = lipgloss.AdaptiveColor{Dark: dark.Overlay0().Hex, Light: light.Overlay0().Hex}
}

func applyDracula() {
	ColorHighlight = lipgloss.AdaptiveColor{Dark: "#bd93f9", Light: "#7c3aed"}
	ColorSubtle = lipgloss.AdaptiveColor{Dark: "#44475a", Light: "#D9DCCF"}
	ColorGreen = lipgloss.AdaptiveColor{Dark: "#50fa7b", Light: "#027A4F"}
	ColorYellow = lipgloss.AdaptiveColor{Dark: "#f1fa8c", Light: "#C47D10"}
	ColorRed = lipgloss.AdaptiveColor{Dark: "#ff5555", Light: "#CC2222"}
	ColorFaint = lipgloss.AdaptiveColor{Dark: "#6272a4", Light: "#9A9A9A"}
}

func applyCharm() {
	ColorHighlight = lipgloss.AdaptiveColor{Dark: "#7D56F4", Light: "#5A3ECC"}
	ColorSubtle = lipgloss.AdaptiveColor{Dark: "#383838", Light: "#D9DCCF"}
	ColorGreen = lipgloss.AdaptiveColor{Dark: "#04B575", Light: "#027A4F"}
	ColorYellow = lipgloss.AdaptiveColor{Dark: "#F5A623", Light: "#C47D10"}
	ColorRed = lipgloss.AdaptiveColor{Dark: "#FF4F4F", Light: "#CC2222"}
	ColorFaint = lipgloss.AdaptiveColor{Dark: "#626262", Light: "#9A9A9A"}
}

func applyBase16() {
	ColorHighlight = lipgloss.AdaptiveColor{Dark: "#a16946", Light: "#a16946"}
	ColorSubtle = lipgloss.AdaptiveColor{Dark: "#383838", Light: "#D9DCCF"}
	ColorGreen = lipgloss.AdaptiveColor{Dark: "#a1b56c", Light: "#027A4F"}
	ColorYellow = lipgloss.AdaptiveColor{Dark: "#f7ca88", Light: "#C47D10"}
	ColorRed = lipgloss.AdaptiveColor{Dark: "#ab4642", Light: "#CC2222"}
	ColorFaint = lipgloss.AdaptiveColor{Dark: "#585858", Light: "#9A9A9A"}
}

func initStyles() {
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

	// Re-render glyphs with new colors.
	GlyphLogin = lipgloss.NewStyle().Foreground(ColorHighlight).Render("󰌾")
	GlyphCard = lipgloss.NewStyle().Foreground(ColorGreen).Render("󰁯")
	GlyphNote = lipgloss.NewStyle().Foreground(ColorYellow).Render("󱙒")
	GlyphIdentity = lipgloss.NewStyle().Foreground(ColorHighlight).Render("󰀄")
	GlyphSSHKey = lipgloss.NewStyle().Foreground(ColorGreen).Render("󰣀")
}
