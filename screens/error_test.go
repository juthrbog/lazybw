package screens

import (
	"errors"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("NO_COLOR", "1")
	// Point PATH at an empty directory, so that a test that runs a bwcmd
	// command can never reach the real bw.
	dir, err := os.MkdirTemp("", "lazybw-nobw-")
	if err != nil {
		os.Exit(1)
	}
	_ = os.Setenv("PATH", dir)
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestErrorViewContent(t *testing.T) {
	m := NewErrorModel(errors.New("connection timed out"), false)
	out := m.ViewContent(80)
	if !strings.Contains(out, "connection timed out") {
		t.Error("should contain error message")
	}
}

func TestErrorViewContentNilError(t *testing.T) {
	m := NewErrorModel(nil, false)
	out := m.ViewContent(80)
	if !strings.Contains(out, "unknown error") {
		t.Error("nil error should show 'unknown error'")
	}
}

func TestErrorViewContentWrapsLongError(t *testing.T) {
	words := strings.Repeat("bw failed to reach the server ", 10)
	m := NewErrorModel(errors.New(words+"\x1b[8mhidden\r\nsecond line"), false)
	for _, width := range []int{120, 80, 40} {
		out := m.ViewContent(width)
		lines := strings.Split(out, "\n")
		if len(lines) < 2 {
			t.Fatalf("width %d: the long error is not wrapped:\n%s", width, out)
		}
		limit := min(width-4, maxErrorWidth)
		for _, l := range lines {
			if w := lipgloss.Width(l); w > limit {
				t.Errorf("width %d: line is %d cells wide, want at most %d: %q", width, w, limit, l)
			}
			if strings.ContainsAny(ansi.Strip(l), "\x1b\r") {
				t.Errorf("width %d: line has control characters: %q", width, l)
			}
		}
		plain := strings.Join(strings.Fields(ansi.Strip(out)), " ")
		if !strings.Contains(plain, strings.TrimSpace(words)) || !strings.Contains(plain, "hidden second line") {
			t.Errorf("width %d: the wrapped error lost text:\n%s", width, plain)
		}
	}
}

func TestErrorFooterHintsNonFatal(t *testing.T) {
	m := NewErrorModel(errors.New("err"), false)
	hints := m.FooterHints()
	if !hasHintKey(hints, "r") {
		t.Error("non-fatal should show retry hint")
	}
	if !hasHintKey(hints, "q") {
		t.Error("should show quit hint")
	}
}

func TestErrorFooterHintsFatal(t *testing.T) {
	m := NewErrorModel(errors.New("err"), true)
	hints := m.FooterHints()
	if hasHintKey(hints, "r") {
		t.Error("fatal should not show retry")
	}
	if !hasHintKey(hints, "q") {
		t.Error("should show quit hint")
	}
}

func TestErrorUpdateRetryNonFatal(t *testing.T) {
	m := NewErrorModel(errors.New("err"), false)
	updated, cmd := m.Update(tea.KeyPressMsg{Text: "r"})
	_ = updated
	if cmd == nil {
		t.Fatal("expected command from retry key")
	}
	msg := cmd()
	if _, ok := msg.(RetryMsg); !ok {
		t.Errorf("expected RetryMsg, got %T", msg)
	}
}

func TestErrorUpdateRetryFatal(t *testing.T) {
	m := NewErrorModel(errors.New("err"), true)
	_, cmd := m.Update(tea.KeyPressMsg{Text: "r"})
	if cmd != nil {
		t.Error("fatal error should not respond to retry key")
	}
}

func TestErrorUpdateQuit(t *testing.T) {
	m := NewErrorModel(errors.New("err"), false)
	_, cmd := m.Update(tea.KeyPressMsg{Text: "q"})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	// The root model must see the quit, so that it can lock the vault
	// and clear the clipboard. tea.Quit would skip both.
	if _, ok := cmd().(QuitMsg); !ok {
		t.Errorf("q returned %T, want QuitMsg", cmd())
	}
}
