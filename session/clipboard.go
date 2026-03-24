package session

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
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
	Err   error
}

// ClipboardClearedMsg is sent after the auto-clear timer fires.
type ClipboardClearedMsg struct{}

// CopyToClipboard returns a tea.Cmd that writes value to the system clipboard
// and schedules an auto-clear after 60 seconds.
func CopyToClipboard(value string, field CopyField) tea.Cmd {
	return func() tea.Msg {
		err := writeClipboard(value)
		return CopiedMsg{Field: field, Err: err}
	}
}

// ScheduleClipboardClear returns a Cmd that fires ClipboardClearedMsg after 60s.
func ScheduleClipboardClear() tea.Cmd {
	return tea.Tick(60*time.Second, func(time.Time) tea.Msg {
		return ClipboardClearedMsg{}
	})
}

// ClearClipboard overwrites the clipboard with an empty string.
func ClearClipboard() {
	_ = writeClipboard("")
}

func writeClipboard(value string) error {
	// Wayland-first.
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		cmd := exec.Command("wl-copy")
		cmd.Stdin = strings.NewReader(value)
		return cmd.Run()
	}
	// Fallback to atotto/clipboard (xclip/xsel/pbcopy).
	return clipboard.WriteAll(value)
}
