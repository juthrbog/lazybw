package screens

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/session"
	"github.com/juthrbog/lazybw/ui"
)

// How long a toast shows. A copy toast stays until the clipboard clear.
// Tests shorten them.
var (
	toastDuration      = 4 * time.Second
	errorToastDuration = 8 * time.Second
)

// toastExpiredMsg ends toast number gen. Each toast gets a higher number,
// so the timer of an older toast does not end a newer one.
type toastExpiredMsg struct{ gen uint64 }

// setToast shows msg in the footer. It returns the command that ends the
// toast after toastDuration.
func (m *VaultModel) setToast(msg string) tea.Cmd {
	m.showToast(msg, false, false)
	return m.expireToast(toastDuration)
}

// setErrorToast shows an error in the footer. The message often holds bw
// output, so it is sanitized to one line. It returns the command that ends
// the toast after errorToastDuration.
func (m *VaultModel) setErrorToast(msg string) tea.Cmd {
	m.showToast(ui.GlyphError+" "+ui.SanitizeLine(msg), true, false)
	return m.expireToast(errorToastDuration)
}

// setCopyToast shows a copy toast. The root model clears the clipboard
// after session.ClipboardClearDelay, and the toast ends then.
func (m *VaultModel) setCopyToast(msg string) {
	m.showToast(msg, false, true)
}

// clearToast removes the toast, so that the footer shows the sync age.
func (m *VaultModel) clearToast() {
	m.showToast("", false, false)
}

func (m *VaultModel) showToast(msg string, isError, isCopy bool) {
	m.toastGen++
	m.toast = msg
	m.toastIsError = isError
	m.toastIsCopy = isCopy
}

// expireToast returns the command that ends the current toast after d.
func (m *VaultModel) expireToast(d time.Duration) tea.Cmd {
	gen := m.toastGen
	return tea.Tick(d, func(time.Time) tea.Msg { return toastExpiredMsg{gen: gen} })
}

// footerStatus returns the status text on the right of the footer: the
// toast, the sync spinner, or the time since the last sync.
func (m VaultModel) footerStatus() string {
	toast := m.toast
	if m.syncing {
		if toast == "" {
			toast = "Syncing…"
		}
		toast = m.syncSpinner.View() + " " + toast
	}
	switch {
	case toast == "":
		return ui.StyleFaint.Render(formatLastSync(m.sess.LastSync))
	case m.toastIsError:
		return ui.StyleError.Render(toast)
	}
	return ui.StyleToast.Render(toast)
}

func formatLastSync(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "synced just now"
	case d < time.Hour:
		return fmt.Sprintf("synced %dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("synced %dh ago", int(d.Hours()))
	}
}

// copyLabel returns the name of the field f for the copy toast.
func copyLabel(f session.CopyField) string {
	switch f {
	case session.CopyFieldTOTP:
		return "TOTP"
	case session.CopyFieldUsername:
		return "Username"
	case session.CopyFieldNote:
		return "Note"
	case session.CopyFieldCardNumber:
		return "Card number"
	case session.CopyFieldCVV:
		return "Security code"
	case session.CopyFieldSSN:
		return "SSN"
	case session.CopyFieldPrivateKey:
		return "SSH private key"
	case session.CopyFieldPublicKey:
		return "Public key"
	case session.CopyFieldEmail:
		return "Email"
	}
	return "Password"
}
