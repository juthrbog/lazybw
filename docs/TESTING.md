# lazybw testing

The tests use the standard Go `testing` package. No test uses teatest, golden files, a `-update` flag or an assertion library. A test reports a failure with `t.Errorf` or `t.Fatalf`.

Most tests call `Update` or `View` on a model directly, without a terminal. A test then runs the commands that `Update` returns and looks at their messages. Functions that only compute a value, such as the bw output parsers and the TOTP code, have table tests. A table test loops over a list of input and output cases.

## Run the tests

Run these commands from the root of the repository:

```sh
# Run all tests.
go test ./...

# Run all tests with the race detector. CI runs this command.
go test -race ./...

# Run go test -race ./... through Taskfile.yml.
task test

# Run one test and show its log.
go test -run TestLockPathsRunBwLock -v .

# Run the static analysis of the Go tool.
go vet ./...
```

The race detector (`-race`) reports a data race. A data race is two goroutines that use the same variable at the same time, and one of them writes it.

## Linters, formatting and govulncheck

```sh
# Run the linters and the gofmt formatter from .golangci.yml.
golangci-lint run ./...

# Run golangci-lint run ./... through Taskfile.yml.
task lint

# Format all Go files. The task runs gofmt -l -w .
task fmt

# Install the govulncheck version that CI uses, then look for known
# vulnerabilities in the code and its dependencies.
go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
govulncheck ./...
```

The golangci-lint configuration in `.golangci.yml` turns on the linters gosec, govet, staticcheck, errcheck, ineffassign and unused. It also turns on the gofmt formatter. `golangci-lint run` reports a `gofmt` issue for each file that does not match the gofmt format.

golangci-lint also lints the `_test.go` files. gosec reports some lines in the tests. Examples are a write of an executable file, and a file read or a command with a path from a variable. The tests mark these lines with a `//nolint:gosec` comment that gives the reason, as `helpers_test.go` does.

Use govulncheck v1.9.0, as CI does. An older govulncheck can fail to load the packages of a newer Go toolchain.

## Coverage

```sh
# Write the coverage profile.
go test -coverprofile=coverage.out ./...

# Show the coverage of each function and the total.
go tool cover -func=coverage.out

# Write a page that marks the covered lines.
go tool cover -html=coverage.out -o coverage.html
```

`.gitignore` has the patterns `*.out` and `coverage.*`, so git does not track these files.

## Fuzz test and benchmarks

A fuzz test runs a function with inputs that Go generates. `totp/totp_test.go` has one fuzz test, `FuzzParse`. It makes sure that each secret that `totp.Parse` accepts gives a code. `go test ./...` runs only its seed inputs. To generate new inputs, run:

```sh
go test -run '^$' -fuzz FuzzParse -fuzztime 30s ./totp
```

If the fuzzer finds an input that fails, Go writes it to `totp/testdata/fuzz/FuzzParse`. Each later `go test` run then uses that file as a seed.

The benchmarks are `BenchmarkToggleExpand` and `BenchmarkGroupItems` in `screens/grouping_bench_test.go`, and `BenchmarkRenderDrawerLongNoteLine` in `ui/drawer_test.go`. The pattern `-run '^$'` matches no test, so only the benchmarks run:

```sh
go test -run '^$' -bench . ./screens ./ui
```

## What CI runs

The workflows are in `.github/workflows`. Each workflow pins its actions by commit SHA and takes the Go version from `go.mod` (`go-version-file`).

`ci.yml` runs on a push to `main` and on a pull request to `main`. It has two jobs:

- `ci` runs on `ubuntu-latest`. It runs `go build ./...`, `go vet ./...`, `go vet` with `GOOS=darwin` and with `GOOS=windows`, and `go test -race ./...`. Then it runs golangci-lint v2.14.0 through `golangci-lint-action`, with the linters and the gofmt formatter from `.golangci.yml`.
- `macos` runs on `macos-latest`. It runs `go build ./...` and `go test ./...`, without the race detector.

`security.yml` runs on a push to `main`, on each pull request, each Monday at 06:00 UTC, and when you start it by hand (`workflow_dispatch`). It has three jobs:

