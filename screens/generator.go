package screens

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/session"
	"github.com/juthrbog/lazybw/ui"
)

// Generator modes.
const (
	genModePassword   = "password"
	genModePassphrase = "passphrase"
)

// genSeparators are the word separators that key 3 cycles through in
// passphrase mode.
var genSeparators = []string{"-", "_", ".", " "}

// genPlaceholder is the generator output while bw generate runs.
const genPlaceholder = "Generating…"

// genMaxWidth is the widest that a line of the generated value gets.
const genMaxWidth = 72

// genMinWords is the smallest word count for a passphrase. bw generate
// raises a smaller count to 3, so the card would show the wrong count.
const genMinWords = 3

// GeneratorModel is the password generator card of the vault screen. It
// runs `bw generate` for each change of its options.
type GeneratorModel struct {
	mode     string // genModePassword or genModePassphrase
	password string // the latest generated value
	err      string // the error of the latest generate request
	// seq is the sequence number of the latest generate request. It goes on
	// over each Open, so that a late result of an earlier card is dropped.
	seq    uint64
	closed bool

	length    int
	uppercase bool
	lowercase bool
	numbers   bool
	special   bool

	words      int
	separator  string
	capitalize bool
	includeNum bool
}

// Open resets the options to the defaults and starts `bw generate`.
func (g GeneratorModel) Open() (GeneratorModel, tea.Cmd) {
	g = GeneratorModel{
		mode:       genModePassword,
		seq:        g.seq,
		length:     20,
		uppercase:  true,
		lowercase:  true,
		numbers:    true,
		special:    true,
		words:      4,
		separator:  "-",
		capitalize: true,
		includeNum: true,
	}
	return g, g.generate()
}

// Closed reports whether the user closed the card.
func (g GeneratorModel) Closed() bool { return g.closed }

// Clear drops the generated value.
func (g *GeneratorModel) Clear() {
	g.password = ""
}

// updateGenerator gives a key to the generator card of the vault, and
// closes the card when the generator closed.
func (m VaultModel) updateGenerator(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.gen, cmd = m.gen.Update(msg)
	if m.gen.Closed() {
		m.overlay = overlayNone
	}
	return m, cmd
}

// Update handles a key press while the card shows, and the result of a
// `bw generate` request.
func (g GeneratorModel) Update(msg tea.Msg) (GeneratorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case bwcmd.GenerateResult:
		if msg.Seq != g.seq {
			return g, nil // a newer request is in flight
		}
		// The error goes in its own field, so that c cannot copy it.
		g.password = msg.Password
		g.err = userError(msg.Err, actionOther)
		return g, nil
	case tea.KeyPressMsg:
		return g.updateKey(msg)
	}
	return g, nil
}

func (g GeneratorModel) updateKey(msg tea.KeyPressMsg) (GeneratorModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		g.closed = true
		return g, nil
	case "enter", "r":
		return g, g.generate()
	case "c":
		if g.err == "" && g.password != "" && g.password != genPlaceholder {
			return g, session.CopyToClipboard(g.password, session.CopyFieldPassword)
		}
	case "m", "tab":
		if g.mode == genModePassword {
			g.mode = genModePassphrase
		} else {
			g.mode = genModePassword
		}
		return g, g.generate()
	case "+", "=":
		if g.mode == genModePassword {
			if g.length < 128 {
				g.length++
			}
		} else {
			if g.words < 20 {
				g.words++
			}
		}
		return g, g.generate()
	case "-":
		if g.mode == genModePassword {
			if g.length > 5 {
				g.length--
			}
		} else {
			if g.words > genMinWords {
				g.words--
			}
		}
		return g, g.generate()
	case "1", "2", "3", "4":
		if g.mode == genModePassphrase {
			return g.togglePassphraseOption(msg.String())
		}
		switch msg.String() {
		case "1":
			g.uppercase = !g.uppercase
		case "2":
			g.lowercase = !g.lowercase
		case "3":
			g.numbers = !g.numbers
		case "4":
			g.special = !g.special
		}
		return g, g.generate()
	}
	return g, nil
}

// togglePassphraseOption changes the passphrase option of key k: 1 toggles
// the capitals, 2 toggles the number, and 3 moves to the next separator.
// Key 4 has no passphrase option.
func (g GeneratorModel) togglePassphraseOption(k string) (GeneratorModel, tea.Cmd) {
	switch k {
	case "1":
		g.capitalize = !g.capitalize
	case "2":
		g.includeNum = !g.includeNum
	case "3":
		next := 0
		for i, sep := range genSeparators {
			if sep == g.separator {
				next = (i + 1) % len(genSeparators)
			}
		}
		g.separator = genSeparators[next]
	default:
		return g, nil
	}
	return g, g.generate()
}

