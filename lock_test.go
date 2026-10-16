package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/screens"
	"github.com/juthrbog/lazybw/session"
	"github.com/juthrbog/lazybw/ui"
)

func TestLateItemsResultAfterLockIsDropped(t *testing.T) {
	m := unlockedVault(t, testItems())
	staleEpoch := m.sess.Epoch

	m, _ = update(t, m, screens.LockMsg{})
	late := []bwcmd.Item{{ID: "9", Type: bwcmd.ItemTypeLogin, Name: "Late", Login: &bwcmd.Login{Password: "hunter2"}}}
	m, cmd := update(t, m, bwcmd.ItemsResult{Items: late, Epoch: staleEpoch})
	if m.state != stateQuitting {
		t.Fatalf("state = %d, want stateQuitting", m.state)
	}
	if cmd != nil {
		t.Error("a stale ItemsResult must not return a command")
	}
	if late[0].Login.Password != "" {
		t.Error("a dropped ItemsResult must be scrubbed")
	}

	m, _ = update(t, m, bwcmd.LockResult{})
	if m.state != stateLocked {
		t.Fatalf("state = %d, want stateLocked", m.state)
	}

	// A result from the old session also stays out of the next session.
	m, _ = update(t, m, screens.UnlockedMsg{Token: "tok2"})
	late2 := []bwcmd.Item{{ID: "9", Type: bwcmd.ItemTypeLogin, Name: "Late"}}
	m, _ = update(t, m, bwcmd.ItemsResult{Items: late2, Epoch: staleEpoch})
	if m.state != stateLoading {
		t.Errorf("state = %d, want stateLoading", m.state)
	}
}

func TestItemsResultOutsideLoadingAndVaultIsDropped(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.sess.SetToken("tok")
	m.state = stateError
	items := []bwcmd.Item{{ID: "1", Login: &bwcmd.Login{Password: "pw"}}}
	m, _ = update(t, m, bwcmd.ItemsResult{Items: items, Epoch: m.sess.Epoch})
	if m.state != stateError {
		t.Errorf("state = %d, want stateError", m.state)
	}
	if items[0].Login.Password != "" {
		t.Error("a dropped ItemsResult must be scrubbed")
	}
}

func TestSyncRefreshKeepsVaultState(t *testing.T) {
	items := testItems()
	m := unlockedVault(t, items)
	m, _ = update(t, m, keyPress("j"))
	if _, typ := m.vault.HeaderInfo(); typ != ui.ItemTypeName(bwcmd.ItemTypeCard) {
		t.Fatalf("selected type = %q, want card", typ)
	}

	m, _ = update(t, m, bwcmd.SyncResult{Epoch: m.sess.Epoch})
	m, _ = update(t, m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch})
	if m.state != stateVault {
		t.Fatalf("state = %d, want stateVault", m.state)
	}
	if _, typ := m.vault.HeaderInfo(); typ != ui.ItemTypeName(bwcmd.ItemTypeCard) {
		t.Errorf("after refresh, selected type = %q, want the same card", typ)
	}
	if items[0].Login.Password != "" {
		t.Error("the replaced items must be scrubbed")
	}
}

func TestStaleSyncResultIsIgnored(t *testing.T) {
	m := unlockedVault(t, testItems())
	_, cmd := update(t, m, bwcmd.SyncResult{Epoch: m.sess.Epoch - 1})
	if cmd != nil {
		t.Error("a SyncResult from an old session must not start a fetch")
	}
}

func TestCopyThenLockClearsClipboard(t *testing.T) {
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, session.CopiedMsg{Field: session.CopyFieldPassword})
	if !m.clipDirty {
		t.Fatal("clipDirty must be set after a copy")
	}
	m, cmd := update(t, m, screens.LockMsg{})
	if !hasMsg(collectMsgs(cmd), clipboardClear) {
		t.Error("lock must clear the clipboard")
	}
	if m.clipDirty {
		t.Error("clipDirty must be reset after the clear")
	}
}

func TestCopyThenIdleLockClearsClipboard(t *testing.T) {
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, session.CopiedMsg{Field: session.CopyFieldPassword})
	m.sess.LastActive = time.Now().Add(-time.Hour)
	m, cmd := update(t, m, idleCheckMsg{})
	if m.state != stateLocked {
		t.Fatalf("state = %d, want stateLocked", m.state)
	}
	if !hasMsg(collectMsgs(cmd), clipboardClear) {
		t.Error("idle lock must clear the clipboard")
	}
}

