package ui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
)

const DefaultDrawerHeight = 8

// DrawerProps carries everything the drawer renderer needs.
type DrawerProps struct {
	Item         *bwcmd.Item
	TOTPCode     string
	TOTPSecsLeft int
	TOTPPeriod   int  // TOTP time step in seconds; 0 means 30
	TOTPInvalid  bool // the TOTP key does not parse
	Width        int
	Height       int  // 0 means DefaultDrawerHeight
	AutoHeight   bool // true = size to content (no padding/truncation)
	ScrollOffset int
	// RepromptVerified is true when the user entered the master password
	// for the item. Until then, an item with master password re-prompt
	// shows no TOTP code and no note text.
	RepromptVerified bool
	// Group is the selected group header when no item is selected. The
	// drawer then shows a summary of the group.
	Group *GroupSummary
}

// GroupSummary describes a group header of the vault list for the drawer.
type GroupSummary struct {
	Name     string
	Count    int
	Expanded bool
}

// Drawer texts for an item with master password re-prompt.
const (
	repromptMarker = "re-prompt"
	hiddenValue    = "(hidden)"
	hiddenNote     = "(hidden: master password re-prompt)"
)

// secretsHidden reports whether the drawer hides the TOTP code and the
// note text of the item in props.
func secretsHidden(props DrawerProps) bool {
	return props.Item != nil && props.Item.Reprompt == bwcmd.RepromptPassword && !props.RepromptVerified
}

// RenderDrawer renders the detail drawer for the selected vault item.
func RenderDrawer(props DrawerProps) string {
	h := props.Height
	if h <= 0 {
		h = DefaultDrawerHeight
	}

	if props.Item == nil {
		if props.Group != nil {
			return renderGroupSummary(*props.Group, props, h)
		}
		return renderNoSelection(props.Width, h)
	}

	// A secure note has no field rows, so its copy hint goes in the header.
	headerHint := ""
	if props.Item.Type == bwcmd.ItemTypeSecureNote && props.Item.Notes != "" {
		headerHint = noteCopyHint
	}
	marker := ""
	if props.Item.Reprompt == bwcmd.RepromptPassword {
		marker = lipgloss.NewStyle().Foreground(ColorYellow).Render(repromptMarker)
	}
	header := renderHeaderCard(props.Item.Name, props.Item.Type, headerHint, marker, props.Width)
	// In inline mode, add a gradient separator above the header card
	// to visually distinguish the drawer from the list.
	var sep string
	if props.AutoHeight {
		sep = header
	} else {
		sep = RenderGradientLine(props.Width) + "\n" + header
	}
	maxFields := bodyRows(props)
	var fields []string

	switch props.Item.Type {
	case bwcmd.ItemTypeLogin:
		fields = renderLoginFields(props)
	case bwcmd.ItemTypeCard:
		fields = renderCardFields(props.Item, props.Width)
	case bwcmd.ItemTypeSecureNote:
		fields = renderNoteFields(props, maxFields)
	case bwcmd.ItemTypeIdentity:
		fields = renderIdentityFields(props.Item, props.Width)
	case bwcmd.ItemTypeSSHKey:
		fields = renderSSHKeyFields(props.Item, props.Width)
	default:
		fields = []string{StyleFaint.Render("  (unsupported item type)")}
	}

	if !props.AutoHeight {
		switch {
		case maxFields <= 0:
			fields = nil
		case len(fields) > maxFields:
			// Replace the last row with a marker, so that the user knows
			// that fields are hidden.
			hidden := len(fields) - maxFields + 1
			more := Truncate(fmt.Sprintf("  %s %d more", Glyphs.Down, hidden), props.Width)
			fields = append(fields[:maxFields-1], StyleFaint.Render(more))
		}
		for len(fields) < maxFields {
			fields = append(fields, "")
		}
	}
	if len(fields) == 0 {
		// The drawer has rows only for the header.
		return sep
	}

	return sep + "\n" + strings.Join(fields, "\n")
}

// mask returns n cells of the mask glyph, which hide a secret value.
func mask(n int) string {
	return strings.Repeat(Glyphs.Mask, n)
}

// drawerKeys are the vault keys that the drawer hints name.
var drawerKeys = DefaultVaultKeyMap()

// keyHint returns the drawer hint for the key of b, for example "[o] open".
func keyHint(b key.Binding, action string) string {
	return "[" + b.Help().Key + "] " + action
}

