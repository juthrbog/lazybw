package bwcmd

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fakeBw writes script as the bw program in a new directory and makes that
// directory the only PATH entry, so that no test can run the real bw. The
// script sets its own PATH first, so that it can run tools such as env
// and sleep.
func fakeBw(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake bw is a shell script")
	}
	dir := t.TempDir()
	body := "#!/bin/sh\nPATH=/usr/bin:/bin\n" + script
	//nolint:gosec // the fake bw must be executable
	if err := os.WriteFile(filepath.Join(dir, "bw"), []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

// useRunner makes Run call r for one test.
func useRunner(t *testing.T, r Runner) {
	t.Helper()
	old := Run
	t.Cleanup(func() { Run = old })
	Run = r
}

// bwCall is one call of a Runner that a test installed.
type bwCall struct {
	args        []string
	env         []string
	hasDeadline bool
}

// envValue returns the value of the variable name in env, and whether env
// sets it.
func envValue(env []string, name string) (string, bool) {
	for _, kv := range env {
		if n, v, ok := strings.Cut(kv, "="); ok && n == name {
			return v, true
		}
	}
	return "", false
}

// TestRunnerScriptsOutput replaces Run, so that the commands get scripted
// bw output. PATH does not change, and no bw program runs.
func TestRunnerScriptsOutput(t *testing.T) {
	var calls []bwCall
	useRunner(t, func(ctx context.Context, env []string, _ []byte, args ...string) ([]byte, []byte, error) {
		_, hasDeadline := ctx.Deadline()
		calls = append(calls, bwCall{args: args, env: env, hasDeadline: hasDeadline})
		switch args[0] {
		case "status":
			return []byte(`{"status":"locked","userEmail":"u@t.com"}`), nil, nil
		case "unlock":
			return []byte("session-token\n"), nil, nil
		case "list":
			return []byte(`[{"id":"1","type":1,"name":"Gmail","login":{"username":"u","password":"p"}}]`), nil, nil
		case "sync":
			return nil, []byte("Syncing failed.\nYou are not logged in.\n"), errors.New("exit status 1")
		}
		t.Errorf("unexpected bw %q", args)
		return nil, nil, errors.New("unexpected command")
	})

	status := CheckStatus()().(StatusResult)
	if status.Err != nil || status.Status.Status != "locked" || status.Status.UserEmail != "u@t.com" {
		t.Errorf("CheckStatus = %+v, want the scripted status", status)
	}
	unlock := Unlock("hunter2")().(UnlockResult)
	if unlock.Err != nil || unlock.Token != "session-token" {
		t.Errorf("Unlock = %+v, want the scripted token", unlock)
	}
	items := FetchItems("session-token", 3)().(ItemsResult)
	if items.Err != nil || len(items.Items) != 1 || items.Items[0].Name != "Gmail" || items.Epoch != 3 {
		t.Errorf("FetchItems = %+v, want the scripted item and epoch 3", items)
	}
	sync := Sync("session-token", 3)().(SyncResult)
	var bwErr *Error
	if !errors.As(sync.Err, &bwErr) || bwErr.Msg != "You are not logged in." || !errors.Is(sync.Err, ErrNotLoggedIn) {
		t.Errorf("Sync err = %#v, want an *Error for the last stderr line", sync.Err)
	}

	if len(calls) != 4 {
		t.Fatalf("Run got %d calls, want 4", len(calls))
	}
	for _, c := range calls {
		if !c.hasDeadline {
			t.Errorf("bw %q: the context has no deadline", c.args)
		}
		if v, _ := envValue(c.env, "BW_NOINTERACTION"); v != "true" {
			t.Errorf("bw %q: BW_NOINTERACTION = %q, want true", c.args, v)
		}
	}
	unlockCall := calls[1]
	if want := []string{"unlock", "--passwordenv", "BW_PASSWORD", "--raw"}; !reflect.DeepEqual(unlockCall.args, want) {
		t.Errorf("unlock args = %q, want %q", unlockCall.args, want)
	}
	if v, _ := envValue(unlockCall.env, "BW_PASSWORD"); v != "hunter2" {
		t.Errorf("unlock BW_PASSWORD = %q, want the password", v)
	}
	if v, _ := envValue(calls[2].env, "BW_SESSION"); v != "session-token" {
		t.Errorf("list BW_SESSION = %q, want the token", v)
	}
	if _, ok := envValue(calls[0].env, "BW_SESSION"); ok {
		t.Error("status got BW_SESSION, want none")
	}
}

// TestRunnerTimeout checks that a Runner that waits until its context ends
// gives the timeout error, and that the partial stdout is zeroed.
func TestRunnerTimeout(t *testing.T) {
	setTimeouts(t, 20*time.Millisecond, 20*time.Millisecond)
	partial := []byte("partial output")
	useRunner(t, func(ctx context.Context, _ []string, _ []byte, _ ...string) ([]byte, []byte, error) {
		<-ctx.Done()
		return partial, nil, ctx.Err()
	})
	captureLog(t)

	err := Lock("tok", 1)().(LockResult).Err
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if !strings.Contains(err.Error(), "bw lock timed out after 20ms") {
		t.Errorf("err = %q, want a timed out text", err)
	}
	if !bytes.Equal(partial, make([]byte, len(partial))) {
		t.Errorf("stdout = %q, want zeros", partial)
	}
}

// logFile returns the path of a new file in the temp directory of t, and
// sets FAKE_BW_LOG to it for the fake bw.
func logFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bw.log")
	t.Setenv("FAKE_BW_LOG", path)
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // test-owned temp file
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// setTimeouts sets the time limits for one test.
func setTimeouts(t *testing.T, timeout, delay time.Duration) {
	t.Helper()
	short, long, wait := shortTimeout, longTimeout, waitDelay
	t.Cleanup(func() { shortTimeout, longTimeout, waitDelay = short, long, wait })
	shortTimeout, longTimeout, waitDelay = timeout, timeout, delay
}

// captureLog sends the standard logger to a buffer for one test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	out, flags := log.Writer(), log.Flags()
	t.Cleanup(func() { log.SetOutput(out); log.SetFlags(flags) })
	log.SetOutput(&buf)
	return &buf
}