func TestCopyThenQuitClearsClipboardBeforeExit(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  tea.Msg
	}{
		{"QuitMsg", screens.QuitMsg{}},
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := unlockedVault(t, testItems())
			m, _ = update(t, m, session.CopiedMsg{Field: session.CopyFieldPassword})
			m, cmd := update(t, m, tc.msg)
			if m.state != stateQuitting {
				t.Fatalf("state = %d, want stateQuitting", m.state)
			}
			msgs := collectMsgs(cmd)
			if !hasMsg(msgs, clipboardClear) {
				t.Error("quit must clear the clipboard at once")
			}
			if !hasMsgType[bwcmd.LockResult](msgs) {
				t.Error("quit must run bw lock")
			}

			_, cmd = update(t, m, bwcmd.LockResult{})
			seq := cmd()
			if !isSequence(seq) {
				t.Fatalf("exit command = %T, want a tea.Sequence", seq)
			}
			cmds, _ := asCmds(seq)
			if len(cmds) != 2 || cmds[0]() != clipboardClear {
				t.Fatal("the sequence must clear the clipboard first")
			}
			if _, ok := cmds[1]().(tea.QuitMsg); !ok {
				t.Error("the sequence must end with tea.Quit")
			}
		})
	}
}

func TestQuitWithoutCopyLeavesClipboardAlone(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.state = stateLocked
	_, cmd := update(t, m, screens.QuitMsg{})
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("quit without a copy must return tea.Quit only")
	}
}

func TestOlderClipboardTimerDoesNotClearNewerCopy(t *testing.T) {
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, session.CopiedMsg{Field: session.CopyFieldPassword})
	first := m.clipGen
	m, _ = update(t, m, session.CopiedMsg{Field: session.CopyFieldTOTP})

	m, cmd := update(t, m, session.ClipboardClearedMsg{Gen: first})
	if hasMsg(collectMsgs(cmd), clipboardClear) {
		t.Error("the timer of an older copy must not clear the clipboard")
	}
	if !m.clipDirty {
		t.Error("clipDirty must stay set until the latest timer fires")
	}

	m, cmd = update(t, m, session.ClipboardClearedMsg{Gen: m.clipGen})
	if !hasMsg(collectMsgs(cmd), clipboardClear) {
		t.Error("the timer of the latest copy must clear the clipboard")
	}
	if m.clipDirty {
		t.Error("clipDirty must be reset")
	}
}

func TestClipboardTimerAfterLockDoesNothing(t *testing.T) {
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, session.CopiedMsg{Field: session.CopyFieldPassword})
	m, _ = update(t, m, screens.LockMsg{})
	_, cmd := update(t, m, session.ClipboardClearedMsg{Gen: m.clipGen})
	if cmd != nil {
		t.Error("a timer for an already cleared copy must not clear again")
	}
}

func TestKeyPressAfterIdleTimeoutLocks(t *testing.T) {
	m := unlockedVault(t, testItems())
	m.sess.LastActive = time.Now().Add(-time.Hour)
	m, _ = update(t, m, keyPress("j"))
	if m.state != stateLocked {
		t.Errorf("state = %d, want stateLocked", m.state)
	}
	if !m.sess.IsLocked() {
		t.Error("token must be cleared")
	}
}

func TestLockResultAfterIdleLockIsIgnored(t *testing.T) {
	m := unlockedVault(t, testItems())
	m.sess.LastActive = time.Now().Add(-time.Hour)
	m, _ = update(t, m, idleCheckMsg{})
	if m.state != stateLocked {
		t.Fatalf("state = %d, want stateLocked", m.state)
	}
	m, cmd := update(t, m, bwcmd.LockResult{})
	if m.state != stateLocked || cmd != nil {
		t.Error("a LockResult after an idle lock must not rebuild the unlock screen")
	}
}

