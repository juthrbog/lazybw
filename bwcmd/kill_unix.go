//go:build unix

package bwcmd

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// killTreeOnCancel starts cmd in a new process group, and makes the end of
// its context kill the whole group. A wrapper script that starts bw
// without exec then cannot leave bw running, with BW_PASSWORD in its
// environment, after a timeout. bw reads no terminal (stdin is /dev/null
// and BW_NOINTERACTION is set), so it does not need the foreground group.
func killTreeOnCancel(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// A negative PID names a process group. Setpgid gave the group
		// the PID of the program that lazybw started.
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
}