// envOf parses the output of env into a map.
func envOf(s string) map[string]string {
	env := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		if name, value, ok := strings.Cut(line, "="); ok {
			env[name] = value
		}
	}
	return env
}

func TestChildEnvStripsBwVariables(t *testing.T) {
	fakeBw(t, `env > "$FAKE_BW_LOG"; echo token`)
	path := logFile(t)
	for _, kv := range []string{
		"BW_SESSION=parent-session", "BW_PASSWORD=parent-password",
		"BW_RESPONSE=true", "BW_QUIET=true", "BW_RAW=true", "BW_PRETTY=true",
		"BW_CLEANEXIT=true", "BW_NOINTERACTION=false", "Bw_Quiet=true",
		"BITWARDENCLI_DEBUG=true",
		"BITWARDENCLI_APPDATA_DIR=/data", "NODE_EXTRA_CA_CERTS=/ca.pem",
		"HTTPS_PROXY=http://proxy:3128",
	} {
		name, value, _ := strings.Cut(kv, "=")
		t.Setenv(name, value)
	}

	tests := []struct {
		name     string
		run      func() error
		session  string // "" means no BW_SESSION
		password string // "" means no BW_PASSWORD
	}{
		{"status", func() error { _, err := execBw(time.Second, "", "", "status"); return err }, "", ""},
		{"list with token", func() error { _, err := execBw(time.Second, "tok", "", "list", "items"); return err }, "tok", ""},
		{"unlock", func() error { return Unlock("hunter2")().(UnlockResult).Err }, "", "hunter2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err != nil {
				t.Fatal(err)
			}
			env := envOf(readFile(t, path))
			for _, name := range []string{"BW_RESPONSE", "BW_QUIET", "BW_RAW", "BW_PRETTY", "BW_CLEANEXIT", "Bw_Quiet", "BITWARDENCLI_DEBUG"} {
				if v, ok := env[name]; ok {
					t.Errorf("bw got %s=%s, want it removed", name, v)
				}
			}
			if env["BW_NOINTERACTION"] != "true" {
				t.Errorf("BW_NOINTERACTION = %q, want true", env["BW_NOINTERACTION"])
			}
			if v, ok := env["BW_SESSION"]; ok != (tt.session != "") || v != tt.session {
				t.Errorf("BW_SESSION = %q (set: %v), want %q", v, ok, tt.session)
			}
			if v, ok := env["BW_PASSWORD"]; ok != (tt.password != "") || v != tt.password {
				t.Errorf("BW_PASSWORD = %q (set: %v), want %q", v, ok, tt.password)
			}
			for name, want := range map[string]string{
				"BITWARDENCLI_APPDATA_DIR": "/data",
				"NODE_EXTRA_CA_CERTS":      "/ca.pem",
				"HTTPS_PROXY":              "http://proxy:3128",
			} {
				if env[name] != want {
					t.Errorf("%s = %q, want %q", name, env[name], want)
				}
			}
		})
	}
}

