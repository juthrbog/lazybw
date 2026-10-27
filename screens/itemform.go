package screens

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/totp"
	"github.com/juthrbog/lazybw/ui"
)

// fieldKind is the item value that a line of the item form edits.
type fieldKind int

const (
	fieldName fieldKind = iota
	fieldUsername
	fieldPassword
	fieldTOTP
	fieldURI
)

// formLabelWidth is the width of the label column of the item form.
const formLabelWidth = 10

// formNotesMinHeight is the least number of rows of the notes field.
const formNotesMinHeight = 3

// Texts of the form checks.
const (
	nameRequiredText = "Enter a name"
	badTOTPText      = "Enter an otpauth:// URI or a base32 key"
)

// formLine is a one-line field of the item form.
type formLine struct {
	kind  fieldKind
	label string
	input textinput.Model
	// uri is the URI object of the item that a URI line edits, or nil for a
	// new URI. Its other keys, such as "match", stay when the form saves.
	uri map[string]any
}

// itemForm is the full-screen form that creates an item. It edits a copy of
// the item JSON, so that a save keeps the keys that the form does not show.
type itemForm struct {
	itemType bwcmd.ItemType
	item     map[string]any
	lines    []formLine
	notes    textarea.Model
	// focus is the index of the focused line. len(lines) is the notes field.
	focus  int
	reveal bool // the password shows as text
	dirty  bool // the user changed a value

	// errLine is the line that failed a check, and errText its error.
	errLine int
	errText string
	// bwErr is the error of the last save.
	bwErr  string
	saving bool

	keys          ui.FormKeyMap
	width, height int
}

// newItemForm returns the form for a new item of type t, for a content
// area of the given size.
func newItemForm(t bwcmd.ItemType, width, height int) (itemForm, tea.Cmd) {
	f := itemForm{
		itemType: t,
		item:     bwcmd.NewItemJSON(t),
		keys:     ui.DefaultFormKeyMap(),
		errLine:  -1,
	}
	f.lines = append(f.lines, newFormLine(fieldName, "Name", ""))
	if t == bwcmd.ItemTypeLogin {
		f.lines = append(f.lines,
			newFormLine(fieldUsername, "Username", ""),
			newFormLine(fieldPassword, "Password", ""),
			newFormLine(fieldTOTP, "TOTP", ""),
			newFormLine(fieldURI, "URI", ""),
		)
	}
	f.notes = textarea.New()
	f.notes.ShowLineNumbers = false
	f.notes.Prompt = ""
	f.notes.SetVirtualCursor(true)
	f.setReveal(false)
	f.resize(width, height)
	return f, f.focusLine(0)
}

// newFormLine returns a line of the form with the value v.
func newFormLine(kind fieldKind, label, v string) formLine {
	in := textinput.New()
	in.Prompt = ""
	in.SetValue(v)
	return formLine{kind: kind, label: label, input: in}
}

// title returns the title of the form.
func (f itemForm) title() string {
	if f.itemType == bwcmd.ItemTypeSecureNote {
		return "New secure note"
	}
	return "New login"
}

// clear drops the values of the form. Go strings cannot be zeroed, so it
// only drops the references.
func (f *itemForm) clear() {
	*f = itemForm{}
}

// resize sets the width of the fields for a content area of the given size.
func (f *itemForm) resize(width, height int) {
	f.width, f.height = width, height
	w := max(1, width-formLabelWidth-4)
	for i := range f.lines {
		f.lines[i].input.SetWidth(w)
	}
	f.notes.SetWidth(w)
	f.notes.SetHeight(max(formNotesMinHeight, height-len(f.lines)-4))
}

// setReveal shows the password as text when on is true, and masks it
// otherwise.
func (f *itemForm) setReveal(on bool) {
	f.reveal = on
	for i := range f.lines {
		if f.lines[i].kind != fieldPassword {
			continue
		}
		f.lines[i].input.EchoMode = textinput.EchoPassword
		f.lines[i].input.EchoCharacter = []rune(ui.Glyphs.Mask)[0]
		if on {
			f.lines[i].input.EchoMode = textinput.EchoNormal
		}
	}
}