// copyHint returns the drawer hint for the key of b, which copies a field,
// for example "[c] copy".
func copyHint(b key.Binding) string {
	return keyHint(b, "copy")
}

// noteCopyHint is the drawer hint for the key that copies the notes.
var noteCopyHint = copyHint(drawerKeys.CopyNote)

// bodyRows returns the number of drawer rows below the header card. In
// inline mode, the gradient line and the header card use 2 of the Height
// rows. Overlay mode (AutoHeight) has no gradient line, so only the header
// card uses a row. That mode cuts only a note to bodyRows: it sizes the card
// to the other fields.
func bodyRows(props DrawerProps) int {
	h := props.Height
	if h <= 0 {
		h = DefaultDrawerHeight
	}
	if props.AutoHeight {
		return h - 1
	}
	return h - 2
}

// MaxScrollOffset returns the largest useful ScrollOffset for props: the
// offset that shows the last full page of a secure note. It returns 0 for
// other item types, because only notes scroll, and for a drawer with no
// rows for the note. RenderDrawer uses the same line count, so the limit
// matches what the drawer shows.
func MaxScrollOffset(props DrawerProps) int {
	item := props.Item
	rows := bodyRows(props)
	if item == nil || item.Type != bwcmd.ItemTypeSecureNote || item.Notes == "" || rows <= 0 || secretsHidden(props) {
		return 0
	}
	return max(0, len(NoteLines(item.Notes, props.Width))-rows)
}

