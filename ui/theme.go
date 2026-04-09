package ui

import (
	"image/color"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

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

// IsDark tracks whether the terminal has a dark background.
// Updated by the root model on tea.BackgroundColorMsg.
var IsDark = true

// Colour palette — set by ApplyTheme.
var (
	ColorHighlight color.Color
	ColorSubtle    color.Color
	ColorGreen     color.Color
	ColorYellow    color.Color
	ColorRed       color.Color
	ColorFaint     color.Color
	GradientFrom   color.Color
	GradientTo     color.Color
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
	StyleHintKey   lipgloss.Style
	StyleHintDesc  lipgloss.Style
	StyleHintSep   lipgloss.Style
	StyleHelpGroup   lipgloss.Style
	StyleHelpKey     lipgloss.Style
	StyleHeaderBadge lipgloss.Style
)

// HuhTheme is applied to huh forms (unlock/login screen).
var HuhTheme huh.Theme

// Glyph variables — re-rendered by initStyles after theme change.
var (
	GlyphLogin    string
	GlyphCard     string
	GlyphNote     string
	GlyphIdentity string
	GlyphSSHKey   string
	GlyphSuccess  string
	GlyphError    string
	GlyphCopy     string
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
		HuhTheme = huh.ThemeFunc(huh.ThemeCatppuccin)
	case "catppuccin-frappe":
		applyCatppuccin(catppuccin.Frappe, catppuccin.Latte)
		HuhTheme = huh.ThemeFunc(huh.ThemeCatppuccin)
	case "catppuccin-macchiato":
		applyCatppuccin(catppuccin.Macchiato, catppuccin.Latte)
		HuhTheme = huh.ThemeFunc(huh.ThemeCatppuccin)
	case "catppuccin-latte":
		applyCatppuccin(catppuccin.Latte, catppuccin.Latte)
		HuhTheme = huh.ThemeFunc(huh.ThemeCatppuccin)
	case "dracula":
		applyDracula()
		HuhTheme = huh.ThemeFunc(huh.ThemeDracula)
	case "charm":
		applyCharm()
		HuhTheme = huh.ThemeFunc(huh.ThemeCharm)
	case "base16":
		applyBase16()
		HuhTheme = huh.ThemeFunc(huh.ThemeBase16)
	default:
		applyCatppuccin(catppuccin.Mocha, catppuccin.Latte)
		HuhTheme = huh.ThemeFunc(huh.ThemeCatppuccin)
	}

	initStyles()
}

func applyCatppuccin(dark, light catppuccin.Flavor) {
	ld := lipgloss.LightDark(IsDark)
	ColorHighlight = ld(lipgloss.Color(light.Mauve().Hex), lipgloss.Color(dark.Mauve().Hex))
	ColorSubtle = ld(lipgloss.Color(light.Surface0().Hex), lipgloss.Color(dark.Surface0().Hex))
	ColorGreen = ld(lipgloss.Color(light.Green().Hex), lipgloss.Color(dark.Green().Hex))
	ColorYellow = ld(lipgloss.Color(light.Yellow().Hex), lipgloss.Color(dark.Yellow().Hex))
	ColorRed = ld(lipgloss.Color(light.Red().Hex), lipgloss.Color(dark.Red().Hex))
	ColorFaint = ld(lipgloss.Color(light.Overlay0().Hex), lipgloss.Color(dark.Overlay0().Hex))
	GradientFrom = ld(lipgloss.Color(light.Mauve().Hex), lipgloss.Color(dark.Mauve().Hex))
	GradientTo = ld(lipgloss.Color(light.Blue().Hex), lipgloss.Color(dark.Blue().Hex))
}

