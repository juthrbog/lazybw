package ui

import (
	"os"
	"strings"
	"testing"

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

func TestCenterInAreaZeroDimensions(t *testing.T) {
	// Should not panic.
	out := CenterInArea("hello", 0, 0)
	if !strings.Contains(out, "hello") {
		t.Error("output should contain content")
	}
}
