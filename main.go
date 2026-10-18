package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/screens"
	"github.com/juthrbog/lazybw/ui"
)

var version = "dev"

func main() {
	debug := flag.Bool("debug", false, "write debug log to $XDG_CACHE_HOME/lazybw/debug.log")
	ver := flag.Bool("version", false, "print version and exit")
	idleTimeout := flag.Duration("idle-timeout", 15*time.Minute, "lock vault after this duration of inactivity (minimum 30s)")
	theme := flag.String("theme", "", "color theme ("+strings.Join(ui.ThemeNames, ", ")+")")
	flag.Parse()

	if *ver {
		fmt.Println(versionString())
		return
	}

	// Apply theme: flag > env var > default (catppuccin-mocha).
	themeName := *theme
	if themeName == "" {
		themeName = os.Getenv("LAZYBW_THEME")
	}
	setTheme(themeName, os.Stderr)

	if *idleTimeout < 30*time.Second {
		fmt.Fprintf(os.Stderr, "Warning: idle-timeout minimum is 30s (got %s), using 30s\n", *idleTimeout)
		*idleTimeout = 30 * time.Second
	}

	if _, err := exec.LookPath("bw"); err != nil {
		fmt.Fprintln(os.Stderr, "lazybw requires the Bitwarden CLI (bw) but it was not found in your PATH.")
		fmt.Fprintln(os.Stderr, "Install it from: https://bitwarden.com/help/cli/#download-and-install")
		os.Exit(1)
	}

	// Without --debug, drop log output. The default target is stderr,
	// which would draw over the TUI.
	log.SetOutput(io.Discard)
	if *debug {
		if err := setupDebugLog(); err != nil {
			fmt.Fprintln(os.Stderr, "failed to open debug log:", err)
			os.Exit(1)
		}
	}

	root := NewRootModel(*idleTimeout)
	p := newProgram(root, os.Stderr)

	// Bubble Tea turns SIGTERM into a quit, but it does not handle SIGHUP,
	// which the system sends when the terminal closes. Quit on SIGHUP too,
	// so that the cleanup below runs.
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		if _, ok := <-hup; ok {
			p.Quit()
		}
	}()

	final, err := p.Run()
	signal.Stop(hup)
	close(hup)
	// Ignore further signals until the cleanup is done. A child process
	// inherits the ignored signals, so a closed terminal cannot stop the
	// `bw lock` that the cleanup runs.
	signal.Ignore(syscall.SIGHUP, syscall.SIGTERM, os.Interrupt)
	finish(final, root.lastCleanup, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// readBuildInfo returns the build info of the binary. Tests replace it.
var readBuildInfo = debug.ReadBuildInfo

// versionString returns the text that --version prints. The release build
// sets version with -ldflags. Other builds have no ldflags, so for the
// "dev" version it uses the module version from the build info. A go
// install build has the version of the module, for example v0.3.0. A go
// build in a git clone gets a version from git: the tag of the commit or a
// pseudo-version, with "+dirty" for changed files. A build with no VCS
// data, for example with -buildvcs=false or from a source archive, reports
// "(devel)" and stays "dev".
func versionString() string {
	v := version
	if v == "dev" {
		if info, ok := readBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
	}
	return "lazybw " + v
}

// newProgram returns the Bubble Tea program for m. tea.NewProgram opens
// the TEA_TRACE file, so newProgram drops the variable first and writes
// the warning to stderr.
func newProgram(m tea.Model, stderr io.Writer) *tea.Program {
	dropTeaTrace(stderr)
	return tea.NewProgram(m)
}

// finish runs the exit cleanup after the program stops, and then writes
// the bw lock errors of the exit to stderr. final is the model that
// tea.Program.Run returned, and last is the cleanup that the last Update
// recorded. The cleanup clears the clipboard through stdout.
func finish(final tea.Model, last *exitCleanup, stdout, stderr io.Writer) {
	lockErr := cleanupFor(final, last).run(stdout)
	printLockErrors(stderr, final, lockErr)
}

// dropTeaTrace unsets TEA_TRACE and writes a warning to w when the
// variable is set. With TEA_TRACE set, tea.NewProgram opens that file, and
// Bubble Tea writes all input and output to it. That includes each key of
// the master password and the OSC 52 clipboard writes.
func dropTeaTrace(w io.Writer) {
	if _, ok := os.LookupEnv("TEA_TRACE"); !ok {
		return
	}
	_ = os.Unsetenv("TEA_TRACE")
	_, _ = fmt.Fprintln(w, "Warning: lazybw ignores TEA_TRACE. A Bubble Tea trace file records every key press, including the master password, and every copied value.")
}

// printLockErrors writes the bw lock errors of the exit to w: the error of
// the lock on quit, which the final model holds, and cleanupErr, the error
// of the lock in the exit cleanup. main calls it after the terminal is
// restored, so that the text stays on the screen. The text can hold a bw
// message, so control characters are removed.
func printLockErrors(w io.Writer, final tea.Model, cleanupErr error) {
	var texts []string
	if m, ok := final.(RootModel); ok {
		texts = append(texts, m.quitError)
	}
	texts = append(texts, screens.LockErrorText(cleanupErr))
	for _, text := range texts {
		if text != "" {
			_, _ = fmt.Fprintln(w, ui.SanitizeLine(text))
		}
	}
}

// setTheme applies the theme from --theme or LAZYBW_THEME. An empty name
// keeps the default theme. For a name that is not a theme, it writes a
// warning with the theme names to w and applies the default theme.
func setTheme(name string, w io.Writer) {
	if name == "" {
		return
	}
	if !ui.IsTheme(name) {
		_, _ = fmt.Fprintf(w, "Warning: unknown theme %q, using %s. Themes: %s\n",
			name, ui.DefaultTheme, strings.Join(ui.ThemeNames, ", "))
		name = ui.DefaultTheme
	}
	ui.ApplyTheme(name)
}

// setupDebugLog sends the log output to lazybw/debug.log in
// $XDG_CACHE_HOME, or in ~/.cache when XDG_CACHE_HOME is not set. The file
// has mode 0600. RootModel logs state changes, and bwcmd logs failed bw
// commands. Neither logs a secret.
func setupDebugLog() error {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		cacheDir = filepath.Join(home, ".cache")
	}

	logDir := filepath.Join(cacheDir, "lazybw")
	if err := os.MkdirAll(logDir, 0o700); err != nil { //nolint:gosec // the path is $XDG_CACHE_HOME or ~/.cache, plus fixed names
		return err
	}

	f, err := os.OpenFile( //nolint:gosec // the path is $XDG_CACHE_HOME or ~/.cache, plus fixed names
		filepath.Join(logDir, "debug.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o600,
	)
	if err != nil {
		return err
	}

	log.SetOutput(f)
	log.SetFlags(log.Ltime | log.Lshortfile)
	return nil
}
