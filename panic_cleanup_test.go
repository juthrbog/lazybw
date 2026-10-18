package main

import (
	"errors"
	"io"
	"os"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/screens"
	"github.com/juthrbog/lazybw/session"
)

func TestUpdateRecordsExitCleanup(t *testing.T) {
	m := unlockedVault(t, testItems())
	if got := *m.lastCleanup; got != (exitCleanup{lockToken: "tok"}) {
		t.Errorf("after unlock, lastCleanup = %+v, want the token", got)
	}
	m, _ = update(t, m, session.CopiedMsg{Field: session.CopyFieldPassword})
	if got := *m.lastCleanup; !got.clipDirty || got.lockToken != "tok" {
		t.Errorf("after a copy, lastCleanup = %+v, want a dirty clipboard and the token", got)
	}
	// While bw lock runs, the cleanup must lock again.
	m, _ = update(t, m, screens.LockMsg{})
	if got := m.lastCleanup.lockToken; got != "tok" {
		t.Errorf("while bw lock runs, lockToken = %q, want tok", got)
	}
	m, _ = update(t, m, bwcmd.LockResult{})
	if got := m.lastCleanup.lockToken; got != "" {
		t.Errorf("after bw lock returned, lockToken = %q, want none", got)
	}
}

func TestCleanupForPrefersFinalModel(t *testing.T) {
	m := unlockedVault(t, testItems())
	last := &exitCleanup{clipDirty: true, lockToken: "old"}
	if got := cleanupFor(m, last); got != (exitCleanup{lockToken: "tok"}) {
		t.Errorf("cleanupFor(model) = %+v, want the cleanup of the model", got)
	}
	if got := cleanupFor(nil, last); got != *last {
		t.Errorf("cleanupFor(nil) = %+v, want the recorded cleanup", got)
	}
	if got := cleanupFor(nil, nil); got != (exitCleanup{}) {
		t.Errorf("cleanupFor(nil, nil) = %+v, want no cleanup", got)
	}
}

// panicMsg makes panicModel panic in Update.
type panicMsg struct{}

// panicModel runs a RootModel and panics on panicMsg. It drops the
// commands of the root, so that no bw command runs.
type panicModel struct{ root RootModel }

func (p panicModel) Init() tea.Cmd { return nil }

func (p panicModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(panicMsg); ok {
		panic("test panic")
	}
	next, _ := p.root.Update(msg)
	p.root = next.(RootModel)
	return p, nil
}

func (p panicModel) View() tea.View { return tea.NewView("") }

// TestPanicInUpdateStillCleansUp runs a real program that panics in
// Update. Run then returns no model, and the cleanup must come from the
// last Update.
func TestPanicInUpdateStillCleansUp(t *testing.T) {
	// Bubble Tea writes the panic and a stack trace to os.Stderr.
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	stderr := os.Stderr
	os.Stderr = devNull
	t.Cleanup(func() {
		os.Stderr = stderr
		_ = devNull.Close()
	})

	root := NewRootModel(15 * time.Minute)
	p := tea.NewProgram(panicModel{root: root},
		tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutSignalHandler())
	go func() {
		p.Send(screens.UnlockedMsg{Token: "tok"})
		p.Send(session.CopiedMsg{Field: session.CopyFieldPassword})
		p.Send(panicMsg{})
	}()
	final, err := p.Run()
	if !errors.Is(err, tea.ErrProgramPanic) {
		t.Fatalf("Run error = %v, want ErrProgramPanic", err)
	}
	if final != nil {
		t.Logf("Run returned a model after a panic: %T", final)
	}
	want := exitCleanup{clipDirty: true, lockToken: "tok"}
	if got := cleanupFor(final, root.lastCleanup); got != want {
		t.Errorf("cleanup after a panic = %+v, want %+v", got, want)
	}
}