func TestEveryCommandSetsNoInteraction(t *testing.T) {
	fakeBw(t, `echo "$1 $BW_NOINTERACTION" >> "$FAKE_BW_LOG"
case "$1" in
status) echo '{"status":"locked"}' ;;
list) echo '[]' ;;
*) echo out ;;
esac`)
	path := logFile(t)
	t.Setenv("BW_NOINTERACTION", "false")

	_ = CheckStatus()()
	_ = FetchItems("tok", 1)()
	_ = Unlock("pw")()
	_ = LoginUser("u@t.com", "pw", TwoStep{})()
	_ = Lock("tok", 1)()
	_ = Sync("tok", 1)()
	_ = Generate(1, "--length", "20")()

	lines := strings.Split(strings.TrimSpace(readFile(t, path)), "\n")
	want := []string{"status", "list", "unlock", "login", "lock", "sync", "generate"}
	if len(lines) != len(want) {
		t.Fatalf("bw ran %d times, want %d: %q", len(lines), len(want), lines)
	}
	for i, line := range lines {
		if line != want[i]+" true" {
			t.Errorf("run %d: got %q, want %q", i, line, want[i]+" true")
		}
	}
}

func TestOutputVariablesDoNotChangeResults(t *testing.T) {
	// The fake fails like bw does when it sees an output variable: with no
	// usable output. With BITWARDENCLI_DEBUG, bw writes its debug log to
	// stdout in front of the output.
	fakeBw(t, `if [ -n "$BW_RESPONSE$BW_QUIET$BW_CLEANEXIT" ]; then exit 0; fi
if [ "$BITWARDENCLI_DEBUG" = true ]; then echo 'Initializing lowdb storage service.'; fi
case "$1" in
status) echo '{"status":"locked","userEmail":"u@t.com"}' ;;
unlock) echo 'session-token' ;;
list) echo '[{"id":"1","type":1,"name":"A"}]' ;;
esac`)
	t.Setenv("BW_RESPONSE", "true")
	t.Setenv("BW_QUIET", "true")
	t.Setenv("BW_CLEANEXIT", "true")
	t.Setenv("BITWARDENCLI_DEBUG", "true")

	status := CheckStatus()().(StatusResult)
	if status.Err != nil || status.Status.Status != "locked" {
		t.Errorf("CheckStatus = %+v, want locked", status)
	}
	unlock := Unlock("pw")().(UnlockResult)
	if unlock.Err != nil || unlock.Token != "session-token" {
		t.Errorf("Unlock = %+v, want the token", unlock)
	}
	items := FetchItems("session-token", 0)().(ItemsResult)
	if items.Err != nil || len(items.Items) != 1 {
		t.Errorf("FetchItems = %+v, want one item", items)
	}
}

func TestEmptyTokenIsAnError(t *testing.T) {
	fakeBw(t, `echo`) // exit 0 with only a newline on stdout
	tests := map[string]UnlockResult{
		"login":  LoginUser("u@t.com", "pw", TwoStep{})().(UnlockResult),
		"unlock": Unlock("pw")().(UnlockResult),
	}
	for name, res := range tests {
		if !errors.Is(res.Err, errNoToken) {
			t.Errorf("%s: err = %v, want errNoToken", name, res.Err)
		}
		if res.Token != "" {
			t.Errorf("%s: token = %q, want empty", name, res.Token)
		}
	}
}

func TestTokenResultZerosOutput(t *testing.T) {
	out := []byte("secret-token\n")
	res := tokenResult(out)
	if res.Token != "secret-token" || res.Err != nil {
		t.Fatalf("tokenResult = %+v, want the token", res)
	}
	if !bytes.Equal(out, make([]byte, len(out))) {
		t.Errorf("stdout = %q, want zeros", out)
	}
}

