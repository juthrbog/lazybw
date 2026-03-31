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
	out := RenderHeader("user@test.com", 60)
	if !strings.Contains(out, "lazybw") {
		t.Error("header should contain 'lazybw'")
	}
	if !strings.Contains(out, "user@test.com") {
		t.Error("header should contain email")
	}
}

func TestRenderHeaderNoEmail(t *testing.T) {
	out := RenderHeader("", 60)
	if !strings.Contains(out, "lazybw") {
		t.Error("header should contain 'lazybw'")
	}
}

func TestRenderHeaderNarrowWidth(t *testing.T) {
	// Should not panic.
	out := RenderHeader("user@test.com", 10)
	if out == "" {
		t.Error("header should not be empty")
	}
}

func TestRenderFooterWithContent(t *testing.T) {
	out := RenderFooter("j/k navigate", "synced 2m ago", 60)
	if !strings.Contains(out, "j/k navigate") {
		t.Error("footer should contain hints")
	}
	if !strings.Contains(out, "synced 2m ago") {
		t.Error("footer should contain status")
	}
}

func TestRenderFooterEmpty(t *testing.T) {
	out := RenderFooter("", "", 60)
	if out == "" {
		t.Error("footer should render even when empty")
	}
}

func TestRenderFooterHintsOnly(t *testing.T) {
	out := RenderFooter("q quit", "", 60)
	if !strings.Contains(out, "q quit") {
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