func renderNoSelection(width, height int) string {
	lines := []string{
		RenderGradientLine(width),
		StyleFaint.Render(Truncate("  No item selected "+Glyphs.Dash+" navigate with j/k or search with /", width)),
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// renderGroupSummary renders the drawer for a selected group header: the
// group name, the number of items, and the key that expands or collapses
// the group. Inline, the drawer is exactly height rows.
func renderGroupSummary(g GroupSummary, props DrawerProps, height int) string {
	action := "expand"
	if g.Expanded {
		action = "collapse"
	}
	width := props.Width
	lines := []string{
		"  " + StyleTitle.Render(Truncate(SanitizeLine(g.Name), width-2)),
		StyleFaint.Render(Truncate(fmt.Sprintf("  %d items%senter to %s", g.Count, Glyphs.Dot, action), width)),
	}
	if props.AutoHeight {
		return strings.Join(lines, "\n")
	}
	lines = append([]string{RenderGradientLine(width)}, lines...)
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines[:height], "\n")
}

// renderHeaderCard renders the item name on the left and the type badge on
// the right. A hint and a styled marker, when given, go before the badge.
// When they would leave the name less than rowMinNameW cells, the hint is
// dropped first, and then the marker.
func renderHeaderCard(name string, itemType bwcmd.ItemType, hint, marker string, width int) string {
	prefix := "  " + ItemGlyph(itemType) + "  "
	badge := StyleHeaderBadge.Render(ItemTypeName(itemType))
	name = SanitizeLine(name)

	var extras []string
	if hint != "" {
		extras = append(extras, StyleFaint.Render(hint))
	}
	if marker != "" {
		extras = append(extras, marker)
	}
	right := strings.Join(append(extras, badge), "  ")
	// Keep at least one space between the name and the right side.
	maxName := width - lipgloss.Width(prefix) - lipgloss.Width(right) - 1
	for len(extras) > 0 && maxName < min(lipgloss.Width(name), rowMinNameW) {
		extras = extras[1:]
		right = strings.Join(append(extras, badge), "  ")
		maxName = width - lipgloss.Width(prefix) - lipgloss.Width(right) - 1
	}

	left := prefix + StyleTitle.Render(Truncate(name, maxName))
	fill := max(1, width-lipgloss.Width(left)-lipgloss.Width(right))
	line := left + strings.Repeat(" ", fill) + right
	if lipgloss.Width(line) > width {
		// The drawer is narrower than the glyph and the badge.
		line = ansi.Truncate(line, max(0, width), "")
	}
	return line
}

// ItemGlyph returns the themed glyph string for the given item type.
func ItemGlyph(t bwcmd.ItemType) string {
	switch t {
	case bwcmd.ItemTypeLogin:
		return GlyphLogin
	case bwcmd.ItemTypeCard:
		return GlyphCard
	case bwcmd.ItemTypeSecureNote:
		return GlyphNote
	case bwcmd.ItemTypeIdentity:
		return GlyphIdentity
	case bwcmd.ItemTypeSSHKey:
		return GlyphSSHKey
	default:
		return " "
	}
}

func ItemTypeName(t bwcmd.ItemType) string {
	switch t {
	case bwcmd.ItemTypeLogin:
		return "Login"
	case bwcmd.ItemTypeSecureNote:
		return "Note"
	case bwcmd.ItemTypeCard:
		return "Card"
	case bwcmd.ItemTypeIdentity:
		return "Identity"
	case bwcmd.ItemTypeSSHKey:
		return "SSH Key"
	default:
		return "Item"
	}
}

func renderLoginFields(props DrawerProps) []string {
	item := props.Item
	w := props.Width
	var fields []string

	username := ""
	password := ""
	if item.Login != nil {
		username = item.Login.Username
		password = item.Login.Password
	}

	// An empty field shows (none) and no copy hint, because its copy key
	// copies nothing.
	if username == "" {
		fields = append(fields, renderedFieldRow("Username", StyleFaint.Render("(none)"), "", w))
	} else {
		fields = append(fields, fieldRow("Username", username, copyHint(drawerKeys.CopyUsername), w))
	}

	if password == "" {
		fields = append(fields, renderedFieldRow("Password", StyleFaint.Render("(none)"), "", w))
	} else {
		fields = append(fields, fieldRow("Password", mask(10), copyHint(drawerKeys.Copy), w))
	}

	if item.Login != nil && item.Login.Totp != "" {
		if secretsHidden(props) && !props.TOTPInvalid {
			fields = append(fields, renderedFieldRow("TOTP", StyleFaint.Render(hiddenValue), copyHint(drawerKeys.CopyTOTP), w))
		} else {
			fields = append(fields, totpRow(props, w))
		}
	}

	if item.Login != nil && len(item.Login.URIs) > 0 {
		fields = append(fields, fieldRow("URL", item.Login.URIs[0].URI, keyHint(drawerKeys.OpenURL, "open"), w))
	}

	switch {
	case item.Notes == "":
		fields = append(fields, renderedFieldRow("Notes", StyleFaint.Render("(none)"), "", w))
	case secretsHidden(props):
		fields = append(fields, renderedFieldRow("Notes", StyleFaint.Render(hiddenValue), noteCopyHint, w))
	default:
		// Cut the note at its first "\r" or "\n", so that the row shows
		// only the first line. An escape sequence ends at a line break, so
		// the cut gives the same text as sanitizing all of the note.
		first := item.Notes[:nextLineBreak(item.Notes, 0)]
		fields = append(fields, styledFieldRow("Notes", first, StyleFaint, noteCopyHint, w))
	}

	return fields
}

func renderCardFields(item *bwcmd.Item, width int) []string {
	if item.Card == nil {
		return []string{"  " + GlyphCard + " " + StyleFaint.Render("(no card data)")}
	}
	c := item.Card
	var fields []string

	fields = append(fields, fieldRow("Cardholder", c.CardholderName, "", width))

	numDisplay := mask(12)
	if len(c.Number) >= 4 {
		numDisplay = mask(4) + " " + mask(4) + " " + mask(4) + " " + c.Number[len(c.Number)-4:]
	}
	fields = append(fields, fieldRow("Number", numDisplay, copyHint(drawerKeys.Copy), width))

	expiry := ""
	if c.ExpMonth != "" && c.ExpYear != "" {
		expiry = fmt.Sprintf("%s/%s", c.ExpMonth, c.ExpYear)
	}
	fields = append(fields, fieldRow("Expiry", expiry, "", width))
	if c.Code == "" {
		fields = append(fields, renderedFieldRow("CVV", StyleFaint.Render("(none)"), "", width))
	} else {
		fields = append(fields, fieldRow("CVV", mask(3), copyHint(drawerKeys.CopyCVV), width))
	}

	return fields
}

func renderNoteFields(props DrawerProps, maxLines int) []string {
	notes := props.Item.Notes
	if notes == "" {
		return []string{"  " + GlyphNote + " " + StyleFaint.Render("(empty note)")}
	}
	if secretsHidden(props) {
		return []string{Truncate("  "+GlyphNote+" "+StyleFaint.Render(hiddenNote), props.Width)}
	}

	lines := NoteLines(notes, props.Width)

	// Apply the scroll offset. An offset past MaxScrollOffset shows the
	// last full page, not blank rows.
	if maxLines > 0 {
		offset := min(max(0, props.ScrollOffset), max(0, len(lines)-maxLines))
		lines = lines[offset:]
		if len(lines) > maxLines {
			lines = lines[:maxLines]
		}
	}

	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = "  " + line
	}
	return result
}

