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
	s := State{
		Token:    "test-token",
		Email:    "user@example.com",
		LastSync: time.Now(),
	}
	s.Lock()
	if !s.IsLocked() {
		t.Error("should be locked after Lock()")
	}
	if s.Token != "" {
		t.Errorf("Token = %q, want empty", s.Token)
	}
	if !s.LastSync.IsZero() {
		t.Error("LastSync should be zero after Lock()")
	}
	// Email is intentionally preserved for the unlock screen.
	if s.Email != "user@example.com" {
		t.Errorf("Email = %q, want preserved", s.Email)
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

func TestEpochChangesOnUnlockAndLock(t *testing.T) {
	s := State{}
	e0 := s.Epoch
	s.SetToken("tok")
	e1 := s.Epoch
	s.Lock()
	e2 := s.Epoch
	if e0 == e1 || e1 == e2 || e0 == e2 {
		t.Errorf("epochs %d, %d, %d must all differ", e0, e1, e2)
	}
}

func TestIsIdleAfterSuspend(t *testing.T) {
	// During a suspend the monotonic clock stops, but the wall clock moves
	// on. Simulate 20 minutes of sleep right after the last key press.
	orig := wallNow
	t.Cleanup(func() { wallNow = orig })
	wallNow = func() time.Time { return time.Now().Round(0).Add(20 * time.Minute) }

	s := State{LastActive: time.Now(), IdleTimeout: 15 * time.Minute}
	if time.Since(s.LastActive) > s.IdleTimeout {
		t.Fatal("the monotonic reading must still be under the timeout")
	}
	if !s.IsIdle() {
		t.Error("should be idle when the wall clock shows the timeout passed")
	}
}

func TestIsIdleBothClocksUnderTimeout(t *testing.T) {
	orig := wallNow
	t.Cleanup(func() { wallNow = orig })
	wallNow = func() time.Time { return time.Now().Round(0).Add(time.Minute) }

	s := State{LastActive: time.Now(), IdleTimeout: 15 * time.Minute}
	if s.IsIdle() {
		t.Error("should not be idle when both clocks are under the timeout")
	}
}