// errorAfterUnlock returns a root model on the error screen after the item
// fetch failed. The session is still unlocked.
func errorAfterUnlock(t *testing.T) RootModel {
	t.Helper()
	m := NewRootModel(15 * time.Minute)
	m.width, m.height = 80, 24
	m, _ = update(t, m, screens.UnlockedMsg{Token: "tok"})
	m, _ = update(t, m, bwcmd.ItemsResult{Err: errors.New("timeout"), Epoch: m.sess.Epoch})
	if m.state != stateError {
		t.Fatalf("state = %d, want stateError", m.state)
	}
	return m
}

func TestErrorScreenQuitLocksVault(t *testing.T) {
	log := fakeBwLog(t)
	m := errorAfterUnlock(t)
	m, cmd := update(t, m, keyPress("q"))
	msgs := collectMsgs(cmd)
	if !hasMsgType[screens.QuitMsg](msgs) {
		t.Fatal("q on the error screen must send QuitMsg")
	}
	m, cmd = update(t, m, screens.QuitMsg{})
	if m.state != stateQuitting || !m.sess.IsLocked() {
		t.Fatal("QuitMsg on the error screen must lock the session")
	}
	collectMsgs(cmd)
	if !ranBwLock(log(), "tok") {
		t.Errorf("quit from the error screen must run bw lock, bw log:\n%s", log())
	}
}

// TestLockPathsRunBwLock runs the commands of each path that ends an
// unlocked session. Each one must run `bw lock` for the session token and
// return its LockResult.
func TestLockPathsRunBwLock(t *testing.T) {
	for _, tc := range []struct {
		name string
		lock func(t *testing.T, m RootModel) (RootModel, tea.Cmd)
	}{
		{"lock key", func(t *testing.T, m RootModel) (RootModel, tea.Cmd) {
			return update(t, m, screens.LockMsg{})
		}},
		{"idle lock", func(t *testing.T, m RootModel) (RootModel, tea.Cmd) {
			m.sess.LastActive = time.Now().Add(-time.Hour)
			return update(t, m, idleCheckMsg{})
		}},
		{"quit", func(t *testing.T, m RootModel) (RootModel, tea.Cmd) {
			return update(t, m, screens.QuitMsg{})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log := fakeBwLog(t)
			m, cmd := tc.lock(t, unlockedVault(t, testItems()))
			if !m.sess.IsLocked() {
				t.Error("the session must be locked")
			}
			if _, ok := waitForMsg[bwcmd.LockResult](cmd, 5*time.Second); !ok {
				t.Error("the commands must return the LockResult of bw lock")
			}
			if !ranBwLock(log(), "tok") {
				t.Errorf("bw lock must run with the session token, bw log:\n%s", log())
			}
		})
	}
}

func TestIdleLockOnErrorScreen(t *testing.T) {
	m := errorAfterUnlock(t)
	m.sess.LastActive = time.Now().Add(-time.Hour)
	m, _ = update(t, m, idleCheckMsg{})
	if m.state != stateLocked || !m.sess.IsLocked() {
		t.Error("idle lock must also run on the error screen")
	}
}

func TestRetryAfterFetchErrorFetchesAgain(t *testing.T) {
	m := errorAfterUnlock(t)
	m, cmd := update(t, m, screens.RetryMsg{})
	if m.state != stateLoading {
		t.Fatalf("state = %d, want stateLoading", m.state)
	}
	msgs := collectMsgs(cmd)
	if !hasMsgType[bwcmd.ItemsResult](msgs) {
		t.Error("retry with an unlocked session must fetch the items again")
	}
	if !hasMsgType[spinner.TickMsg](msgs) {
		t.Error("retry must restart the spinner")
	}
}

func TestRetryWhenLockedChecksStatus(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.state = stateError
	_, cmd := update(t, m, screens.RetryMsg{})
	msgs := collectMsgs(cmd)
	if !hasMsgType[bwcmd.StatusResult](msgs) {
		t.Error("retry with a locked session must run bw status")
	}
	if !hasMsgType[spinner.TickMsg](msgs) {
		t.Error("retry must restart the spinner")
	}
}

func TestCleanupAfterExit(t *testing.T) {
	log := fakeBwLog(t)
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, session.CopiedMsg{Field: session.CopyFieldPassword})

	// SIGTERM ends the program without a call to Update.
	var buf bytes.Buffer
	m.cleanupAfterExit(&buf)
	if buf.String() != ansi.SetSystemClipboard(session.ClearPlaceholder) {
		t.Errorf("cleanup wrote %q, want an OSC 52 clear", buf.String())
	}
	if !ranBwLock(log(), "tok") {
		t.Errorf("cleanup must run bw lock for an unlocked session, bw log:\n%s", log())
	}
}

