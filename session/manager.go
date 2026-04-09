package session

import "time"

// State holds all runtime session data. The zero value represents an
// unauthenticated, locked state.
type State struct {
	Token       string        // BW_SESSION token; empty when locked
	Email       string        // authenticated user email
	LastSync    time.Time     // zero when never synced
	LastActive  time.Time     // updated on every keypress
	IdleTimeout time.Duration // default 15min
}

// IsLocked reports whether the session has no valid token.
func (s *State) IsLocked() bool {
	return s.Token == ""
}

// Lock clears the token and any other sensitive in-memory state.
// Email is intentionally retained for the unlock screen display.
func (s *State) Lock() {
	s.Token = ""
	s.LastSync = time.Time{}
}

// SetToken stores the session token obtained from `bw unlock`.
func (s *State) SetToken(token string) {
	s.Token = token
}

// Touch updates the last-active timestamp.
func (s *State) Touch() {
	s.LastActive = time.Now()
}

// IsIdle reports whether the session has been idle longer than IdleTimeout.
func (s *State) IsIdle() bool {
	if s.IdleTimeout <= 0 || s.LastActive.IsZero() {
		return false
	}
	return time.Since(s.LastActive) > s.IdleTimeout
}
