package ui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestSanitizeLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain ASCII", "GitHub (Work)", "GitHub (Work)"},
		{"CJK and accents", "東京 café", "東京 café"},
		{"emoji with joiner", "a👩‍💻b", "a👩‍💻b"},
		{"CRLF", "first\r\nsecond", "first second"},
		{"CR", "first\rsecond", "first second"},
		{"LF", "first\nsecond", "first second"},
		{"tab", "a\tb", "a    b"},
		{"NUL, BEL and BS", "a\x00b\x07c\x08d", "abcd"},
		{"other C0", "a\x01\x1f\x0bb", "ab"},
		{"DEL", "a\x7fb", "ab"},
		{"C1 NEL and CSI", "a\u0085b\u009bc", "abc"},
		{"SGR hidden text", "a\x1b[8mhidden\x1b[0mb", "ahiddenb"},
		{"OSC 8 link", "\x1b]8;;https://evil.example\x07Spoof\x1b]8;;\x07", "Spoof"},
		{"OSC 52 clipboard", "a\x1b]52;c;ZXZpbA==\x07b", "ab"},
		{"cursor movement", "a\x1b[2Ab\x1b[10Gc", "abc"},
		{"ESC sequence", "a\x1bcb", "ab"},
		{"OSC with ST", "a\x1b]0;title\x1b\\b", "ab"},
		// A sequence with no end loses only its ESC byte, so that no text
		// is hidden.
		{"unterminated OSC", "codes \x1b]0;x\ncode1\ncode2", "codes ]0;x code1 code2"},
		{"unterminated DCS", "a \x1bPdata\nb", "a Pdata b"},
		{"unterminated APC", "a \x1b_apc\nb", "a _apc b"},
		{"cancelled OSC", "a\x1b]0;shown\x1b[0mb", "a]0;shownb"},
		{"unterminated CSI", "a\x1b[31", "a[31"},
		{"trailing ESC", "price 5\x1b", "price 5"},
		{"ESC before a line break", "price 5\x1b\nline2", "price 5 line2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeLine(tt.in)
			if got != tt.want {
				t.Errorf("SanitizeLine(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("SanitizeLine(%q) = %q is not valid UTF-8", tt.in, got)
			}
		})
	}
}

func TestSanitizeLineInvalidUTF8(t *testing.T) {
	for _, in := range []string{"a\xffb", "a\xe6\x97b", "a\xc2b", "\x9b31mx"} {
		got := SanitizeLine(in)
		if !utf8.ValidString(got) {
			t.Errorf("SanitizeLine(%q) = %q is not valid UTF-8", in, got)
		}
		if strings.IndexByte(got, 0x1b) >= 0 || strings.IndexByte(got, 0x9b) >= 0 {
			t.Errorf("SanitizeLine(%q) = %q keeps a control byte", in, got)
		}
	}
}

func TestSanitizeTextKeepsLineBreaks(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"line breaks", "one\r\ntwo\rthree\nfour\x1b[31m\tred\x07", "one\ntwo\nthree\nfour    red"},
		// A broken sequence must not hide the lines after it.
		{"unterminated OSC", "Recovery codes \x1b]0;x\ncode1\r\ncode2", "Recovery codes ]0;x\ncode1\ncode2"},
		{"unterminated DCS", "line1 \x1bPdata\nline2\nline3", "line1 Pdata\nline2\nline3"},
		{"unterminated APC", "line1 \x1b_apc\rline2", "line1 _apc\nline2"},
		{"ESC before a line break", "price 5\x1b\nline2", "price 5\nline2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeText(tt.in); got != tt.want {
				t.Errorf("SanitizeText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width int
	}{
		{"ASCII", strings.Repeat("a", 50), 10},
		{"CJK", strings.Repeat("漢字", 20), 9},
		{"accents", strings.Repeat("é", 30), 7},
		{"styled", StyleFaint.Render(strings.Repeat("x", 40)), 12},
		{"one cell", "abc", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.in, tt.width)
			if w := lipgloss.Width(got); w > tt.width {
				t.Errorf("width = %d, want at most %d: %q", w, tt.width, got)
			}
			if !utf8.ValidString(got) {
				t.Errorf("result is not valid UTF-8: %q", got)
			}
			if !strings.HasSuffix(ansi.Strip(got), "…") {
				t.Errorf("cut string must end with an ellipsis: %q", got)
			}
		})
	}
	if got := Truncate("short", 10); got != "short" {
		t.Errorf("Truncate of a short string = %q, want it unchanged", got)
	}
	if got := Truncate("abc", 0); got != "" {
		t.Errorf("Truncate to 0 cells = %q, want empty", got)
	}
}

func TestNoteLines(t *testing.T) {
	long := strings.TrimSpace(strings.Repeat("lorem ipsum dolor sit amet ", 6))
	lines := NoteLines(long+"\r\n\r\nlast\tline", 40)
	if len(lines) < 4 {
		t.Fatalf("got %d lines, want the long line wrapped: %q", len(lines), lines)
	}
	for _, l := range lines {
		if w := lipgloss.Width(l); w > 38 {
			t.Errorf("line is %d cells wide, want at most 38: %q", w, l)
		}
		if strings.ContainsAny(l, "\r\t\n") {
			t.Errorf("line has control characters: %q", l)
		}
	}
	if lines[len(lines)-2] != "" {
		t.Errorf("the empty line of the note must stay, got %q", lines)
	}
	joined := strings.Join(strings.Fields(strings.Join(lines, " ")), " ")
	if !strings.Contains(joined, long) || !strings.HasSuffix(joined, "last line") {
		t.Errorf("wrapped lines lost text: %q", joined)
	}

	// A sequence with no end must not hide the lines after it.
	if got := NoteLines("Recovery codes \x1b]0;x\ncode1\ncode2", 40); len(got) != 3 || got[2] != "code2" {
		t.Errorf("NoteLines with an unterminated OSC = %q, want 3 lines", got)
	}

	// A character wider than the space must not make a line too wide.
	for _, l := range NoteLines("漢字", 3) {
		if lipgloss.Width(l) > 1 {
			t.Errorf("line %q is wider than 1 cell", l)
		}
	}
}
