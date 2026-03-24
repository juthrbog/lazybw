package ui

import "github.com/charmbracelet/lipgloss"

// StatusBarProps carries everything the status bar renderer needs.
type StatusBarProps struct {
	Email    string
	LastSync string // e.g. "synced 2m ago"; empty → "never synced"
	Toast    string // transient message; empty → hidden
	Width    int
}

// RenderStatusBar returns a single-line status bar string padded to Width columns.
func RenderStatusBar(props StatusBarProps) string {
	sync := props.LastSync
	if sync == "" {
		sync = "never synced"
	}

	left := props.Email
	if left == "" {
		left = "not logged in"
	}

	var right string
	if props.Toast != "" {
		right = StyleToast.Render(props.Toast)
	} else {
		right = StyleFaint.Render(sync)
	}

	// Calculate padding between left and right sections.
	// lipgloss.Width strips ANSI codes for accurate measurement.
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := props.Width - leftWidth - rightWidth - 2 // 2 for padding(0,1)
	if gap < 1 {
		gap = 1
	}

	content := left + spacer(gap) + right
	return StyleStatusBar.Width(props.Width).Render(content)
}

func spacer(n int) string {
	s := make([]byte, n)
	for i := range s {
		s[i] = ' '
	}
	return string(s)
}
