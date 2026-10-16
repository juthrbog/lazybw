package ui

import (
	"strings"
	"testing"

	"github.com/juthrbog/lazybw/bwcmd"
)

func TestRenderItemRowHidesNoteText(t *testing.T) {
	item := bwcmd.Item{
		Type:  bwcmd.ItemTypeSecureNote,
		Name:  "Deploy",
		Notes: "API_KEY=abc123\nsecond line",
	}
	out := RenderItemRow(item, false, 80, false)
	if strings.Contains(out, "abc123") || strings.Contains(out, "API_KEY") {
		t.Errorf("the list row shows note text: %q", out)
	}
	if !strings.Contains(out, "(note)") {
		t.Errorf("the list row must show (note): %q", out)
	}
}
