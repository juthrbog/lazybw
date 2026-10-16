package ui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestRenderDrawerNilItem(t *testing.T) {
	out := RenderDrawer(DrawerProps{Width: 60})
	if !strings.Contains(out, "No item selected") {
		t.Error("nil item should show 'No item selected'")
	}
	assertMinLineCount(t, out, DefaultDrawerHeight-1)
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
	assertMinLineCount(t, out, DefaultDrawerHeight-1)
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
	assertMinLineCount(t, out, DefaultDrawerHeight-1)
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
	assertMinLineCount(t, out, DefaultDrawerHeight-1)
}

func TestRenderDrawerSecureNoteScroll(t *testing.T) {
	// The note has more lines than the 6 rows of the default drawer body.
	item := &bwcmd.Item{
		Type:  bwcmd.ItemTypeSecureNote,
		Name:  "Keys",
		Notes: "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8",
	}
	out := RenderDrawer(DrawerProps{Item: item, Width: 60, ScrollOffset: 1})
	if strings.Contains(out, "line1") {
		t.Error("scrolled note should not show first line")
	}
	if !strings.Contains(out, "line2") {
		t.Error("scrolled note should show second line")
	}
}

func TestRenderDrawerTOTPCountdownBar(t *testing.T) {
	tests := []struct {
		name     string
		secsLeft int
		bar      string
	}{
		{"full", 30, "████"},
		{"most", 28, "███▋"},
		{"half", 15, "██░░"},
		{"quarter", 8, "█░░░"},
		{"almost empty", 1, "▏░░░"},
		{"empty", 0, "░░░░"},
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
			plain := ansiRe.ReplaceAllString(out, "")
			if !strings.Contains(plain, tt.bar) {
				t.Errorf("secsLeft=%d: expected bar %q in output, got:\n%s", tt.secsLeft, tt.bar, plain)
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
	assertMinLineCount(t, out, DefaultDrawerHeight-1)
}

func TestRenderDrawerSSHKey(t *testing.T) {
	item := &bwcmd.Item{
		Type: bwcmd.ItemTypeSSHKey,
		Name: "Server Key",
		SSHKey: &bwcmd.SSHKey{
			KeyFingerprint: "SHA256:abc123def456",
			PublicKey:      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...",
			PrivateKey:     "-----BEGIN OPENSSH PRIVATE KEY-----\nb3Blbn...",
		},
	}
	out := RenderDrawer(DrawerProps{Item: item, Width: 80})
	if !strings.Contains(out, "Fingerprint") {
		t.Error("SSH key drawer should contain 'Fingerprint'")
	}
	if !strings.Contains(out, "Public Key") {
		t.Error("SSH key drawer should contain 'Public Key'")
	}
	if !strings.Contains(out, "Private Key") {
		t.Error("SSH key drawer should contain 'Private Key'")
	}
	if !strings.Contains(out, "SSH Key") {
		t.Error("separator should contain 'SSH Key'")
	}
	if strings.Contains(out, "BEGIN OPENSSH") {
		t.Error("private key value should NOT appear in output")
	}
	if !strings.Contains(out, "••••••••••") {
		t.Error("private key should be masked")
	}
	assertMinLineCount(t, out, DefaultDrawerHeight-1)
}

func TestFieldRowTruncation(t *testing.T) {
	longURL := "https://amazonwebservices.benchprep.com/purchasing/checkout/very-long-path"
	out := fieldRow("URL", longURL, "[o] open", 60)
	if !strings.Contains(out, "[o] open") {
		t.Error("hint should always be visible after truncation")
	}
	if !strings.Contains(out, "…") {
		t.Error("long value should be truncated with ellipsis")
	}
}

func TestFieldRowNoTruncation(t *testing.T) {
	out := fieldRow("URL", "https://short.io", "[o] open", 80)
	if !strings.Contains(out, "https://short.io") {
		t.Error("short value should not be truncated")
	}
	if strings.Contains(out, "…") {
		t.Error("short value should not have ellipsis")
	}
}

func TestRenderDrawerCustomHeight(t *testing.T) {
	item := &bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Test",
		Login: &bwcmd.Login{Username: "user"},
	}
	out := RenderDrawer(DrawerProps{Item: item, Width: 60, Height: 12})
	lines := strings.Split(out, "\n")
	if len(lines) != 12 {
		t.Errorf("drawer with Height=12 should have 12 lines, got %d", len(lines))
	}
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

func TestRenderDrawerTOTPBarUsesPeriod(t *testing.T) {
	item := &bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Test",
		Login: &bwcmd.Login{Totp: "seed"},
	}
	out := RenderDrawer(DrawerProps{
		Item:         item,
		TOTPCode:     "123456",
		TOTPSecsLeft: 45,
		TOTPPeriod:   60,
		Width:        80,
	})
	plain := ansiRe.ReplaceAllString(out, "")
	if strings.Contains(plain, "████") {
		t.Errorf("with 45 of 60 seconds left, the bar must not be full:\n%s", plain)
	}
	if !strings.Contains(plain, "███░") {
		t.Errorf("with 45 of 60 seconds left, the bar must be three quarters full:\n%s", plain)
	}
}

func TestRenderDrawerTOTPInvalid(t *testing.T) {
	item := &bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Test",
		Login: &bwcmd.Login{Totp: "!!!!"},
	}
	out := RenderDrawer(DrawerProps{Item: item, TOTPInvalid: true, Width: 60})
	if !strings.Contains(out, "invalid TOTP secret") || strings.Contains(out, "loading") {
		t.Error("an invalid TOTP key must show 'invalid TOTP secret', not 'loading'")
	}
}

