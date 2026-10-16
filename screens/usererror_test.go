package screens

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/juthrbog/lazybw/bwcmd"
)

func TestUserError(t *testing.T) {
	unlockWrong := &bwcmd.Error{Msg: "The provided key is not the expected type"}
	loginWrong := &bwcmd.Error{Msg: "Invalid master password. Confirm your email is correct and your account was created on vault.example."}
	timeout := fmt.Errorf("bw unlock timed out after 2m0s: %w", context.DeadlineExceeded)
	notLoggedIn := &bwcmd.Error{Msg: "You are not logged in."}
	unknown := &bwcmd.Error{Msg: "Vault is locked."}

	tests := []struct {
		name   string
		err    error
		action errAction
		want   string
	}{
		{"wrong password on unlock", unlockWrong, actionUnlock, unlockFailedText},
		{"wrong password on login", loginWrong, actionLogin, loginFailedText},
		{"wrong email on login", &bwcmd.Error{Msg: "Username or password is incorrect. Try again."}, actionLogin, loginFailedText},
		{"wrong password on re-prompt", unlockWrong, actionReprompt, wrongPasswordText},
		{"timeout on re-prompt", timeout, actionReprompt, timeoutText},
		{"unknown bw error on re-prompt", unknown, actionReprompt, "Vault is locked."},
		{"password error elsewhere", unlockWrong, actionOther, "The provided key is not the expected type"},
		{"timeout on unlock", timeout, actionUnlock, timeoutText},
		{"timeout on login", timeout, actionLogin, timeoutText},
		{"timeout elsewhere", timeout, actionOther, timeoutText},
		{"not logged in", notLoggedIn, actionOther, notLoggedInText},
		{"unknown bw error", unknown, actionUnlock, "Vault is locked."},
		{"other error", errors.New("parse bw status output: got null"), actionOther, "parse bw status output: got null"},
		{"no error", nil, actionUnlock, ""},
	}
	for _, tt := range tests {
		if got := userError(tt.err, tt.action); got != tt.want {
			t.Errorf("%s: userError = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLoginFailedTextNamesBothFields(t *testing.T) {
	// The text must not say which of the two was wrong, so it names both.
	if !strings.Contains(loginFailedText, "email") || !strings.Contains(loginFailedText, "master password") {
		t.Errorf("loginFailedText = %q, want both the email and the master password", loginFailedText)
	}
}

func TestErrorScreenShowsTimeoutText(t *testing.T) {
	err := fmt.Errorf("bw list items timed out after 2m0s: %w", context.DeadlineExceeded)
	if got := NewErrorModel(err, false).ViewContent(80); !strings.Contains(got, timeoutText) {
		t.Errorf("error screen = %q, want the timeout text", got)
	}
}

func TestLockErrorText(t *testing.T) {
	timeout := fmt.Errorf("bw lock timed out after 15s: %w", context.DeadlineExceeded)
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"no error", nil, ""},
		{"not logged in", &bwcmd.Error{Msg: "You are not logged in."}, ""},
		{"timeout", timeout, fmt.Sprintf(lockFailedText, lockTimeoutReason)},
		{"bw message", &bwcmd.Error{Msg: "Something went wrong. "}, fmt.Sprintf(lockFailedText, "Something went wrong")},
		{"other error", errors.New("exit status 1"), fmt.Sprintf(lockFailedText, "exit status 1")},
	}
	for _, tt := range tests {
		if got := LockErrorText(tt.err); got != tt.want {
			t.Errorf("%s: LockErrorText = %q, want %q", tt.name, got, tt.want)
		}
	}
	if got := LockErrorText(timeout); !strings.Contains(got, `Run "bw lock" in a terminal.`) {
		t.Errorf("LockErrorText = %q, want the step that locks bw", got)
	}
}
