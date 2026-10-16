package screens

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// fakeOpener replaces openerCommand for one test with a command that runs
// script. It returns the targets that the opener got and the commands that
// it built.
func fakeOpener(t *testing.T, script string) (*[]string, *[]*exec.Cmd) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake opener is a shell script")
	}
	path := filepath.Join(t.TempDir(), "opener")
	//nolint:gosec // the fake opener must be executable
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o700); err != nil {
		t.Fatal(err)
	}
	targets := &[]string{}
	cmds := &[]*exec.Cmd{}
	orig := openerCommand
	t.Cleanup(func() { openerCommand = orig })
	openerCommand = func(target string) *exec.Cmd {
		*targets = append(*targets, target)
		c := exec.Command(path, target) //nolint:gosec // test-owned script
		*cmds = append(*cmds, c)
		return c
	}
	return targets, cmds
}

func TestBrowserURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
		err  error
	}{
		{"https://example.com/login", "https://example.com/login", nil},
		{"http://example.com", "http://example.com", nil},
		{"HTTPS://Example.com", "https://Example.com", nil},
		{"example.com", "https://example.com", nil},
		{" example.com/path?q=1 ", "https://example.com/path?q=1", nil},
		{"example.com:8443/app", "https://example.com:8443/app", nil},
		{"192.168.1.1:8080", "https://192.168.1.1:8080", nil},
		{"file:///etc/passwd", "", errUnsupportedScheme},
		{"file:/etc/passwd", "", errUnsupportedScheme},
		{"smb://server/share", "", errUnsupportedScheme},
		{"javascript:alert(1)", "", errUnsupportedScheme},
		{"mailto:u@t.com", "", errUnsupportedScheme},
		{"", "", errNoHost},
		{"https://", "", errNoHost},
		// A URI with a scheme that does not parse gets no "https://" in
		// front, which would make its scheme the host.
		{"https://ex ample.com/", "", errInvalidURL},
		{"http://example.com:abc/login", "", errInvalidURL},
		{"smb://ser ver/share", "", errInvalidURL},
		{"ex ample.com", "", errInvalidURL},
		// The query gets the escapes that url.URL.String leaves out.
		{`https://example.com/?q="x" & calc`, "https://example.com/?q=%22x%22%20&%20calc", nil},
		{"https://example.com/?a=<b>|^`{}\\", "https://example.com/?a=%3Cb%3E%7C%5E%60%7B%7D%5C", nil},
		{"https://example.com/?q=%20ok&r=a+b", "https://example.com/?q=%20ok&r=a+b", nil},
		{"example.com/?q=\u00fc", "https://example.com/?q=%C3%BC", nil},
		{"https://example.com/p a#f g", "https://example.com/p%20a#f%20g", nil},
	}
	for _, tt := range tests {
		got, err := browserURL(tt.in)
		if got != tt.want || (tt.err != nil && !errors.Is(err, tt.err)) || (tt.err == nil && err != nil) {
			t.Errorf("browserURL(%q) = %q, %v; want %q, %v", tt.in, got, err, tt.want, tt.err)
		}
	}
}

func TestOpenURLWaitsForOpener(t *testing.T) {
	targets, cmds := fakeOpener(t, "exit 0\n")
	msg := openURL("example.com")()
	res, ok := msg.(OpenURLResult)
	if !ok || res.Err != nil {
		t.Fatalf("openURL returned %#v, want OpenURLResult with no error", msg)
	}
	if !reflect.DeepEqual(*targets, []string{"https://example.com"}) {
		t.Errorf("opener got %q, want the URL with https added", *targets)
	}
	// Only Wait sets ProcessState. A set ProcessState means that the child
	// was reaped and is no zombie.
	if c := (*cmds)[0]; c.ProcessState == nil || !c.ProcessState.Exited() {
		t.Error("openURL must wait for the opener")
	}
}

func TestOpenURLRejectsFileURI(t *testing.T) {
	targets, _ := fakeOpener(t, "exit 0\n")
	msg := openURL("file:///etc/passwd")()
	if res, ok := msg.(OpenURLResult); !ok || !errors.Is(res.Err, errUnsupportedScheme) {
		t.Errorf("openURL returned %#v, want errUnsupportedScheme", msg)
	}
	if len(*targets) != 0 {
		t.Errorf("the opener ran for %q", *targets)
	}
}

func TestOpenURLReportsOpenerFailure(t *testing.T) {
	fakeOpener(t, "exit 4\n")
	res, _ := openURL("https://example.com")().(OpenURLResult)
	var exitErr *exec.ExitError
	if !errors.As(res.Err, &exitErr) {
		t.Errorf("err = %v, want the exit error of the opener", res.Err)
	}

	orig := openerCommand
	t.Cleanup(func() { openerCommand = orig })
	openerCommand = func(target string) *exec.Cmd {
		return exec.Command(filepath.Join(t.TempDir(), "missing"), target) //nolint:gosec // test path
	}
	if res, _ := openURL("https://example.com")().(OpenURLResult); res.Err == nil {
		t.Error("a missing opener must give an error")
	}
}

func TestSystemOpener(t *testing.T) {
	want := map[string][]string{
		"darwin":  {"open", "https://example.com"},
		"windows": {"rundll32", "url.dll,FileProtocolHandler", "https://example.com"},
	}[runtime.GOOS]
	if want == nil {
		want = []string{"xdg-open", "https://example.com"}
	}
	if got := systemOpener("https://example.com").Args; !reflect.DeepEqual(got, want) {
		t.Errorf("opener args = %q, want %q", got, want)
	}
}
