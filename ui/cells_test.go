package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// cell is one column of a rendered line, with the SGR attributes that a
// terminal would draw it with. The text of the second column of a wide
// character is empty.
type cell struct {
	text  string
	bold  bool
	faint bool
}

// cellsOf splits one rendered line into cells. It follows the SGR bold and
// faint state and skips all other escape sequences.
func cellsOf(line string) []cell {
	var (
		out         []cell
		bold, faint bool
		state       byte
	)
	p := ansi.NewParser()
	for len(line) > 0 {
		seq, width, n, newState := ansi.DecodeSequence(line, state, p)
		state = newState
		line = line[n:]
		if width > 0 {
			out = append(out, cell{text: seq, bold: bold, faint: faint})
			// A wide character covers more columns. Add empty cells for
			// them, so that the index of a cell is its column.
			for range width - 1 {
				out = append(out, cell{bold: bold, faint: faint})
			}
			continue
		}
		cmd := ansi.Cmd(p.Command())
		if !ansi.HasCsiPrefix(seq) || cmd.Final() != 'm' || cmd.Prefix() != 0 || cmd.Intermediate() != 0 {
			continue
		}
		params := p.Params()
		if len(params) == 0 {
			bold, faint = false, false
			continue
		}
		for i := 0; i < len(params); i++ {
			switch params[i].Param(0) {
			case 0:
				bold, faint = false, false
			case 1:
				bold = true
			case 2:
				faint = true
			case 22:
				bold, faint = false, false
			case 38, 48, 58:
				// Skip the color arguments: 5;n or 2;r;g;b.
				if next, _, ok := params.Param(i+1, 0); ok {
					switch next {
					case 5:
						i += 2
					case 2:
						i += 4
					}
				}
			}
		}
	}
	return out
}

// cellsText joins the text of cells.
func cellsText(cells []cell) string {
	var b strings.Builder
	for _, c := range cells {
		b.WriteString(c.text)
	}
	return b.String()
}
