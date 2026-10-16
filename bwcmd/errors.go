package bwcmd

import (
	"errors"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Known bw failures. A failed bw command returns an *Error, and errors.Is
// matches it against these errors by the message that bw printed.
var (
	// ErrCodeRequired means that bw needs a two-step login code. bw also
	// prints this message when it needs a new-device code.
	ErrCodeRequired = errors.New("bw needs a two-step login code")
	// ErrMethodRequired means that the account has more than one two-step
	// login method, and bw needs to know which one to use.
	ErrMethodRequired = errors.New("bw needs a two-step login method")
	// ErrInvalidPassword means that bw did not accept the master password,
	// or the email and master password.
	ErrInvalidPassword = errors.New("bw did not accept the master password")
	// ErrNotLoggedIn means that bw has no logged-in account.
	ErrNotLoggedIn = errors.New("bw is not logged in")
)

// knownMessages maps the start of a bw message to a known failure. The
// texts come from bw 2026.2.0.
var knownMessages = []struct {
	prefix string
	err    error
}{
	{"Code is required.", ErrCodeRequired},
	{"Login failed. No provider selected.", ErrMethodRequired},
	// bw login, and bw unlock in older bw versions.
	{"Invalid master password.", ErrInvalidPassword},
	{"Username or password is incorrect.", ErrInvalidPassword},
	// bw unlock, when the master password cannot decrypt the user key.
	{"The provided key is not the expected type", ErrInvalidPassword},
	{"You are not logged in.", ErrNotLoggedIn},
}

// Error is a bw command that failed. Error returns Msg, the last line that
// bw wrote to stderr, which holds the message for the user.
type Error struct {
	Msg    string // last non-empty stderr line, without ANSI sequences
	Stderr string // all of stderr, for the debug log
	Err    error  // the error from os/exec
}

func (e *Error) Error() string { return e.Msg }

func (e *Error) Unwrap() error { return e.Err }

// Is reports whether the bw message belongs to the known failure target.
func (e *Error) Is(target error) bool {
	for _, k := range knownMessages {
		if k.err == target && strings.HasPrefix(e.Msg, k.prefix) {
			return true
		}
	}
	return false
}

// newError builds the *Error for a bw command that failed with err and
// wrote stderr.
func newError(err error, stderr []byte) *Error {
	msg := lastLine(string(stderr))
	if msg == "" {
		msg = err.Error()
	}
	return &Error{Msg: msg, Stderr: string(stderr), Err: err}
}

// lastLine returns the last non-empty line of s without ANSI sequences.
// bw writes stack traces and warnings before its message. Both "\n" and
// "\r" end a line.
func lastLine(s string) string {
	lines := strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' })
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(ansi.Strip(lines[i])); line != "" {
			return line
		}
	}
	return ""
}
