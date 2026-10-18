package session

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// ClipboardClearDelay is how long a copied value stays on the clipboard.
const ClipboardClearDelay = 60 * time.Second

// ClearPlaceholder is the value that a clipboard clear writes. tmux ignores
// an OSC 52 write with an empty payload, and other multiplexers can do the
// same, so an empty clear can leave the secret on the clipboard. One space
// overwrites it.
const ClearPlaceholder = " "

// CopyField enumerates what kind of value is being copied.
type CopyField int

const (
	CopyFieldPassword CopyField = iota
	CopyFieldTOTP
	CopyFieldUsername
	CopyFieldNote
	CopyFieldCardNumber
	CopyFieldCVV // the card security code
	CopyFieldSSN
	CopyFieldPrivateKey // the private key of an SSH key item
	CopyFieldPublicKey  // the public key of an SSH key item
	CopyFieldEmail      // the email of an identity item
)

// CopiedMsg is sent back to the model after a copy operation completes.
type CopiedMsg struct {
	Field CopyField
}

// ClipboardClearedMsg is sent when the auto-clear timer for copy number Gen
// fires. A timer whose Gen is older than the latest copy must be ignored,
// so that it does not clear a newer value early.
type ClipboardClearedMsg struct {
	Gen uint64
}

// CopyToClipboard returns a tea.Cmd that writes value to the system clipboard
// via OSC 52 and notifies the model with a CopiedMsg. The terminal sends no
// reply to OSC 52, so the CopiedMsg means only that the request was sent.
func CopyToClipboard(value string, field CopyField) tea.Cmd {
	return tea.Batch(
		tea.SetClipboard(value),
		func() tea.Msg { return CopiedMsg{Field: field} },
	)
}

// ScheduleClipboardClear returns a Cmd that fires ClipboardClearedMsg for
// copy number gen after ClipboardClearDelay.
func ScheduleClipboardClear(gen uint64) tea.Cmd {
	return tea.Tick(ClipboardClearDelay, func(time.Time) tea.Msg {
		return ClipboardClearedMsg{Gen: gen}
	})
}

// ClearClipboard returns a Cmd that overwrites the clipboard with
// ClearPlaceholder via OSC 52.
func ClearClipboard() tea.Cmd {
	return tea.SetClipboard(ClearPlaceholder)
}