func TestFieldRowWidthSafe(t *testing.T) {
	values := map[string]func(width int) string{
		"CJK value": func(width int) string {
			return fieldRow("Username", strings.Repeat("東京都庁", 20), "[u] copy", width)
		},
		"password mask": func(width int) string {
			return fieldRow("Password", "••••••••••", "[c] copy", width)
		},
		"styled value": func(width int) string {
			return fieldRow("Notes", StyleFaint.Render(strings.Repeat("é", 80)), "", width)
		},
		"styled TOTP": func(width int) string {
			return totpRow(DrawerProps{TOTPCode: "843291", TOTPSecsLeft: 20}, width)
		},
	}
	for name, render := range values {
		for width := 10; width <= 80; width++ {
			row := render(width)
			if w := lipgloss.Width(row); w > width {
				t.Errorf("%s at %d: width = %d: %q", name, width, w, row)
			}
			if !utf8.ValidString(row) {
				t.Errorf("%s at %d: row is not valid UTF-8: %q", name, width, row)
			}
		}
	}
}

func TestRenderDrawerLoginNotesWithCRLF(t *testing.T) {
	item := &bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Gmail",
		Login: &bwcmd.Login{Username: "user"},
		Notes: "first line\r\nsecond",
	}
	for _, auto := range []bool{false, true} {
		out := RenderDrawer(DrawerProps{Item: item, Width: 60, AutoHeight: auto})
		var notesRow string
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "Notes") {
				notesRow = ansi.Strip(line)
			}
		}
		if !strings.Contains(notesRow, "first line") {
			t.Errorf("Notes row = %q, want it to contain %q", notesRow, "first line")
		}
		if !strings.Contains(notesRow, "[n] copy") {
			t.Errorf("Notes row = %q, want the copy hint", notesRow)
		}
		if strings.ContainsAny(out, "\r\t") {
			t.Errorf("drawer has a carriage return or a tab: %q", out)
		}
	}
}

