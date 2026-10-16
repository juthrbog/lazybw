package main

import (
	"reflect"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/screens"
)

func TestUnlockDropsUnlockScreen(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width, m.height = 80, 24
	m, _ = update(t, m, bwcmd.StatusResult{Status: bwcmd.VaultStatus{Status: "locked"}})
	m, _ = update(t, m, screens.UnlockedMsg{Token: "tok"})
	if !reflect.ValueOf(m.locked).IsZero() {
		t.Error("after unlock, the root must not keep the unlock screen and its form")
	}
	m, _ = update(t, m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch})
	if m.state != stateVault {
		t.Fatalf("state = %d, want stateVault", m.state)
	}
}

func TestEveryLockPathDropsVault(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*testing.T, RootModel) (RootModel, tea.Cmd)
	}{
		{"LockMsg", func(t *testing.T, m RootModel) (RootModel, tea.Cmd) { return update(t, m, screens.LockMsg{}) }},
		{"QuitMsg", func(t *testing.T, m RootModel) (RootModel, tea.Cmd) { return update(t, m, screens.QuitMsg{}) }},
		{"ctrl+c", func(t *testing.T, m RootModel) (RootModel, tea.Cmd) {
			return update(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		}},
		{"idle", func(t *testing.T, m RootModel) (RootModel, tea.Cmd) {
			m.sess.LastActive = time.Now().Add(-time.Hour)
			return update(t, m, idleCheckMsg{})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := testItems()
			m := unlockedVault(t, items)
			m, _ = tc.run(t, m)
			if !reflect.ValueOf(m.vault).IsZero() {
				t.Error("the root must drop the vault model")
			}
			if items[0].Login.Password != "" || items[2].Notes != "" {
				t.Error("the vault items must be scrubbed")
			}
		})
	}
}
