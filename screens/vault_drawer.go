package screens

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/juthrbog/lazybw/totp"
	"github.com/juthrbog/lazybw/ui"
)

// handleDrawerKey handles the keys that scroll, resize and move the
// drawer. It reports false for any other key.
func (m VaultModel) handleDrawerKey(msg tea.KeyPressMsg) (VaultModel, tea.Cmd, bool) {
	switch {
	// Only secure notes scroll. For other items the limit is 0, so J and
	// K do nothing. A resize or a drawer change can lower the limit, so K
	// clamps first and then moves.
	case key.Matches(msg, m.keymap.ScrollDown):
		m.drawerScroll = min(m.drawerScroll+1, m.maxDrawerScroll())
		return m, nil, true

	case key.Matches(msg, m.keymap.ScrollUp):
		m.drawerScroll = max(0, min(m.drawerScroll, m.maxDrawerScroll())-1)
		return m, nil, true

	case key.Matches(msg, m.keymap.DrawerGrow):
		if !m.drawerOverlay {
			m.drawerHeight = m.clampDrawerHeight(m.drawerHeight + 1)
			m.resizeList()
		}
		return m, nil, true

	case key.Matches(msg, m.keymap.DrawerShrink):
		if !m.drawerOverlay {
			m.drawerHeight = m.clampDrawerHeight(m.drawerHeight - 1)
			m.resizeList()
		}
		return m, nil, true

	case key.Matches(msg, m.keymap.DrawerReset):
		if !m.drawerOverlay {
			m.drawerHeight = m.clampDrawerHeight(ui.DefaultDrawerHeight)
			m.resizeList()
		}
		return m, nil, true

	case key.Matches(msg, m.keymap.DrawerToggle):
		m.drawerOverlay = !m.drawerOverlay
		m.resizeList()
		if m.drawerOverlay {
			return m, m.setToast("Drawer: overlay"), true
		}
		return m, m.setToast("Drawer: inline"), true
	}
	return m, nil, false
}

// loadTOTP parses the TOTP key of the selected item and computes its code.
// It scrubs the params of the previous item.
func (m *VaultModel) loadTOTP() {
	if m.totpParams != nil {
		m.totpParams.Clear()
		m.totpParams = nil
	}
	m.totpCode = ""
	m.totpInvalid = false
	m.totpItemID = ""

	item := m.selectedItem()
	if item != nil {
		m.totpItemID = item.ID
	}
	if item != nil && item.Login != nil && item.Login.Totp != "" {
		p, err := totp.Parse(item.Login.Totp)
		if err == nil {
			m.totpParams = &p
		} else {
			m.totpInvalid = true
		}
	}
	m.computeTOTP()
}

// computeTOTP recomputes the TOTP code and countdown from the cached
// params. It is a no-op when no TOTP params are available.
func (m *VaultModel) computeTOTP() {
	if m.totpParams == nil {
		m.totpCode = ""
		m.totpSecsLeft = 0
		return
	}
	m.totpCode = totp.Code(*m.totpParams)
	m.totpSecsLeft = totp.SecsLeft(m.totpParams.Period)
}

// totpPeriod returns the TOTP period of the selected item, or 0 when it
// has no TOTP.
func (m VaultModel) totpPeriod() int {
	if m.totpParams == nil {
		return 0
	}
	return m.totpParams.Period
}

func (m *VaultModel) clampDrawerHeight(h int) int {
	return clampDrawerHeight(h, m.height)
}

// minListRows is the number of rows that the list always draws: its title
// line, where the filter input shows, and one item. The bubbles list draws
// them even when its height is smaller.
const minListRows = 2

// clampDrawerHeight limits a drawer height h to half of the screen height.
// The drawer keeps at least 3 rows. On a screen that is too short for that,
// the drawer gets the rows of the content area that the list leaves, and at
// least 1 row.
func clampDrawerHeight(h, height int) int {
	const minDrawerHeight = 3
	maxDrawerHeight := max(height/2, minDrawerHeight)
	// A taller drawer would make the list and the drawer together taller
	// than the content area, and the frame would cut the drawer.
	maxDrawerHeight = min(maxDrawerHeight, height-ui.ChromeHeight-minListRows)
	return max(1, min(max(h, minDrawerHeight), maxDrawerHeight))
}

// resizeList sizes the list to the rows that it gets in the content area:
// all of them in overlay mode, and the rows above the drawer in inline
// mode. The list pages by the size that it has, so this size must be the
// size that the view draws. Otherwise PgDn skips items.
func (m *VaultModel) resizeList() {
	listH := m.height - ui.ChromeHeight
	if !m.drawerOverlay {
		listH -= m.drawerHeight
	}
	m.list.SetSize(m.width, max(0, listH))
}

// drawerProps returns the drawer props for a content area of the given
// size. The view and the scroll limit both use it, so that they count the
// same note lines.
func (m VaultModel) drawerProps(width, contentHeight int) ui.DrawerProps {
	props := ui.DrawerProps{
		Item:         m.selectedItem(),
		TOTPCode:     m.totpCode,
		TOTPSecsLeft: m.totpSecsLeft,
		TOTPPeriod:   m.totpPeriod(),
		TOTPInvalid:  m.totpInvalid,
		ScrollOffset: m.drawerScroll,
	}
	props.RepromptVerified = m.repromptVerified(props.Item)
	if gh, ok := m.list.SelectedItem().(GroupHeaderItem); ok {
		props.Group = &ui.GroupSummary{Name: gh.Name, Count: gh.Count, Expanded: gh.Expanded}
	}
	if m.drawerOverlay {
		props.Width = overlayDrawerWidth(width)
		props.AutoHeight = true
		// RenderOverlay cuts the card content to the rows inside the border
		// and the padding. Size the note page to those rows, so that J can
		// scroll to the last line. Height counts the header, and the page
		// keeps at least one note row.
		props.Height = max(2, min(ui.DefaultDrawerHeight, contentHeight-ui.OverlayEdgeHeight))
	} else {
		props.Width = width
		props.Height = m.drawerHeight
	}
	return props
}

// overlayDrawerWidth returns the drawer width in overlay mode.
func overlayDrawerWidth(width int) int {
	// Constrain overlay drawer width so it doesn't span the full terminal.
	w := width * 3 / 4
	if w > 80 {
		w = 80
	}
	if w < 40 {
		// Leave room for the card border and padding.
		w = width - ui.OverlayEdgeWidth
	}
	return w
}

// maxDrawerScroll returns the largest drawer scroll offset for the
// selected item.
func (m VaultModel) maxDrawerScroll() int {
	return ui.MaxScrollOffset(m.drawerProps(m.width, m.height-ui.ChromeHeight))
}

// renderVaultContent renders the normal vault layout (list + drawer). The
// list keeps the size that resizeList gave it, so that the view shows the
// page that the list moves by.
func (m VaultModel) renderVaultContent(width, contentHeight int) string {
	drawer := ui.RenderDrawer(m.drawerProps(width, contentHeight))
	listView := m.list.View()

	if m.drawerOverlay {
		return ui.RenderOverlay(listView, drawer, width, contentHeight)
	}

	return lipgloss.JoinVertical(lipgloss.Left, listView, drawer)
}