func TestRenderDrawerSanitizesText(t *testing.T) {
	item := &bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Spoof\n\x1b]8;;https://evil.example\x07link\x1b]8;;\x07",
		Login: &bwcmd.Login{Username: "a\tb\rc", URIs: []bwcmd.URI{{URI: "https://x.example/\x1b[8mhidden"}}},
		Notes: "\x1b[2Jline",
	}
	out := RenderDrawer(DrawerProps{Item: item, Width: 60})
	if lines := strings.Split(out, "\n"); len(lines) != DefaultDrawerHeight {
		t.Errorf("drawer has %d lines, want %d", len(lines), DefaultDrawerHeight)
	}
	for _, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > 60 {
			t.Errorf("line is %d cells wide: %q", w, line)
		}
		for _, b := range []byte(ansi.Strip(line)) {
			if b < 0x20 || b == 0x7f {
				t.Errorf("line has control byte %#x: %q", b, line)
			}
		}
	}
	if strings.Contains(out, "\x1b]") || strings.Contains(out, "\x1b[8m") || strings.Contains(out, "\x1b[2J") {
		t.Errorf("drawer keeps an escape sequence from the item: %q", out)
	}
}

func TestRenderDrawerWrapsLongNoteLine(t *testing.T) {
	words := strings.Fields(strings.Repeat("alpha bravo charlie delta echo foxtrot golf ", 4))
	line := strings.Join(words, " ")[:157]
	item := &bwcmd.Item{Type: bwcmd.ItemTypeSecureNote, Name: "Pasted", Notes: line}
	out := RenderDrawer(DrawerProps{Item: item, Width: 80})
	for _, l := range strings.Split(out, "\n") {
		if w := lipgloss.Width(l); w > 80 {
			t.Errorf("line is %d cells wide, want at most 80: %q", w, l)
		}
	}
	plain := strings.Join(strings.Fields(ansi.Strip(out)), " ")
	if !strings.Contains(plain, line) {
		t.Errorf("drawer does not show all of the note:\n%s", plain)
	}
}

func TestRenderDrawerSecureNoteScrollStopsAtLastPage(t *testing.T) {
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("line%02d", i))
	}
	props := DrawerProps{
		Item:   &bwcmd.Item{Type: bwcmd.ItemTypeSecureNote, Name: "Keys", Notes: strings.Join(lines, "\n")},
		Width:  60,
		Height: 8,
	}
	if got := MaxScrollOffset(props); got != 4 {
		t.Fatalf("MaxScrollOffset = %d, want 4 (10 lines, 6 rows)", got)
	}
	// An offset past the end shows the last full page, not blank rows.
	props.ScrollOffset = 50
	out := RenderDrawer(props)
	if !strings.Contains(out, "line04") || !strings.Contains(out, "line09") || strings.Contains(out, "line03") {
		t.Errorf("an offset past the end must show the last page:\n%s", out)
	}
	login := DrawerProps{Item: &bwcmd.Item{Type: bwcmd.ItemTypeLogin, Name: "x", Notes: strings.Join(lines, "\n")}, Width: 60}
	if got := MaxScrollOffset(login); got != 0 {
		t.Errorf("MaxScrollOffset for a login = %d, want 0", got)
	}
}

func TestRenderDrawerMarksHiddenFields(t *testing.T) {
	item := &bwcmd.Item{
		Type: bwcmd.ItemTypeIdentity,
		Name: "Personal",
		Identity: &bwcmd.Identity{
			FirstName: "John", LastName: "Doe", Email: "john@test.com", Phone: "555",
			Company: "ACME", SSN: "123", PassportNumber: "P1", LicenseNumber: "L1", City: "Springfield",
		},
	}
	out := RenderDrawer(DrawerProps{Item: item, Width: 80})
	if lines := strings.Split(out, "\n"); len(lines) != DefaultDrawerHeight {
		t.Errorf("drawer has %d lines, want %d", len(lines), DefaultDrawerHeight)
	}
	// 8 fields in 6 rows: 5 fields and the marker for the other 3.
	if !strings.Contains(out, "↓ 3 more") {
		t.Errorf("drawer must mark the 3 hidden fields:\n%s", ansi.Strip(out))
	}
}

func TestRenderDrawerSecureNoteShowsCopyHint(t *testing.T) {
	item := &bwcmd.Item{Type: bwcmd.ItemTypeSecureNote, Name: "Keys", Notes: "secret"}
	if out := RenderDrawer(DrawerProps{Item: item, Width: 60}); !strings.Contains(out, "[n] copy") {
		t.Errorf("note drawer must show the copy hint:\n%s", out)
	}
}

