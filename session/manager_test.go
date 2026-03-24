package session

import (
	"testing"
	"time"
)

func TestStateZeroValueIsLocked(t *testing.T) {
	s := State{}
	if !s.IsLocked() {
		t.Error("zero value State should be locked")
	}
}

func TestSetToken(t *testing.T) {
	s := State{}
	s.SetToken("test-token")
	if s.IsLocked() {
		t.Error("should not be locked after SetToken")
	}
	if s.Token != "test-token" {
		t.Errorf("Token = %q, want %q", s.Token, "test-token")
	}
}

func TestLock(t *testing.T) {
	s := State{Token: "test-token"}
	s.Lock()
	if !s.IsLocked() {
		t.Error("should be locked after Lock()")
	}
	if s.Token != "" {
		t.Errorf("Token = %q, want empty", s.Token)
	}
}

func TestTouch(t *testing.T) {
	s := State{}
	if !s.LastActive.IsZero() {
		t.Error("LastActive should be zero initially")
	}
	s.Touch()
	if s.LastActive.IsZero() {
		t.Error("LastActive should be non-zero after Touch()")
	}
	if time.Since(s.LastActive) > time.Second {
		t.Error("LastActive should be recent")
	}
}

func TestIsIdleNoTimeout(t *testing.T) {
	s := State{LastActive: time.Now().Add(-time.Hour)}
	if s.IsIdle() {
		t.Error("should not be idle when IdleTimeout is zero")
	}
}

func TestIsIdleBeforeTimeout(t *testing.T) {
	s := State{
		LastActive:  time.Now(),
		IdleTimeout: 15 * time.Minute,
	}
	if s.IsIdle() {
		t.Error("should not be idle when just touched")
	}
}

func TestIsIdleAfterTimeout(t *testing.T) {
	s := State{
		LastActive:  time.Now().Add(-20 * time.Minute),
		IdleTimeout: 15 * time.Minute,
	}
	if !s.IsIdle() {
		t.Error("should be idle after timeout exceeded")
	}
}

func TestIsIdleZeroLastActive(t *testing.T) {
	s := State{IdleTimeout: 15 * time.Minute}
	if s.IsIdle() {
		t.Error("should not be idle when LastActive is zero")
	}
}