func renderIdentityFields(item *bwcmd.Item, width int) []string {
	if item.Identity == nil {
		return []string{"  " + GlyphIdentity + " " + StyleFaint.Render("(no identity data)")}
	}
	id := item.Identity
	var fields []string

	// Full name.
	var parts []string
	for _, p := range []string{id.Title, id.FirstName, id.MiddleName, id.LastName} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) > 0 {
		fields = append(fields, fieldRow("Name", strings.Join(parts, " "), "", width))
	}

	if id.Email != "" {
		fields = append(fields, fieldRow("Email", id.Email, copyHint(drawerKeys.CopyUsername), width))
	}
	if id.Phone != "" {
		fields = append(fields, fieldRow("Phone", id.Phone, "", width))
	}
	if id.Company != "" {
		fields = append(fields, fieldRow("Company", id.Company, "", width))
	}

	// Sensitive fields — masked.
	if id.SSN != "" {
		fields = append(fields, fieldRow("SSN", mask(9), copyHint(drawerKeys.Copy), width))
	}
	if id.PassportNumber != "" {
		fields = append(fields, fieldRow("Passport", mask(9), "", width))
	}
	if id.LicenseNumber != "" {
		fields = append(fields, fieldRow("License", mask(9), "", width))
	}

	// Address.
	addr := formatAddress(id)
	if addr != "" {
		fields = append(fields, fieldRow("Address", addr, "", width))
	}

	if len(fields) == 0 {
		fields = []string{"  " + GlyphIdentity + " " + StyleFaint.Render("(empty identity)")}
	}
	return fields
}

func formatAddress(id *bwcmd.Identity) string {
	var parts []string
	for _, line := range []string{id.Address1, id.Address2, id.Address3} {
		if line != "" {
			parts = append(parts, line)
		}
	}
	var cityState []string
	if id.City != "" {
		cityState = append(cityState, id.City)
	}
	if id.State != "" {
		cityState = append(cityState, id.State)
	}
	if len(cityState) > 0 {
		cs := strings.Join(cityState, ", ")
		if id.PostalCode != "" {
			cs += " " + id.PostalCode
		}
		parts = append(parts, cs)
	} else if id.PostalCode != "" {
		parts = append(parts, id.PostalCode)
	}
	if id.Country != "" {
		parts = append(parts, id.Country)
	}
	return strings.Join(parts, ", ")
}

func renderSSHKeyFields(item *bwcmd.Item, width int) []string {
	if item.SSHKey == nil {
		return []string{"  " + GlyphSSHKey + " " + StyleFaint.Render("(no SSH key data)")}
	}
	k := item.SSHKey
	var fields []string

	if k.KeyFingerprint != "" {
		fields = append(fields, fieldRow("Fingerprint", k.KeyFingerprint, "", width))
	}

	if k.PublicKey != "" {
		// fieldRow handles truncation, so pass the full key.
		fields = append(fields, fieldRow("Public Key", k.PublicKey, copyHint(drawerKeys.CopyUsername), width))
	}

	fields = append(fields, fieldRow("Private Key", mask(10), copyHint(drawerKeys.Copy), width))

	if len(fields) == 0 {
		fields = []string{"  " + GlyphSSHKey + " " + StyleFaint.Render("(empty SSH key)")}
	}
	return fields
}

// Field row layout, in cells.
const (
	fieldIndent    = 2
	fieldLabelW    = 12
	fieldHintGap   = 2 // space between value and hint
	fieldMinValueW = 4 // the hint is dropped when it leaves less for the value
)

// fieldRow renders a labeled row for a plain value from the vault. It
// sanitizes the value for display and cuts it to fit the width.
func fieldRow(label, value, hint string, width int) string {
	return styledFieldRow(label, value, lipgloss.NewStyle(), hint, width)
}

// styledFieldRow is fieldRow with a style for the value. It cuts the plain
// value before it applies the style.
func styledFieldRow(label, value string, style lipgloss.Style, hint string, width int) string {
	hint, maxValue := fieldLayout(hint, width)
	value = Truncate(SanitizeLine(value), maxValue)
	return joinFieldRow(label, style.Render(value), hint, width)
}

