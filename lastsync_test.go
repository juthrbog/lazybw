package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/screens"
)

// footerLine renders m and returns its footer without styles.
func footerLine(m RootModel) string {
	lines := viewLines(m)
	return ansi.Strip(lines[len(lines)-1])
}

func TestUnlockShowsSyncAgeFromStatus(t *testing.T) {
	m := NewRootModel(15 * time.Minute)
	m.width, m.height = 80, 24
	// bw writes the time with milliseconds.
	synced := time.Now().Add(-5*time.Minute - 30*time.Second).UTC().Format("2006-01-02T15:04:05.000Z07:00")
	m, _ = update(t, m, bwcmd.StatusResult{Status: bwcmd.VaultStatus{Status: "locked", UserEmail: "u@t.com", LastSync: synced}})
	m, _ = update(t, m, screens.UnlockedMsg{Token: "tok"})
	m, _ = update(t, m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch})
	if footer := footerLine(m); !strings.Contains(footer, "synced 5m ago") {
		t.Errorf("footer after unlock = %q, want the sync age from bw status", footer)
	}

	// A lock clears the time in the session. The next unlock shows it
	// again.
	m, _ = update(t, m, screens.LockMsg{})
	m, _ = update(t, m, bwcmd.LockResult{})
	if !m.sess.LastSync.IsZero() {
		t.Error("a lock must clear the sync time in the session")
	}
	m, _ = update(t, m, screens.UnlockedMsg{Token: "tok2"})
	m, _ = update(t, m, bwcmd.ItemsResult{Items: testItems(), Epoch: m.sess.Epoch})
	if footer := footerLine(m); !strings.Contains(footer, "synced 5m ago") {
		t.Errorf("footer after a second unlock = %q, want the sync age", footer)
	}
}

func TestParseLastSync(t *testing.T) {
	want := time.Date(2026, 10, 16, 8, 30, 0, 123e6, time.UTC)
	for in, out := range map[string]time.Time{
		"2026-10-16T08:30:00.123Z": want,
		"2026-10-16T08:30:00Z":     want.Truncate(time.Second),
		"":                         {},
		"yesterday":                {},
	} {
		if got := parseLastSync(in); !got.Equal(out) {
			t.Errorf("parseLastSync(%q) = %v, want %v", in, got, out)
		}
	}
}
