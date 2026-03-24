package screens

import (
	"strings"
	"testing"
)

func TestLockedFooterContentUnlock(t *testing.T) {
	m := NewLockedModel(false)
	hints, status := m.FooterContent()
	if !strings.Contains(hints, "unlock") {
		t.Errorf("unlock mode hints should contain 'unlock', got %q", hints)
	}
	if status != "" {
		t.Errorf("status should be empty, got %q", status)
	}
}

func TestLockedFooterContentLogin(t *testing.T) {
	m := NewLockedModel(true)
	hints, _ := m.FooterContent()
	if !strings.Contains(hints, "submit") {
		t.Errorf("login mode hints should contain 'submit', got %q", hints)
	}
}

func TestLockedFooterContentUnlocking(t *testing.T) {
	m := NewLockedModel(false)
	m.state = lockedUnlocking
	hints, status := m.FooterContent()
	if hints != "" {
		t.Errorf("unlocking hints should be empty, got %q", hints)
	}
	if status != "" {
		t.Errorf("unlocking status should be empty, got %q", status)
	}
}

func TestLockedViewContentUnlock(t *testing.T) {
	m := NewLockedModel(false)
	out := m.ViewContent(80, 24)
	if !strings.Contains(out, "Vault is locked") {
		t.Error("unlock view should contain 'Vault is locked'")
	}
}

func TestLockedViewContentLogin(t *testing.T) {
	m := NewLockedModel(true)
	out := m.ViewContent(80, 24)
	if !strings.Contains(out, "Log in to Bitwarden") {
		t.Error("login view should contain 'Log in to Bitwarden'")
	}
}
