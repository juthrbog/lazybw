package ui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/juthrbog/lazybw/bwcmd"
)

const DefaultDrawerHeight = 8

// DrawerProps carries everything the drawer renderer needs.
type DrawerProps struct {
	Item         *bwcmd.Item
	TOTPCode     string
	TOTPSecsLeft int
	Width        int
	Height       int  // 0 means DefaultDrawerHeight
	AutoHeight   bool // true = size to content (no padding/truncation)
	ScrollOffset int
}

// RenderDrawer renders the detail drawer for the selected vault item.
func RenderDrawer(props DrawerProps) string {
	h := props.Height
	if h <= 0 {
		h = DefaultDrawerHeight
	}

	if props.Item == nil {
		return renderNoSelection(props.Width, h)
	}

	header := renderHeaderCard(props.Item.Name, props.Item.Type, props.Width)
	// In inline mode, add a gradient separator above the header card
	// to visually distinguish the drawer from the list.
	var sep string
	if props.AutoHeight {
		sep = header
	} else {
		sep = RenderGradientLine(props.Width) + "\n" + header
	}
	// Header lines: 2 for inline (gradient + card), 1 for overlay (card only).
	headerLines := 2
	if props.AutoHeight {
		headerLines = 1
	}
	maxFields := h - headerLines
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
		if len(fields) > maxFields {
			fields = fields[:maxFields]
		}
		for len(fields) < maxFields {
			fields = append(fields, "")
		}
	}

	return sep + "\n" + strings.Join(fields, "\n")
}

