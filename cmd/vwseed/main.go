package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/juthrbog/lazybw/bwcmd"
)

// httpClient skips TLS verification for the self-signed Caddy cert.
var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}, //nolint:gosec // dev-only self-signed cert
	},
}

const (
	serverURL     = "https://localhost:8443"
	email         = "test@lazybw.dev"
	password      = "master-password-for-dev"
	kdfIterations = 600_000
	bwDataDir     = ".bw-dev" // isolated bw CLI config directory
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// bwEnv returns the base environment for all bw subprocess calls,
// with BITWARDENCLI_APPDATA_DIR set to isolate dev config from production.
func bwEnv() []string {
	abs, err := filepath.Abs(bwDataDir)
	if err != nil {
		abs = bwDataDir
	}
	return append(os.Environ(),
		"BITWARDENCLI_APPDATA_DIR="+abs,
		"NODE_TLS_REJECT_UNAUTHORIZED=0", // accept Caddy's self-signed cert
	)
}

func run() error {
	if err := os.MkdirAll(bwDataDir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", bwDataDir, err)
	}

	step("Waiting for Vaultwarden")
	if err := waitForServer(serverURL+"/alive", 30, time.Second); err != nil {
		return err
	}
	done()

	step("Registering account")
	if err := registerAccount(serverURL, email, password, kdfIterations); err != nil {
		return err
	}
	done()

	token, err := authenticate()
	if err != nil {
		return err
	}

	step("Syncing vault")
	if err := bwWithSession(token, "sync"); err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	done()

	step("Checking existing items")
	existing, err := bwCaptureWithSession(token, "list", "items")
	if err != nil {
		return fmt.Errorf("list items: %w", err)
	}
	if strings.TrimSpace(existing) != "[]" && strings.TrimSpace(existing) != "" {
		fmt.Println(" vault already has items, skipping seed")
		return lockAndSummary(token, nil)
	}
	done()

	step("Generating seed items")
	items, err := seedItems()
	if err != nil {
		return err
	}
	done()

	for _, item := range items {
		step(fmt.Sprintf("Creating %s", item.Name))
		itemJSON, err := marshalItemForCreate(item)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", item.Name, err)
		}
		encoded := base64.StdEncoding.EncodeToString(itemJSON)
		if _, err := bwCaptureWithSession(token, "create", "item", encoded); err != nil {
			return fmt.Errorf("create %s: %w", item.Name, err)
		}
		done()
	}

	return lockAndSummary(token, items)
}

func lockAndSummary(token string, items []bwcmd.Item) error {
	step("Locking vault")
	if err := bwWithSession(token, "lock"); err != nil {
		return err
	}
	done()

	fmt.Println()
	fmt.Println("Seed complete!")
	fmt.Printf("  Server:   %s\n", serverURL)
	fmt.Printf("  Email:    %s\n", email)
	fmt.Printf("  Password: %s\n", password)
	if copyToClipboard(password) {
		fmt.Println("             (copied to clipboard)")
	}
	if len(items) > 0 {
		fmt.Printf("  Items:    %d\n", len(items))
	}
	fmt.Println()
	fmt.Printf("Run lazybw against local Vaultwarden:\n")
	fmt.Printf("  BITWARDENCLI_APPDATA_DIR=%s go run .\n", bwDataDir)
	return nil
}

