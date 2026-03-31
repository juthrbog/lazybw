package screens

import (
	"errors"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("NO_COLOR", "1")
	os.Exit(m.Run())
}

func TestErrorViewContent(t *testing.T) {
	m := NewErrorModel(errors.New("connection timed out"), false)
	out := m.ViewContent()
	if !strings.Contains(out, "connection timed out") {
		t.Error("should contain error message")
	}
}

func TestErrorViewContentNilError(t *testing.T) {
	m := NewErrorModel(nil, false)
	out := m.ViewContent()
	if !strings.Contains(out, "unknown error") {
		t.Error("nil error should show 'unknown error'")
	}
}

func TestErrorFooterHintsNonFatal(t *testing.T) {
	m := NewErrorModel(errors.New("err"), false)
	hints := m.FooterHints()
	if !strings.Contains(hints, "r retry") {
		t.Error("non-fatal should show retry hint")
	}
	if !strings.Contains(hints, "q quit") {
		t.Error("should show quit hint")
	}
}

func TestErrorFooterHintsFatal(t *testing.T) {
	m := NewErrorModel(errors.New("err"), true)
	hints := m.FooterHints()
	if strings.Contains(hints, "retry") {
		t.Error("fatal should not show retry")
	}
	if !strings.Contains(hints, "q quit") {
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
}