func renderNoSelection(width, height int) string {
	lines := []string{
		RenderGradientLine(width),
		StyleFaint.Render("  No item selected — navigate with j/k or search with /"),
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func renderHeaderCard(name string, itemType bwcmd.ItemType, width int) string {
	glyph := ItemGlyph(itemType)
	badge := StyleHeaderBadge.Render(ItemTypeName(itemType))
	badgeW := lipgloss.Width(badge)

	// Truncate name if it won't fit.
	left := "  " + glyph + "  " + StyleTitle.Render(name)
	leftW := lipgloss.Width(left)
	maxLeft := width - badgeW - 1
	if leftW > maxLeft {
		for lipgloss.Width(name) > 0 && lipgloss.Width("  "+glyph+"  "+StyleTitle.Render(name+"…")) > maxLeft {
			name = name[:len(name)-1]
		}
		left = "  " + glyph + "  " + StyleTitle.Render(name+"…")
		leftW = lipgloss.Width(left)
	}

	fill := width - leftW - badgeW
	if fill < 1 {
		fill = 1
	}
	return left + strings.Repeat(" ", fill) + badge
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

	fields = append(fields, fieldRow("Username", username, "[u] copy", w))

	pwDisplay := "••••••••••" //nolint:gosec // display mask, not a credential
	if password == "" {
		pwDisplay = StyleFaint.Render("(none)")
	}
	fields = append(fields, fieldRow("Password", pwDisplay, "[c] copy", w))

	if item.Login != nil && item.Login.Totp != "" {
		totpDisplay := renderTOTP(props.TOTPCode, props.TOTPSecsLeft)
		fields = append(fields, fieldRow("TOTP", totpDisplay, "[t] copy", w))
	}

	if item.Login != nil && len(item.Login.URIs) > 0 {
		fields = append(fields, fieldRow("URL", item.Login.URIs[0].URI, "[o] open", w))
	}

	notes := "(none)"
	if item.Notes != "" {
		first, _, _ := strings.Cut(item.Notes, "\n")
		notes = first
	}
	fields = append(fields, fieldRow("Notes", StyleFaint.Render(notes), "", w))

	return fields
}

func renderCardFields(item *bwcmd.Item, width int) []string {
	if item.Card == nil {
		return []string{StyleFaint.Render("  (no card data)")}
	}
	c := item.Card
	var fields []string

	fields = append(fields, fieldRow("Cardholder", c.CardholderName, "", width))

	numDisplay := "••••••••••••"
	if len(c.Number) >= 4 {
		numDisplay = "•••• •••• •••• " + c.Number[len(c.Number)-4:]
	}
	fields = append(fields, fieldRow("Number", numDisplay, "[c] copy", width))

	expiry := ""
	if c.ExpMonth != "" && c.ExpYear != "" {
		expiry = fmt.Sprintf("%s/%s", c.ExpMonth, c.ExpYear)
	}
	fields = append(fields, fieldRow("Expiry", expiry, "", width))
	fields = append(fields, fieldRow("CVV", "•••", "[v] copy", width))

	return fields
}

func renderNoteFields(props DrawerProps, maxLines int) []string {
	notes := props.Item.Notes
	if notes == "" {
		return []string{StyleFaint.Render("  (empty note)")}
	}

	lines := strings.Split(notes, "\n")

	// Apply scroll offset.
	if props.ScrollOffset > 0 && props.ScrollOffset < len(lines) {
		lines = lines[props.ScrollOffset:]
	}

	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = "  " + line
	}
	return result
}

func renderIdentityFields(item *bwcmd.Item, width int) []string {
	if item.Identity == nil {
		return []string{StyleFaint.Render("  (no identity data)")}
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
		fields = append(fields, fieldRow("Email", id.Email, "[u] copy", width))
	}
	if id.Phone != "" {
		fields = append(fields, fieldRow("Phone", id.Phone, "", width))
	}
	if id.Company != "" {
		fields = append(fields, fieldRow("Company", id.Company, "", width))
	}

	// Sensitive fields — masked.
	if id.SSN != "" {
		fields = append(fields, fieldRow("SSN", "•••••••••", "[c] copy", width))
	}
	if id.PassportNumber != "" {
		fields = append(fields, fieldRow("Passport", "•••••••••", "", width))
	}
	if id.LicenseNumber != "" {
		fields = append(fields, fieldRow("License", "•••••••••", "", width))
	}

	// Address.
	addr := formatAddress(id)
	if addr != "" {
		fields = append(fields, fieldRow("Address", addr, "", width))
	}

	if len(fields) == 0 {
		fields = []string{StyleFaint.Render("  (empty identity)")}
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
		return []string{StyleFaint.Render("  (no SSH key data)")}
	}
	k := item.SSHKey
	var fields []string

	if k.KeyFingerprint != "" {
		fields = append(fields, fieldRow("Fingerprint", k.KeyFingerprint, "", width))
	}

	if k.PublicKey != "" {
		// fieldRow handles truncation, so pass the full key.
		fields = append(fields, fieldRow("Public Key", k.PublicKey, "[u] copy", width))
	}

	fields = append(fields, fieldRow("Private Key", "••••••••••", "[c] copy", width))

	if len(fields) == 0 {
		fields = []string{StyleFaint.Render("  (empty SSH key)")}
	}
	return fields
}

func fieldRow(label, value, hint string, width int) string {
	const indent = 2
	const labelWidth = 12
	const gap = 2 // space between value and hint

	labelStyle := lipgloss.NewStyle().Width(labelWidth).Align(lipgloss.Left)
	renderedHint := ""
	hintWidth := 0
	if hint != "" {
		renderedHint = StyleFaint.Render(hint)
		hintWidth = gap + lipgloss.Width(renderedHint)
	}

	// Truncate value if it exceeds available space.
	maxValue := width - indent - labelWidth - hintWidth
	if maxValue < 1 {
		maxValue = 1
	}
	if lipgloss.Width(value) > maxValue {
		// Strip to fit, accounting for the ellipsis character.
		for lipgloss.Width(value) > maxValue-1 && len(value) > 0 {
			value = value[:len(value)-1]
		}
		value += "…"
	}

	left := "  " + labelStyle.Render(label) + value
	if renderedHint == "" {
		return left
	}
	return left + "  " + renderedHint
}

func renderTOTP(code string, secsLeft int) string {
	if code == "" {
		return StyleFaint.Render("loading…")
	}

	// Format code with mid-space: "843 291"
	display := code
	if len(code) == 6 {
		display = code[:3] + " " + code[3:]
	}

	// Countdown circle: depletes as time runs out, like Bitwarden's extension.
	// ● (full) → ◕ (3/4) → ◑ (1/2) → ◔ (1/4) → ○ (empty)
	var circle string
	switch {
	case secsLeft > 24:
		circle = "●"
	case secsLeft > 18:
		circle = "◕"
	case secsLeft > 12:
		circle = "◑"
	case secsLeft > 6:
		circle = "◔"
	default:
		circle = "○"
	}

	// Color based on urgency.
	var color color.Color
	switch {
	case secsLeft > 15:
		color = ColorGreen
	case secsLeft > 10:
		color = ColorYellow
	default:
		color = ColorRed
	}

	styled := lipgloss.NewStyle().Foreground(color).Render(circle)
	return fmt.Sprintf("%s  %s %ds", display, styled, secsLeft)
}
