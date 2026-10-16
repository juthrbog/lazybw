package session

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// ClipboardClearDelay is how long a copied value stays on the clipboard.
const ClipboardClearDelay = 60 * time.Second

// CopyField enumerates what kind of value is being copied.
type CopyField int

const (
	CopyFieldPassword CopyField = iota
	CopyFieldTOTP
	CopyFieldUsername
	CopyFieldNote
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
// via OSC 52 and notifies the model with a CopiedMsg.
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

// ClearClipboard returns a Cmd that clears the clipboard via OSC 52.
func ClearClipboard() tea.Cmd {
	return tea.SetClipboard("")
}
