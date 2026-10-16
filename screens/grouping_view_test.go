package screens

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/ui"
)

func TestHeaderCountIsVaultItems(t *testing.T) {
	m := newTestVault(groupTestItems())
	check := func(state string) {
		t.Helper()
		if count, _ := m.HeaderInfo(); count != len(groupTestItems()) {
			t.Errorf("%s: header count = %d, want %d (rows %q)", state, count, len(groupTestItems()), rowNames(m))
		}
	}
	check("grouping off")
	m = press(t, m, ctrlG)
	check("group collapsed")
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	check("group expanded")
}

// groupRowLine returns the line of out that shows the group row.
func groupRowLine(t *testing.T, out, name string) string {
	t.Helper()
	for _, line := range strings.Split(ansi.Strip(out), "\n") {
		if strings.Contains(line, name+" (") {
			return line
		}
	}
	t.Fatalf("no group row %q in:\n%s", name, ansi.Strip(out))
	return ""
}

func TestSelectedCollapsedGroupRow(t *testing.T) {
	m := press(t, vaultAt(groupTestItems(), 80, 24), ctrlG)
	m = selectRow(t, m, "group:github")
	out := m.ViewContent(80, 24-ui.ChromeHeight)
	line := groupRowLine(t, out, "GitHub")
	if !strings.HasPrefix(line, "▶ ▸  GitHub (3)") {
		t.Errorf("group row = %q, want %q", line, "▶ ▸  GitHub (3)")
	}
	if strings.Count(line, "▶") != 1 || strings.Count(line, "▸") != 1 {
		t.Errorf("group row = %q, want the cursor and the chevron once each", line)
	}
	if strings.Contains(ansi.Strip(out), "github") {
		t.Errorf("the view must show the item name, not the lowercase key:\n%s", ansi.Strip(out))
	}
}

func TestDrawerShowsGroupSummary(t *testing.T) {
	m := press(t, vaultAt(groupTestItems(), 80, 24), ctrlG)
	m = selectRow(t, m, "group:github")
	for _, overlay := range []bool{false, true} {
		if overlay {
			m = press(t, m, keyMsg("d"))
		}
		out := m.ViewContent(80, 24-ui.ChromeHeight)
		plain := ansi.Strip(out)
		if strings.Contains(plain, "No item selected") || !strings.Contains(plain, "3 items · enter to expand") {
			t.Errorf("overlay %v: the drawer must show the group summary:\n%s", overlay, plain)
		}
		assertArea(t, out, 80, 24-ui.ChromeHeight)
	}

	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if plain := ansi.Strip(m.ViewContent(80, 24-ui.ChromeHeight)); !strings.Contains(plain, "3 items · enter to collapse") {
		t.Errorf("an expanded group must offer to collapse:\n%s", plain)
	}
}
