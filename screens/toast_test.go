package screens

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/session"
)

func TestCopyToastStaysUntilClipboardClear(t *testing.T) {
	m := newTestVault(testItems())
	m.sess.LastSync = time.Now()
	updated, cmd := m.Update(session.CopiedMsg{Field: session.CopyFieldPassword})
	m = updated.(VaultModel)
	if cmd != nil {
		t.Error("a copy toast must not start its own timer")
	}
	// The terminal does not confirm the write, so the toast must not say
	// that the copy worked.
	if _, status := m.FooterContent(); !strings.Contains(status, "Password sent to clipboard") || strings.Contains(status, "copied") {
		t.Errorf("status = %q, want a toast that says the password was sent", status)
	}
	updated, _ = m.Update(session.ClipboardClearedMsg{})
	m = updated.(VaultModel)
	if _, status := m.FooterContent(); !strings.Contains(status, "synced just now") {
		t.Errorf("status = %q, want the sync age after the clipboard clear", status)
	}
}

func TestClipboardClearKeepsNewerToast(t *testing.T) {
	m := newTestVault(testItems())
	updated, _ := m.Update(session.CopiedMsg{Field: session.CopyFieldPassword})
	m = updated.(VaultModel)
	updated, _ = m.Update(keyMsg("d"))
	m = updated.(VaultModel)
	updated, _ = m.Update(session.ClipboardClearedMsg{})
	m = updated.(VaultModel)
	if _, status := m.FooterContent(); !strings.Contains(status, "Drawer: overlay") {
		t.Errorf("status = %q: the clipboard clear removed a newer toast", status)
	}
}

func TestErrorToastExpires(t *testing.T) {
	orig := errorToastDuration
	t.Cleanup(func() { errorToastDuration = orig })
	errorToastDuration = time.Millisecond

	m := newTestVault(testItems())
	updated, cmd := m.Update(bwcmd.SyncResult{Err: errors.New("offline")})
	m = updated.(VaultModel)
	if !m.toastIsError || cmd == nil {
		t.Fatalf("toast = %q, cmd = %v, want an error toast and its timer", m.toast, cmd)
	}
	updated, _ = m.Update(cmd())
	m = updated.(VaultModel)
	if m.toast != "" || m.toastIsError {
		t.Errorf("toast = %q after its timer, want none", m.toast)
	}
}

func TestSyncShowsSyncingUntilResult(t *testing.T) {
	m := newTestVault(testItems())
	updated, _ := m.Update(session.CopiedMsg{Field: session.CopyFieldPassword})
	m = updated.(VaultModel)
	updated, _ = m.Update(keyMsg("r")) // the command would run bw sync
	m = updated.(VaultModel)
	if _, status := m.FooterContent(); !strings.Contains(status, "Syncing…") || strings.Contains(status, "clipboard") {
		t.Errorf("status = %q, want Syncing… in place of the old toast", status)
	}
}
