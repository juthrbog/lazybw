package bwcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Time limits for bw commands. Tests shorten them.
var (
	// shortTimeout is the limit for bw status, bw lock and bw generate.
	shortTimeout = 15 * time.Second
	// longTimeout is the limit for bw list items, bw sync, bw login and
	// bw unlock. These commands wait for the network, read the whole
	// vault or run the key derivation.
	longTimeout = 2 * time.Minute
	// waitDelay is how long the pipes of a killed bw can stay open. A
	// process that the kill does not reach keeps the pipes open. On
	// Windows, the kill stops only the program that lazybw started, such
	// as a wrapper script, and not the bw that the wrapper started.
	waitDelay = 2 * time.Second
)

// errNoToken is the error for a bw unlock or bw login that exits 0 but
// prints no session key.
var errNoToken = errors.New("bw did not return a session key")

// Typed result messages for pattern matching in the root model.

// StatusResult is returned by CheckStatus.
type StatusResult struct {
	Status VaultStatus
	Err    error
}

// ItemsResult is returned by FetchItems. Epoch is the session epoch that
// was passed to FetchItems.
type ItemsResult struct {
	Items []Item
	Err   error
	Epoch uint64
}

// UnlockResult is returned by Unlock/Login.
type UnlockResult struct {
	Token string
	Err   error
}

// SyncResult is returned by Sync. Epoch is the session epoch that was
// passed to Sync.
type SyncResult struct {
	Err   error
	Epoch uint64
}

// LockResult is returned by Lock.
type LockResult struct {
	Err error
}

// GenerateResult is returned by Generate. Seq is the sequence number that
// was passed to Generate.
type GenerateResult struct {
	Password string
	Err      error
	Seq      uint64
}

// TwoStepMethod is a two-step login method that LoginUser can pass to bw.
type TwoStepMethod int

const (
	MethodAuto          TwoStepMethod = iota // bw picks the method
	MethodAuthenticator                      // authenticator app, bw method 0
	MethodEmail                              // code sent by email, bw method 1
	MethodYubiKey                            // YubiKey OTP, bw method 3
)

// arg returns the value for the bw --method option, or "" for MethodAuto.
func (m TwoStepMethod) arg() string {
	switch m {
	case MethodAuthenticator:
		return "0"
	case MethodEmail:
		return "1"
	case MethodYubiKey:
		return "3"
	}
	return ""
}

// TwoStep holds the two-step login options for LoginUser. The zero value
// sends no two-step options.
type TwoStep struct {
	Method TwoStepMethod
	Code   string // empty sends no code
}

// strippedEnv lists the variables that childEnv does not copy to bw. The
// BW_ output variables change what bw prints. BITWARDENCLI_DEBUG makes bw
// write its debug log to stdout, in front of the JSON and the session key.
// lazybw sets BW_SESSION, BW_PASSWORD and BW_NOINTERACTION itself.
var strippedEnv = []string{
	"BW_SESSION", "BW_PASSWORD", "BW_RESPONSE", "BW_QUIET", "BW_RAW",
	"BW_PRETTY", "BW_CLEANEXIT", "BW_NOINTERACTION", "BITWARDENCLI_DEBUG",
}

// childEnv returns the environment for bw. It copies the environment of
// lazybw without the variables in strippedEnv, so variables such as
// BITWARDENCLI_APPDATA_DIR, NODE_EXTRA_CA_CERTS and the proxy variables
// stay. It adds BW_NOINTERACTION=true, so that bw fails at once when it
// needs an answer that nobody can type. It adds BW_SESSION when token is
// not empty, and BW_PASSWORD when password is not empty.
func childEnv(token, password string) []string {
	parent := os.Environ()
	env := make([]string, 0, len(parent)+3)
	for _, kv := range parent {
		if !isStripped(kv) {
			env = append(env, kv)
		}
	}
	env = append(env, "BW_NOINTERACTION=true")
	if token != "" {
		env = append(env, "BW_SESSION="+token)
	}
	if password != "" {
		env = append(env, "BW_PASSWORD="+password)
	}
	return env
}

// isStripped reports whether the entry kv ("NAME=value") names a variable
// in strippedEnv. Names on Windows ignore case, so the match ignores case.
func isStripped(kv string) bool {
	name, _, _ := strings.Cut(kv, "=")
	for _, s := range strippedEnv {
		if strings.EqualFold(name, s) {
			return true
		}
	}
	return false
}

// execBw runs bw with args and returns its stdout. The token and the
// password go to bw in its environment, not in args, which ps and
// /proc/PID/cmdline show to other users. bw is killed after timeout.
func execBw(timeout time.Duration, token, password string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bw", args...) //nolint:gosec // G204: the program is always bw and no shell runs; user text goes only after "--" or in "--code="
	cmd.Env = childEnv(token, password)
	cmd.WaitDelay = waitDelay
	killTreeOnCancel(cmd)

	var stdout secretBuffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		clear(stdout.Bytes())
		log.Printf("bw %s failed: %v\nstderr:\n%s", subcommand(args), err,
			redact(stderr.String(), privateValues(token, password, args)))
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("bw %s timed out after %s: %w", subcommand(args), timeout, ctx.Err())
		}
		return nil, newError(err, stderr.Bytes())
	}
	return stdout.Bytes(), nil
}

// secretBuffer collects the stdout of bw, which can hold the whole
// decrypted vault. Unlike bytes.Buffer, it zeros the old array each time
// it grows, so that no partial copy stays behind in memory. The copy
// buffers inside os/exec are out of reach.
type secretBuffer struct {
	b []byte
}

