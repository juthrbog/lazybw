package screens

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/ui"
)

// openedGenerator returns a vault screen of the given size with the
// generator open. The bw generate command is not run.
func openedGenerator(width, height int) VaultModel {
	m := vaultAt(testItems(), width, height)
	updated, _ := m.openGenerator()
	return updated.(VaultModel)
}

// genKey sends the key k to the generator of m. It does not run the
// command, which would start bw generate.
func genKey(m VaultModel, k string) VaultModel {
	updated, _ := m.Update(keyMsg(k))
	return updated.(VaultModel)
}

// cardText returns the text inside the overlay card of out, one row after
// the other, without the border and the padding.
func cardText(out string) string {
	var b strings.Builder
	inCard := false
	for _, line := range strings.Split(ansi.Strip(out), "\n") {
		if strings.Contains(line, "╭") {
			inCard = true
			continue
		}
		if !inCard {
			continue
		}
		if strings.Contains(line, "╰") {
			break
		}
		l, r := strings.Index(line, "│"), strings.LastIndex(line, "│")
		if l >= 0 && r > l {
			b.WriteString(strings.TrimSpace(line[l+len("│") : r]))
		}
	}
	return b.String()
}

func TestGeneratorShowsAllOfLongPassword(t *testing.T) {
	pw := strings.Repeat("0123456789", 10)
	for _, size := range [][2]int{{80, 24}, {120, 30}} {
		width, height := size[0], size[1]
		m := openedGenerator(width, height)
		updated, _ := m.Update(bwcmd.GenerateResult{Password: pw, Seq: m.genSeq})
		m = updated.(VaultModel)
		out := m.ViewContent(width, height-ui.ChromeHeight)
		assertArea(t, out, width, height-ui.ChromeHeight)
		left, right := cardColumns(t, out)
		if !strings.Contains(cardText(out), pw) {
			t.Errorf("%dx%d: the card must show all 100 characters:\n%s", width, height, ansi.Strip(out))
		}
		// The value wraps at genMaxWidth on a wide screen.
		if w := right - left + 1; w > genMaxWidth+ui.OverlayEdgeWidth {
			t.Errorf("%dx%d: card is %d cells wide, want at most %d", width, height, w, genMaxWidth+ui.OverlayEdgeWidth)
		}
	}
}

func TestGeneratorPassphraseKeysChangeOptions(t *testing.T) {
	m := openedGenerator(80, 24)
	m = genKey(m, "m")
	if m.genMode != "passphrase" {
		t.Fatalf("mode = %q, want passphrase", m.genMode)
	}
	args := func() []string { return m.genArgs() }
	separator := func() string {
		a := args()
		i := slices.Index(a, "--separator")
		if i < 0 || i+1 >= len(a) {
			t.Fatalf("args %q have no separator", a)
		}
		return a[i+1]
	}
	if !slices.Contains(args(), "--capitalize") || !slices.Contains(args(), "--includeNumber") || separator() != "-" {
		t.Fatalf("default passphrase args = %q", args())
	}

	m = genKey(m, "1")
	if slices.Contains(args(), "--capitalize") {
		t.Errorf("after 1, args = %q, want no --capitalize", args())
	}
	m = genKey(m, "2")
	if slices.Contains(args(), "--includeNumber") {
		t.Errorf("after 2, args = %q, want no --includeNumber", args())
	}
	// 3 cycles through the separators and back. bw reads "space" as a
	// space.
	for _, want := range []string{"_", ".", "space", "-"} {
		m = genKey(m, "3")
		if got := separator(); got != want {
			t.Errorf("after 3, separator = %q, want %q", got, want)
		}
	}
	m = genKey(m, "1")
	if !slices.Contains(args(), "--capitalize") {
		t.Errorf("a second 1 must turn --capitalize on again, args = %q", args())
	}
	// The password options did not change.
	if !m.genUppercase || !m.genLowercase || !m.genNumbers || !m.genSpecial {
		t.Error("passphrase keys must not change the password options")
	}
	// 4 has no passphrase option.
	before := m.genSeq
	if m = genKey(m, "4"); m.genSeq != before {
		t.Error("4 in passphrase mode must not start bw generate")
	}
}

func TestGeneratorPassphraseWordsStopAtMinimum(t *testing.T) {
	m := genKey(openedGenerator(80, 24), "m")
	for range 5 {
		m = genKey(m, "-")
	}
	if m.genWords != genMinWords {
		t.Errorf("genWords = %d, want %d", m.genWords, genMinWords)
	}
	args := m.genArgs()
	if i := slices.Index(args, "--words"); i < 0 || i+1 >= len(args) || args[i+1] != "3" {
		t.Errorf("args = %q, want --words 3", args)
	}
}

func TestGeneratorPassphraseCardShowsSeparator(t *testing.T) {
	m := openedGenerator(80, 24)
	m = genKey(m, "m")
	for range 3 {
		m = genKey(m, "3") // to the space separator
	}
	if plain := ansi.Strip(m.renderGeneratorCard(80)); !strings.Contains(plain, "Separator   space") {
		t.Errorf("the card must name the space separator:\n%s", plain)
	}
}

func TestGeneratorPassphraseOptionsFitNarrowCards(t *testing.T) {
	for _, width := range []int{40, 45, 50} {
		m := openedGenerator(width, 30)
		m = genKey(m, "m")
		for range 3 {
			m = genKey(m, "3") // to the space separator, the widest
		}
		out := m.ViewContent(width, 30-ui.ChromeHeight)
		assertArea(t, out, width, 30-ui.ChromeHeight)
		cardColumns(t, out)
		plain := ansi.Strip(out)
		for _, want := range []string{"Capitalize  ✓   Include #  ✓", "Separator   space"} {
			if !strings.Contains(plain, want) {
				t.Errorf("%d columns: the card must show %q:\n%s", width, want, plain)
			}
		}
	}
}

func TestGeneratorFooterFollowsMode(t *testing.T) {
	m := openedGenerator(80, 24)
	hints, _ := m.FooterContent()
	if !hasHint(hints, "+/-", "length") || !hasHint(hints, "1-4", "toggles") {
		t.Errorf("password mode hints = %v, want +/- length and 1-4 toggles", hints)
	}
	m = genKey(m, "m")
	hints, _ = m.FooterContent()
	if !hasHint(hints, "+/-", "words") || !hasHint(hints, "1-3", "options") {
		t.Errorf("passphrase mode hints = %v, want +/- words and 1-3 options", hints)
	}
	if hasHintDesc(hints, "length") || hasHintDesc(hints, "toggles") || hasHintKey(hints, "1-4") {
		t.Errorf("passphrase mode hints = %v, want no password hints", hints)
	}
	// The copy and close hints fit at 80 columns.
	out := ansi.Strip(ui.RenderFooter(hints, "", 80))
	for _, want := range []string{"+/- words", "c copy", "esc close"} {
		if !strings.Contains(out, want) {
			t.Errorf("footer at 80 columns = %q, want %q", out, want)
		}
	}
}

func hasHint(hints []ui.HintBinding, key, desc string) bool {
	for _, h := range hints {
		if h.Key == key && h.Desc == desc {
			return true
		}
	}
	return false
}
