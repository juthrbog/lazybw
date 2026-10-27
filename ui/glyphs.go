package ui

import (
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
)

// GlyphSet holds each glyph that lazybw draws itself. The Unicode set uses
// Nerd Font icons and Unicode symbols. Many of them have the East Asian
// Width "Ambiguous", so a terminal with a CJK locale can draw them 2 cells
// wide and break the alignment. The ASCII set has only printable ASCII,
// which every terminal draws 1 cell wide.
type GlyphSet struct {
	// Icons, without styles. initStyles colors the item icons.
	Logo, Login, Card, Note, Identity, SSHKey, Copy string
	Save, Success, Error                            string

	Cursor    string // in front of the selected row, with its space
	Expanded  string // an expanded group
	Collapsed string // a collapsed group
	Mask      string // one cell of a hidden value
	Ellipsis  string // the end of a cut text, and of a busy text
	Sep       string // between two footer hints, with its spaces
	Dot       string // between two parts of one text, with its spaces
	On, Off   string // a generator option that is on or off
	Up, Down  string // the arrow keys and scroll cues
	Dash      string // a dash in a sentence
	Rule      string // one cell of the line under the header

	// The TOTP bar: a full cell, an empty cell, and the partial cells from
	// 7/8 full down to 1/8 full.
	BarFull, BarEmpty string
	BarSteps          [7]string

	SpinnerLoad, SpinnerLock, SpinnerUnlock spinner.Spinner

	CardBorder lipgloss.Border // the border of an overlay card
	FormBorder lipgloss.Border // the left border of a focused form field
}

// UpDown returns the label of the up and down arrow keys, for example "↑/↓".
func (g GlyphSet) UpDown() string { return g.Up + "/" + g.Down }

var unicodeGlyphs = GlyphSet{
	Logo:     "\U000F099D", // nf-md-shield_lock
	Login:    "\U000F033E", // nf-md-lock
	Card:     "\U000F0FEF", // nf-md-credit_card
	Note:     "\U000F039E", // nf-md-note_text
	Identity: "\U000F0004", // nf-md-account
	SSHKey:   "\U000F08C0", // nf-md-ssh
	Copy:     "\U000F018F", // nf-md-content_copy
	Save:     "\U000F0193", // nf-md-content_save
	Success:  "✓",
	Error:    "✗",

	Cursor:    "▶ ",
	Expanded:  "▾",
	Collapsed: "▸",
	Mask:      "•",
	Ellipsis:  "…",
	Sep:       " · ",
	Dot:       " · ",
	On:        "✓",
	Off:       "·",
	Up:        "↑",
	Down:      "↓",
	Dash:      "—",
	Rule:      "▄",

	BarFull:  "█",
	BarEmpty: "░",
	BarSteps: [7]string{"▉", "▊", "▋", "▌", "▍", "▎", "▏"},

	SpinnerLoad: spinner.Dot,
	SpinnerLock: spinner.Spinner{
		Frames: []string{"󰌿", "󰌆", "󰷖", "󰌾"},
		FPS:    time.Second / 3,
	},
	SpinnerUnlock: spinner.Spinner{
		Frames: []string{"󰌾", "󰷖", "󰌆", "󰌿"},
		FPS:    time.Second / 3,
	},

	CardBorder: lipgloss.RoundedBorder(),
	FormBorder: lipgloss.ThickBorder(),
}

var asciiGlyphs = GlyphSet{
	Logo:     "#",
	Login:    "L",
	Card:     "C",
	Note:     "N",
	Identity: "I",
	SSHKey:   "K",
	Copy:     "*",
	Save:     "S",
	Success:  "+",
	Error:    "!",

	Cursor:    "> ",
	Expanded:  "-",
	Collapsed: "+",
	Mask:      "*",
	Ellipsis:  "...",
	Sep:       " | ",
	Dot:       " - ",
	On:        "x",
	Off:       "-",
	Up:        "^",
	Down:      "v",
	Dash:      "-",
	Rule:      "-",

	BarFull:  "#",
	BarEmpty: ".",
	BarSteps: [7]string{"=", "=", "=", "=", "-", "-", "-"},

	SpinnerLoad:   spinner.Line,
	SpinnerLock:   spinner.Line,
	SpinnerUnlock: spinner.Line,

	CardBorder: lipgloss.ASCIIBorder(),
	FormBorder: lipgloss.ASCIIBorder(),
}

// Glyphs is the active glyph set. UseASCII changes it.
var Glyphs = unicodeGlyphs

// UseASCII makes the ASCII set active when on is true, and the Unicode set
// otherwise. It restyles the colored glyphs. Call it before the program
// starts, because the screens copy the spinners when they are built.
func UseASCII(on bool) {
	Glyphs = unicodeGlyphs
	if on {
		Glyphs = asciiGlyphs
	}
	SpinnerLoad = Glyphs.SpinnerLoad
	SpinnerLock = Glyphs.SpinnerLock
	SpinnerUnlock = Glyphs.SpinnerUnlock
	initStyles()
}

// ASCII reports whether the ASCII set is active.
func ASCII() bool { return Glyphs.Logo == asciiGlyphs.Logo }