// onNotes reports whether the notes field has the focus.
func (f itemForm) onNotes() bool { return f.focus == len(f.lines) }

// onPassword reports whether the password field has the focus.
func (f itemForm) onPassword() bool {
	return f.focus < len(f.lines) && f.lines[f.focus].kind == fieldPassword
}

// focusLine moves the focus to line i, where len(lines) is the notes field.
func (f *itemForm) focusLine(i int) tea.Cmd {
	n := len(f.lines) + 1
	f.focus = (i%n + n) % n
	for j := range f.lines {
		f.lines[j].input.Blur()
	}
	f.notes.Blur()
	if f.onNotes() {
		return f.notes.Focus()
	}
	return f.lines[f.focus].input.Focus()
}

// setPassword puts v in the password field.
func (f *itemForm) setPassword(v string) {
	for i := range f.lines {
		if f.lines[i].kind == fieldPassword {
			f.lines[i].input.SetValue(v)
			f.dirty = true
		}
	}
}

// update gives a key or another message to the focused field. Keys that
// the vault takes, such as save and cancel, do not reach it.
func (f itemForm) update(msg tea.Msg) (itemForm, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(k, f.keys.Next):
			return f, f.focusLine(f.focus + 1)
		case key.Matches(k, f.keys.Prev):
			return f, f.focusLine(f.focus - 1)
		case key.Matches(k, f.keys.Reveal):
			f.setReveal(!f.reveal)
			return f, nil
		case k.String() == "enter" && !f.onNotes():
			return f, f.focusLine(f.focus + 1)
		}
	}

	var cmd tea.Cmd
	if f.onNotes() {
		before := f.notes.Value()
		f.notes, cmd = f.notes.Update(msg)
		f.dirty = f.dirty || f.notes.Value() != before
		return f, cmd
	}
	line := &f.lines[f.focus]
	before := line.input.Value()
	line.input, cmd = line.input.Update(msg)
	if line.input.Value() != before {
		f.dirty = true
		if f.errLine == f.focus {
			f.errLine, f.errText = -1, ""
		}
	}
	f.addURILine()
	return f, cmd
}

// addURILine adds an empty URI line after the last URI line when the last
// URI line has text, so that the form always has an empty URI line.
func (f *itemForm) addURILine() {
	last := -1
	for i, l := range f.lines {
		if l.kind == fieldURI {
			last = i
		}
	}
	if last < 0 || f.lines[last].input.Value() == "" {
		return
	}
	line := newFormLine(fieldURI, "", "")
	line.input.SetWidth(f.lines[last].input.Width())
	f.lines = append(f.lines[:last+1], append([]formLine{line}, f.lines[last+1:]...)...)
	if f.focus > last {
		f.focus++
	}
	f.resize(f.width, f.height)
}

// validate checks the values. On an error, it moves the focus to the
// field and reports false.
func (f *itemForm) validate() (bool, tea.Cmd) {
	f.errLine, f.errText = -1, ""
	for i, l := range f.lines {
		v := strings.TrimSpace(l.input.Value())
		switch {
		case l.kind == fieldName && v == "":
			f.errLine, f.errText = i, nameRequiredText
		case l.kind == fieldTOTP && v != "":
			p, err := totp.Parse(v)
			p.Clear()
			if err != nil {
				f.errLine, f.errText = i, badTOTPText
			}
		}
		if f.errLine >= 0 {
			return false, f.focusLine(i)
		}
	}
	return true, nil
}

// value returns the value of the first line of kind, or nil when it is
// empty, as Bitwarden stores an empty field.
func (f itemForm) value(kind fieldKind) any {
	for _, l := range f.lines {
		if l.kind == kind {
			if v := l.input.Value(); v != "" {
				return v
			}
			return nil
		}
	}
	return nil
}