func TestCleanupAfterNormalExitDoesNothing(t *testing.T) {
	log := fakeBwLog(t)
	m := NewRootModel(15 * time.Minute)
	var buf bytes.Buffer
	m.cleanupAfterExit(&buf)
	if buf.Len() != 0 || log() != "" {
		t.Error("cleanup after a normal exit must not write or run bw")
	}
}

func TestLateItemsResultAfterLockResultIsDropped(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  func(epoch uint64) bwcmd.ItemsResult
	}{
		{"items", func(e uint64) bwcmd.ItemsResult {
			return bwcmd.ItemsResult{Items: testItems(), Epoch: e}
		}},
		{"error", func(e uint64) bwcmd.ItemsResult {
			return bwcmd.ItemsResult{Err: errors.New("vault is locked"), Epoch: e}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := unlockedVault(t, testItems())
			syncEpoch := m.sess.Epoch
			m, _ = update(t, m, bwcmd.SyncResult{Epoch: syncEpoch})
			m, _ = update(t, m, screens.LockMsg{})
			m, _ = update(t, m, bwcmd.LockResult{})
			late := tc.msg(syncEpoch)
			m, _ = update(t, m, late)
			if m.state != stateLocked {
				t.Errorf("state = %d, want stateLocked", m.state)
			}
			for _, item := range late.Items {
				if item.Notes != "" || (item.Login != nil && item.Login.Password != "") {
					t.Errorf("late item %s must be zeroed", item.Name)
				}
			}
		})
	}
}

