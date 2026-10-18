package ui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
)

func TestRenderItemRowLoginSelected(t *testing.T) {
	item := bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Gmail",
		Login: &bwcmd.Login{Username: "user@gmail.com"},
	}
	out := RenderItemRow(item, true, 60, false)
	if !strings.Contains(out, "▶") {
		t.Error("selected row should contain cursor")
	}
	if !strings.Contains(out, "Gmail") {
		t.Error("row should contain item name")
	}
	if !strings.Contains(out, "user@gmail.com") {
		t.Error("row should contain username")
	}
}

func TestRenderItemRowLoginUnselected(t *testing.T) {
	item := bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Gmail",
		Login: &bwcmd.Login{Username: "user@gmail.com"},
	}
	out := RenderItemRow(item, false, 60, false)
	if strings.Contains(out, "▶") {
		t.Error("unselected row should not contain cursor")
	}
	if !strings.Contains(out, "Gmail") {
		t.Error("row should contain item name")
	}
}

func TestRenderItemRowCard(t *testing.T) {
	item := bwcmd.Item{
		Type: bwcmd.ItemTypeCard,
		Name: "Visa Debit",
		Card: &bwcmd.Card{Number: "4242424242424242"},
	}
	out := RenderItemRow(item, false, 60, false)
	if !strings.Contains(out, "Visa Debit") {
		t.Error("row should contain card name")
	}
	if !strings.Contains(out, "4242") {
		t.Error("row should contain last 4 digits")
	}
}

func TestRenderItemRowNote(t *testing.T) {
	item := bwcmd.Item{
		Type:  bwcmd.ItemTypeSecureNote,
		Name:  "API Keys",
		Notes: "some secret",
	}
	out := RenderItemRow(item, false, 60, false)
	if !strings.Contains(out, "API Keys") {
		t.Error("row should contain note name")
	}
}

func TestRenderItemRowIdentity(t *testing.T) {
	item := bwcmd.Item{
		Type:     bwcmd.ItemTypeIdentity,
		Name:     "Personal ID",
		Identity: &bwcmd.Identity{FirstName: "John", LastName: "Doe"},
	}
	out := RenderItemRow(item, false, 60, false)
	if !strings.Contains(out, "Personal ID") {
		t.Error("row should contain identity name")
	}
	if !strings.Contains(out, "John Doe") {
		t.Error("row should contain full name as description")
	}
}

func TestRenderItemRowSSHKey(t *testing.T) {
	item := bwcmd.Item{
		Type:   bwcmd.ItemTypeSSHKey,
		Name:   "Server Key",
		SSHKey: &bwcmd.SSHKey{KeyFingerprint: "SHA256:abc123"},
	}
	out := RenderItemRow(item, false, 60, false)
	if !strings.Contains(out, "Server Key") {
		t.Error("row should contain SSH key name")
	}
	if !strings.Contains(out, "SHA256:abc123") {
		t.Error("row should contain fingerprint as description")
	}
}

func TestRenderItemRowIndented(t *testing.T) {
	item := bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Gmail",
		Login: &bwcmd.Login{Username: "user@gmail.com"},
	}
	normal := RenderItemRow(item, false, 60, false)
	indented := RenderItemRow(item, false, 60, true)

	// Indented row should start with more spaces.
	if !strings.HasPrefix(indented, "    ") {
		t.Error("indented row should start with 4 spaces")
	}
	if strings.HasPrefix(normal, "    ") {
		t.Error("normal row should not start with 4 spaces")
	}
}

func TestRenderItemRowIndentedSelected(t *testing.T) {
	item := bwcmd.Item{
		Type: bwcmd.ItemTypeLogin,
		Name: "Gmail",
	}
	out := RenderItemRow(item, true, 60, true)
	if !strings.Contains(out, "▶") {
		t.Error("selected indented row should contain cursor")
	}
}

func TestRenderGroupRowCollapsed(t *testing.T) {
	out := RenderGroupRow("Discord", 3, false, false, 60)
	if !strings.Contains(out, "▸") {
		t.Error("collapsed group should show ▸")
	}
	if !strings.Contains(out, "Discord") {
		t.Error("group row should contain base name")
	}
	if !strings.Contains(out, "(3)") {
		t.Error("group row should contain count")
	}
}

func TestRenderGroupRowExpanded(t *testing.T) {
	out := RenderGroupRow("Discord", 3, true, false, 60)
	if !strings.Contains(out, "▾") {
		t.Error("expanded group should show ▾")
	}
}

func TestRenderGroupRowSelected(t *testing.T) {
	out := ansi.Strip(RenderGroupRow("Discord", 3, false, true, 60))
	if !strings.HasPrefix(out, "▶ ▸  Discord (3)") {
		t.Errorf("selected collapsed group = %q, want the cursor and the chevron once each", out)
	}
	if n := strings.Count(out, "▶"); n != 1 {
		t.Errorf("selected collapsed group shows ▶ %d times, want once: %q", n, out)
	}
}