// itemJSON returns the item JSON with the values of the form. Empty URI
// lines are left out.
func (f itemForm) itemJSON() map[string]any {
	item := make(map[string]any, len(f.item))
	for k, v := range f.item {
		item[k] = v
	}
	item["name"] = strings.TrimSpace(f.lines[0].input.Value())
	item["notes"] = nil
	if v := f.notes.Value(); v != "" {
		item["notes"] = v
	}
	if f.itemType != bwcmd.ItemTypeLogin {
		return item
	}
	login := make(map[string]any)
	if old, ok := item["login"].(map[string]any); ok {
		for k, v := range old {
			login[k] = v
		}
	}
	login["username"] = f.value(fieldUsername)
	login["password"] = f.value(fieldPassword)
	login["totp"] = f.value(fieldTOTP)
	uris := []any{}
	for _, l := range f.lines {
		v := strings.TrimSpace(l.input.Value())
		if l.kind != fieldURI || v == "" {
			continue
		}
		uri := map[string]any{"match": nil}
		for k, old := range l.uri {
			uri[k] = old
		}
		uri["uri"] = v
		uris = append(uris, uri)
	}
	login["uris"] = uris
	item["login"] = login
	return item
}

// private returns the values of the form, which the debug log must not
// show.
func (f itemForm) private() []string {
	var out []string
	for _, l := range f.lines {
		if v := l.input.Value(); v != "" {
			out = append(out, v)
		}
	}
	if v := f.notes.Value(); v != "" {
		out = append(out, v)
	}
	return out
}

// view renders the form for a content area of the given size. When the
// form is taller than the area, it keeps the focused field in view.
func (f itemForm) view(width, height int, spinner string) string {
	labelStyle := lipgloss.NewStyle().Width(formLabelWidth).Foreground(ui.ColorFaint)
	focusStyle := labelStyle.Foreground(ui.ColorHighlight).Bold(true)
	textW := max(1, width-4)

	rows := []string{ui.StyleTitle.Render(ansi.Truncate(f.title(), textW, ui.Glyphs.Ellipsis)), ""}
	switch {
	case f.saving:
		rows = append(rows, spinner+" "+ui.StyleFaint.Render("Saving"+ui.Glyphs.Ellipsis), "")
	case f.bwErr != "":
		rows = append(rows, strings.Split(errorBlock(f.bwErr, width, textW), "\n")...)
		rows = append(rows, "")
	}

	focusRow := 0
	for i, l := range f.lines {
		marker, style := "  ", labelStyle
		if i == f.focus {
			marker, style = ui.StyleTitle.Render(ui.Glyphs.Cursor), focusStyle
			focusRow = len(rows)
		}
		rows = append(rows, marker+style.Render(l.label)+l.input.View())
		if i == f.errLine {
			rows = append(rows, strings.Repeat(" ", formLabelWidth+2)+ui.StyleError.Render(f.errText))
		}
	}

	marker, style := "  ", labelStyle
	if f.onNotes() {
		marker, style = ui.StyleTitle.Render(ui.Glyphs.Cursor), focusStyle
		focusRow = len(rows)
	}
	notes := strings.Split(f.notes.View(), "\n")
	for i, n := range notes {
		label := strings.Repeat(" ", formLabelWidth)
		m := "  "
		if i == 0 {
			label, m = style.Render("Notes"), marker
		}
		rows = append(rows, m+label+n)
	}

	// Keep the focused row in view. For the notes field, show as much of
	// it as fits.
	start := 0
	if height > 0 && len(rows) > height && focusRow >= height {
		start = min(focusRow, len(rows)-height)
	}
	rows = rows[start:]
	if height > 0 && len(rows) > height {
		rows = rows[:height]
	}
	for len(rows) < height {
		rows = append(rows, "")
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = ansi.Truncate(" "+r, width, "")
	}
	return strings.Join(out, "\n")
}

// hints returns the footer hints of the form.
func (f itemForm) hints() []ui.HintBinding {
	if f.saving {
		return nil
	}
	k := f.keys
	hints := []ui.HintBinding{
		ui.Hint(k.Save, "save"),
		ui.Hint(k.Cancel, "cancel"),
		ui.Hint(k.Next, "next"),
	}
	if f.onPassword() {
		hints = append(hints, ui.Hint(k.Generate, "generate"), ui.Hint(k.Reveal, "show"))
	}
	return append(hints, ui.Hint(k.Prev, "previous"))
}
