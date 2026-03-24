package ui

import (
	"strings"
	"testing"

	"github.com/juthrbog/lazybw/bwcmd"
)

func TestRenderDrawerNilItem(t *testing.T) {
	out := RenderDrawer(DrawerProps{Width: 60})
	if !strings.Contains(out, "No item selected") {
		t.Error("nil item should show 'No item selected'")
	}
	assertMinLineCount(t, out, DrawerHeight-1)
}

func TestRenderDrawerLogin(t *testing.T) {
	item := &bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Gmail",
		Login: &bwcmd.Login{Username: "user@gmail.com", Password: "secret"},
	}
	out := RenderDrawer(DrawerProps{Item: item, Width: 60})
	if !strings.Contains(out, "Username") {
		t.Error("login drawer should contain 'Username'")
	}
	if !strings.Contains(out, "Password") {
		t.Error("login drawer should contain 'Password'")
	}
	if !strings.Contains(out, "Gmail") {
		t.Error("separator should contain item name")
	}
	if !strings.Contains(out, "Login") {
		t.Error("separator should contain type name")
	}
	assertMinLineCount(t, out, DrawerHeight-1)
}

func TestRenderDrawerLoginWithTOTP(t *testing.T) {
	item := &bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "AWS",
		Login: &bwcmd.Login{Username: "admin", Totp: "seed"},
	}
	out := RenderDrawer(DrawerProps{
		Item:         item,
		TOTPCode:     "843291",
		TOTPSecsLeft: 20,
		Width:        60,
	})
	if !strings.Contains(out, "843 291") {
		t.Error("should contain formatted TOTP code")
	}
	if !strings.Contains(out, "TOTP") {
		t.Error("should contain TOTP label")
	}
}

func TestRenderDrawerCard(t *testing.T) {
	item := &bwcmd.Item{
		Type: bwcmd.ItemTypeCard,
		Name: "Visa",
		Card: &bwcmd.Card{
			CardholderName: "John Smith",
			Number:         "4242424242424242",
			ExpMonth:       "12",
			ExpYear:        "27",
			Code:           "123",
		},
	}
	out := RenderDrawer(DrawerProps{Item: item, Width: 60})
	if !strings.Contains(out, "Cardholder") {
		t.Error("card drawer should contain 'Cardholder'")
	}
	if !strings.Contains(out, "4242") {
		t.Error("should show last 4 digits")
	}
	if !strings.Contains(out, "Card") {
		t.Error("separator should contain type 'Card'")
	}
	assertMinLineCount(t, out, DrawerHeight-1)
}

func TestRenderDrawerSecureNote(t *testing.T) {
	item := &bwcmd.Item{
		Type:  bwcmd.ItemTypeSecureNote,
		Name:  "Keys",
		Notes: "line1\nline2\nline3",
	}
	out := RenderDrawer(DrawerProps{Item: item, Width: 60})
	if !strings.Contains(out, "line1") {
		t.Error("note drawer should contain note text")
	}
	assertMinLineCount(t, out, DrawerHeight-1)
}

func TestRenderDrawerSecureNoteScroll(t *testing.T) {
	item := &bwcmd.Item{
		Type:  bwcmd.ItemTypeSecureNote,
		Name:  "Keys",
		Notes: "line1\nline2\nline3",
	}
	out := RenderDrawer(DrawerProps{Item: item, Width: 60, ScrollOffset: 1})
	if strings.Contains(out, "line1") {
		t.Error("scrolled note should not show first line")
	}
	if !strings.Contains(out, "line2") {
		t.Error("scrolled note should show second line")
	}
}

func TestRenderDrawerTOTPCountdownCircles(t *testing.T) {
	tests := []struct {
		name     string
		secsLeft int
		circle   string
	}{
		{"full", 28, "●"},
		{"three quarter", 20, "◕"},
		{"half", 14, "◑"},
		{"quarter", 8, "◔"},
		{"empty", 3, "○"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &bwcmd.Item{
				Type:  bwcmd.ItemTypeLogin,
				Name:  "Test",
				Login: &bwcmd.Login{Totp: "seed"},
			}
			out := RenderDrawer(DrawerProps{
				Item:         item,
				TOTPCode:     "123456",
				TOTPSecsLeft: tt.secsLeft,
				Width:        80,
			})
			if !strings.Contains(out, tt.circle) {
				t.Errorf("secsLeft=%d: expected circle %q in output", tt.secsLeft, tt.circle)
			}
		})
	}
}

func TestRenderDrawerTOTPLoading(t *testing.T) {
	item := &bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Test",
		Login: &bwcmd.Login{Totp: "seed"},
	}
	out := RenderDrawer(DrawerProps{Item: item, TOTPCode: "", Width: 60})
	if !strings.Contains(out, "loading") {
		t.Error("empty TOTP code should show 'loading'")
	}
}

func TestRenderDrawerIdentity(t *testing.T) {
	item := &bwcmd.Item{
		Type: bwcmd.ItemTypeIdentity,
		Name: "Personal",
		Identity: &bwcmd.Identity{
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john@test.com",
			Phone:     "+1-555-0123",
			SSN:       "123-45-6789",
			City:      "Springfield",
			State:     "IL",
		},
	}
	out := RenderDrawer(DrawerProps{Item: item, Width: 80})
	if !strings.Contains(out, "Name") {
		t.Error("identity drawer should contain 'Name'")
	}
	if !strings.Contains(out, "Email") {
		t.Error("identity drawer should contain 'Email'")
	}
	if !strings.Contains(out, "Identity") {
		t.Error("separator should contain 'Identity'")
	}
	if !strings.Contains(out, "•••••••••") {
		t.Error("SSN should be masked")
	}
	if strings.Contains(out, "123-45-6789") {
		t.Error("SSN value should NOT appear in output")
	}
	assertMinLineCount(t, out, DrawerHeight-1)
}

func assertMinLineCount(t *testing.T, output string, minExpected int) {
	t.Helper()
	// Count newline-separated segments. Trailing empty lines from padding
	// are valid — they ensure the drawer occupies a fixed height.
	count := strings.Count(output, "\n")
	if count < minExpected {
		t.Errorf("got %d newlines, want at least %d", count, minExpected)
	}
}
