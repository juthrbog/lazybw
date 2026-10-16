package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
)

// assertArea fails when out is not exactly width x height cells.
func assertArea(t *testing.T, out string, width, height int) {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != height {
		t.Errorf("got %d lines, want %d", len(lines), height)
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != width {
			t.Errorf("line %d is %d cells wide, want %d: %q", i, w, width, ansi.Strip(l))
		}
	}
}

func TestRenderOverlayClipsLargeCard(t *testing.T) {
	bg := strings.Repeat(strings.Repeat("b", 50)+"\n", 20)
	fg := strings.Repeat(strings.Repeat("f", 100)+"\n", 40)
	out := RenderOverlay(bg, fg, 30, 10)
	assertArea(t, out, 30, 10)

	// The content is cut, not the card, so all four corners show.
	plain := ansi.Strip(out)
	for _, corner := range []string{"╭", "╮", "╰", "╯"} {
		if !strings.Contains(plain, corner) {
			t.Errorf("the card lost its %s corner:\n%s", corner, plain)
		}
	}
}

func TestRenderOverlayTinyArea(t *testing.T) {
	// The area is smaller than the card border and padding.
	out := RenderOverlay("bg", "content", 4, 3)
	assertArea(t, out, 4, 3)
	if out := RenderOverlay("bg", "fg", 0, 0); out != "" {
		t.Errorf("an empty area must render nothing, got %q", out)
	}
}

func TestRenderOverlayDimsWholeBackground(t *testing.T) {
	item := bwcmd.Item{Type: bwcmd.ItemTypeLogin, Name: "Gmail", Login: &bwcmd.Login{Username: "user@gmail.com"}}
	rows := []string{
		RenderItemRow(item, true, 40, false),
		RenderItemRow(item, false, 40, false),
		StyleTitle.Render("title") + " plain " + StyleFaint.Render("faint") + StyleSelected.Render("bold"),
		GlyphNote + "  " + StyleError.Render("error") + " tail",
	}
	bg := strings.Join(append(rows, rows...), "\n")
	const width, height = 40, 8
	fg := "card"
	out := RenderOverlay(bg, fg, width, height)
	assertArea(t, out, width, height)

	// The card is centered: 4 cells of content, 6 of border and padding.
	cardW, cardH := lipgloss.Width(fg)+OverlayEdgeWidth, lipgloss.Height(fg)+OverlayEdgeHeight
	x0, y0 := (width-cardW)/2, (height-cardH)/2
	for y, line := range strings.Split(out, "\n") {
		cells := cellsOf(line)
		if len(cells) != width {
			t.Fatalf("line %d has %d cells, want %d", y, len(cells), width)
		}
		for x, c := range cells {
			inCard := x >= x0 && x < x0+cardW && y >= y0 && y < y0+cardH
			if inCard {
				continue
			}
			if !c.faint {
				t.Errorf("background cell (%d,%d) %q is not faint", x, y, c.text)
			}
			if c.bold {
				t.Errorf("background cell (%d,%d) %q is bold", x, y, c.text)
			}
		}
	}
}

// cardRows returns the rows of the overlay card in out, as cells.
func cardRows(t *testing.T, out string) [][]cell {
	t.Helper()
	var rows [][]cell
	inCard := false
	for _, line := range strings.Split(out, "\n") {
		plain := ansi.Strip(line)
		if strings.Contains(plain, "╭") {
			inCard = true
		}
		if inCard {
			rows = append(rows, cellsOf(line))
		}
		if strings.Contains(plain, "╰") {
			break
		}
	}
	if len(rows) < 3 {
		t.Fatalf("no card found in:\n%s", ansi.Strip(out))
	}
	return rows
}

// assertCardBorders fails when a row of the card does not start and end
// with the border at the same columns as the top row.
func assertCardBorders(t *testing.T, out string) {
	t.Helper()
	rows := cardRows(t, out)
	left, right := -1, -1
	for i, c := range rows[0] {
		switch c.text {
		case "╭":
			left = i
		case "╮":
			right = i
		}
	}
	if left < 0 || right < 0 {
		t.Fatalf("top border has no corners: %q", cellsText(rows[0]))
	}
	for i, row := range rows[1 : len(rows)-1] {
		if len(row) <= right || row[left].text != "│" || row[right].text != "│" {
			t.Errorf("card row %d lost a border: %q", i+1, cellsText(row))
		}
	}
	bottom := rows[len(rows)-1]
	if len(bottom) <= right || bottom[left].text != "╰" || bottom[right].text != "╯" {
		t.Errorf("bottom border is not under the top border: %q", cellsText(bottom))
	}
}

func TestRenderOverlayDrawerKeepsBorders(t *testing.T) {
	items := []*bwcmd.Item{
		{Type: bwcmd.ItemTypeLogin, Name: "Spoof\nFAKE-ROW", Login: &bwcmd.Login{Username: "a\tb"}, Notes: "first line\r\nsecond"},
		{Type: bwcmd.ItemTypeLogin, Name: "\x1b]8;;https://evil.example\x07link\x1b]8;;\x07", Login: &bwcmd.Login{Password: "p"}, Notes: "\x1b[8mhidden\r"},
		{Type: bwcmd.ItemTypeSecureNote, Name: "Pasted", Notes: strings.Repeat("a long pasted paragraph ", 8) + "\rend\tof note"},
		{Type: bwcmd.ItemTypeLogin, Name: strings.Repeat("東京", 30), Login: &bwcmd.Login{Username: strings.Repeat("é", 90)}},
	}
	for _, item := range items {
		for _, width := range []int{80, 50, 30} {
			drawerW := width - OverlayEdgeWidth
			drawer := RenderDrawer(DrawerProps{Item: item, Width: drawerW, AutoHeight: true})
			out := RenderOverlay("background", drawer, width, 20)
			assertArea(t, out, width, 20)
			assertCardBorders(t, out)
		}
	}
}
