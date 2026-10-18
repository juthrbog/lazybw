package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// tabSpaces replaces a tab in sanitized text. lipgloss expands a tab to the
// same 4 spaces.
const tabSpaces = "    "

// SanitizeLine makes vault text safe to draw on one line. It removes ANSI
// escape sequences (SGR, OSC 8 links and others), replaces "\r\n", "\r" and
// "\n" with a space, expands tabs to spaces, and removes all other C0 and C1
// control characters. A broken escape sequence cannot hide text: a sequence
// ends at a line break, and when it has no end, only its ESC byte is
// removed. The result is valid UTF-8: an invalid byte in the text is
// replaced with U+FFFD.
//
// Use it only for display. Copy actions must use the stored value.
func SanitizeLine(s string) string {
	return sanitize(s, false)
}

// SanitizeText is SanitizeLine for text with more than one line, such as
// notes. It keeps the line breaks: "\r\n" and "\r" become "\n".
func SanitizeText(s string) string {
	return sanitize(s, true)
}

func sanitize(s string, keepNewlines bool) string {
	if isPrintableASCII(s) {
		return s
	}

	lineBreak := " "
	if keepNewlines {
		lineBreak = "\n"
	}

	var b strings.Builder
	b.Grow(len(s))
	lineEnd := -1 // the index of the next line break, or len(s) when none
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == ansi.ESC:
			if lineEnd < i {
				lineEnd = nextLineBreak(s, i)
			}
			size = escapeLen(s[i:lineEnd])
		case r == '\r':
			// "\r\n" is one line break.
			if i+1 < len(s) && s[i+1] == '\n' {
				size = 2
			}
			b.WriteString(lineBreak)
		case r == '\n':
			b.WriteString(lineBreak)
		case r == '\t':
			b.WriteString(tabSpaces)
		case r < 0x20 || (r >= 0x7f && r <= 0x9f):
			// Other C0 controls, DEL and C1 controls.
		default:
			// An invalid byte decodes as utf8.RuneError, which writes U+FFFD.
			b.WriteRune(r)
		}
		i += size
	}
	return b.String()
}

// nextLineBreak returns the index of the first "\r" or "\n" in s at or
// after i, or len(s) when there is none.
func nextLineBreak(s string, i int) int {
	if j := strings.IndexAny(s[i:], "\r\n"); j >= 0 {
		return i + j
	}
	return len(s)
}

// escapeLen returns the length in bytes of the escape sequence at the start
// of s, which starts with ESC and ends before the next line break. A
// sequence that does not end in s counts as only its ESC byte, so that the
// text after it stays visible. So does a string sequence (OSC, DCS, SOS, PM
// or APC) that another control cancels before its terminator.
func escapeLen(s string) int {
	seq, _, n, state := ansi.DecodeSequence(s, ansi.NormalState, nil)
	if state != ansi.NormalState || n < 1 {
		return 1
	}
	if isStringSequence(seq) && !hasStringTerminator(seq) {
		return 1
	}
	return n
}

// isStringSequence reports whether seq starts with the 7-bit introducer of
// an OSC, DCS, SOS, PM or APC sequence.
func isStringSequence(seq string) bool {
	return len(seq) >= 2 && strings.IndexByte("]PX^_", seq[1]) >= 0
}

// hasStringTerminator reports whether the string sequence seq ends with ST,
// or with BEL for an OSC sequence.
func hasStringTerminator(seq string) bool {
	if seq[1] == ']' && strings.HasSuffix(seq, "\a") {
		return true
	}
	return strings.HasSuffix(seq, "\x1b\\") || strings.HasSuffix(seq, "\x9c")
}

// isPrintableASCII reports whether s has only printable ASCII characters,
// so that it needs no sanitizing.
func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

// Truncate cuts s to at most width terminal cells and ends a cut string
// with "…". It measures display width, not bytes, so it never splits a
// multi-byte character, a wide character or an ANSI escape sequence.
func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(s, width, "…")
}

// NoteLines splits a note into the lines that the drawer shows at the given
// drawer width. It sanitizes each line and wraps it to width-2 cells, the
// width that is left after the 2-cell indent. An empty line in the note
// stays an empty line.
func NoteLines(notes string, width int) []string {
	w := max(1, width-2)
	var out []string
	for _, line := range strings.Split(SanitizeText(notes), "\n") {
		if line == "" {
			out = append(out, "")
			continue
		}
		for _, l := range strings.Split(ansi.Wrap(line, w, ""), "\n") {
			// Wrap cannot split a character that is wider than w. Cut it,
			// so that no line is wider than the drawer.
			out = append(out, ansi.Truncate(l, w, ""))
		}
	}
	return out
}
