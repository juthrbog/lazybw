package session

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// CopyField enumerates what kind of value is being copied.
type CopyField int

const (
	CopyFieldPassword CopyField = iota
	CopyFieldTOTP
	CopyFieldUsername
)

// CopiedMsg is sent back to the model after a copy operation completes.
type CopiedMsg struct {
	Field CopyField
}

// ClipboardClearedMsg is sent after the auto-clear timer fires.
type ClipboardClearedMsg struct{}

// CopyToClipboard returns a tea.Cmd that writes value to the system clipboard
// via OSC 52 and notifies the model with a CopiedMsg.
func CopyToClipboard(value string, field CopyField) tea.Cmd {
	return tea.Batch(
		tea.SetClipboard(value),
		func() tea.Msg { return CopiedMsg{Field: field} },
	)
}

// ScheduleClipboardClear returns a Cmd that fires ClipboardClearedMsg after 60s.
func ScheduleClipboardClear() tea.Cmd {
	return tea.Tick(60*time.Second, func(time.Time) tea.Msg {
		return ClipboardClearedMsg{}
	})
}

// ClearClipboard returns a Cmd that clears the clipboard via OSC 52.
func ClearClipboard() tea.Cmd {
	return tea.SetClipboard("")
}
