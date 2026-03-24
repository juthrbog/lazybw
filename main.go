package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	debug := flag.Bool("debug", false, "write debug log to $XDG_CACHE_HOME/lazybw/debug.log")
	ver := flag.Bool("version", false, "print version and exit")
	idleTimeout := flag.Duration("idle-timeout", 15*time.Minute, "lock vault after this duration of inactivity")
	flag.Parse()

	if *ver {
		fmt.Println("lazybw", version)
		return
	}

	if _, err := exec.LookPath("bw"); err != nil {
		fmt.Fprintln(os.Stderr, "lazybw requires the Bitwarden CLI (bw) but it was not found in your PATH.")
		fmt.Fprintln(os.Stderr, "Install it from: https://bitwarden.com/help/cli/#download-and-install")
		os.Exit(1)
	}

	if *debug {
		if err := setupDebugLog(); err != nil {
			fmt.Fprintln(os.Stderr, "failed to open debug log:", err)
			os.Exit(1)
		}
	}

	p := tea.NewProgram(
		NewRootModel(*idleTimeout),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return err
	}

	f, err := os.OpenFile(
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
