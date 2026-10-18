package ui

import (
	"os"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("NO_COLOR", "1")
	os.Exit(m.Run())
}

func TestRenderHeaderWithEmail(t *testing.T) {
	out := RenderHeader(HeaderData{Email: "user@test.com", Width: 80})
	if !strings.Contains(out, "user@test.com") {
		t.Error("header should contain email")
	}
	if !strings.Contains(out, "▄") {
		t.Error("header should contain gradient separator")
	}
	// Header is 2 lines.
	if !strings.Contains(out, "\n") {
		t.Error("header should be 2 lines")
	}
}

func TestRenderHeaderNoEmail(t *testing.T) {
	out := RenderHeader(HeaderData{Width: 60})
	if out == "" {
		t.Error("header should not be empty")
	}
	if !strings.Contains(out, "▄") {
		t.Error("header should contain gradient separator")
	}
}

func TestRenderHeaderNarrowWidth(t *testing.T) {
	// Should not panic.
	out := RenderHeader(HeaderData{Email: "user@test.com", Width: 10})
	if out == "" {
		t.Error("header should not be empty")
	}
}

func TestRenderHeaderWideShowsItemCount(t *testing.T) {
	out := RenderHeader(HeaderData{Email: "user@test.com", ItemCount: 42, Width: 100})
	if !strings.Contains(out, "42 items") {
		t.Error("wide header should show item count")
	}
}

func TestRenderHeaderMediumHidesItemCount(t *testing.T) {
	out := RenderHeader(HeaderData{Email: "user@test.com", ItemCount: 42, Width: 70})
	if strings.Contains(out, "42 items") {
		t.Error("medium header should not show item count")
	}
}

func TestRenderFooterWithContent(t *testing.T) {
	hints := []HintBinding{{Key: "j/k", Desc: "navigate"}}
	out := RenderFooter(hints, "synced 2m ago", 60)
	if !strings.Contains(out, "j/k") || !strings.Contains(out, "navigate") {
		t.Error("footer should contain hints")
	}
	if !strings.Contains(out, "synced 2m ago") {
		t.Error("footer should contain status")
	}
}

func TestRenderFooterEmpty(t *testing.T) {
	out := RenderFooter(nil, "", 60)
	if out == "" {
		t.Error("footer should render even when empty")
	}
}

func TestRenderFooterHintsOnly(t *testing.T) {
	hints := []HintBinding{{Key: "q", Desc: "quit"}}
	out := RenderFooter(hints, "", 60)
	if !strings.Contains(out, "q") || !strings.Contains(out, "quit") {
		t.Error("footer should contain hints")
	}
}

func TestCenterInAreaSingleLine(t *testing.T) {
	out := CenterInArea("hello", 40, 10)
	if !strings.Contains(out, "hello") {
		t.Error("output should contain content")
	}
	lines := strings.Split(out, "\n")
	// Should have vertical padding.
	if len(lines) < 5 {
		t.Errorf("expected vertical padding, got %d lines", len(lines))
	}
}

func TestCenterInAreaMultiLine(t *testing.T) {
	out := CenterInArea("line1\nline2", 40, 10)
	if !strings.Contains(out, "line1") || !strings.Contains(out, "line2") {
		t.Error("output should contain both lines")
	}
}

func TestCenterInAreaContentTallerThanArea(t *testing.T) {
	tall := strings.Repeat("line\n", 20)
	// Should not panic.
	out := CenterInArea(tall, 40, 5)
	if !strings.Contains(out, "line") {
		t.Error("output should contain content")
	}
}

func TestCenterInAreaExactHeight(t *testing.T) {
	tests := []struct {
		name    string
		content string
		height  int
	}{
		{"one line", "x", 5},
		{"fills the area", "a\nb\nc", 3},
		{"taller than the area", strings.Repeat("line\n", 20), 5},
		{"one row", "a\nb", 1},
	}
	for _, tt := range tests {
		out := CenterInArea(tt.content, 10, tt.height)
		if got := len(strings.Split(out, "\n")); got != tt.height {
			t.Errorf("%s: got %d lines, want %d", tt.name, got, tt.height)
		}
	}
}

func TestRenderHeaderLongEmail(t *testing.T) {
	email := strings.Repeat("e", 28) + "@example.com" // 40 characters
	for _, width := range []int{10, 20, 30, 45, 60, 80, 120} {
		out := RenderHeader(HeaderData{Email: email, ItemCount: 42, SelectedType: "Login", Width: width})
		lines := strings.Split(out, "\n")
		if len(lines) != 2 {
			t.Fatalf("width %d: header has %d lines, want 2", width, len(lines))
		}
		for _, l := range lines {
			if w := lipgloss.Width(l); w > width {
				t.Errorf("width %d: line is %d cells wide: %q", width, w, l)
			}
		}
	}
	plain := ansi.Strip(RenderHeader(HeaderData{Email: email, Width: 30}))
	if !strings.Contains(plain, "eeee") || !strings.Contains(plain, "…") {
		t.Errorf("at width 30 the email must show cut, got %q", plain)
	}
}

func TestRenderFooterFitsWidth(t *testing.T) {
	hints := []HintBinding{
		{Key: "j/k", Desc: "navigate"}, {Key: "/", Desc: "search"}, {Key: "c", Desc: "pwd"},
		{Key: "?", Desc: "help"}, {Key: "q", Desc: "quit"},
	}
	status := StyleError.Render("✗ Sync failed: " + strings.Repeat("bw could not reach the server ", 5))
	for _, width := range []int{8, 20, 40, 60, 80, 120} {
		out := RenderFooter(hints, status, width)
		if strings.Contains(out, "\n") {
			t.Errorf("width %d: footer has more than one line", width)
		}
		if w := lipgloss.Width(out); w != width {
			t.Errorf("width %d: footer is %d cells wide: %q", width, w, out)
		}
	}
	out := ansi.Strip(RenderFooter(hints, status, 80))
	if !strings.Contains(out, "j/k navigate") || !strings.Contains(out, "Sync failed") {
		t.Errorf("footer must keep the first hint and the start of the status, got %q", out)
	}
}

func TestCenterInAreaZeroDimensions(t *testing.T) {
	// Should not panic. An area with no rows has no content.
	if out := CenterInArea("hello", 0, 0); out != "" {
		t.Errorf("CenterInArea with height 0 = %q, want empty", out)
	}
}
