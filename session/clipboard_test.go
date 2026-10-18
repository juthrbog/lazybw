package session

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestClearClipboardPayloadIsNotEmpty(t *testing.T) {
	msg := ClearClipboard()()
	// tea.SetClipboard returns an unexported string type that holds the
	// payload.
	v := reflect.ValueOf(msg)
	if v.Kind() != reflect.String {
		t.Fatalf("ClearClipboard produced %T, want a clipboard write", msg)
	}
	payload := v.String()
	if msg != tea.SetClipboard(payload)() {
		t.Fatalf("ClearClipboard produced %#v, want a clipboard write", msg)
	}
	if payload == "" {
		t.Error("the clear payload is empty: tmux skips an empty OSC 52 write and keeps the secret")
	}
	if strings.TrimSpace(payload) != "" {
		t.Errorf("the clear payload is %q, want only white space", payload)
	}
}