func TestRetryOutsideErrorScreenIsIgnored(t *testing.T) {
	m := errorAfterUnlock(t)
	// r and ctrl+c arrive in one read: ctrl+c is handled first.
	m, _ = update(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m, cmd := update(t, m, screens.RetryMsg{})
	if m.state != stateQuitting || cmd != nil {
		t.Errorf("state = %d, want stateQuitting and no command", m.state)
	}
	m, _ = update(t, m, bwcmd.LockResult{})
	if m.state != stateQuitting {
		t.Error("the quit must go on after a stale RetryMsg")
	}
}

func TestSecondFetchFailureAfterRetryLocks(t *testing.T) {
	log := fakeBwLog(t)
	m := errorAfterUnlock(t)
	m, _ = update(t, m, screens.RetryMsg{})
	m, cmd := update(t, m, bwcmd.ItemsResult{Err: errors.New("Vault is locked."), Epoch: m.sess.Epoch})
	if m.state != stateError {
		t.Fatalf("state = %d, want stateError", m.state)
	}
	if !m.sess.IsLocked() {
		t.Fatal("a second failure must end the session")
	}
	collectMsgs(cmd)
	if !ranBwLock(log(), "tok") {
		t.Errorf("a second failure must run bw lock, bw log:\n%s", log())
	}
	_, cmd = update(t, m, screens.RetryMsg{})
	if !hasMsgType[bwcmd.StatusResult](collectMsgs(cmd)) {
		t.Error("the next retry must run bw status, so the unlock screen shows")
	}
}

func TestCtrlCAfterIdleTimeoutQuits(t *testing.T) {
	m := unlockedVault(t, testItems())
	m.sess.LastActive = time.Now().Add(-time.Hour)
	m, _ = update(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if m.state != stateQuitting || m.lockFor != intentQuit {
		t.Errorf("state = %d, want stateQuitting with intentQuit", m.state)
	}
	if !m.sess.IsLocked() {
		t.Error("quit must lock the session")
	}
}

func TestIdleLockWhenVaultOpenWithoutToken(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width, m.height = 80, 24
	m, _ = update(t, m, bwcmd.StatusResult{Status: bwcmd.VaultStatus{Status: "unlocked"}})
	items := testItems()
	m, _ = update(t, m, bwcmd.ItemsResult{Items: items, Epoch: m.sess.Epoch})
	if m.state != stateVault {
		t.Skipf("state = %d: the unlocked status does not open the vault", m.state)
	}
	m.sess.LastActive = time.Now().Add(-time.Hour)
	m, _ = update(t, m, idleCheckMsg{})
	if m.state != stateLocked || items[0].Login.Password != "" {
		t.Error("an idle vault must lock, even without a token")
	}
}

func TestCleanupRunsPendingLock(t *testing.T) {
	log := fakeBwLog(t)
	m := unlockedVault(t, testItems())
	// The quit started bw lock, but the program stopped before it returned.
	m, _ = update(t, m, screens.QuitMsg{})
	if m.pendingLock == "" {
		t.Fatal("quit must remember the pending lock")
	}
	m.cleanupAfterExit(&bytes.Buffer{})
	if !ranBwLock(log(), "tok") {
		t.Errorf("cleanup must run bw lock again for a pending lock, bw log:\n%s", log())
	}
	m, _ = update(t, m, bwcmd.LockResult{})
	if m.pendingLock != "" {
		t.Error("LockResult must forget the pending lock")
	}
}

func TestLockMsgWithoutTokenClearsVault(t *testing.T) {
	items := testItems()
	m := unlockedVault(t, items)
	m.sess.Lock() // the session ended, but the vault still shows
	m, _ = update(t, m, screens.LockMsg{})
	if m.state != stateLocked {
		t.Errorf("state = %d, want stateLocked", m.state)
	}
	if items[0].Login.Password != "" {
		t.Error("LockMsg must clear the vault items")
	}
}

func TestClipboardTimerClearsWhileQuitting(t *testing.T) {
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, session.CopiedMsg{Field: session.CopyFieldPassword})
	m, _ = update(t, m, screens.QuitMsg{})
	if m.state != stateQuitting {
		t.Fatalf("state = %d, want stateQuitting", m.state)
	}
	_, cmd := update(t, m, session.ClipboardClearedMsg{Gen: m.clipGen})
	if !hasMsg(collectMsgs(cmd), clipboardClear) {
		t.Error("the clear timer must still clear the clipboard while quitting")
	}
}

func TestLockResultDoesNotStartAnotherIdleCheck(t *testing.T) {
	orig := idleCheckInterval
	t.Cleanup(func() { idleCheckInterval = orig })
	idleCheckInterval = time.Millisecond

	m := unlockedVault(t, testItems())
	m, _ = update(t, m, screens.LockMsg{})
	_, cmd := update(t, m, bwcmd.LockResult{})
	if hasMsgType[idleCheckMsg](collectMsgsWithin(cmd, 100*time.Millisecond)) {
		t.Error("LockResult must not start a second idle-check chain")
	}
}

func TestLockResultAfterIdleLockKeepsTypedText(t *testing.T) {
	m := unlockedVault(t, testItems())
	m.sess.LastActive = time.Now().Add(-time.Hour)
	m, _ = update(t, m, idleCheckMsg{})
	fresh := m.locked.ViewContent(80, 20)
	for _, k := range "abc" {
		m, _ = update(t, m, keyPress(string(k)))
	}
	typed := m.locked.ViewContent(80, 20)
	if typed == fresh {
		t.Fatal("typing must change the unlock form")
	}
	m, _ = update(t, m, bwcmd.LockResult{})
	if got := m.locked.ViewContent(80, 20); got != typed {
		t.Error("a LockResult after an idle lock must keep the typed text")
	}
}

func TestSyncFetchErrorKeepsVaultOpen(t *testing.T) {
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, bwcmd.ItemsResult{Err: errors.New("timeout"), Epoch: m.sess.Epoch})
	if m.state != stateVault {
		t.Fatalf("state = %d, want stateVault", m.state)
	}
	if _, status := m.vault.FooterContent(); !strings.Contains(status, "Load failed") {
		t.Errorf("status = %q, want a Load failed toast", status)
	}
}

func TestSyncRefreshStartsNoTOTPTicker(t *testing.T) {
	m := unlockedVault(t, testItems())
	_, cmd := update(t, m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch})
	// The TOTP tick fires after one second. Wait a bit longer.
	if hasMsgType[screens.TOTPTickMsg](collectMsgsWithin(cmd, 1500*time.Millisecond)) {
		t.Error("a refresh must not start another TOTP tick chain")
	}
}
