package ui

import (
	"image/color"
	"slices"

	"charm.land/bubbles/v2/list"
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

// DefaultTheme is the theme that lazybw uses when no theme is set, or when
// the name of the theme is not known.
const DefaultTheme = "catppuccin-mocha"

// CurrentTheme holds the name of the active theme.
var CurrentTheme = DefaultTheme

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

// Spinner definitions for loading states. UseASCII changes them.
var (
	SpinnerUnlock = Glyphs.SpinnerUnlock
	SpinnerLock   = Glyphs.SpinnerLock
	SpinnerLoad   = Glyphs.SpinnerLoad
)

// Pre-built styles — set by initStyles, called from ApplyTheme.
var (
	StyleTitle       lipgloss.Style
	StyleSelected    lipgloss.Style
	StyleFaint       lipgloss.Style
	StyleToast       lipgloss.Style
	StyleError       lipgloss.Style
	StyleHintKey     lipgloss.Style
	StyleHintDesc    lipgloss.Style
	StyleHintSep     lipgloss.Style
	StyleHelpGroup   lipgloss.Style
	StyleHelpKey     lipgloss.Style
	StyleHeaderBadge lipgloss.Style
)

// HuhTheme is the theme of the huh forms: the unlock, login and two-step
// forms, the master password prompt and the theme picker. It uses the light
// or the dark styles for IsDark at the time that a form draws, so a form
// that was built before the background was known also matches it.
var HuhTheme huh.Theme

// The glyphs in the colors of the theme. initStyles renders them again
// after a theme change.
var (
	GlyphLogin    string
	GlyphCard     string
	GlyphNote     string
	GlyphIdentity string
	GlyphSSHKey   string
	GlyphSuccess  string
	GlyphError    string
	GlyphCopy     string
	GlyphSave     string
)

func init() {
	ApplyTheme(DefaultTheme)
}

// IsTheme reports whether name is one of ThemeNames.
func IsTheme(name string) bool {
	return slices.Contains(ThemeNames, name)
}

// ApplyTheme sets the color palette, styles, and glyphs for the given theme.
// For a name that is not in ThemeNames, it applies DefaultTheme. It sets
// CurrentTheme to the theme that it applies.
func ApplyTheme(name string) {
	if !IsTheme(name) {
		name = DefaultTheme
	}
	CurrentTheme = name

	switch name {
	case "catppuccin-frappe":
		applyCatppuccin(catppuccin.Frappe, catppuccin.Latte)
		HuhTheme = followBackground(huh.ThemeCatppuccin)
	case "catppuccin-macchiato":
		applyCatppuccin(catppuccin.Macchiato, catppuccin.Latte)
		HuhTheme = followBackground(huh.ThemeCatppuccin)
	case "catppuccin-latte":
		applyCatppuccin(catppuccin.Latte, catppuccin.Latte)
		HuhTheme = followBackground(huh.ThemeCatppuccin)
	case "dracula":
		applyDracula()
		HuhTheme = followBackground(huh.ThemeDracula)
	case "charm":
		applyCharm()
		HuhTheme = followBackground(huhThemeCharm)
	case "base16":
		applyBase16()
		HuhTheme = followBackground(huh.ThemeBase16)
	default:
		applyCatppuccin(catppuccin.Mocha, catppuccin.Latte)
		HuhTheme = followBackground(huh.ThemeCatppuccin)
	}

	initStyles()
}

// followBackground returns a huh theme that builds its styles for IsDark.
// huh learns the background only from a tea.BackgroundColorMsg that reaches
// the form, and a form that is built later never gets that message. So the
// theme ignores the background that huh passes in.
func followBackground(styles func(isDark bool) *huh.Styles) huh.Theme {
	return huh.ThemeFunc(func(bool) *huh.Styles {
		s := styles(IsDark)
		// The focused field has a border on its left. Use the border of the
		// active glyph set.
		s.Focused.Base = s.Focused.Base.BorderStyle(Glyphs.FormBorder)
		return s
	})
}

// huhThemeCharm is huh.ThemeCharm with readable option text. huh.ThemeCharm
// uses its light option color (ANSI 252) on light backgrounds and its dark
// one (ANSI 235) on dark backgrounds, so the text almost disappears on both.
func huhThemeCharm(isDark bool) *huh.Styles {
	s := huh.ThemeCharm(isDark)
	fg := lipgloss.LightDark(isDark)(lipgloss.Color("235"), lipgloss.Color("252"))
	s.Focused.Option = s.Focused.Option.Foreground(fg)
	s.Focused.UnselectedOption = s.Focused.UnselectedOption.Foreground(fg)
	s.Blurred.Option = s.Blurred.Option.Foreground(fg)
	s.Blurred.UnselectedOption = s.Blurred.UnselectedOption.Foreground(fg)
	return s
}

// Colors that replace palette colors with too little contrast. Each one has
// a contrast of at least 4.5:1 on its background: the light colors on
// #ffffff, and faintDark on #1e1e2e.
var (
	faintLight  = lipgloss.Color("#767676")
	faintDark   = lipgloss.Color("#8a8a8a")
	yellowLatte = lipgloss.Color("#a56a00") // a darker Latte yellow
)

func applyCatppuccin(dark, light catppuccin.Flavor) {
	ld := lipgloss.LightDark(IsDark)
	ColorHighlight = ld(lipgloss.Color(light.Mauve().Hex), lipgloss.Color(dark.Mauve().Hex))
	ColorSubtle = ld(lipgloss.Color(light.Surface0().Hex), lipgloss.Color(dark.Surface0().Hex))
	ColorGreen = ld(lipgloss.Color(light.Green().Hex), lipgloss.Color(dark.Green().Hex))
	ColorYellow = ld(yellowLatte, lipgloss.Color(dark.Yellow().Hex))
	ColorRed = ld(lipgloss.Color(light.Red().Hex), lipgloss.Color(dark.Red().Hex))
	ColorFaint = ld(lipgloss.Color(light.Overlay2().Hex), lipgloss.Color(dark.Overlay1().Hex))
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
	ColorFaint = ld(faintLight, lipgloss.Color("#6272a4"))
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
	ColorFaint = ld(faintLight, faintDark)
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
	ColorFaint = ld(faintLight, faintDark)
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
		Foreground(textOn(ColorHighlight)).
		Background(ColorHighlight).
		Padding(0, 1)

	// Render the glyphs of the active set in the new colors. glyphs.go
	// gives the Nerd Font names of the code points, and docs/GLYPHS.md
	// lists them.
	g := Glyphs
	GlyphLogin = lipgloss.NewStyle().Foreground(ColorHighlight).Render(g.Login)
	GlyphCard = lipgloss.NewStyle().Foreground(ColorGreen).Render(g.Card)
	GlyphNote = lipgloss.NewStyle().Foreground(ColorYellow).Render(g.Note)
	GlyphIdentity = lipgloss.NewStyle().Foreground(ColorHighlight).Render(g.Identity)
	GlyphSSHKey = lipgloss.NewStyle().Foreground(ColorGreen).Render(g.SSHKey)
	GlyphSuccess = lipgloss.NewStyle().Foreground(ColorGreen).Render(g.Success)
	GlyphError = lipgloss.NewStyle().Foreground(ColorRed).Render(g.Error)
	GlyphCopy = lipgloss.NewStyle().Foreground(ColorGreen).Render(g.Copy)
	GlyphSave = lipgloss.NewStyle().Foreground(ColorGreen).Render(g.Save)
}

// ListStyles returns the styles of the vault list and its filter input for
// the active theme and the background. The list of bubbles uses its dark
// styles by default.
func ListStyles() list.Styles {
	s := list.DefaultStyles(IsDark)
	prompt := lipgloss.NewStyle().Foreground(ColorHighlight)
	s.Filter.Focused.Prompt = prompt
	s.Filter.Blurred.Prompt = prompt
	s.Filter.Cursor.Color = ColorHighlight
	s.NoItems = s.NoItems.Foreground(ColorFaint)
	s.StatusEmpty = s.StatusEmpty.Foreground(ColorFaint)
	return s
}
