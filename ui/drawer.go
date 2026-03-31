package ui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/juthrbog/lazybw/bwcmd"
)

const DrawerHeight = 8

// DrawerProps carries everything the drawer renderer needs.
type DrawerProps struct {
	Item         *bwcmd.Item
	TOTPCode     string
	TOTPSecsLeft int
	Width        int
	ScrollOffset int
}

// RenderDrawer renders the detail drawer for the selected vault item.
func RenderDrawer(props DrawerProps) string {
	if props.Item == nil {
		return renderNoSelection(props.Width)
	}

	sep := renderSeparator(props.Item.Name, itemTypeName(props.Item.Type), props.Width)
	var fields []string

	switch props.Item.Type {
	case bwcmd.ItemTypeLogin:
		fields = renderLoginFields(props)
	case bwcmd.ItemTypeCard:
		fields = renderCardFields(props.Item)
	case bwcmd.ItemTypeSecureNote:
		fields = renderNoteFields(props)
	case bwcmd.ItemTypeIdentity:
		fields = renderIdentityFields(props.Item)
	case bwcmd.ItemTypeSSHKey:
		fields = renderSSHKeyFields(props.Item)
	default:
		fields = []string{StyleFaint.Render("  (unsupported item type)")}
	}

	// Pad or truncate to fixed height (DrawerHeight - 1 for separator).
	maxFields := DrawerHeight - 1
	if len(fields) > maxFields {
		fields = fields[:maxFields]
	}
	for len(fields) < maxFields {
		fields = append(fields, "")
	}

	return sep + "\n" + strings.Join(fields, "\n")
}

func renderNoSelection(width int) string {
	sep := StyleFaint.Render("── No item selected " + strings.Repeat("─", max(0, width-21)))
	hint := StyleFaint.Render("  Navigate with j/k or search with /")
	lines := []string{sep, hint}
	for len(lines) < DrawerHeight {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func renderSeparator(name, typeName string, width int) string {
	left := "── " + name + " "
	right := " " + typeName + " ─"
	fill := width - lipgloss.Width(left) - lipgloss.Width(right)
	if fill < 1 {
		fill = 1
	}
	return StyleFaint.Render(left + strings.Repeat("─", fill) + right)
}

func itemTypeName(t bwcmd.ItemType) string {
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
	var fields []string

	username := ""
	password := ""
	if item.Login != nil {
		username = item.Login.Username
		password = item.Login.Password
	}

	fields = append(fields, fieldRow("Username", username, "[u] copy"))

	pwDisplay := "••••••••••"
	if password == "" {
		pwDisplay = StyleFaint.Render("(none)")
	}
	fields = append(fields, fieldRow("Password", pwDisplay, "[c] copy"))

	if item.Login != nil && item.Login.Totp != "" {
		totpDisplay := renderTOTP(props.TOTPCode, props.TOTPSecsLeft)
		fields = append(fields, fieldRow("TOTP", totpDisplay, "[t] copy"))
	}

	if item.Login != nil && len(item.Login.URIs) > 0 {
		fields = append(fields, fieldRow("URL", item.Login.URIs[0].URI, "[o] open"))
	}

	notes := "(none)"
	if item.Notes != "" {
		first, _, _ := strings.Cut(item.Notes, "\n")
		notes = first
	}
	fields = append(fields, fieldRow("Notes", StyleFaint.Render(notes), ""))

	return fields
}

func renderCardFields(item *bwcmd.Item) []string {
	if item.Card == nil {
		return []string{StyleFaint.Render("  (no card data)")}
	}
	c := item.Card
	var fields []string

	fields = append(fields, fieldRow("Cardholder", c.CardholderName, ""))

	numDisplay := "••••••••••••"
	if len(c.Number) >= 4 {
		numDisplay = "•••• •••• •••• " + c.Number[len(c.Number)-4:]
	}
	fields = append(fields, fieldRow("Number", numDisplay, "[c] copy"))

	expiry := ""
	if c.ExpMonth != "" && c.ExpYear != "" {
		expiry = fmt.Sprintf("%s/%s", c.ExpMonth, c.ExpYear)
	}
	fields = append(fields, fieldRow("Expiry", expiry, ""))
	fields = append(fields, fieldRow("CVV", "•••", "[v] copy"))

	return fields
}

func renderNoteFields(props DrawerProps) []string {
	notes := props.Item.Notes
	if notes == "" {
		return []string{StyleFaint.Render("  (empty note)")}
	}

	lines := strings.Split(notes, "\n")

	// Apply scroll offset.
	if props.ScrollOffset > 0 && props.ScrollOffset < len(lines) {
		lines = lines[props.ScrollOffset:]
	}

	maxLines := DrawerHeight - 1
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = "  " + line
	}
	return result
}

func renderIdentityFields(item *bwcmd.Item) []string {
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
		fields = append(fields, fieldRow("Name", strings.Join(parts, " "), ""))
	}

	if id.Email != "" {
		fields = append(fields, fieldRow("Email", id.Email, "[u] copy"))
	}
	if id.Phone != "" {
		fields = append(fields, fieldRow("Phone", id.Phone, ""))
	}
	if id.Company != "" {
		fields = append(fields, fieldRow("Company", id.Company, ""))
	}

	// Sensitive fields — masked.
	if id.SSN != "" {
		fields = append(fields, fieldRow("SSN", "•••••••••", "[c] copy"))
	}
	if id.PassportNumber != "" {
		fields = append(fields, fieldRow("Passport", "•••••••••", ""))
	}
	if id.LicenseNumber != "" {
		fields = append(fields, fieldRow("License", "•••••••••", ""))
	}

	// Address.
	addr := formatAddress(id)
	if addr != "" {
		fields = append(fields, fieldRow("Address", addr, ""))
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

func renderSSHKeyFields(item *bwcmd.Item) []string {
	if item.SSHKey == nil {
		return []string{StyleFaint.Render("  (no SSH key data)")}
	}
	k := item.SSHKey
	var fields []string

	if k.KeyFingerprint != "" {
		fields = append(fields, fieldRow("Fingerprint", k.KeyFingerprint, ""))
	}

	if k.PublicKey != "" {
		display := k.PublicKey
		if len(display) > 40 {
			display = display[:40] + "…"
		}
		fields = append(fields, fieldRow("Public Key", display, "[u] copy"))
	}

	fields = append(fields, fieldRow("Private Key", "••••••••••", "[c] copy"))

	if len(fields) == 0 {
		fields = []string{StyleFaint.Render("  (empty SSH key)")}
	}
	return fields
}

func fieldRow(label, value, hint string) string {
	labelStyle := lipgloss.NewStyle().Width(12).Align(lipgloss.Left)
	left := "  " + labelStyle.Render(label) + value
	if hint == "" {
		return left
	}
	return left + "  " + StyleFaint.Render(hint)
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
