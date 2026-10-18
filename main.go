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
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
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
		fmt.Println("lazybw", version)
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
	p := tea.NewProgram(root)

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
	cleanupFor(final, root.lastCleanup).run(os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
	if err := os.MkdirAll(logDir, 0o700); err != nil { //nolint:gosec // path is built from os.UserCacheDir + hardcoded name
		return err
	}

	f, err := os.OpenFile( //nolint:gosec // path is built from os.UserCacheDir + hardcoded name
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
