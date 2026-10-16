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

	// Epoch changes every time the session is unlocked or locked. A bw
	// command records the epoch it started in, so its result can be
	// dropped when it arrives after the session it belongs to has ended.
	Epoch uint64
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
	s.Epoch++
}

// SetToken stores the session token obtained from `bw unlock`.
func (s *State) SetToken(token string) {
	s.Token = token
	s.Epoch++
}

// Touch updates the last-active timestamp.
func (s *State) Touch() {
	s.LastActive = time.Now()
}

// wallNow returns the current wall-clock time without a monotonic reading.
// Tests replace it to simulate a system suspend.
var wallNow = func() time.Time { return time.Now().Round(0) }

// IsIdle reports whether the session has been idle longer than IdleTimeout.
//
// On Linux and macOS the monotonic clock stops while the system is
// suspended, so time.Since alone ignores time spent asleep. The check also
// compares wall-clock time (Round(0) strips the monotonic reading), so a
// vault left unlocked before a suspend counts as idle after resume.
func (s *State) IsIdle() bool {
	if s.IdleTimeout <= 0 || s.LastActive.IsZero() {
		return false
	}
	mono := time.Since(s.LastActive)
	wall := wallNow().Sub(s.LastActive.Round(0))
	return mono > s.IdleTimeout || wall > s.IdleTimeout
}
