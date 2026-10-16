package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/screens"
	"github.com/juthrbog/lazybw/session"
)

func TestVersionString(t *testing.T) {
	prevVersion, prevRead := version, readBuildInfo
	t.Cleanup(func() { version, readBuildInfo = prevVersion, prevRead })

	tests := []struct {
		name    string
		version string // the value that -ldflags sets, or "dev"
		info    *debug.BuildInfo
		ok      bool
		want    string
	}{
		{"release build", "0.3.0", &debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}}, true, "lazybw 0.3.0"},
		{"go install build", "dev", &debug.BuildInfo{Main: debug.Module{Version: "v0.3.0"}}, true, "lazybw v0.3.0"},
		{"go build in a git clone", "dev", &debug.BuildInfo{Main: debug.Module{Version: "v0.2.1-0.20261016083000-959df31a2b3c+dirty"}}, true, "lazybw v0.2.1-0.20261016083000-959df31a2b3c+dirty"},
		{"no VCS data", "dev", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true, "lazybw dev"},
		{"no module version", "dev", &debug.BuildInfo{}, true, "lazybw dev"},
		{"no build info", "dev", nil, false, "lazybw dev"},
	}
	for _, tt := range tests {
		version = tt.version
		readBuildInfo = func() (*debug.BuildInfo, bool) { return tt.info, tt.ok }
		if got := versionString(); got != tt.want {
			t.Errorf("%s: versionString() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// TestNewProgramWritesNoTraceFile builds the program as main does, with
// TEA_TRACE set. No trace file must exist after that.
func TestNewProgramWritesNoTraceFile(t *testing.T) {
	dir := t.TempDir()
	if runtime.GOOS != "windows" {
		// Without the check, tea.NewProgram opens the trace file. The
		// program keeps the file open, and Windows cannot remove an open
		// file from the test directory.
		control := filepath.Join(dir, "control.log")
		t.Setenv("TEA_TRACE", control)
		_ = tea.NewProgram(NewRootModel(time.Minute))
		if _, err := os.Stat(control); err != nil {
			t.Fatalf("tea.NewProgram did not open the trace file of TEA_TRACE: %v", err)
		}
	}

	trace := filepath.Join(dir, "trace.log")
	t.Setenv("TEA_TRACE", trace)
	var out bytes.Buffer
	_ = newProgram(NewRootModel(time.Minute), &out)
	if _, err := os.Stat(trace); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("trace file: stat error = %v, want that the file does not exist", err)
	}
	if _, ok := os.LookupEnv("TEA_TRACE"); ok {
		t.Error("TEA_TRACE must be unset")
	}
	if got := out.String(); !strings.HasPrefix(got, "Warning:") || !strings.Contains(got, "TEA_TRACE") {
		t.Errorf("stderr = %q, want a warning that names TEA_TRACE", got)
	}
}

func TestDropTeaTraceWithoutVariable(t *testing.T) {
	t.Setenv("TEA_TRACE", "")
	if err := os.Unsetenv("TEA_TRACE"); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dropTeaTrace(&out)
	if out.Len() != 0 {
		t.Errorf("stderr = %q, want no warning when TEA_TRACE is not set", out.String())
	}
}

func TestPrintLockErrorsRemovesControlCharacters(t *testing.T) {
	m := NewRootModel(time.Minute)
	m.quitError = "bw lock failed: \x1b]52;c;aGk=\x07 boom"
	var out bytes.Buffer
	printLockErrors(&out, m, nil)
	if got := out.String(); strings.ContainsAny(got, "\x1b\x07") || !strings.Contains(got, "bw lock failed") {
		t.Errorf("printed %q, want the text with no control characters", got)
	}
}

func TestExitCleanupReportsLockError(t *testing.T) {
	// The fake bw fails bw lock for this token.
	err := exitCleanup{lockToken: failLockToken}.run(&bytes.Buffer{})
	if err == nil {
		t.Fatal("run must return the error of bw lock")
	}
	var out bytes.Buffer
	printLockErrors(&out, nil, err)
	for _, want := range []string{"bw lock failed: " + strings.TrimSuffix(failLockMessage, "."), `Run "bw lock" in a terminal.`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("printed %q, want it to contain %q", out.String(), want)
		}
	}
	if err := (exitCleanup{lockToken: "tok"}).run(&bytes.Buffer{}); err != nil {
		t.Errorf("run with a working bw lock = %v, want no error", err)
	}
}

// TestFinishPrintsLockErrors runs the exit as main does. The fake bw fails
// bw lock for failLockToken.
func TestFinishPrintsLockErrors(t *testing.T) {
	// After a panic, Run returns no model, and the cleanup that the last
	// Update recorded runs.
	var stdout, stderr bytes.Buffer
	finish(nil, &exitCleanup{clipDirty: true, lockToken: failLockToken}, &stdout, &stderr)
	if stdout.String() != ansi.SetSystemClipboard(session.ClearPlaceholder) {
		t.Errorf("stdout = %q, want an OSC 52 clear", stdout.String())
	}
	for _, want := range []string{"bw lock failed: " + strings.TrimSuffix(failLockMessage, "."), `Run "bw lock" in a terminal.`} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr after the exit cleanup = %q, want it to contain %q", stderr.String(), want)
		}
	}

	// A quit after a failed bw lock: the final model holds the error.
	m := unlockedVault(t, testItems())
	m, _ = update(t, m, screens.QuitMsg{})
	m, _ = update(t, m, lockResult(m, errLockTimedOut))
	stdout.Reset()
	stderr.Reset()
	finish(m, m.lastCleanup, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "bw lock failed: bw lock timed out") {
		t.Errorf("stderr after a quit = %q, want the bw lock error", stderr.String())
	}

	// A working bw lock prints nothing.
	stderr.Reset()
	finish(nil, &exitCleanup{lockToken: "tok"}, &stdout, &stderr)
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want nothing after a working bw lock", stderr.String())
	}
}