func TestLoginUserArgs(t *testing.T) {
	fakeBw(t, `for a in "$@"; do printf '%s\n' "$a"; done > "$FAKE_BW_LOG"; echo token`)
	path := logFile(t)
	login := func(rest ...string) []string {
		return append([]string{"login", "--passwordenv", "BW_PASSWORD", "--raw"}, rest...)
	}
	tests := []struct {
		name  string
		email string
		step  TwoStep
		want  []string
	}{
		{"email that looks like an option", "--help", TwoStep{}, login("--", "--help")},
		{"no two-step options", "u@t.com", TwoStep{}, login("--", "u@t.com")},
		{"authenticator code", "u@t.com", TwoStep{Method: MethodAuthenticator, Code: "123456"},
			login("--method", "0", "--code=123456", "--", "u@t.com")},
		{"email method without code", "u@t.com", TwoStep{Method: MethodEmail},
			login("--method", "1", "--", "u@t.com")},
		{"yubikey", "u@t.com", TwoStep{Method: MethodYubiKey, Code: "cccc"},
			login("--method", "3", "--code=cccc", "--", "u@t.com")},
		{"code without method", "u@t.com", TwoStep{Code: "-1"},
			login("--code=-1", "--", "u@t.com")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := LoginUser(tt.email, "x", tt.step)().(UnlockResult)
			if res.Err != nil || res.Token != "token" {
				t.Fatalf("LoginUser = %+v, want the token", res)
			}
			got := strings.Split(strings.TrimSuffix(readFile(t, path), "\n"), "\n")
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("args = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestUnlockArgs checks that the master password goes to bw in the
// BW_PASSWORD variable and never in argv, where other users can read it.
func TestUnlockArgs(t *testing.T) {
	fakeBw(t, `for a in "$@"; do printf 'arg=%s\n' "$a"; done > "$FAKE_BW_LOG"; printf 'env=%s\n' "$BW_PASSWORD" >> "$FAKE_BW_LOG"; echo token`)
	path := logFile(t)
	res := Unlock("hunter2")().(UnlockResult)
	if res.Err != nil || res.Token != "token" {
		t.Fatalf("Unlock = %+v, want the token", res)
	}
	got := strings.Split(strings.TrimSuffix(readFile(t, path), "\n"), "\n")
	want := []string{"arg=unlock", "arg=--passwordenv", "arg=BW_PASSWORD", "arg=--raw", "env=hunter2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bw got %q, want %q", got, want)
	}
	for _, line := range got {
		if strings.HasPrefix(line, "arg=") && strings.Contains(line, "hunter2") {
			t.Errorf("the password is in argv: %q", line)
		}
	}
}

func TestTimeoutError(t *testing.T) {
	fakeBw(t, `exec sleep 1`)
	setTimeouts(t, 50*time.Millisecond, 100*time.Millisecond)

	err := FetchItems("tok", 0)().(ItemsResult).Err
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if !strings.Contains(err.Error(), "bw list items timed out after 50ms") {
		t.Errorf("err = %q, want a timed out text", err)
	}
}

// stackTrace is the shape of what bw writes to stderr when it cannot
// reach the server.
const stackTrace = `Unable to fetch ServerConfig from https://vault.example/api FetchError: request to https://vault.example/api/config failed, reason: connect ECONNREFUSED 127.0.0.1:443
    at ClientRequest.<anonymous> (/snapshot/clients/node_modules/node-fetch/lib/index.js:1501:11)
    at ClientRequest.emit (node:events:518:28)
    at emitErrorEvent (node:_http_client:104:11)
    at TLSSocket.socketErrorListener (node:_http_client:518:5)
    at process.processTicksAndRejections (node:internal/process/task_queues:90:21) {
  type: 'system',
  errno: 'ECONNREFUSED',
  code: 'ECONNREFUSED'
}
You are not logged in.

`

func TestStackTraceGivesLastLine(t *testing.T) {
	fakeBw(t, `cat >&2 <<'EOF'
`+stackTrace+`EOF
exit 1`)
	captureLog(t)

	err := FetchItems("tok", 0)().(ItemsResult).Err
	if err == nil || err.Error() != "You are not logged in." {
		t.Fatalf("err = %q, want %q", err, "You are not logged in.")
	}
	if !errors.Is(err, ErrNotLoggedIn) {
		t.Error("errors.Is(err, ErrNotLoggedIn) = false")
	}
	var bwErr *Error
	if !errors.As(err, &bwErr) || !strings.Contains(bwErr.Stderr, "ECONNREFUSED") {
		t.Errorf("err = %#v, want an *Error with all of stderr", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Error("the error must wrap the *exec.ExitError")
	}
}

func TestErrorTextHasNoANSI(t *testing.T) {
	fakeBw(t, `printf '\033[32m?\033[39m \033[1mTwo-step login code:\033[22m\033[0m \033[0m\033[29D\033[29C\n\033[31mCode is required.\033[0m\r\n' >&2
exit 1`)
	err := LoginUser("u@t.com", "pw", TwoStep{})().(UnlockResult).Err
	if err == nil || err.Error() != "Code is required." {
		t.Fatalf("err = %q, want %q", err, "Code is required.")
	}
	if strings.Contains(err.Error(), "\x1b") {
		t.Errorf("err = %q, want no escape sequences", err)
	}
	if !errors.Is(err, ErrCodeRequired) {
		t.Error("errors.Is(err, ErrCodeRequired) = false")
	}
}

func TestEmptyStderrGivesExecError(t *testing.T) {
	fakeBw(t, `exit 3`)
	res := Lock("tok", 7)().(LockResult)
	if err := res.Err; err == nil || err.Error() != "exit status 3" {
		t.Errorf("err = %q, want %q", err, "exit status 3")
	}
	if res.Seq != 7 {
		t.Errorf("Seq = %d, want the sequence number 7 that was passed to Lock", res.Seq)
	}
}

func TestMissingBw(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := CheckStatus()().(StatusResult).Err
	if !errors.Is(err, exec.ErrNotFound) {
		t.Errorf("err = %v, want exec.ErrNotFound", err)
	}
}

func TestDebugLogHidesPrivateValues(t *testing.T) {
	fakeBw(t, `echo "args: $*" >&2
echo "env: $BW_PASSWORD $BW_SESSION" >&2
echo "Login failed." >&2
exit 1`)
	buf := captureLog(t)

	_ = LoginUser("user@example.com", "hunter2", TwoStep{Method: MethodAuthenticator, Code: "424242"})()
	_ = FetchItems("session-secret", 0)()

	got := buf.String()
	for _, secret := range []string{"user@example.com", "hunter2", "424242", "session-secret", "PATH="} {
		if strings.Contains(got, secret) {
			t.Errorf("debug log contains %q:\n%s", secret, got)
		}
	}
	for _, want := range []string{"bw login failed", "bw list items failed", "Login failed."} {
		if !strings.Contains(got, want) {
			t.Errorf("debug log has no %q:\n%s", want, got)
		}
	}
}

func TestErrorIsKnownMessage(t *testing.T) {
	tests := []struct {
		msg  string
		want error // nil: no known error
	}{
		{"Code is required.", ErrCodeRequired},
		{"Login failed. No provider selected.", ErrMethodRequired},
		{"Invalid master password. Confirm your email is correct and your account was created on vault.example.", ErrInvalidPassword},
		{"Invalid master password.", ErrInvalidPassword},
		{"Username or password is incorrect. Try again.", ErrInvalidPassword},
		{"The provided key is not the expected type", ErrInvalidPassword},
		{"You are not logged in.", ErrNotLoggedIn},
		{"Vault is locked.", nil},
		{"Login failed.", nil},
	}
	known := []error{ErrCodeRequired, ErrMethodRequired, ErrInvalidPassword, ErrNotLoggedIn}
	for _, tt := range tests {
		err := error(&Error{Msg: tt.msg})
		for _, k := range known {
			if got := errors.Is(err, k); got != (k == tt.want) {
				t.Errorf("errors.Is(%q, %v) = %v", tt.msg, k, got)
			}
		}
	}
}

func TestLastLine(t *testing.T) {
	tests := map[string]string{
		"":                           "",
		"\n\n":                       "",
		"one\ntwo\n\n":               "two",
		"warning\r\nmessage\r\n":     "message",
		"prompt\rmessage":            "message",
		"text\n\x1b[2K\x1b[G\n":      "text",
		"\x1b[31m  red text \x1b[0m": "red text",
	}
	for in, want := range tests {
		if got := lastLine(in); got != want {
			t.Errorf("lastLine(%q) = %q, want %q", in, got, want)
		}
	}
}