func TestTOTPRowKeepsFullCountdown(t *testing.T) {
	totpRowAt := func(secsLeft, width int) string {
		return ansi.Strip(totpRow(DrawerProps{TOTPCode: "907064", TOTPSecsLeft: secsLeft}, width))
	}
	// "907 064  ███▋" is 13 cells, so from 27 cells the code and the bar fit
	// after the label.
	for width := 27; width <= 80; width++ {
		rows := map[int]string{9: totpRowAt(9, width), 23: totpRowAt(23, width)}
		for secs, row := range rows {
			if strings.Contains(row, "…") {
				t.Errorf("width %d, %ds left: the countdown is cut: %q", width, secs, row)
			}
			if !strings.Contains(row, "907 064") {
				t.Errorf("width %d, %ds left: the code is missing: %q", width, secs, row)
			}
			if lipgloss.Width(row) > width {
				t.Errorf("width %d, %ds left: row is %d cells wide: %q", width, secs, lipgloss.Width(row), row)
			}
		}
		// The layout must not change as the seconds tick.
		for _, part := range []string{"[t] copy", "s"} {
			if strings.HasSuffix(strings.TrimSpace(rows[9]), part) != strings.HasSuffix(strings.TrimSpace(rows[23]), part) {
				t.Errorf("width %d: the row changes between 9s and 23s: %q, %q", width, rows[9], rows[23])
			}
		}
	}
	if row := totpRowAt(23, 40); !strings.Contains(row, "23s") {
		t.Errorf("at width 40 the row must show the full countdown: %q", row)
	}
	if row := totpRowAt(23, 80); !strings.Contains(row, "23s") || !strings.Contains(row, "[t] copy") {
		t.Errorf("at width 80 the row must show the countdown and the hint: %q", row)
	}
}

func TestRenderDrawerOverlayNotePage(t *testing.T) {
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("line%02d", i))
	}
	// An overlay card with room for the header and 4 note lines.
	props := DrawerProps{
		Item:       &bwcmd.Item{Type: bwcmd.ItemTypeSecureNote, Name: "Keys", Notes: strings.Join(lines, "\n")},
		Width:      40,
		Height:     5,
		AutoHeight: true,
	}
	if got := MaxScrollOffset(props); got != 6 {
		t.Fatalf("MaxScrollOffset = %d, want 6 (10 lines, 4 rows)", got)
	}
	props.ScrollOffset = 6
	out := RenderDrawer(props)
	if got := strings.Split(out, "\n"); len(got) != 5 {
		t.Errorf("drawer has %d lines, want 5:\n%s", len(got), ansi.Strip(out))
	}
	if !strings.Contains(out, "line06") || !strings.Contains(out, "line09") || strings.Contains(out, "line05") {
		t.Errorf("the last page must show lines 6 to 9:\n%s", ansi.Strip(out))
	}
}

func TestRenderDrawerHeaderOnly(t *testing.T) {
	note := &bwcmd.Item{Type: bwcmd.ItemTypeSecureNote, Name: "Keys", Notes: "a\nb\nc"}
	props := DrawerProps{Item: note, Width: 40, Height: 2}
	if got := strings.Split(RenderDrawer(props), "\n"); len(got) != 2 {
		t.Errorf("a 2-row drawer has %d lines, want 2", len(got))
	}
	if got := MaxScrollOffset(props); got != 0 {
		t.Errorf("MaxScrollOffset with no note rows = %d, want 0", got)
	}
}

func BenchmarkRenderDrawerLongNoteLine(b *testing.B) {
	item := &bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Gmail",
		Login: &bwcmd.Login{Username: "user", URIs: []bwcmd.URI{{URI: "https://example.com/" + strings.Repeat("p", 10000)}}},
		Notes: strings.Repeat("é", 10000) + "\nsecond",
	}
	props := DrawerProps{Item: item, Width: 80}
	for b.Loop() {
		RenderDrawer(props)
	}
}
