package main

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/screens"
)

func TestMain(m *testing.M) {
	os.Setenv("NO_COLOR", "1")
	os.Exit(m.Run())
}

func TestNewRootModel(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	if m.state != stateLoading {
		t.Errorf("initial state = %d, want stateLoading (%d)", m.state, stateLoading)
	}
	if m.sess == nil {
		t.Fatal("session should not be nil")
	}
	if m.sess.IdleTimeout != 15*time.Minute {
		t.Errorf("IdleTimeout = %v, want 15m", m.sess.IdleTimeout)
	}
}

func TestRootStatusResultLocked(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	msg := bwcmd.StatusResult{
		Status: bwcmd.VaultStatus{Status: "locked", UserEmail: "u@t.com"},
	}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateLocked {
		t.Errorf("state = %d, want stateLocked (%d)", m.state, stateLocked)
	}
	if m.sess.Email != "u@t.com" {
		t.Errorf("email = %q, want %q", m.sess.Email, "u@t.com")
	}
}

func TestRootStatusResultUnauthenticated(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	msg := bwcmd.StatusResult{
		Status: bwcmd.VaultStatus{Status: "unauthenticated"},
	}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateLogin {
		t.Errorf("state = %d, want stateLogin (%d)", m.state, stateLogin)
	}
}

func TestRootStatusResultError(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	msg := bwcmd.StatusResult{Err: errors.New("bw not found")}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateError {
		t.Errorf("state = %d, want stateError (%d)", m.state, stateError)
	}
}

func TestRootUnlockedMsg(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	msg := screens.UnlockedMsg{Token: "test-token", Email: "u@t.com"}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateLoading {
		t.Errorf("state = %d, want stateLoading (%d)", m.state, stateLoading)
	}
	if m.sess.Token != "test-token" {
		t.Errorf("token = %q, want %q", m.sess.Token, "test-token")
	}
}

func TestRootItemsResult(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	items := []bwcmd.Item{
		{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Gmail"},
	}
	msg := bwcmd.ItemsResult{Items: items}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateVault {
		t.Errorf("state = %d, want stateVault (%d)", m.state, stateVault)
	}
}

func TestRootItemsResultError(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width = 80
	m.height = 24

	msg := bwcmd.ItemsResult{Err: errors.New("parse error")}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateError {
		t.Errorf("state = %d, want stateError (%d)", m.state, stateError)
	}
}

func TestRootLockMsg(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.sess.SetToken("test-token")

	msg := screens.LockMsg{}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateQuitting {
		t.Errorf("state = %d, want stateQuitting (%d)", m.state, stateQuitting)
	}
	if m.lockFor != intentLock {
		t.Errorf("lockFor = %d, want intentLock (%d)", m.lockFor, intentLock)
	}
}

func TestRootQuitMsg(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.sess.SetToken("test-token")

	msg := screens.QuitMsg{}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateQuitting {
		t.Errorf("state = %d, want stateQuitting (%d)", m.state, stateQuitting)
	}
	if m.lockFor != intentQuit {
		t.Errorf("lockFor = %d, want intentQuit (%d)", m.lockFor, intentQuit)
	}
}

func TestRootIdleLock(t *testing.T) {
	m := NewRootModel(1 * time.Millisecond)
	m.state = stateVault
	m.sess.SetToken("test-token")
	m.sess.LastActive = time.Now().Add(-time.Second)
	m.width = 80
	m.height = 24

	updated, _ := m.Update(idleCheckMsg{})
	m = updated.(RootModel)

	if m.state != stateLocked {
		t.Errorf("state = %d, want stateLocked (%d)", m.state, stateLocked)
	}
	if m.sess.Token != "" {
		t.Error("token should be cleared after idle lock")
	}
	if m.lockFor != intentLock {
		t.Errorf("lockFor = %d, want intentLock (%d)", m.lockFor, intentLock)
	}
}

func TestRootRetryMsg(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.state = stateError

	msg := screens.RetryMsg{}
	updated, _ := m.Update(msg)
	m = updated.(RootModel)

	if m.state != stateLoading {
		t.Errorf("state = %d, want stateLoading (%d)", m.state, stateLoading)
	}
}
