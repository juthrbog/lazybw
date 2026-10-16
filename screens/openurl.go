package screens

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

// OpenURLResult is the result of an attempt to open a URL in the browser.
type OpenURLResult struct {
	Err error
}

var (
	errNoHost            = errors.New("the URL has no host")
	errInvalidURL        = errors.New("the URL is not valid")
	errUnsupportedScheme = errors.New("lazybw opens only http and https URLs")
)

// openerCommand returns the command that opens target in the default
// browser. Tests replace it.
var openerCommand = systemOpener

// systemOpener returns the command of the system that opens target.
func systemOpener(target string) *exec.Cmd {
	// G204: target is an http or https URL that browserURL built, so it
	// cannot start with "-", and no shell runs it.
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", target) //nolint:gosec // see the comment above
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target) //nolint:gosec // see the comment above
	default:
		return exec.Command("xdg-open", target) //nolint:gosec // see the comment above
	}
}

// openURL returns a command that opens raw in the default browser. It
// opens only http and https URLs, and it adds "https://" when raw has no
// scheme.
func openURL(raw string) tea.Cmd {
	target, err := browserURL(raw)
	if err != nil {
		return func() tea.Msg { return OpenURLResult{Err: err} }
	}
	return func() tea.Msg {
		cmd := openerCommand(target)
		if err := cmd.Start(); err != nil {
			return OpenURLResult{Err: err}
		}
		// Wait reaps the opener, so that it does not stay as a zombie.
		return OpenURLResult{Err: cmd.Wait()}
	}
}

// browserURL checks raw and returns the URL to open. A URI without "://"
// that does not parse, such as "192.168.1.1:8080", gets "https://" in front
// too.
func browserURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	switch {
	case err != nil && strings.Contains(raw, "://"):
		// raw names a scheme. With "https://" in front, the scheme of raw
		// becomes the host.
		return "", errInvalidURL
	case err != nil || !hasScheme(u):
		if u, err = url.Parse("https://" + raw); err != nil {
			return "", errInvalidURL
		}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errUnsupportedScheme
	}
	if u.Host == "" {
		return "", errNoHost
	}
	// String escapes the path and the fragment, but not the query.
	u.RawQuery = escapeQuery(u.RawQuery)
	return u.String(), nil
}

// escapeQuery percent-encodes each byte of the query q that RFC 3986 does
// not allow in a query, such as a space, a quote or "|". It keeps "%", so
// that the escapes in q stay as they are.
func escapeQuery(q string) string {
	var b strings.Builder
	for i := range len(q) {
		if c := q[i]; queryByte(c) {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// queryByte reports whether c can stay as it is in a query: RFC 3986
// allows it, or it is "%".
func queryByte(c byte) bool {
	switch {
	case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		return true
	}
	return strings.IndexByte("-._~!$&'()*+,;=:@/?%", c) >= 0
}

// hasScheme reports whether the parsed URI u names a scheme. url.Parse
// reads "example.com:8080" as the scheme "example.com". When the part
// after the colon starts with a digit, the URI is a host and a port.
func hasScheme(u *url.URL) bool {
	if u.Scheme == "" {
		return false
	}
	r := []rune(u.Opaque)
	return len(r) == 0 || !unicode.IsDigit(r[0])
}