func (s *secretBuffer) Write(p []byte) (int, error) {
	if len(s.b)+len(p) > cap(s.b) {
		grown := make([]byte, len(s.b), 2*cap(s.b)+len(p))
		copy(grown, s.b)
		clear(s.b)
		s.b = grown
	}
	s.b = append(s.b, p...)
	return len(p), nil
}

// Bytes returns the collected output. The caller owns it and zeros it.
func (s *secretBuffer) Bytes() []byte { return s.b }

// subcommand returns the bw subcommand in args, such as "unlock" or
// "list items". It leaves out options and user values.
func subcommand(args []string) string {
	if len(args) == 0 {
		return ""
	}
	if args[0] == "list" && len(args) > 1 {
		return "list " + args[1]
	}
	return args[0]
}

// privateValues returns the values that the debug log must not show: the
// session token, the password, the two-step code and the args after "--",
// which hold the email.
func privateValues(token, password string, args []string) []string {
	private := []string{token, password}
	for i, a := range args {
		if code, ok := strings.CutPrefix(a, "--code="); ok {
			private = append(private, code)
		}
		if a == "--" {
			return append(private, args[i+1:]...)
		}
	}
	return private
}

// redact replaces each non-empty value of private in s with "[hidden]".
func redact(s string, private []string) string {
	for _, p := range private {
		if p != "" {
			s = strings.ReplaceAll(s, p, "[hidden]")
		}
	}
	return s
}

// tokenResult turns the stdout of `bw unlock --raw` or `bw login --raw`
// into an UnlockResult, and then zeros out. An exit code of 0 with no
// token still means that the unlock or login failed.
func tokenResult(out []byte) UnlockResult {
	token := strings.TrimSpace(string(out))
	clear(out)
	if token == "" {
		return UnlockResult{Err: errNoToken}
	}
	return UnlockResult{Token: token}
}

// CheckStatus runs `bw status` and returns a StatusResult.
func CheckStatus() tea.Cmd {
	return func() tea.Msg {
		out, err := execBw(shortTimeout, "", "", "status")
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

// FetchItems runs `bw list items` and returns an ItemsResult tagged with
// epoch, so the caller can drop a result that belongs to an ended session.
func FetchItems(token string, epoch uint64) tea.Cmd {
	return func() tea.Msg {
		out, err := execBw(longTimeout, token, "", "list", "items")
		if err != nil {
			return ItemsResult{Err: err, Epoch: epoch}
		}
		return itemsResult(out, epoch)
	}
}

// itemsResult parses the stdout of `bw list items` and then zeros it. The
// output holds the whole decrypted vault, including fields that lazybw
// does not parse, such as the password history.
func itemsResult(out []byte, epoch uint64) ItemsResult {
	defer clear(out)
	items, err := ParseItems(out)
	if err != nil {
		return ItemsResult{Err: err, Epoch: epoch}
	}
	return ItemsResult{Items: items, Epoch: epoch}
}

// Unlock runs `bw unlock --passwordenv BW_PASSWORD --raw` and returns
// an UnlockResult. The password is passed via a process-local environment
// variable instead of a CLI argument to avoid exposing it in
// /proc/PID/cmdline.
func Unlock(password string) tea.Cmd {
	return func() tea.Msg {
		out, err := execBw(longTimeout, "", password, "unlock", "--passwordenv", "BW_PASSWORD", "--raw")
		if err != nil {
			return UnlockResult{Err: err}
		}
		return tokenResult(out)
	}
}

// LoginUser runs `bw login --passwordenv BW_PASSWORD --raw [--method N]
// [--code=X] -- email` and returns an UnlockResult. The password is passed
// via a process-local environment variable instead of a CLI argument. The
// email comes after "--", so that bw cannot read it as an option.
func LoginUser(email, password string, step TwoStep) tea.Cmd {
	args := loginArgs(email, step)
	return func() tea.Msg {
		out, err := execBw(longTimeout, "", password, args...)
		if err != nil {
			return UnlockResult{Err: err}
		}
		return tokenResult(out)
	}
}

// loginArgs returns the bw arguments for LoginUser.
func loginArgs(email string, step TwoStep) []string {
	args := []string{"login", "--passwordenv", "BW_PASSWORD", "--raw"}
	if m := step.Method.arg(); m != "" {
		args = append(args, "--method", m)
	}
	if step.Code != "" {
		// One argument, so that a code that starts with "-" stays a value.
		args = append(args, "--code="+step.Code)
	}
	return append(args, "--", email)
}

// Lock runs `bw lock`.
func Lock(token string) tea.Cmd {
	return func() tea.Msg {
		_, err := execBw(shortTimeout, token, "", "lock")
		return LockResult{Err: err}
	}
}

// Sync runs `bw sync` and returns a SyncResult tagged with epoch.
func Sync(token string, epoch uint64) tea.Cmd {
	return func() tea.Msg {
		_, err := execBw(longTimeout, token, "", "sync")
		return SyncResult{Err: err, Epoch: epoch}
	}
}

// Generate runs `bw generate` with the given args and returns a
// GenerateResult tagged with seq.
func Generate(seq uint64, args ...string) tea.Cmd {
	return func() tea.Msg {
		fullArgs := append([]string{"generate"}, args...)
		out, err := execBw(shortTimeout, "", "", fullArgs...)
		if err != nil {
			return GenerateResult{Err: err, Seq: seq}
		}
		return GenerateResult{Password: strings.TrimSpace(string(out)), Seq: seq}
	}
}
