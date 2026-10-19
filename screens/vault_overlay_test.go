package screens

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/ui"
)

// overlayCase is a card of the vault screen: the key that opens it, its
// title in the view and one of its footer hints.
type overlayCase struct {
	name  string
	open  tea.KeyPressMsg
	want  overlay
	title string
	hint  string
}

var overlayCases = []overlayCase{
	{"re-prompt", keyMsg("c"), overlayReprompt, "Master password re-prompt", "submit"},
	{"generator", keyMsg("p"), overlayGenerator, "Password Generator", "regen"},
	{"theme picker", keyMsg("T"), overlayThemePicker, "Choose theme", "select"},
	{"help", keyMsg("?"), overlayHelp, "Keybindings", "scroll"},
	{"quit", keyMsg("q"), overlayQuit, "Quit lazybw?", "yes"},
}

// sendKey gives msg to m. It does not run the command, which could start
// bw.
func sendKey(m VaultModel, msg tea.KeyPressMsg) VaultModel {
	updated, _ := m.Update(msg)
	return updated.(VaultModel)
}

// TestEachOverlayOwnsViewFooterAndKeys opens each card. The view shows only
// that card, the footer shows its hints, and the keys that open the other
// cards go to it. Esc closes it, and the vault list gets the keys again.
func TestEachOverlayOwnsViewFooterAndKeys(t *testing.T) {
	items := []bwcmd.Item{{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Bank", Reprompt: bwcmd.RepromptPassword,
		Login: &bwcmd.Login{Username: "me", Password: "bank-pw"}}}
	contentH := 24 - ui.ChromeHeight
	for _, tc := range overlayCases {
		t.Run(tc.name, func(t *testing.T) {
			keepTheme(t)
			m := sendKey(vaultAt(items, 80, 24), tc.open)
			if m.overlay != tc.want {
				t.Fatalf("overlay = %d, want %d", m.overlay, tc.want)
			}
			check := func(when string) {
				t.Helper()
				plain := ansi.Strip(m.ViewContent(80, contentH))
				if !strings.Contains(plain, tc.title) {
					t.Errorf("%s: the view must show %q:\n%s", when, tc.title, plain)
				}
				for _, other := range overlayCases {
					if other.want != tc.want && strings.Contains(plain, other.title) {
						t.Errorf("%s: the view must not show %q:\n%s", when, other.title, plain)
					}
				}
				if hints, _ := m.FooterContent(); !hasHintDesc(hints, tc.hint) {
					t.Errorf("%s: footer hints = %v, want %q", when, hints, tc.hint)
				}
			}
			check("after the open key")

			for _, other := range overlayCases {
				if other.want == tc.want {
					continue
				}
				if m = sendKey(m, other.open); m.overlay != tc.want {
					t.Fatalf("the open key of the %s changed the overlay to %d", other.name, m.overlay)
				}
			}
			check("after the open keys of the other cards")

			m = sendKey(m, tea.KeyPressMsg{Code: tea.KeyEscape})
			if m.overlay != overlayNone {
				t.Fatalf("after esc, overlay = %d, want none", m.overlay)
			}
			plain := ansi.Strip(m.ViewContent(80, contentH))
			for _, c := range overlayCases {
				if strings.Contains(plain, c.title) {
					t.Errorf("after esc, the view must show no card, found %q:\n%s", c.title, plain)
				}
			}
			if hints, _ := m.FooterContent(); !hasHintDesc(hints, "navigate") {
				t.Errorf("after esc, footer hints = %v, want the vault hints", hints)
			}
		})
	}
}

// TestHelpKeyClosesHelpOverlay checks that the overlay closes when the help
// card closes itself on its own key.
func TestHelpKeyClosesHelpOverlay(t *testing.T) {
	m := sendKey(newTestVault(testItems()), keyMsg("?"))
	if m.overlay != overlayHelp {
		t.Fatalf("overlay = %d, want help", m.overlay)
	}
	if m = sendKey(m, keyMsg("?")); m.overlay != overlayNone || m.help.Visible() {
		t.Errorf("after a second ?, overlay = %d and help visible = %v, want both closed", m.overlay, m.help.Visible())
	}
}
