package bwcmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

const cmdTimeout = 10 * time.Second

// CmdResult carries the stdout bytes or an error from a bw subprocess.
type CmdResult struct {
	Output []byte
	Err    error
}

// Typed result messages for pattern matching in the root model.

// StatusResult is returned by CheckStatus.
type StatusResult struct {
	Status VaultStatus
	Err    error
}

// ItemsResult is returned by FetchItems.
type ItemsResult struct {
	Items []Item
	Err   error
}

// PasswordResult is returned by GetPassword.
type PasswordResult struct {
	Password string
	Err      error
}

// TOTPResult is returned by GetTOTP.
type TOTPResult struct {
	Code string
	Err  error
}

// UnlockResult is returned by Unlock/Login.
type UnlockResult struct {
	Token string
	Err   error
}

// SyncResult is returned by Sync.
type SyncResult struct {
	Err error
}

// LockResult is returned by Lock.
type LockResult struct {
	Err error
}

// GenerateResult is returned by Generate.
type GenerateResult struct {
	Password string
	Err      error
}

// execBw runs the bw CLI with the given args and session token.
func execBw(sessionToken string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bw", args...) //nolint:gosec // args are constructed internally, not from user input
	cmd.Env = append(os.Environ(), "BW_SESSION="+sessionToken)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("%s", errMsg)
	}
	return stdout.Bytes(), nil
}

// CheckStatus runs `bw status` and returns a StatusResult.
func CheckStatus() tea.Cmd {
	return func() tea.Msg {
		out, err := execBw("", "status")
		if err != nil {
			return StatusResult{Err: err}
		}
		status, err := ParseStatus(out)
		if err != nil {
			return StatusResult{Err: err}
		}
		return StatusResult{Status: status}
	}
}

// FetchItems runs `bw list items` and returns an ItemsResult.
func FetchItems(token string) tea.Cmd {
	return func() tea.Msg {
		out, err := execBw(token, "list", "items")
		if err != nil {
			return ItemsResult{Err: err}
		}
		items, err := ParseItems(out)
		if err != nil {
			return ItemsResult{Err: err}
		}
		return ItemsResult{Items: items}
	}
}

// Unlock runs `bw unlock [password] --raw` and returns an UnlockResult.
func Unlock(password string) tea.Cmd {
	return func() tea.Msg {
		out, err := execBw("", "unlock", password, "--raw")
		if err != nil {
			return UnlockResult{Err: err}
		}
		return UnlockResult{Token: strings.TrimSpace(string(out))}
	}
}

// Login runs `bw login [email] [password] --raw` and returns an UnlockResult.
func LoginUser(email, password string) tea.Cmd {
	return func() tea.Msg {
		out, err := execBw("", "login", email, password, "--raw")
		if err != nil {
			return UnlockResult{Err: err}
		}
		return UnlockResult{Token: strings.TrimSpace(string(out))}
	}
}

// Lock runs `bw lock`.
func Lock(token string) tea.Cmd {
	return func() tea.Msg {
		_, err := execBw(token, "lock")
		return LockResult{Err: err}
	}
}

// Sync runs `bw sync`.
func Sync(token string) tea.Cmd {
	return func() tea.Msg {
		_, err := execBw(token, "sync")
		return SyncResult{Err: err}
	}
}

// GetPassword runs `bw get password [id]`.
func GetPassword(token, id string) tea.Cmd {
	return func() tea.Msg {
		out, err := execBw(token, "get", "password", id)
		if err != nil {
			return PasswordResult{Err: err}
		}
		return PasswordResult{Password: string(out)}
	}
}

// GetTOTP runs `bw get totp [id]`.
func GetTOTP(token, id string) tea.Cmd {
	return func() tea.Msg {
		out, err := execBw(token, "get", "totp", id)
		if err != nil {
			return TOTPResult{Err: err}
		}
		return TOTPResult{Code: strings.TrimSpace(string(out))}
	}
}

// Generate runs `bw generate` with the given args.
func Generate(args ...string) tea.Cmd {
	return func() tea.Msg {
		fullArgs := append([]string{"generate"}, args...)
		out, err := execBw("", fullArgs...)
		if err != nil {
			return GenerateResult{Err: err}
		}
		return GenerateResult{Password: strings.TrimSpace(string(out))}
	}
}