// renderedFieldRow renders a labeled row for a value that lazybw built,
// such as the TOTP countdown. The value can hold ANSI styles, and it is cut
// to fit without breaking them.
func renderedFieldRow(label, value, hint string, width int) string {
	hint, maxValue := fieldLayout(hint, width)
	return joinFieldRow(label, Truncate(value, maxValue), hint, width)
}

// fieldLayout returns the hint to show and the width for the value. It
// drops the hint when the value would get less than fieldMinValueW cells.
func fieldLayout(hint string, width int) (string, int) {
	maxValue := width - fieldIndent - fieldLabelW
	if hint != "" {
		if maxValue-fieldHintGap-lipgloss.Width(hint) < fieldMinValueW {
			return "", maxValue
		}
		maxValue -= fieldHintGap + lipgloss.Width(hint)
	}
	return hint, maxValue
}

func joinFieldRow(label, value, hint string, width int) string {
	labelStyle := lipgloss.NewStyle().Width(fieldLabelW).Align(lipgloss.Left)
	row := strings.Repeat(" ", fieldIndent) + labelStyle.Render(label) + value
	if hint != "" {
		row += strings.Repeat(" ", fieldHintGap) + StyleFaint.Render(hint)
	}
	if lipgloss.Width(row) > width {
		// The drawer is narrower than the label.
		row = ansi.Truncate(row, max(0, width), "")
	}
	return row
}

// totpRow renders the TOTP row. A cut countdown such as "2…" looks like a
// wrong time, so a row that is too narrow drops the copy hint first and
// then the seconds text. Only a drawer that is too narrow for the code and
// the bar cuts the value. The check uses the widest seconds text of the
// period, so that the row does not change as the seconds tick.
func totpRow(props DrawerProps, width int) string {
	copyText := copyHint(drawerKeys.CopyTOTP)
	period := props.TOTPPeriod
	if period <= 0 {
		period = 30
	}
	widest := lipgloss.Width(renderTOTP(props.TOTPCode, period, period, props.TOTPInvalid, true))
	valueW := width - fieldIndent - fieldLabelW

	hint, showSecs := "", true
	switch {
	case widest <= valueW-fieldHintGap-lipgloss.Width(copyText):
		hint = copyText
	case widest > valueW:
		showSecs = false
	}
	display := renderTOTP(props.TOTPCode, props.TOTPSecsLeft, period, props.TOTPInvalid, showSecs)
	return renderedFieldRow("TOTP", display, hint, width)
}

// renderTOTP renders the TOTP code and its countdown bar. showSecs adds the
// seconds left as text after the bar.
func renderTOTP(code string, secsLeft, period int, invalid, showSecs bool) string {
	if invalid {
		return StyleError.Render("invalid TOTP secret")
	}
	if code == "" {
		return StyleFaint.Render("loading" + Glyphs.Ellipsis)
	}
	if period <= 0 {
		period = 30
	}

	// Format code with mid-space: "843 291"
	display := code
	if len(code) == 6 {
		display = code[:3] + " " + code[3:]
	}

	// Micro-bar countdown: 4 cells of the bar glyphs of the active set, for
	// example block elements (U+2588-258F) with a light shade (░) for the
	// empty track. 4 cells * 8 levels = 32 states for the period, giving a
	// visible change every second at 30 seconds.
	const barWidth = 4
	g := Glyphs

	fill := min(secsLeft*barWidth*8/period, barWidth*8)
	full := fill / 8
	frac := fill % 8

	var filled, track strings.Builder
	for i := range barWidth {
		if i < full {
			filled.WriteString(g.BarFull)
		} else if i == full && frac > 0 {
			filled.WriteString(g.BarSteps[7-frac])
		} else {
			track.WriteString(g.BarEmpty)
		}
	}

	// Color based on urgency: green → yellow → red. For 30 seconds, yellow
	// starts at 15 and red at 10.
	var clr color.Color
	switch {
	case secsLeft > period/2:
		clr = ColorGreen
	case secsLeft > period/3:
		clr = ColorYellow
	default:
		clr = ColorRed
	}

	style := lipgloss.NewStyle().Foreground(clr)
	bar := style.Render(filled.String()) + StyleFaint.Render(track.String())
	if !showSecs {
		return display + "  " + bar
	}
	return fmt.Sprintf("%s  %s %s", display, bar, style.Render(fmt.Sprintf("%ds", secsLeft)))
}
