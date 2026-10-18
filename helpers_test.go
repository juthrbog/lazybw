package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/screens"
)

// fakeBwScript stands in for the Bitwarden CLI in tests. It appends its
// arguments to $FAKE_BW_LOG when that variable is set, and prints canned
// output for the commands the root model runs.
const fakeBwScript = `#!/bin/sh
if [ -n "$FAKE_BW_LOG" ]; then echo "$*" >> "$FAKE_BW_LOG"; fi
case "$1" in
status) echo '{"status":"locked","userEmail":"u@t.com"}' ;;
list) echo '[{"id":"1","type":1,"name":"Gmail","login":{"username":"u","password":"p"}}]' ;;
lock) echo 'Your vault is locked.' ;;
sync) echo 'Syncing complete.' ;;
esac
`

// installFakeBw writes the fake bw to a temporary directory and makes that
// directory the only PATH entry, so that no test can run the real bw.
func installFakeBw() (string, error) {
	dir, err := os.MkdirTemp("", "lazybw-fakebw-")
	if err != nil {
		return "", err
	}
	//nolint:gosec // the fake bw must be executable
	if err := os.WriteFile(filepath.Join(dir, "bw"), []byte(fakeBwScript), 0o700); err != nil {
		return "", err
	}
	return dir, os.Setenv("PATH", dir)
}

// update sends msg to m and returns the new root model and command.
func update(t *testing.T, m RootModel, msg tea.Msg) (RootModel, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	rm, ok := updated.(RootModel)
	if !ok {
		t.Fatalf("Update returned %T, want RootModel", updated)
	}
	return rm, cmd
}

// collectMsgs runs cmd and returns the messages it produces. It expands
// batches and sequences in order. A command that does not return within
// a short time (a timer) is skipped.
func collectMsgs(cmd tea.Cmd) []tea.Msg {
	return collectMsgsWithin(cmd, 500*time.Millisecond)
}

// collectMsgsWithin is collectMsgs with a given wait for each command.
func collectMsgsWithin(cmd tea.Cmd, wait time.Duration) []tea.Msg {
	var out []tea.Msg
	var run func(tea.Cmd)
	run = func(c tea.Cmd) {
		if c == nil {
			return
		}
		ch := make(chan tea.Msg, 1)
		go func() { ch <- c() }()
		select {
		case msg := <-ch:
			if cmds, ok := asCmds(msg); ok {
				for _, sub := range cmds {
					run(sub)
				}
				return
			}
			if msg != nil {
				out = append(out, msg)
			}
		case <-time.After(wait):
		}
	}
	run(cmd)
	return out
}

var cmdType = reflect.TypeOf((tea.Cmd)(nil))

// asCmds returns the commands inside a tea.BatchMsg or the unexported
// sequence message that tea.Sequence produces.
func asCmds(msg tea.Msg) ([]tea.Cmd, bool) {
	if msg == nil {
		return nil, false
	}
	v := reflect.ValueOf(msg)
	if v.Kind() != reflect.Slice || v.Type().Elem() != cmdType {
		return nil, false
	}
	cmds := make([]tea.Cmd, v.Len())
	for i := range cmds {
		cmds[i], _ = v.Index(i).Interface().(tea.Cmd)
	}
	return cmds, true
}

// isSequence reports whether msg comes from tea.Sequence.
func isSequence(msg tea.Msg) bool {
	return msg != nil && reflect.TypeOf(msg).Name() == "sequenceMsg"
}

// clipboardClear is the message that session.ClearClipboard produces.
var clipboardClear = tea.SetClipboard("")()

func hasMsg(msgs []tea.Msg, want tea.Msg) bool {
	for _, m := range msgs {
		if m == want {
			return true
		}
	}
	return false
}

func hasMsgType[T any](msgs []tea.Msg) bool {
	for _, m := range msgs {
		if _, ok := m.(T); ok {
			return true
		}
	}
	return false
}

func testItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Gmail", Login: &bwcmd.Login{Username: "user@gmail.com", Password: "pw1"}},
		{ID: "2", Type: bwcmd.ItemTypeCard, Name: "Visa", Card: &bwcmd.Card{Number: "4242424242424242"}},
		{ID: "3", Type: bwcmd.ItemTypeSecureNote, Name: "Keys", Notes: "secret"},
	}
}

// unlockedVault returns a root model that shows the vault with items.
func unlockedVault(t *testing.T, items []bwcmd.Item) RootModel {
	t.Helper()
	m := NewRootModel(15 * time.Minute)
	m.width, m.height = 80, 24
	m, _ = update(t, m, screens.UnlockedMsg{Token: "tok", Email: "u@t.com"})
	m, _ = update(t, m, bwcmd.ItemsResult{Items: items, Epoch: m.sess.Epoch})
	if m.state != stateVault {
		t.Fatalf("state = %d, want stateVault", m.state)
	}
	return m
}

// fakeBwLog points the fake bw at a fresh log file and returns a function
// that reads the log.
func fakeBwLog(t *testing.T) func() string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bw.log")
	t.Setenv("FAKE_BW_LOG", path)
	return func() string {
		b, _ := os.ReadFile(path) //nolint:gosec // test-owned temp file
		return string(b)
	}
}

func keyPress(s string) tea.KeyPressMsg {
	r := []rune(s)
	return tea.KeyPressMsg{Code: r[0], Text: s}
}