- `govulncheck` installs govulncheck v1.9.0 and runs `govulncheck ./...`.
- `trivy` scans the files of the repository for vulnerable dependencies and secrets. A CRITICAL or HIGH finding fails the job. The job uploads its SARIF report to GitHub code scanning, also when it fails.
- `gitleaks` scans the git commits for secrets.

golangci-lint, with gosec, runs only in `ci.yml`.

`release.yml` runs when you push a tag that starts with `v`. Its `test` job runs `go vet`, `go test -race -shuffle=on` and govulncheck, with the Go cache turned off. The `release` job starts only after the `test` job passes, so a failed check stops the release. A pull request that changes the release setup runs the `snapshot` and `verify` jobs instead. They build every archive, check the Homebrew formula and run `lazybw --version` on Linux, macOS and Windows. They publish nothing. [RELEASING.md](RELEASING.md) has the details.

No workflow measures coverage or runs the fuzzer.

## No test runs the real bw

A test must never run the real Bitwarden CLI (`bw`). A real `bw` uses the vault and the login of the person who runs the tests. Each package that runs commands keeps the real `bw` out of reach:

| Package | How the tests keep the real `bw` out of reach |
|---|---|
| root (`main`) | `TestMain` in `model_test.go` calls `installFakeBw` from `helpers_test.go`. It writes `fakeBwScript` as `bw` to a new temporary directory. Then it makes that directory the only `PATH` entry for the whole run. `TestMain` also sends the log output to `io.Discard`, as `main` does without `--debug`. |
| `screens` | `TestMain` in `screens/error_test.go` sets `PATH` to an empty temporary directory. A bwcmd command in a screens test fails with `exec.ErrNotFound`. A test that needs a bw result replaces a command variable (see [Package variables that tests replace](#package-variables-that-tests-replace)). |
| `bwcmd` | The package has no `TestMain`. Each test that runs `bw` calls `fakeBw(t, script)` or `useRunner(t, r)` first. `fakeBw` writes the script as `bw` to `t.TempDir()` and sets `PATH` to that directory with `t.Setenv`. `useRunner` replaces `Run`, so that no process starts. `TestMissingBw` sets `PATH` to an empty directory. |
| `session`, `totp`, `ui`, `secutil` | The tests start no process. |

The fake `bw` programs are shell scripts. On Windows, `fakeBw` and `fakeOpener` skip the test. `bwcmd/kill_unix_test.go` builds only on Unix (`//go:build unix`).

`TestPanicInUpdateStillCleansUp` in `panic_cleanup_test.go` runs a real `tea.Program`. Its `panicModel` drops the commands of the root model, so that no bw command runs.

Follow these rules in a new test:

1. In the root package, use the fake `bw`. Call `fakeBwLog(t)` to see what the fake got.
2. In `screens`, replace `unlockCmd`, `loginCmd` or `repromptUnlockCmd` with a stub. Do not change `PATH`.
3. In `bwcmd`, call `useRunner(t, r)` before the first call to `bw`, so that `r` gives the output of `bw`. To test how `execRun` starts the process, for example its environment or the kill after a timeout, call `fakeBw(t, script)` instead.

## Test files

| Package | File | What it tests |
|---|---|---|
| root (`main`) | `model_test.go` | The state machine of `RootModel`: bw status, unlock, item load, lock, quit and idle lock. Also the frame size, the footer, the `bw lock` errors on the unlock screen and the debug log, which holds no secrets. It holds the `TestMain`. |
| root (`main`) | `main_test.go` | The `--version` text, the `TEA_TRACE` check, and the `bw lock` errors that `main` prints after the exit. |
| root (`main`) | `lock_test.go` | The lock key, the idle lock, quit, retry, the clipboard clear, the exit cleanup and late bw results. |
| root (`main`) | `secrets_test.go` | Unlock drops the unlock screen, and each lock path drops the vault. |
| root (`main`) | `panic_cleanup_test.go` | The exit cleanup after a panic in `Update`. |
| root (`main`) | `lastsync_test.go` | The sync age in the footer, from bw status. |
| root (`main`) | `theme_test.go` | The warning for an unknown theme, and the restyle after a theme or background change. |
| root (`main`) | `helpers_test.go` | No tests. The fake `bw` and the helpers. |
| `bwcmd` | `exec_test.go`, `kill_unix_test.go` | How lazybw runs `bw`: arguments, environment, time limits, errors and the debug log. |
| `bwcmd` | `parser_test.go`, `types_test.go` | The parsers of the bw output, and `Description`, `FilterValue` and `Title` of an item. |
| `bwcmd` | `secret_test.go` | bwcmd sets the bw output buffers to zeros after use. |
| `screens` | `error_test.go` | The error screen. It holds the `TestMain`. |
| `screens` | `locked_test.go`, `twostep_test.go` | The unlock and login forms, and the two-step login. |
| `screens` | `vault_test.go` | The vault screen: footer, quit dialog, copy keys, overlays, drawer, toasts and filter. |
| `screens` | `filter_paging_test.go`, `selection_test.go` | The filter, paging, and the selection after a filter, a regroup or a refresh. |
| `screens` | `grouping_test.go`, `grouping_view_test.go`, `grouping_bench_test.go` | The grouping of items with the same base name. |
| `screens` | `vault_overlay_test.go` | Each overlay card owns the view, the footer and the keys until Esc closes it. |
| `screens` | `generator_test.go` | The password generator. |
| `screens` | `reprompt_test.go` | The master password re-prompt. |
| `screens` | `toast_test.go` | The lifetime of the toasts. |
| `screens` | `totp_cache_test.go` | The TOTP code of the selected item. |
| `screens` | `refresh_test.go` | The vault after a sync replaces its items. |
| `screens` | `openurl_test.go` | The `o` key, which opens the URL of an item. |
| `screens` | `usererror_test.go` | The text that lazybw shows for a bw error. |
| `screens` | `theme_test.go` | The theme picker. |
| `ui` | `drawer_test.go`, `drawer_reprompt_test.go` | The drawer for each item type, and the hidden fields of a re-prompt item. |
| `ui` | `itemrow_test.go`, `itemrow_note_test.go` | The rows of the item list and the group rows. |
| `ui` | `frame_test.go`, `hints_test.go`, `gradient_test.go` | The header, the footer, the footer hints and the gradient line. |
| `ui` | `help_test.go`, `keymap_test.go` | The help overlay and the key map. |
| `ui` | `keybindings_doc_test.go` | `docs/KEYBINDINGS.md` lists each key of `DefaultVaultKeyMap`, `DefaultGeneratorKeyMap` and `DefaultConfirmKeyMap`. |
| `ui` | `overlay_test.go`, `text_test.go` | Overlays, and the text helpers `SanitizeLine`, `Truncate` and `NoteLines`. |
| `ui` | `theme_test.go` | The themes and their contrast. It holds the `TestMain`. |
| `ui` | `cells_test.go` | No tests. The `cellsOf` helper. |
| `session` | `manager_test.go`, `clipboard_test.go` | The session state (token, lock, idle time and epoch) and the clipboard clear. |
| `totp` | `totp_test.go` | TOTP secrets and codes. |
| `secutil` | `zero_test.go` | `ZeroItem` and `ZeroItems`. |

## Test helpers

### Root package

| Helper | File | What it does |
|---|---|---|
| `fakeBwScript` | `helpers_test.go` | The fake `bw`. It answers `status` (locked, `u@t.com`), `list` (one login named Gmail), `lock` and `sync`. For the token `failLockToken`, `lock` writes `Locking failed.` (`failLockMessage`) to stderr and exits with 1. When `FAKE_BW_LOG` is set, it adds one line to that file for each run: its arguments, and ` session=<token>` when `BW_SESSION` is set. |
| `fakeBwLog(t)` | `helpers_test.go` | Sets `FAKE_BW_LOG` to a new file in `t.TempDir()`. Returns a function that reads the log. |
| `ranBwLock(log, token)` | `helpers_test.go` | Reports whether the log has the line `lock session=<token>`. |
| `update(t, m, msg)` | `helpers_test.go` | Calls `RootModel.Update`. Fails the test when the result is not a `RootModel`. |
| `unlockedVault(t, items)` | `helpers_test.go` | Returns a root model at 80x24 that shows the vault. The session token is `tok`. |
| `testItems()` | `helpers_test.go` | Returns a login (Gmail), a card (Visa) and a secure note (Keys). |
| `collectMsgs(cmd)` | `helpers_test.go` | Runs `cmd` and returns its messages. It opens each `tea.Batch` and `tea.Sequence` and runs their commands in order. It skips a command that does not return in 500 ms, such as a timer. `collectMsgsWithin(cmd, wait)` takes the wait as an argument. |
| `waitForMsg[T](cmd, wait)` | `helpers_test.go` | Runs `cmd` and all commands in its batches and sequences at the same time. Returns the first message of type `T`, or false after `wait`. A timer in `cmd` does not delay the result. |
| `hasMsg(msgs, want)`, `hasMsgType[T](msgs)` | `helpers_test.go` | Report whether `msgs` has the message `want`, or a message of type `T`. |
| `keyPress(s)` | `helpers_test.go` | Returns the `tea.KeyPressMsg` for the printable key `s`. |
| `lockResult(m, err)` | `helpers_test.go` | Returns the `bwcmd.LockResult` of the latest `bw lock` of `m`, with the error `err`. Its `Seq` is `m.lockSeq`, so `RootModel` does not drop it as the result of an earlier lock. |
| `clipboardClear` | `helpers_test.go` | The message that `session.ClearClipboard` produces. |
| `rootIn(t, state, width, height)` | `model_test.go` | Returns a root model in the given state at the given terminal size. |
| `viewLines(m)` | `model_test.go` | Renders `m` and returns the lines of the frame. |
| `captureLog(t)` | `model_test.go` | Sends the log output to a buffer that is safe to use from more than one goroutine. It restores the old output after the test. A command of an earlier test can still run and write to the log. |

This test in package `main` sends `screens.LockMsg`, the message of the lock key. It makes sure that `bw lock` runs with the session token:

```go
func TestLockMsgRunsBwLock(t *testing.T) {
	log := fakeBwLog(t)
	m := unlockedVault(t, testItems())
	m, cmd := update(t, m, screens.LockMsg{})
	if !m.sess.IsLocked() {
		t.Fatal("the session must be locked")
	}
	if _, ok := waitForMsg[bwcmd.LockResult](cmd, 5*time.Second); !ok {
		t.Fatal("the commands must return a bwcmd.LockResult")
	}
	if !ranBwLock(log(), "tok") {
		t.Errorf("bw lock did not get the token, bw log:\n%s", log())
	}
}
```

Use `waitForMsg` when the commands hold a timer or a slow command. `collectMsgs` waits up to 500 ms for each command in turn.

### screens

| Helper | File | What it does |
|---|---|---|
| `newTestVault(items)` | `vault_test.go` | Returns a `VaultModel` at 80x24. |
| `vaultAt(items, width, height)` | `vault_test.go` | Returns a `VaultModel` and sends it a `tea.WindowSizeMsg` of that size. |
| `testItems()` | `vault_test.go` | Returns a login (Gmail), a card (Visa) and a secure note (Keys). |
| `keyMsg(s)` | `vault_test.go` | Returns the `tea.KeyPressMsg` for the printable key `s`. |
| `press(t, m, key)` | `totp_cache_test.go` | Sends `key` to `m`. Then it sends back the `list.FilterMatchesMsg` that the list computes in a command. |
| `typeKeys(t, m, s)` | `totp_cache_test.go` | Calls `press` for each character of `s`. |
| `sendKey(m, key)` | `vault_overlay_test.go` | Sends `key` to `m`, and does not run the command. Use it for a key that can start `bw`. |
| `openGenerator(m)`, `openedGenerator(width, height)` | `generator_test.go` | Press `p`, and do not run the `bw generate` command. |
| `runMsgs(cmd)` | `totp_cache_test.go` | Runs `cmd` and returns its messages. It opens each `tea.Batch`. It skips a command that does not return in 50 ms. |
| `hasMsgOfType[T](msgs)` | `theme_test.go` | Reports whether `msgs` has a message of type `T`. |
| `clipboardWrites(t, cmd, candidates...)` | `vault_test.go` | Returns the candidates that `cmd` writes to the clipboard, and the fields of its `session.CopiedMsg` values. |
| `assertArea(t, out, width, height)` | `vault_test.go` | See [Frame size](#frame-size). |
| `stubUnlock(t)`, `stubLogin(t, results...)` | `twostep_test.go` | Replace `unlockCmd` and `loginCmd`, and record each call. |
| `stubReprompt(t)` | `reprompt_test.go` | Replaces `repromptUnlockCmd`. The password `right` passes. Other passwords fail as a wrong master password does. |
| `fakeOpener(t, script)` | `openurl_test.go` | Replaces `openerCommand` with a shell script. |
| `pickTheme(t, m, name)` | `theme_test.go` | Opens the theme picker, moves to `name` and presses Enter. |

This test in package `screens` types a filter and makes sure that the frame has the right size and shows only the match:

```go
func TestFilterShowsMatch(t *testing.T) {
	m := vaultAt(testItems(), 80, 24)
	m = press(t, m, keyMsg("/"))
	m = typeKeys(t, m, "visa")
	out := m.ViewContent(80, 24-ui.ChromeHeight)
	assertArea(t, out, 80, 24-ui.ChromeHeight)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "Visa") || strings.Contains(plain, "Gmail") {
		t.Errorf("the filter must show Visa and hide Gmail:\n%s", plain)
	}
}
```

A table test can send each key of a dialog in turn. `TestQuitConfirmationKeys` in `vault_test.go` sends `y`, `n`, Esc and `x` to the quit dialog through `VaultModel.Update`.

### bwcmd

| Helper | File | What it does |
|---|---|---|
| `useRunner(t, r)` | `exec_test.go` | Makes `Run` call the `Runner` `r` for one test. `r` gets the context, the environment and the arguments of each `bw` command, and it returns the stdout, the stderr and the error. No process starts, so the test also runs on Windows. |
| `envValue(env, name)` | `exec_test.go` | Returns the value of the variable `name` in an environment list, and whether the list sets it. |
| `fakeBw(t, script)` | `exec_test.go` | Writes `#!/bin/sh`, `PATH=/usr/bin:/bin` and `script` as `bw` to `t.TempDir()`. The `PATH` line lets the script run tools such as `env` and `sleep`. Makes that directory the only `PATH` entry for the test. |
| `logFile(t)` | `exec_test.go` | Returns the path of a new file in `t.TempDir()` and sets `FAKE_BW_LOG` to it. |
| `readFile(t, path)` | `exec_test.go` | Returns the content of the file. |
| `setTimeouts(t, timeout, delay)` | `exec_test.go` | Sets `shortTimeout` and `longTimeout` to `timeout`, and `waitDelay` to `delay`, for one test. |
| `captureLog(t)` | `exec_test.go` | Sends the standard logger to a buffer for one test. |
| `envOf(s)` | `exec_test.go` | Parses the output of `env` into a map. |

This test in package `bwcmd` makes sure that `Lock` runs `bw lock`:

```go
func TestLockArgs(t *testing.T) {
	var got []string
	useRunner(t, func(_ context.Context, _ []string, args ...string) ([]byte, []byte, error) {
		got = args
		return []byte("Your vault is locked."), nil, nil
	})
	if err := Lock("tok", 1)().(LockResult).Err; err != nil {
		t.Fatal(err)
	}
	if want := []string{"lock"}; !reflect.DeepEqual(got, want) {
		t.Errorf("bw got %q, want %q", got, want)
	}
}
```

These bwcmd tests cover how lazybw runs `bw`:

- `TestUnlockArgs` makes sure that `Unlock` runs `bw unlock --passwordenv BW_PASSWORD --raw` and passes the password only in `BW_PASSWORD`. `TestLoginUserArgs` does the same for `bw login`.
- `TestChildEnvStripsBwVariables` makes sure that `bw` gets no output variable such as `BW_RAW` or `BW_QUIET` from the parent process, and gets `BW_NOINTERACTION=true`.
- `TestEveryCommandSetsNoInteraction` runs each bwcmd command once and makes sure that each run gets `BW_NOINTERACTION=true`.
- `TestDebugLogHidesPrivateValues` uses `captureLog` and makes sure that the log has no email, password, two-step code or session token.
- `bwcmd/kill_unix_test.go` uses `setTimeouts`. It makes sure that a timeout stops the child process of a wrapper script. It also makes sure that `waitDelay` ends the wait for a process that the kill does not reach.

### ui

| Helper | File | What it does |
|---|---|---|
| `withTheme(t, name)`, `keepTheme(t)` | `theme_test.go` | See [Themes](#themes). |
| `withBackground(t, dark)` | `theme_test.go` | Sets `IsDark` for one test, and restores `IsDark` and the theme after it. |
| `assertArea(t, out, width, height)` | `overlay_test.go` | See [Frame size](#frame-size). |
| `cellsOf(line)`, `cellsText(cells)` | `cells_test.go` | Split a rendered line into cells, one for each terminal column, with the bold and faint state of each cell. Tests use them to find which text is bold or faint. |

`ui/theme_test.go` also makes sure that each item glyph and the header glyph has the expected rune (Unicode code point).

### session, totp and secutil

`session/manager_test.go` replaces `wallNow`, the wall clock of the session, to simulate a system suspend. `session/clipboard_test.go` tests the clipboard clear (see [Clipboard](#clipboard)). `totp/totp_test.go` tests the RFC 6238 test vectors and has the fuzz test `FuzzParse`. `secutil/zero_test.go` tests `ZeroItem` and `ZeroItems`.

## Key presses and rendered output

Bubble Tea v2 sends a key press as a `tea.KeyPressMsg`. A test builds the message directly:

```go
tea.KeyPressMsg{Code: 'j', Text: "j"}                 // j
tea.KeyPressMsg{Code: tea.KeyEnter}                   // enter
tea.KeyPressMsg{Code: tea.KeyEscape}                  // esc
tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}          // ctrl+c
tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift} // shift+down
```

`keyPress(s)` in the root package and `keyMsg(s)` in `screens` build the first form for a printable key.

A rendered string holds ANSI escape codes, the terminal codes for color and style. No `TestMain` sets `NO_COLOR` or a color profile. In lipgloss v2, `NO_COLOR` does not change what `Render` returns. Tests remove the codes with `ansi.Strip` from `github.com/charmbracelet/x/ansi` before they compare text. Tests measure the width of a line with `lipgloss.Width`, which skips the codes.

## Package variables that tests replace

Some code reads a package variable, so that a test can replace a command, a time limit or a clock:

| Variable | Package | Real value | Replaced in |
|---|---|---|---|
| `shortTimeout`, `longTimeout`, `waitDelay` | `bwcmd` | 15 s, 2 min, 2 s | `setTimeouts` in `exec_test.go` |
| `Run` | `bwcmd` | `execRun`, which starts `bw` from `PATH` | `useRunner` in `exec_test.go` |
| `unlockCmd`, `loginCmd` | `screens` | `bwcmd.Unlock`, `bwcmd.LoginUser` | `stubUnlock`, `stubLogin` in `twostep_test.go` |
| `repromptUnlockCmd` | `screens` | `bwcmd.Unlock` | `stubReprompt` in `reprompt_test.go` |
| `openerCommand` | `screens` | `systemOpener` | `fakeOpener` and `TestOpenURLReportsOpenerFailure` in `openurl_test.go` |
| `toastDuration` | `screens` | 4 s | `TestToastExpiresAndFooterShowsSyncAge` in `vault_test.go` |
| `errorToastDuration` | `screens` | 8 s | `TestErrorToastExpires` in `toast_test.go` |
| `idleCheckInterval` | root (`main`) | 30 s | `TestLockResultDoesNotStartAnotherIdleCheck` in `lock_test.go` |
| `version`, `readBuildInfo` | root (`main`) | `"dev"`, `debug.ReadBuildInfo` | `TestVersionString` in `main_test.go` |
| `wallNow` | `session` | `time.Now().Round(0)` | `TestIsIdleAfterSuspend`, `TestIsIdleBothClocksUnderTimeout` in `manager_test.go` |

Save the old value, restore it in `t.Cleanup`, and then set the new value:

```go
orig := toastDuration
t.Cleanup(func() { toastDuration = orig })
toastDuration = time.Millisecond
```

No test calls `t.Parallel`. `t.Setenv` does not work in a parallel test, and all tests of a package share the replaced variables and the theme.

## Themes

The theme is global state in package `ui`: `ui.CurrentTheme`, `ui.IsDark`, `ui.ColorHighlight` and the styles. A test that changes the theme must restore it. The root package (`theme_test.go`), `screens` (`screens/theme_test.go`) and `ui` (`ui/theme_test.go`) each have two helpers for this.

`keepTheme(t)` saves `ui.IsDark` and `ui.CurrentTheme`. A `t.Cleanup` function restores them after the test. `withTheme(t, name)` calls `keepTheme(t)` and then `ui.ApplyTheme(name)`. Inside package `ui`, the helpers use `IsDark`, `CurrentTheme` and `ApplyTheme` without the `ui.` prefix. This code is for the root package or `screens`:

```go
// Use dracula for this test.
withTheme(t, "dracula")

// Change the background for this test.
keepTheme(t)
ui.IsDark = false
ui.ApplyTheme("catppuccin-mocha")
```

The `TestMain` of the root, `screens` and `ui` packages saves the theme name, `IsDark` and `ColorHighlight` before the tests run. If all tests pass but one of the values changed, `TestMain` prints this line and fails the run:

```text
a test left the theme at <name> (dark <true|false>): use withTheme or keepTheme
```

The line does not name the test. To find it, run the tests of the package in smaller groups with `-run`.

## Frame size

`ui.ChromeHeight` is 3: the header takes 2 rows and the footer takes 1 row. A screen draws its content in `height - ui.ChromeHeight` rows. These helpers and tests make sure that a frame fits the terminal:

- `assertArea` in `screens/vault_test.go` fails when `out` does not have exactly `height` lines, or when a line is wider than `width` cells.
- `assertArea` in `ui/overlay_test.go` is stricter. Each of the `height` lines must be exactly `width` cells wide.
- `TestRootViewFillsTerminal` in `model_test.go` renders each app state at 80x24, 60x20 and 40x10. It fails when the frame does not have exactly `height` lines, or when a line is wider than the terminal.

In `screens`, `TestOverlaysFitSmallTerminals`, `TestHelpFitsAfterResize`, `TestRepromptPromptFitsSmallTerminals` and `TestSlashFrameFitsArea` use `assertArea`, among others. In `ui`, `TestRenderOverlayTinyArea` and `TestHelpOverlayResizeKeepsState` use it.

## Clipboard

lazybw writes the clipboard with `tea.SetClipboard`. Bubble Tea then sends OSC 52 to the terminal. OSC 52 is an escape sequence that asks the terminal to set the system clipboard. lazybw uses no clipboard tool, so no test needs one.

The clipboard code is in `session/clipboard.go`:

- `session.CopyToClipboard(value, field)` returns a batch of `tea.SetClipboard(value)` and a command that returns `session.CopiedMsg`. The terminal sends no reply, so `CopiedMsg` means only that lazybw sent the write.
- `session.ScheduleClipboardClear(gen)` returns a timer. After `session.ClipboardClearDelay` (60 s), it returns `session.ClipboardClearedMsg` with the copy number `Gen`.
- `session.ClearClipboard()` writes `session.ClearPlaceholder`, which is one space (`" "`), not an empty string. tmux ignores an OSC 52 write with an empty payload.

The message of `tea.SetClipboard(v)()` is a comparable value, so a test compares it with `==`. The root helper `clipboardClear` is `tea.SetClipboard(session.ClearPlaceholder)()`. This test body in package `main` makes sure that a lock after a copy clears the clipboard:

```go
m := unlockedVault(t, testItems())
m, _ = update(t, m, session.CopiedMsg{Field: session.CopyFieldPassword})
_, cmd := update(t, m, screens.LockMsg{})
if !hasMsg(collectMsgs(cmd), clipboardClear) {
	t.Error("the lock must clear the clipboard")
}
```

After the program stops, `exitCleanup.run` writes the escape sequence directly to a writer. `TestCleanupAfterExit` calls `m.pendingCleanup().run(&buf)` and compares the output with `ansi.SetSystemClipboard(session.ClearPlaceholder)`. `TestFinishPrintsLockErrors` in `main_test.go` runs the cleanup through `finish`, as `main` does.

These tests cover the clipboard messages:

| Test | File | What it tests |
|---|---|---|
| `TestClearClipboardPayloadIsNotEmpty` | `session/clipboard_test.go` | The clear payload is not empty and holds only white space. |
| `TestCopyThenLockClearsClipboard`, `TestCopyThenIdleLockClearsClipboard` | `lock_test.go` | After a `CopiedMsg`, the lock key and the idle lock clear the clipboard. |
| `TestCopyThenQuitClearsClipboardBeforeExit` | `lock_test.go` | After a `CopiedMsg`, quit and ctrl+c clear the clipboard at once and run `bw lock`. The exit is a `tea.Sequence` of the clear and `tea.Quit`. |
| `TestOlderClipboardTimerDoesNotClearNewerCopy` | `lock_test.go` | The `ClipboardClearedMsg` of an older copy does not clear a newer copy. The message of the latest copy clears it. |
| `TestClipboardTimerAfterLockDoesNothing` | `lock_test.go` | A `ClipboardClearedMsg` after a lock returns no command, because the lock already cleared the clipboard. |
| `TestClipboardTimerClearsWhileQuitting` | `lock_test.go` | A `ClipboardClearedMsg` clears the clipboard while lazybw quits. |
| `TestCleanupAfterExit` | `lock_test.go` | After a `CopiedMsg`, the exit cleanup writes the OSC 52 clear and runs `bw lock`. |
| `TestCopyNoteIsClearedLikeOtherSecrets` | `model_test.go` | `n` copies the note and sends `CopiedMsg`. The `ClipboardClearedMsg` of that copy clears the clipboard. |
| `TestUpdateRecordsExitCleanup`, `TestPanicInUpdateStillCleansUp` | `panic_cleanup_test.go` | After a `CopiedMsg`, the recorded exit cleanup includes a clipboard clear, also after a panic in `Update`. |
| `TestCopiedMsgLeavesClearToRoot` | `screens/vault_test.go` | The vault shows the copy toast for a `CopiedMsg` and starts no timer. The root model starts the clear timer for each `CopiedMsg`. |
| `TestCopyToastStaysUntilClipboardClear`, `TestClipboardClearKeepsNewerToast` | `screens/toast_test.go` | The copy toast stays until the `ClipboardClearedMsg`. The clear does not remove a newer toast. |

`TestCopyKeysNameTheField`, `TestCopyNoteCopiesFullNote` and `TestCopyKeepsControlCharacters` in `screens/vault_test.go` use `clipboardWrites` to read what the copy keys write and the fields of their `CopiedMsg` values. The re-prompt tests in `screens/reprompt_test.go` use it to make sure that a copy waits for the master password.

## Add tests for a new feature

When you add a feature, add tests in these places:

- For a new key binding, add it to the table in `TestDefaultVaultKeyMap`, `TestDefaultGeneratorKeyMap` or `TestDefaultConfirmKeyMap` (`ui/keymap_test.go`). Each test walks each field of its key map with `reflect`. It fails for a field that is not in the table, a binding with no help text, and a key that two bindings use. For a vault binding, also add it to `categorized` in `ui/keymap.go`. `TestHelpOverlayShowsEachVaultBinding` fails until the help overlay shows it. Then add the key to `docs/KEYBINDINGS.md` as inline code. `TestKeybindingsDocListsKeys` (`ui/keybindings_doc_test.go`) fails until the doc lists it.
- For a new bw command, add a test with `fakeBw` in `bwcmd`. Add the command to `TestEveryCommandSetsNoInteraction`.
- For a new item type, add cases to `TestItemDescription` (`bwcmd/types_test.go`). Add drawer and row tests such as `TestRenderDrawerIdentity` (`ui/drawer_test.go`) and `TestRenderItemRowIdentity` (`ui/itemrow_test.go`).
- For a new overlay or screen state, test its size with `assertArea` at small terminal sizes, as `TestOverlaysFitSmallTerminals` does.
- For a new path that locks or quits, add a case to `TestLockPathsRunBwLock` (`lock_test.go`) and `TestEveryLockPathDropsVault` (`secrets_test.go`).
- For a new bw result, make sure that the model drops a result with an old epoch, as `TestLateItemsResultAfterLockIsDropped` does. The epoch is a session counter that changes on each unlock and lock.
