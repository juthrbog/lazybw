//go:build !unix

package bwcmd

import "os/exec"

// killTreeOnCancel keeps the default cancel of exec.CommandContext, which
// kills only the program that lazybw started. waitDelay limits how long
// execRun waits for the processes that this program started.
func killTreeOnCancel(*exec.Cmd) {}
