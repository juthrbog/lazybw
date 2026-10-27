package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
)

func repromptLogin(reprompt int) *bwcmd.Item {
	return &bwcmd.Item{
		Type:     bwcmd.ItemTypeLogin,
		Name:     "Bank",
		Notes:    "pin 4321\nmore",
		Reprompt: reprompt,
		Login:    &bwcmd.Login{Username: "me", Password: "pw", Totp: "seed"},
	}
}

func TestDrawerHidesSecretsOfRepromptItem(t *testing.T) {
	out := ansi.Strip(RenderDrawer(DrawerProps{Item: repromptLogin(bwcmd.RepromptPassword), TOTPCode: "843291", TOTPSecsLeft: 20, Width: 80}))
	if !strings.Contains(out, repromptMarker) {
		t.Errorf("the drawer must show the re-prompt marker:\n%s", out)
	}
	if strings.Contains(out, "843 291") {
		t.Errorf("the drawer must hide the TOTP code until the password check passes:\n%s", out)
	}
	if strings.Contains(out, "pin 4321") {
		t.Errorf("the drawer must hide the note until the password check passes:\n%s", out)
	}
	if strings.Count(out, hiddenValue) != 2 {
		t.Errorf("the TOTP and the notes rows must say %q:\n%s", hiddenValue, out)
	}
	// The copy keys still show: they ask for the master password.
	for _, hint := range []string{"[t] copy", "[m] copy"} {
		if !strings.Contains(out, hint) {
			t.Errorf("the drawer must show %q:\n%s", hint, out)
		}
	}
}

func TestDrawerShowsSecretsAfterRepromptCheck(t *testing.T) {
	out := ansi.Strip(RenderDrawer(DrawerProps{
		Item: repromptLogin(bwcmd.RepromptPassword), TOTPCode: "843291", TOTPSecsLeft: 20, Width: 80,
		RepromptVerified: true,
	}))
	for _, want := range []string{repromptMarker, "843 291", "pin 4321"} {
		if !strings.Contains(out, want) {
			t.Errorf("after the password check, the drawer must show %q:\n%s", want, out)
		}
	}
}

func TestDrawerWithoutRepromptShowsNoMarker(t *testing.T) {
	out := ansi.Strip(RenderDrawer(DrawerProps{Item: repromptLogin(bwcmd.RepromptNone), TOTPCode: "843291", TOTPSecsLeft: 20, Width: 80}))
	if strings.Contains(out, repromptMarker) || strings.Contains(out, hiddenValue) {
		t.Errorf("an item without re-prompt must show no marker and hide nothing:\n%s", out)
	}
	if !strings.Contains(out, "843 291") || !strings.Contains(out, "pin 4321") {
		t.Errorf("an item without re-prompt must show its TOTP code and note:\n%s", out)
	}
}

func TestDrawerHidesRepromptNoteBody(t *testing.T) {
	var lines []string
	for range 20 {
		lines = append(lines, "secret line")
	}
	item := &bwcmd.Item{Type: bwcmd.ItemTypeSecureNote, Name: "Keys", Notes: strings.Join(lines, "\n"), Reprompt: bwcmd.RepromptPassword}
	props := DrawerProps{Item: item, Width: 60}
	out := ansi.Strip(RenderDrawer(props))
	if strings.Contains(out, "secret line") {
		t.Errorf("the drawer must hide the note body until the password check passes:\n%s", out)
	}
	if !strings.Contains(out, hiddenNote) || !strings.Contains(out, repromptMarker) {
		t.Errorf("the drawer must say that the note is hidden:\n%s", out)
	}
	if got := MaxScrollOffset(props); got != 0 {
		t.Errorf("a hidden note must not scroll, MaxScrollOffset = %d", got)
	}
	props.RepromptVerified = true
	if out := ansi.Strip(RenderDrawer(props)); !strings.Contains(out, "secret line") {
		t.Errorf("after the password check, the drawer must show the note:\n%s", out)
	}
	if got := MaxScrollOffset(props); got == 0 {
		t.Error("after the password check, a long note must scroll")
	}
}

func TestDrawerRepromptHeaderFits(t *testing.T) {
	item := &bwcmd.Item{Type: bwcmd.ItemTypeSecureNote, Name: "A long secure note name", Notes: "n", Reprompt: bwcmd.RepromptPassword}
	for _, width := range []int{20, 30, 40, 50, 60, 80} {
		out := RenderDrawer(DrawerProps{Item: item, Width: width})
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: line %d is %d cells wide: %q", width, i, w, ansi.Strip(line))
			}
		}
	}
	// At 80 cells, the hint, the marker and the badge all fit, in that
	// order.
	header := ansi.Strip(renderHeaderCard(item.Name, item.Type, noteCopyHint, repromptMarker, 80))
	hint, marker, badge := strings.Index(header, noteCopyHint), strings.Index(header, repromptMarker), strings.LastIndex(header, "Note")
	if hint < 0 || marker < hint || badge < marker {
		t.Errorf("header = %q, want the hint, the marker and the badge", header)
	}
	// A narrow header drops the hint before the marker.
	header = ansi.Strip(renderHeaderCard(item.Name, item.Type, noteCopyHint, repromptMarker, 40))
	if strings.Contains(header, noteCopyHint) || !strings.Contains(header, repromptMarker) {
		t.Errorf("header = %q, want the marker without the hint", header)
	}
}

func TestDrawerCardCVVHint(t *testing.T) {
	card := func(code string) *bwcmd.Item {
		return &bwcmd.Item{Type: bwcmd.ItemTypeCard, Name: "Visa", Card: &bwcmd.Card{Number: "4242424242424242", Code: code}}
	}
	if out := ansi.Strip(RenderDrawer(DrawerProps{Item: card("123"), Width: 60})); !strings.Contains(out, "[v] copy") {
		t.Errorf("a card with a CVV must show the [v] copy hint:\n%s", out)
	}
	out := ansi.Strip(RenderDrawer(DrawerProps{Item: card(""), Width: 60}))
	if strings.Contains(out, "[v] copy") {
		t.Errorf("a card without a CVV must not show the [v] copy hint:\n%s", out)
	}
	if !strings.Contains(out, "CVV         (none)") {
		t.Errorf("a card without a CVV must show (none):\n%s", out)
	}
}

func TestDrawerLoginEmptyFieldsHaveNoCopyHint(t *testing.T) {
	login := func(username, password string) *bwcmd.Item {
		return &bwcmd.Item{Type: bwcmd.ItemTypeLogin, Name: "Site", Login: &bwcmd.Login{Username: username, Password: password}}
	}
	out := ansi.Strip(RenderDrawer(DrawerProps{Item: login("octocat", "pw"), Width: 60}))
	for _, hint := range []string{"[u] copy", "[c] copy"} {
		if !strings.Contains(out, hint) {
			t.Errorf("a login with a username and a password must show %s:\n%s", hint, out)
		}
	}
	out = ansi.Strip(RenderDrawer(DrawerProps{Item: login("", ""), Width: 60}))
	for _, hint := range []string{"[u] copy", "[c] copy"} {
		if strings.Contains(out, hint) {
			t.Errorf("a login with no username and no password must not show %s:\n%s", hint, out)
		}
	}
	for _, row := range []string{"Username    (none)", "Password    (none)"} {
		if !strings.Contains(out, row) {
			t.Errorf("drawer must show %q:\n%s", row, out)
		}
	}
}