func applyDracula() {
	ld := lipgloss.LightDark(IsDark)
	ColorHighlight = ld(lipgloss.Color("#7c3aed"), lipgloss.Color("#bd93f9"))
	ColorSubtle = ld(lipgloss.Color("#D9DCCF"), lipgloss.Color("#44475a"))
	ColorGreen = ld(lipgloss.Color("#027A4F"), lipgloss.Color("#50fa7b"))
	ColorYellow = ld(lipgloss.Color("#C47D10"), lipgloss.Color("#f1fa8c"))
	ColorRed = ld(lipgloss.Color("#CC2222"), lipgloss.Color("#ff5555"))
	ColorFaint = ld(lipgloss.Color("#9A9A9A"), lipgloss.Color("#6272a4"))
	GradientFrom = ld(lipgloss.Color("#7c3aed"), lipgloss.Color("#bd93f9"))
	GradientTo = ld(lipgloss.Color("#db2777"), lipgloss.Color("#ff79c6"))
}

func applyCharm() {
	ld := lipgloss.LightDark(IsDark)
	ColorHighlight = ld(lipgloss.Color("#5A3ECC"), lipgloss.Color("#7D56F4"))
	ColorSubtle = ld(lipgloss.Color("#D9DCCF"), lipgloss.Color("#383838"))
	ColorGreen = ld(lipgloss.Color("#027A4F"), lipgloss.Color("#04B575"))
	ColorYellow = ld(lipgloss.Color("#C47D10"), lipgloss.Color("#F5A623"))
	ColorRed = ld(lipgloss.Color("#CC2222"), lipgloss.Color("#FF4F4F"))
	ColorFaint = ld(lipgloss.Color("#9A9A9A"), lipgloss.Color("#626262"))
	GradientFrom = ld(lipgloss.Color("#db2777"), lipgloss.Color("#ff70a0"))
	GradientTo = ld(lipgloss.Color("#4338ca"), lipgloss.Color("#7571f9"))
}

func applyBase16() {
	ld := lipgloss.LightDark(IsDark)
	ColorHighlight = ld(lipgloss.Color("#a16946"), lipgloss.Color("#a16946"))
	ColorSubtle = ld(lipgloss.Color("#D9DCCF"), lipgloss.Color("#383838"))
	ColorGreen = ld(lipgloss.Color("#027A4F"), lipgloss.Color("#a1b56c"))
	ColorYellow = ld(lipgloss.Color("#C47D10"), lipgloss.Color("#f7ca88"))
	ColorRed = ld(lipgloss.Color("#CC2222"), lipgloss.Color("#ab4642"))
	ColorFaint = ld(lipgloss.Color("#9A9A9A"), lipgloss.Color("#585858"))
	GradientFrom = ld(lipgloss.Color("#0097a7"), lipgloss.Color("#00bcd4"))
	GradientTo = ld(lipgloss.Color("#1565c0"), lipgloss.Color("#2196f3"))
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

	StyleHintKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorHighlight)

	StyleHintDesc = lipgloss.NewStyle().
		Foreground(ColorFaint)

	StyleHintSep = lipgloss.NewStyle().
		Foreground(ColorFaint)

	StyleHelpGroup = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorHighlight)

	StyleHelpKey = lipgloss.NewStyle().
		Width(14)

	StyleHeaderBadge = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSubtle).
		Background(ColorHighlight).
		Padding(0, 1)

	// Re-render glyphs with new colors.
	GlyphLogin = lipgloss.NewStyle().Foreground(ColorHighlight).Render("󰌾")
	GlyphCard = lipgloss.NewStyle().Foreground(ColorGreen).Render("󰁯")
	GlyphNote = lipgloss.NewStyle().Foreground(ColorYellow).Render("󱙒")
	GlyphIdentity = lipgloss.NewStyle().Foreground(ColorHighlight).Render("󰀄")
	GlyphSSHKey = lipgloss.NewStyle().Foreground(ColorGreen).Render("󰣀")
	GlyphSuccess = lipgloss.NewStyle().Foreground(ColorGreen).Render("✓")
	GlyphError = lipgloss.NewStyle().Foreground(ColorRed).Render("✗")
	GlyphCopy = lipgloss.NewStyle().Foreground(ColorGreen).Render("󰆏")
}