func TestRenderGroupRowFitsWidth(t *testing.T) {
	names := []string{
		"AWS",
		"Corporate single sign-on portal for the finance department",
		"東京都庁の職員ポータルサイトのログイン情報です",
		"Name\x1b[31m with an escape\ttab",
	}
	for _, name := range names {
		for _, width := range []int{4, 12, 30, 60} {
			for _, expanded := range []bool{false, true} {
				for _, selected := range []bool{false, true} {
					row := RenderGroupRow(name, 12, expanded, selected, width)
					if w := lipgloss.Width(row); w != width {
						t.Errorf("%q at %d (expanded %v, selected %v): width = %d", name, width, expanded, selected, w)
					}
					if width >= 30 && !strings.Contains(ansi.Strip(row), "(12)") {
						t.Errorf("%q at %d: the count must stay: %q", name, width, ansi.Strip(row))
					}
					if strings.Contains(row, "\x1b[31m") {
						t.Errorf("%q at %d: the name must be sanitized: %q", name, width, row)
					}
				}
			}
		}
	}
}

func TestRenderItemRowTruncation(t *testing.T) {
	item := bwcmd.Item{
		Type: bwcmd.ItemTypeLogin,
		Name: "This is a very long item name that should be truncated",
	}
	out := RenderItemRow(item, false, 30, false)
	if !strings.Contains(out, "…") {
		t.Error("long name should be truncated with ellipsis")
	}
}

func TestRenderItemRowFitsWidth(t *testing.T) {
	longLine := strings.Repeat("recovery code 1234-5678 ", 5)
	items := map[string]bwcmd.Item{
		"long note": {
			Type: bwcmd.ItemTypeSecureNote, Name: "AWS Root Account Recovery Codes",
			Notes: longLine + "\nsecond line",
		},
		"long username": {
			Type: bwcmd.ItemTypeLogin, Name: "Corporate SSO",
			Login: &bwcmd.Login{Username: "firstname.lastname.department@subsidiary.example-corporation.com"},
		},
		"CJK name": {
			Type: bwcmd.ItemTypeLogin, Name: "東京都庁の職員ポータルサイトのログイン情報です",
			Login: &bwcmd.Login{Username: "職員番号一二三四五六七八九"},
		},
		"accented name": {
			Type: bwcmd.ItemTypeLogin, Name: "Crédit Agricole Île-de-France — accès épargne",
			Login: &bwcmd.Login{Username: "élodie.françois@crédit.example"},
		},
		"SSH fingerprint": {
			Type: bwcmd.ItemTypeSSHKey, Name: "Deploy key",
			SSHKey: &bwcmd.SSHKey{KeyFingerprint: "SHA256:" + strings.Repeat("Ab3/", 11)},
		},
	}
	for name, item := range items {
		for _, width := range []int{40, 60, 80} {
			for _, selected := range []bool{false, true} {
				for _, indent := range []bool{false, true} {
					row := RenderItemRow(item, selected, width, indent)
					if w := lipgloss.Width(row); w != width {
						t.Errorf("%s at %d (selected %v, indent %v): width = %d", name, width, selected, indent, w)
					}
					if !utf8.ValidString(row) {
						t.Errorf("%s at %d: row is not valid UTF-8: %q", name, width, row)
					}
				}
			}
		}
	}
}

func TestRenderItemRowLongNoteKeepsName(t *testing.T) {
	item := bwcmd.Item{
		Type:  bwcmd.ItemTypeSecureNote,
		Name:  "AWS Root Account Recovery Codes",
		Notes: strings.Repeat("x", 100),
	}
	plain := ansi.Strip(RenderItemRow(item, false, 80, false))
	if !strings.Contains(plain, item.Name[:12]) {
		t.Errorf("row must show at least 12 characters of the name, got %q", plain)
	}
}

func TestRenderItemRowSelectedNameIsBold(t *testing.T) {
	item := bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Gmail",
		Login: &bwcmd.Login{Username: "user@gmail.com"},
	}
	cells := cellsOf(RenderItemRow(item, true, 60, false))
	start := -1
	for i := 0; i+len(item.Name) <= len(cells); i++ {
		if cellsText(cells[i:i+len(item.Name)]) == item.Name {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("row has no name: %q", cellsText(cells))
	}
	for _, c := range cells[start : start+len(item.Name)] {
		if !c.bold {
			t.Errorf("name cell %q is not bold", c.text)
		}
	}
	for _, c := range cells[start+len(item.Name):] {
		if c.bold {
			t.Errorf("cell %q after the name is bold", c.text)
		}
	}
}

func TestRenderItemRowSanitizesName(t *testing.T) {
	names := []string{
		"Spoof\nFAKE-ROW",
		"first\rsecond",
		"a\tb\tc",
		"\x1b[8mhidden\x1b[0m",
		"\x1b]8;;https://evil.example\x07link\x1b]8;;\x07",
	}
	for _, name := range names {
		item := bwcmd.Item{Type: bwcmd.ItemTypeLogin, Name: name, Login: &bwcmd.Login{Username: "u\r\nv"}}
		for _, width := range []int{40, 80} {
			row := RenderItemRow(item, false, width, false)
			if w := lipgloss.Width(row); w != width {
				t.Errorf("%q at %d: width = %d", name, width, w)
			}
			for _, b := range []byte(ansi.Strip(row)) {
				if b < 0x20 || b == 0x7f {
					t.Errorf("%q at %d: row has control byte %#x: %q", name, width, b, row)
				}
			}
			if strings.Contains(row, "\x1b]") || strings.Contains(row, "\x1b[8m") {
				t.Errorf("%q at %d: row keeps an escape sequence from the name: %q", name, width, row)
			}
		}
	}
}