func waitForServer(url string, retries int, delay time.Duration) error {
	for range retries {
		resp, err := httpClient.Get(url) //nolint:gosec // url is a hardcoded dev-only constant
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("vaultwarden not responding at %s after %d retries", url, retries)
}

func bw(args ...string) error {
	cmd := exec.Command("bw", args...) //nolint:gosec // bw is a trusted binary, args are hardcoded
	cmd.Env = bwEnv()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func bwWithSession(token string, args ...string) error {
	cmd := exec.Command("bw", args...) //nolint:gosec // bw is a trusted binary, args are hardcoded
	cmd.Env = append(bwEnv(), "BW_SESSION="+token)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func bwCapture(password string, args ...string) (string, error) {
	cmd := exec.Command("bw", args...) //nolint:gosec // bw is a trusted binary, args are hardcoded
	cmd.Env = append(bwEnv(), "BW_PASSWORD="+password) //nolint:gosec // dev-only password from constant
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func bwCaptureRaw(args ...string) (string, error) {
	cmd := exec.Command("bw", args...) //nolint:gosec // bw is a trusted binary, args are hardcoded
	cmd.Env = bwEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func bwCaptureWithSession(token string, args ...string) (string, error) {
	cmd := exec.Command("bw", args...) //nolint:gosec // bw is a trusted binary, args are hardcoded
	cmd.Env = append(bwEnv(), "BW_SESSION="+token)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// authenticate checks bw status and logs in or unlocks as needed.
// Returns a session token.
func authenticate() (string, error) {
	step("Checking bw status")
	statusJSON, err := bwCaptureRaw("status")
	if err != nil {
		// First run — bw not configured yet.
		done()
		step("Configuring bw CLI")
		if err := bw("config", "server", serverURL); err != nil {
			return "", err
		}
		done()

		step("Logging in")
		token, err := bwCapture(password, "login", email, "--passwordenv", "BW_PASSWORD", "--raw")
		if err != nil {
			return "", fmt.Errorf("login: %w", err)
		}
		done()
		return token, nil
	}

	var status struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(statusJSON), &status); err != nil {
		return "", fmt.Errorf("parse status: %w", err)
	}
	done()

	switch status.Status {
	case "unauthenticated":
		step("Configuring bw CLI")
		// Config might already be set; ignore errors.
		_ = bw("config", "server", serverURL)
		done()

		step("Logging in")
		token, err := bwCapture(password, "login", email, "--passwordenv", "BW_PASSWORD", "--raw")
		if err != nil {
			return "", fmt.Errorf("login: %w", err)
		}
		done()
		return token, nil

	case "locked":
		step("Unlocking vault")
		token, err := bwCapture(password, "unlock", "--passwordenv", "BW_PASSWORD", "--raw")
		if err != nil {
			return "", fmt.Errorf("unlock: %w", err)
		}
		done()
		return token, nil

	case "unlocked":
		// Already unlocked — need a session token. Lock and re-unlock.
		step("Re-unlocking vault")
		if err := bw("lock"); err != nil {
			return "", fmt.Errorf("lock: %w", err)
		}
		token, err := bwCapture(password, "unlock", "--passwordenv", "BW_PASSWORD", "--raw")
		if err != nil {
			return "", fmt.Errorf("unlock: %w", err)
		}
		done()
		return token, nil

	default:
		return "", fmt.Errorf("unexpected bw status: %s", status.Status)
	}
}

// marshalItemForCreate converts a bwcmd.Item to JSON suitable for `bw create item`.
// The bw CLI requires organizationId and folderId to be null (not empty string)
// when creating personal vault items, otherwise it tries to look up an org key.
func marshalItemForCreate(item bwcmd.Item) ([]byte, error) {
	raw, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	// Nullify empty ID fields that the bw CLI misinterprets as real IDs.
	for _, key := range []string{"id", "organizationId", "folderId"} {
		if v, ok := m[key].(string); ok && v == "" {
			m[key] = nil
		}
	}
	return json.Marshal(m)
}

// copyToClipboard attempts to copy text to the system clipboard using
// the first available clipboard tool. Returns true on success, false
// if no tool is available or the copy fails. This is best-effort —
// clipboard access is a convenience, not a requirement.
func copyToClipboard(text string) bool {
	candidates := []struct {
		name string
		args []string
	}{
		{"pbcopy", nil},
		{"wl-copy", nil},
		{"xclip", []string{"-selection", "clipboard"}},
		{"xsel", []string{"--clipboard", "--input"}},
	}
	for _, c := range candidates {
		path, err := exec.LookPath(c.name)
		if err != nil {
			continue
		}
		cmd := exec.Command(path, c.args...) //nolint:gosec // clipboard tool path is from LookPath, args are hardcoded
		cmd.Stdin = strings.NewReader(text)
		if cmd.Run() == nil {
			return true
		}
	}
	return false
}

func step(msg string) {
	fmt.Printf("  -> %s...", msg)
}

func done() {
	fmt.Println(" done")
}
