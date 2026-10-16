//go:build unix

package bwcmd

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// childPID reads the PID that the fake bw wrote to path. It returns 0 when
// the file is empty.
func childPID(t *testing.T, path string) int {
	t.Helper()
	s := strings.TrimSpace(readFile(t, path))
	if s == "" {
		return 0
	}
	pid, err := strconv.Atoi(s)
	if err != nil {
		t.Fatalf("PID file holds %q", s)
	}
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	return pid
}

// gone reports whether the process pid ends within one second. A killed
// orphan stays a zombie until init reaps it, and kill(pid, 0) finds a
// zombie too, so gone asks again until the deadline.
func gone(pid int) bool {
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestTimeoutWithWrapperScript(t *testing.T) {
	// The wrapper starts sleep without exec, writes the PID of sleep to
	// FAKE_BW_LOG and waits for it, as a wrapper that starts bw does.
	fakeBw(t, `sleep 5 &
echo $! > "$FAKE_BW_LOG"
wait
echo late`)
	path := logFile(t)
	const timeout, delay = 100 * time.Millisecond, 100 * time.Millisecond
	setTimeouts(t, timeout, delay)

	start := time.Now()
	err := Unlock("pw")().(UnlockResult).Err
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	// The call returns after the timeout plus the wait delay. The slack
	// covers process start, and stays well below the five seconds of sleep.
	if limit := timeout + delay + 400*time.Millisecond; elapsed > limit {
		t.Errorf("Unlock returned after %v, want at most %v", elapsed, limit)
	}

	pid := childPID(t, path)
	if pid == 0 {
		t.Skip("the timeout came before the wrapper started sleep")
	}
	if !gone(pid) {
		t.Errorf("the child of the wrapper (PID %d) still runs after the timeout", pid)
	}
}

func TestWaitDelayEndsWaitForEscapedProcess(t *testing.T) {
	if _, err := os.Stat("/usr/bin/setsid"); err != nil {
		t.Skip("setsid is not installed")
	}
	// sleep runs in a new session, so the kill of the process group does
	// not reach it, and it keeps the output pipes open for one second.
	fakeBw(t, `setsid sleep 1 &
echo $! > "$FAKE_BW_LOG"
wait`)
	path := logFile(t)
	const timeout, delay = 50 * time.Millisecond, 100 * time.Millisecond
	setTimeouts(t, timeout, delay)

	start := time.Now()
	err := Sync("tok", 0)().(SyncResult).Err
	elapsed := time.Since(start)
	childPID(t, path) // the cleanup stops sleep
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if limit := timeout + delay + 400*time.Millisecond; elapsed > limit {
		t.Errorf("Sync returned after %v, want at most %v", elapsed, limit)
	}
}