// separatorName returns the separator as the generator card shows it.
func separatorName(sep string) string {
	if sep == " " {
		return "space"
	}
	return sep
}

// generate starts a new `bw generate` request. Each request gets a new
// sequence number, so that a slow, older result cannot replace a newer one.
func (g *GeneratorModel) generate() tea.Cmd {
	g.seq++
	g.password = genPlaceholder
	g.err = ""
	return bwcmd.Generate(g.seq, g.args()...)
}

// args returns the options of `bw generate` for the current settings.
func (g GeneratorModel) args() []string {
	if g.mode == genModePassphrase {
		// bw reads the word "space" as a space separator.
		args := []string{"--passphrase", "--words", fmt.Sprintf("%d", g.words), "--separator", separatorName(g.separator)}
		if g.capitalize {
			args = append(args, "--capitalize")
		}
		if g.includeNum {
			args = append(args, "--includeNumber")
		}
		return args
	}
	args := []string{"--length", fmt.Sprintf("%d", g.length)}
	if g.uppercase {
		args = append(args, "--uppercase")
	}
	if g.lowercase {
		args = append(args, "--lowercase")
	}
	if g.numbers {
		args = append(args, "--number")
	}
	if g.special {
		args = append(args, "--special")
	}
	return args
}

// View renders the generator card for a content area of the given width.
// The generated value wraps to fit inside the card.
func (g GeneratorModel) View(width int) string {
	var b strings.Builder

	b.WriteString(ui.StyleTitle.Render("Password Generator"))
	b.WriteString("\n\n")

	// Mode indicator.
	if g.mode == genModePassword {
		b.WriteString("Mode        ")
		b.WriteString(ui.StyleTitle.Render("[Password]"))
		b.WriteString("  ")
		b.WriteString(ui.StyleFaint.Render("Passphrase"))
		b.WriteString("\n")
	} else {
		b.WriteString("Mode        ")
		b.WriteString(ui.StyleFaint.Render("Password"))
		b.WriteString("  ")
		b.WriteString(ui.StyleTitle.Render("[Passphrase]"))
		b.WriteString("\n")
	}

	// Options.
	if g.mode == genModePassword {
		fmt.Fprintf(&b, "Length      %d\n", g.length)
		fmt.Fprintf(&b, "Uppercase   %s   Lowercase  %s   Numbers  %s   Special  %s\n",
			checkMark(g.uppercase), checkMark(g.lowercase),
			checkMark(g.numbers), checkMark(g.special))
	} else {
		// The options show in the order of their keys 1 to 3, on rows
		// that stay narrow.
		fmt.Fprintf(&b, "Words       %d\n", g.words)
		fmt.Fprintf(&b, "Capitalize  %s   Include #  %s\n",
			checkMark(g.capitalize), checkMark(g.includeNum))
		fmt.Fprintf(&b, "Separator   %s\n", separatorName(g.separator))
	}

	b.WriteString("\n")

	// Generated output. A long password or passphrase wraps, so that all
	// of it shows inside the card border.
	contentW := max(1, min(width-ui.OverlayEdgeWidth, genMaxWidth))
	if g.err != "" {
		// errorBlock keeps a 2-cell margin on each side.
		b.WriteString(errorBlock(g.err, contentW+4, contentW))
	} else {
		b.WriteString(ui.StyleTitle.Render(ansi.Hardwrap(ui.SanitizeLine(g.password), contentW, true)))
	}

	return b.String()
}

func checkMark(v bool) string {
	if v {
		return ui.StyleToast.Render("✓")
	}
	return ui.StyleFaint.Render("·")
}

// FooterHints returns the footer hints for the mode of the card.
func (g GeneratorModel) FooterHints() []ui.HintBinding {
	if g.mode == genModePassphrase {
		return []ui.HintBinding{
			{Key: "enter", Desc: "regen"},
			{Key: "+/-", Desc: "words"},
			{Key: "m", Desc: "mode"},
			{Key: "1-3", Desc: "options"},
			{Key: "c", Desc: "copy"},
			{Key: "esc", Desc: "close"},
		}
	}
	return []ui.HintBinding{
		{Key: "enter", Desc: "regen"},
		{Key: "+/-", Desc: "length"},
		{Key: "m", Desc: "mode"},
		{Key: "1-4", Desc: "toggles"},
		{Key: "c", Desc: "copy"},
		{Key: "esc", Desc: "close"},
	}
}
