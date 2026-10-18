package screens

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/session"
	"github.com/juthrbog/lazybw/ui"
)

func newTestVault(items []bwcmd.Item) VaultModel {
	sess := &session.State{Email: "test@test.com"}
	return NewVaultModel(items, sess, 80, 24)
}

func testItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Gmail", Login: &bwcmd.Login{Username: "user@gmail.com"}},
		{ID: "2", Type: bwcmd.ItemTypeCard, Name: "Visa", Card: &bwcmd.Card{Number: "4242424242424242"}},
		{ID: "3", Type: bwcmd.ItemTypeSecureNote, Name: "Keys", Notes: "secret"},
	}
}

func TestVaultFooterContentNormalMode(t *testing.T) {
	m := newTestVault(testItems())
	hints, _ := m.FooterContent()
	if !hasHintKey(hints, "j/k") {
		t.Errorf("normal mode hints should contain navigation, got %v", hints)
	}
}

func TestVaultFooterContentWithToast(t *testing.T) {
	m := newTestVault(testItems())
	m.setToast("Password copied")
	_, status := m.FooterContent()
	if !strings.Contains(status, "Password copied") {
		t.Errorf("status should contain toast, got %q", status)
	}
}

func TestVaultItemFilterValue(t *testing.T) {
	item := bwcmd.Item{
		ID:    "1",
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Gmail",
		Login: &bwcmd.Login{Username: "user@gmail.com"},
	}
	vi := VaultItem{Item: item}
	fv := vi.FilterValue()
	if !strings.Contains(fv, "Gmail") {
		t.Errorf("FilterValue should contain name, got %q", fv)
	}
	if !strings.Contains(fv, "user@gmail.com") {
		t.Errorf("FilterValue should contain description, got %q", fv)
	}
}

func TestVaultSelectedItemInRange(t *testing.T) {
	m := newTestVault(testItems())
	m.list.Select(1)
	item := m.selectedItem()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
	if item.Name != "Visa" {
		t.Errorf("expected 'Visa', got %q", item.Name)
	}
}

func TestVaultSelectedItemEmptyList(t *testing.T) {
	m := newTestVault(nil)
	if m.selectedItem() != nil {
		t.Error("expected nil for empty list")
	}
}

func TestToListItems(t *testing.T) {
	items := testItems()
	li := toListItems(items)
	if len(li) != len(items) {
		t.Errorf("expected %d list items, got %d", len(items), len(li))
	}
	for i, item := range li {
		vi := item.(VaultItem)
		if vi.Name != items[i].Name {
			t.Errorf("item %d: expected %q, got %q", i, items[i].Name, vi.Name)
		}
	}
}

func groupTestItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "GitHub", Login: &bwcmd.Login{Username: "dev@example.com"}},
		{ID: "2", Type: bwcmd.ItemTypeLogin, Name: "GitHub (Work)", Login: &bwcmd.Login{Username: "work@company.com"}},
		{ID: "3", Type: bwcmd.ItemTypeLogin, Name: "GitHub (Personal)", Login: &bwcmd.Login{Username: "me@example.com"}},
		{ID: "4", Type: bwcmd.ItemTypeLogin, Name: "Gmail", Login: &bwcmd.Login{Username: "user@gmail.com"}},
	}
}

func TestSelectedItemNilForGroupHeader(t *testing.T) {
	m := newTestVault(groupTestItems())
	m.groups.enabled = true
	m.rebuildListItems()
	m.list.Select(0) // select the group header (collapsed)
	if m.selectedItem() != nil {
		t.Error("selectedItem should return nil for a group header")
	}
}

func TestSelectedItemForExpandedChild(t *testing.T) {
	m := newTestVault(groupTestItems())
	m.groups.enabled = true
	m.groups.toggle("github") // expand
	m.rebuildListItems()
	m.list.Select(1) // first child: "GitHub"
	item := m.selectedItem()
	if item == nil {
		t.Fatal("expected non-nil item for expanded child")
	}
	if item.Name != "GitHub" {
		t.Errorf("expected 'GitHub', got %q", item.Name)
	}
}

func TestToggleGroupingRebuilds(t *testing.T) {
	m := newTestVault(groupTestItems())

	// Initially grouping is off — 4 flat items.
	if len(m.list.Items()) != 4 {
		t.Fatalf("expected 4 items, got %d", len(m.list.Items()))
	}

	// Enable grouping.
	m.groups.toggleGrouping()
	m.rebuildListItems()

	// Expect: Header(Discord) + Gmail = 2 items (collapsed).
	if len(m.list.Items()) != 2 {
		t.Fatalf("expected 2 items with grouping, got %d", len(m.list.Items()))
	}

	// Disable grouping.
	m.groups.toggleGrouping()
	m.rebuildListItems()
	if len(m.list.Items()) != 4 {
		t.Fatalf("expected 4 items without grouping, got %d", len(m.list.Items()))
	}
}

func TestQuitConfirmationShowsHints(t *testing.T) {
	m := newTestVault(testItems())
	m.confirmingQuit = true
	hints, _ := m.FooterContent()
	if !hasHintKey(hints, "y") || !hasHintDesc(hints, "yes") {
		t.Errorf("confirmation hints should contain 'y yes', got %v", hints)
	}
}

func TestQuitConfirmationKeys(t *testing.T) {
	for _, tc := range []struct {
		name     string
		key      tea.KeyPressMsg
		wantOpen bool
		wantQuit bool
	}{
		{"y quits", keyMsg("y"), false, true},
		{"n cancels", keyMsg("n"), false, false},
		{"esc cancels", tea.KeyPressMsg{Code: tea.KeyEscape}, false, false},
		{"other keys keep the dialog", keyMsg("x"), true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestVault(testItems())
			updated, _ := m.Update(keyMsg("q"))
			m = updated.(VaultModel)
			if !m.confirmingQuit {
				t.Fatal("q must open the quit dialog")
			}
			updated, cmd := m.Update(tc.key)
			m = updated.(VaultModel)
			if m.confirmingQuit != tc.wantOpen {
				t.Errorf("confirmingQuit = %v, want %v", m.confirmingQuit, tc.wantOpen)
			}
			if got := hasMsgOfType[QuitMsg](runMsgs(cmd)); got != tc.wantQuit {
				t.Errorf("QuitMsg sent = %v, want %v", got, tc.wantQuit)
			}
		})
	}
}

func TestGenArgsPassword(t *testing.T) {
	m := newTestVault(testItems())
	m.genMode = "password"
	m.genLength = 24
	m.genUppercase = true
	m.genLowercase = true
	m.genNumbers = true
	m.genSpecial = false

	args := m.genArgs()
	if args[0] != "--length" || args[1] != "24" {
		t.Errorf("expected --length 24, got %v", args[:2])
	}
	hasSpecial := false
	for _, a := range args {
		if a == "--special" {
			hasSpecial = true
		}
	}
	if hasSpecial {
		t.Error("--special should not be present when disabled")
	}
}

func TestGenArgsPassphrase(t *testing.T) {
	m := newTestVault(testItems())
	m.genMode = "passphrase"
	m.genWords = 5
	m.genSeparator = "."
	m.genCapitalize = true
	m.genIncludeNum = false

	args := m.genArgs()
	if args[0] != "--passphrase" {
		t.Errorf("first arg should be --passphrase, got %q", args[0])
	}
	hasIncludeNum := false
	for _, a := range args {
		if a == "--includeNumber" {
			hasIncludeNum = true
		}
	}
	if hasIncludeNum {
		t.Error("--includeNumber should not be present when disabled")
	}
}

func TestSyncKeyIgnoredWhileSyncing(t *testing.T) {
	m := newTestVault(testItems())
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m = updated.(VaultModel)
	if cmd == nil || !m.syncing {
		t.Fatal("r must start a sync")
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd != nil {
		t.Error("r during a sync must not start a second sync")
	}
}

func TestReplaceItemsKeepsSelectionAndScrubsOldItems(t *testing.T) {
	old := testItems()
	old[0].Login.Password = "old-pw"
	m := newTestVault(old)
	m.list.Select(1) // Visa

	fresh := testItems()
	// The item moved to the end of the new list.
	fresh = []bwcmd.Item{fresh[0], fresh[2], fresh[1]}
	updated, _ := m.Update(bwcmd.ItemsResult{Items: fresh})
	m = updated.(VaultModel)

	if item := m.selectedItem(); item == nil || item.ID != "2" {
		t.Errorf("selected item = %+v, want ID 2", item)
	}
	if old[0].Login.Password != "" {
		t.Error("the replaced items must be scrubbed")
	}
}

func TestCopiedMsgLeavesClearToRoot(t *testing.T) {
	m := newTestVault(testItems())
	updated, cmd := m.Update(session.CopiedMsg{Field: session.CopyFieldPassword})
	m = updated.(VaultModel)
	if cmd != nil {
		t.Error("the vault must not schedule the clipboard clear")
	}
	if !strings.Contains(m.toast, "Password sent to clipboard") {
		t.Errorf("toast = %q, want a copy toast", m.toast)
	}
}

func TestGenerateResultOutOfOrderIsDropped(t *testing.T) {
	m := newTestVault(testItems())
	updated, _ := m.openGenerator()
	m = updated.(VaultModel)
	first := m.genSeq
	updated, _ = m.updateGenerator(tea.KeyPressMsg{Code: '+', Text: "+"})
	m = updated.(VaultModel)
	latest := m.genSeq

	updated, _ = m.Update(bwcmd.GenerateResult{Password: "new", Seq: latest})
	m = updated.(VaultModel)
	updated, _ = m.Update(bwcmd.GenerateResult{Password: "old", Seq: first})
	m = updated.(VaultModel)
	if m.genPassword != "new" {
		t.Errorf("genPassword = %q, want the result of the latest request", m.genPassword)
	}
}

// vaultWithURI returns a vault whose selected login has uri.
func vaultWithURI(uri string) VaultModel {
	items := []bwcmd.Item{{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Site",
		Login: &bwcmd.Login{Username: "u", URIs: []bwcmd.URI{{URI: uri}}}}}
	return NewVaultModel(items, &session.State{}, 80, 24)
}

// pressOpen presses o on m, runs the command and feeds its result back.
func pressOpen(t *testing.T, m VaultModel) VaultModel {
	t.Helper()
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	m = updated.(VaultModel)
	if cmd == nil {
		t.Fatal("o must return a command")
	}
	updated, _ = m.Update(cmd())
	return updated.(VaultModel)
}

func TestVaultOpenRejectsFileURI(t *testing.T) {
	targets, _ := fakeOpener(t, "exit 0\n")
	m := pressOpen(t, vaultWithURI("file:///etc/passwd"))
	if len(*targets) != 0 {
		t.Errorf("the opener ran for %q", *targets)
	}
	if !m.toastIsError || !strings.Contains(m.toast, "http and https") {
		t.Errorf("toast = %q (error: %v), want an error toast", m.toast, m.toastIsError)
	}
}

func TestVaultOpenFailureShowsToast(t *testing.T) {
	fakeOpener(t, "exit 1\n")
	m := pressOpen(t, vaultWithURI("example.com"))
	if _, status := m.FooterContent(); !m.toastIsError || !strings.Contains(status, "Open URL failed") {
		t.Errorf("status = %q, want an Open URL failed toast", status)
	}
}

func TestVaultOpenSuccessShowsNoError(t *testing.T) {
	targets, _ := fakeOpener(t, "exit 0\n")
	m := pressOpen(t, vaultWithURI("example.com"))
	if !reflect.DeepEqual(*targets, []string{"https://example.com"}) {
		t.Errorf("opener got %q, want https://example.com", *targets)
	}
	if m.toastIsError {
		t.Errorf("toast = %q, want no error", m.toast)
	}
}

func keyMsg(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

// vaultAt returns a vault screen for a terminal of the given size.
func vaultAt(items []bwcmd.Item, width, height int) VaultModel {
	m := NewVaultModel(items, &session.State{Email: "test@test.com"}, width, height)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(VaultModel)
}

// clipboardWrites returns the values that cmd writes to the clipboard and
// the CopiedMsg fields that it reports.
func clipboardWrites(t *testing.T, cmd tea.Cmd, candidates ...string) ([]string, []session.CopyField) {
	t.Helper()
	var values []string
	var fields []session.CopyField
	for _, msg := range runMsgs(cmd) {
		if c, ok := msg.(session.CopiedMsg); ok {
			fields = append(fields, c.Field)
			continue
		}
		for _, want := range candidates {
			if msg == tea.SetClipboard(want)() {
				values = append(values, want)
			}
		}
	}
	return values, fields
}

// assertArea fails when out is not exactly height lines of at most width
// cells.
func assertArea(t *testing.T, out string, width, height int) {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != height {
		t.Errorf("got %d lines, want %d", len(lines), height)
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w > width {
			t.Errorf("line %d is %d cells wide, want at most %d: %q", i, w, width, ansi.Strip(l))
		}
	}
}

// cardColumns finds the overlay card in out and returns the columns of its
// left and right border. It fails when a row of the card does not have
// both borders in those columns.
func cardColumns(t *testing.T, out string) (int, int) {
	t.Helper()
	left, right := -1, -1
	inCard := false
	for _, line := range strings.Split(ansi.Strip(out), "\n") {
		if !inCard {
			i := strings.Index(line, "╭")
			if i < 0 {
				continue
			}
			inCard = true
			left = lipgloss.Width(line[:i])
			right = lipgloss.Width(line[:strings.Index(line, "╮")])
			continue
		}
		edge, endEdge := "│", "│"
		if strings.Contains(line, "╰") {
			edge, endEdge = "╰", "╯"
		}
		l, r := strings.Index(line, edge), strings.LastIndex(line, endEdge)
		if l < 0 || r < 0 || lipgloss.Width(line[:l]) != left || lipgloss.Width(line[:r]) != right {
			t.Errorf("card row lost a border: %q", line)
		}
		if edge == "╰" {
			return left, right
		}
	}
	t.Fatalf("no complete card in:\n%s", ansi.Strip(out))
	return 0, 0
}

func TestCopyNoteCopiesFullNote(t *testing.T) {
	note := "line one\r\n\x1b[8mline two\tend\n"
	items := []bwcmd.Item{{ID: "n", Type: bwcmd.ItemTypeSecureNote, Name: "Keys", Notes: note}}
	m := newTestVault(items)
	_, cmd := m.Update(keyMsg("n"))
	values, fields := clipboardWrites(t, cmd, note)
	if len(values) != 1 {
		t.Fatalf("n must copy the full, unchanged note, got %q", values)
	}
	if len(fields) != 1 || fields[0] != session.CopyFieldNote {
		t.Fatalf("n must report a note copy, got %v", fields)
	}
	updated, _ := m.Update(session.CopiedMsg{Field: session.CopyFieldNote})
	m = updated.(VaultModel)
	if !strings.Contains(m.toast, "Note sent to clipboard") {
		t.Errorf("toast = %q, want it to name the note", m.toast)
	}
}

func TestCopyNoteWithoutNotesCopiesNothing(t *testing.T) {
	m := newTestVault(testItems()) // Gmail has no notes
	_, cmd := m.Update(keyMsg("n"))
	if _, fields := clipboardWrites(t, cmd); len(fields) != 0 {
		t.Errorf("n on an item without notes must not copy, got %v", fields)
	}
}

func TestCopyKeepsControlCharacters(t *testing.T) {
	pw := "p\x1b[8m\r\n\tw\x00rd" //nolint:gosec // fake test password
	user := "user\x1b]8;;x\x07name"
	items := []bwcmd.Item{{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Odd\nname", Login: &bwcmd.Login{Username: user, Password: pw}}}
	m := newTestVault(items)
	_, cmd := m.Update(keyMsg("c"))
	if values, _ := clipboardWrites(t, cmd, pw); len(values) != 1 {
		t.Error("c must copy the stored password without change")
	}
	_, cmd = m.Update(keyMsg("u"))
	if values, _ := clipboardWrites(t, cmd, user); len(values) != 1 {
		t.Error("u must copy the stored username without change")
	}
}

func tenLineNote() bwcmd.Item {
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("line%02d", i))
	}
	return bwcmd.Item{ID: "n", Type: bwcmd.ItemTypeSecureNote, Name: "Keys", Notes: strings.Join(lines, "\n")}
}

func TestDrawerScrollStopsAtLastPage(t *testing.T) {
	m := newTestVault([]bwcmd.Item{tenLineNote()})
	if m.drawerHeight != 8 {
		t.Fatalf("drawer height = %d, want 8", m.drawerHeight)
	}
	for range 20 {
		m = press(t, m, keyMsg("J"))
	}
	// 10 lines in the 6 rows below the drawer header: the last full page
	// starts at line 4.
	if m.drawerScroll != 4 {
		t.Fatalf("after 20 J presses, drawerScroll = %d, want 4", m.drawerScroll)
	}
	m = press(t, m, keyMsg("K"))
	if m.drawerScroll != 3 {
		t.Errorf("after K, drawerScroll = %d, want 3", m.drawerScroll)
	}
	out := m.ViewContent(80, 21)
	if !strings.Contains(out, "line03") || !strings.Contains(out, "line08") || strings.Contains(out, "line02") {
		t.Errorf("drawer must show lines 3 to 8:\n%s", ansi.Strip(out))
	}
}

func TestDrawerScrollIgnoredForLogin(t *testing.T) {
	items := []bwcmd.Item{{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Gmail", Notes: tenLineNote().Notes}}
	m := newTestVault(items)
	for _, k := range []string{"J", "J", "K", "J"} {
		m = press(t, m, keyMsg(k))
		if m.drawerScroll != 0 {
			t.Fatalf("after %s on a login, drawerScroll = %d, want 0", k, m.drawerScroll)
		}
	}
}

func TestOverlaysFitSmallTerminals(t *testing.T) {
	for _, size := range [][2]int{{50, 15}, {30, 10}} {
		width, height := size[0], size[1]
		for _, k := range []string{"?", "p", "q", "d", "T"} {
			m := vaultAt(testItems(), width, height)
			// Do not run the command: p would start bw generate.
			updated, _ := m.Update(keyMsg(k))
			m = updated.(VaultModel)
			out := m.ViewContent(width, height-ui.ChromeHeight)
			assertArea(t, out, width, height-ui.ChromeHeight)
			if width == 50 && k != "d" {
				// 50x15 has room for the whole card of each dialog.
				cardColumns(t, out)
			}
			if width == 50 && k == "?" && !strings.Contains(ansi.Strip(out), "esc close") {
				t.Errorf("at 50x15 the help footer must show:\n%s", ansi.Strip(out))
			}
		}
	}
}

func TestHelpAt80x24ShowsCue(t *testing.T) {
	m := newTestVault(testItems())
	m = press(t, m, keyMsg("?"))
	out := m.ViewContent(80, 24-ui.ChromeHeight)
	assertArea(t, out, 80, 24-ui.ChromeHeight)
	cardColumns(t, out)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "esc close") || !strings.Contains(plain, "↓ more") {
		t.Errorf("help at 80x24 must show its footer and the more cue:\n%s", plain)
	}
}

func TestHelpFitsAfterResize(t *testing.T) {
	m := newTestVault(testItems())
	m = press(t, m, keyMsg("?"))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 15})
	m = updated.(VaultModel)
	out := m.ViewContent(50, 15-ui.ChromeHeight)
	assertArea(t, out, 50, 15-ui.ChromeHeight)
	left, right := cardColumns(t, out)
	if right-left+1 > 50 {
		t.Errorf("help card is %d cells wide, want at most 50", right-left+1)
	}
	// The card is sized for the new area, so nothing in it is cut: the
	// header and the footer of the help both show.
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "Keybindings") || !strings.Contains(plain, "esc close") {
		t.Errorf("help card is cut after the resize:\n%s", plain)
	}
}

func TestOverlayDrawerKeepsBorders(t *testing.T) {
	long := strings.Join(strings.Fields(strings.Repeat("alpha bravo charlie delta echo foxtrot golf ", 4)), " ")[:157]
	items := []bwcmd.Item{
		{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Spoof\nFAKE-ROW", Login: &bwcmd.Login{Username: "a\tb"}, Notes: "first line\r\nsecond"},
		{ID: "2", Type: bwcmd.ItemTypeLogin, Name: "\x1b]8;;https://evil.example\x07link\x1b]8;;\x07", Notes: "\x1b[8mhidden\r"},
		{ID: "3", Type: bwcmd.ItemTypeSecureNote, Name: "Pasted", Notes: long},
	}
	for i := range items {
		m := newTestVault(items)
		m.list.Select(i)
		m.syncSelection()
		m = press(t, m, keyMsg("d"))
		out := m.ViewContent(80, 21)
		assertArea(t, out, 80, 21)
		left, right := cardColumns(t, out)
		if w := right - left + 1; w > overlayDrawerWidth(80)+ui.OverlayEdgeWidth {
			t.Errorf("item %d: card is %d cells wide, want at most %d", i, w, overlayDrawerWidth(80)+ui.OverlayEdgeWidth)
		}
	}
	// The long note line wraps inside the card, so all of it shows.
	m := newTestVault(items)
	m.list.Select(2)
	m.syncSelection()
	m = press(t, m, keyMsg("d"))
	plain := strings.Join(strings.Fields(strings.ReplaceAll(ansi.Strip(m.ViewContent(80, 21)), "│", " ")), " ")
	if !strings.Contains(plain, long) {
		t.Errorf("overlay card does not show all of the note:\n%s", plain)
	}
}

func TestVaultFooterShowsHelpAndQuitAt80(t *testing.T) {
	m := newTestVault(testItems())
	m.sess.LastSync = time.Now()
	hints, status := m.FooterContent()
	out := ansi.Strip(ui.RenderFooter(hints, status, 80))
	for _, want := range []string{"? help", "q quit", "synced just now"} {
		if !strings.Contains(out, want) {
			t.Errorf("footer at 80 columns = %q, want it to contain %q", out, want)
		}
	}
}

func TestVaultFooterShowsAppliedFilter(t *testing.T) {
	m := newTestVault(testItems())
	// The footer needs only the filter text, not the matches, so the
	// commands are not run.
	for _, msg := range []tea.KeyPressMsg{keyMsg("/"), keyMsg("g"), keyMsg("m"), {Code: tea.KeyEnter}} {
		updated, _ := m.Update(msg)
		m = updated.(VaultModel)
	}
	if m.list.FilterState() != list.FilterApplied {
		t.Fatalf("filter state = %v, want FilterApplied", m.list.FilterState())
	}
	m.sess.LastSync = time.Now()
	hints, status := m.FooterContent()
	out := ansi.Strip(ui.RenderFooter(hints, status, 80))
	for _, want := range []string{"filter: gm", "esc clear", "? help", "q quit", "synced just now"} {
		if !strings.Contains(out, want) {
			t.Errorf("footer at 80 columns = %q, want it to contain %q", out, want)
		}
	}
}

func TestVaultFooterLongFilterKeepsHelpAndQuit(t *testing.T) {
	m := newTestVault(testItems())
	// As in TestVaultFooterShowsAppliedFilter, the commands are not run.
	msgs := []tea.KeyPressMsg{keyMsg("/")}
	for _, r := range "a very long filter text that is cut" {
		msgs = append(msgs, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	for _, msg := range append(msgs, tea.KeyPressMsg{Code: tea.KeyEnter}) {
		updated, _ := m.Update(msg)
		m = updated.(VaultModel)
	}
	if m.list.FilterState() != list.FilterApplied {
		t.Fatalf("filter state = %v, want FilterApplied", m.list.FilterState())
	}
	m.sess.LastSync = time.Now()
	hints, status := m.FooterContent()
	out := ansi.Strip(ui.RenderFooter(hints, status, 80))
	for _, want := range []string{"filter: a very long", "…", "esc clear", "? help", "q quit", "synced just now"} {
		if !strings.Contains(out, want) {
			t.Errorf("footer at 80 columns = %q, want it to contain %q", out, want)
		}
	}
}

// pressN sends the key k to m n times.
func pressN(t *testing.T, m VaultModel, k string, n int) VaultModel {
	t.Helper()
	for range n {
		m = press(t, m, keyMsg(k))
	}
	return m
}

func TestOverlayDrawerScrollReachesLastLine(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {80, 14}, {80, 12}, {40, 12}, {80, 10}} {
		width, height := size[0], size[1]
		m := vaultAt([]bwcmd.Item{tenLineNote()}, width, height)
		m = press(t, m, keyMsg("d"))
		m = pressN(t, m, "J", 30)
		out := m.ViewContent(width, height-ui.ChromeHeight)
		assertArea(t, out, width, height-ui.ChromeHeight)
		cardColumns(t, out)
		if !strings.Contains(ansi.Strip(out), "line09") {
			t.Errorf("%dx%d: after 30 J presses, the last note line must show (drawerScroll = %d):\n%s",
				width, height, m.drawerScroll, ansi.Strip(out))
		}
		// One K press moves the note back by one line.
		before := m.drawerScroll
		if m = press(t, m, keyMsg("K")); m.drawerScroll != before-1 {
			t.Errorf("%dx%d: after K, drawerScroll = %d, want %d", width, height, m.drawerScroll, before-1)
		}
	}
}

func TestInlineDrawerFitsShortTerminals(t *testing.T) {
	for height := 7; height <= 24; height++ {
		m := vaultAt([]bwcmd.Item{tenLineNote()}, 80, height)
		m = pressN(t, m, "J", 30)
		// The view must fit the content area by itself. The frame cuts
		// rows past the area, and those would be the last drawer rows.
		out := m.ViewContent(80, height-ui.ChromeHeight)
		if got := len(strings.Split(out, "\n")); got != height-ui.ChromeHeight {
			t.Errorf("height %d: view has %d lines, want %d", height, got, height-ui.ChromeHeight)
		}
		// From 8 rows the drawer has a row for the note.
		if height >= 8 && !strings.Contains(ansi.Strip(out), "line09") {
			t.Errorf("height %d: after 30 J presses, the last note line must show:\n%s", height, ansi.Strip(out))
		}
	}
}

func TestDrawerResetFitsShortTerminal(t *testing.T) {
	m := vaultAt(testItems(), 80, 12)
	m = pressN(t, m, "-", 3)
	m = press(t, m, keyMsg("="))
	if want := clampDrawerHeight(ui.DefaultDrawerHeight, 12); m.drawerHeight != want {
		t.Errorf("after =, drawer height = %d, want %d", m.drawerHeight, want)
	}
	assertArea(t, m.ViewContent(80, 12-ui.ChromeHeight), 80, 12-ui.ChromeHeight)
}

func TestThemePickerCursorVisibleOnShortTerminal(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {60, 14}, {80, 24}} {
		width, height := size[0], size[1]
		m := vaultAt(testItems(), width, height)
		m = press(t, m, keyMsg("T"))
		for range len(ui.ThemeNames) - 1 {
			updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			m = updated.(VaultModel)
		}
		last := ui.ThemeNames[len(ui.ThemeNames)-1]
		if *m.selectedTheme != last {
			t.Fatalf("%dx%d: cursor on %q, want %q", width, height, *m.selectedTheme, last)
		}
		out := m.ViewContent(width, height-ui.ChromeHeight)
		assertArea(t, out, width, height-ui.ChromeHeight)
		cardColumns(t, out)
		if !strings.Contains(ansi.Strip(out), "> "+last) {
			t.Errorf("%dx%d: the cursor row must show:\n%s", width, height, ansi.Strip(out))
		}
	}
	// A tall screen shows all themes.
	m := press(t, vaultAt(testItems(), 80, 24), keyMsg("T"))
	plain := ansi.Strip(m.ViewContent(80, 24-ui.ChromeHeight))
	for _, name := range ui.ThemeNames {
		if !strings.Contains(plain, name) {
			t.Errorf("at 80x24 the picker must show %q:\n%s", name, plain)
		}
	}
}

func TestThemePickerFitsAfterResize(t *testing.T) {
	m := press(t, vaultAt(testItems(), 80, 24), keyMsg("T"))
	for range 4 {
		updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = updated.(VaultModel)
	}
	want := *m.selectedTheme
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	m = updated.(VaultModel)
	if cmd == nil {
		t.Fatal("a resize that changes the picker height must rebuild the picker")
	}
	out := m.ViewContent(40, 12-ui.ChromeHeight)
	assertArea(t, out, 40, 12-ui.ChromeHeight)
	cardColumns(t, out)
	if !strings.Contains(ansi.Strip(out), "> "+want) {
		t.Errorf("after the resize the cursor must stay on %q and show:\n%s", want, ansi.Strip(out))
	}
	// The same size again must not rebuild the picker. Form.Init asks for
	// the window size, so a rebuild would cause another resize message.
	if _, cmd := m.Update(tea.WindowSizeMsg{Width: 40, Height: 12}); cmd != nil {
		t.Error("a resize to the same size must not rebuild the picker")
	}
}

func TestSyncErrorToastIsOneFooterLine(t *testing.T) {
	m := newTestVault(testItems())
	err := errors.New("bw sync: exit status 1\nmore detail about the failure from the server\n\x1b[31mlast line\x1b[0m")
	updated, _ := m.Update(bwcmd.SyncResult{Err: err})
	m = updated.(VaultModel)
	hints, status := m.FooterContent()
	for _, width := range []int{40, 80, 120} {
		out := ui.RenderFooter(hints, status, width)
		if strings.Contains(out, "\n") {
			t.Errorf("width %d: footer has more than one line: %q", width, out)
		}
		if w := lipgloss.Width(out); w != width {
			t.Errorf("width %d: footer is %d cells wide", width, w)
		}
		if !strings.Contains(ansi.Strip(out), "Sync failed: bw sync") {
			t.Errorf("width %d: footer = %q, want the start of the error", width, ansi.Strip(out))
		}
	}
}

// copyTableItems has one item of each type with every copyable field set.
// No item has master password re-prompt.
func copyTableItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "l", Type: bwcmd.ItemTypeLogin, Name: "Login", Login: &bwcmd.Login{Username: "user", Password: "login-pw"}},
		{ID: "c", Type: bwcmd.ItemTypeCard, Name: "Card", Card: &bwcmd.Card{Number: "4242424242424242", Code: "123"}},
		{ID: "i", Type: bwcmd.ItemTypeIdentity, Name: "Identity", Identity: &bwcmd.Identity{SSN: "123-45-6789", Email: "id@example.com"}},
		{ID: "k", Type: bwcmd.ItemTypeSSHKey, Name: "SSH", SSHKey: &bwcmd.SSHKey{PrivateKey: "PRIVATE KEY", PublicKey: "ssh-ed25519 PUBLIC"}},
		{ID: "n", Type: bwcmd.ItemTypeSecureNote, Name: "Note", Notes: "note body"},
	}
}

func TestCopyKeysNameTheField(t *testing.T) {
	candidates := []string{"user", "login-pw", "4242424242424242", "123", "123-45-6789", "id@example.com", "PRIVATE KEY", "ssh-ed25519 PUBLIC", "note body"}
	tests := []struct {
		item  int
		key   string
		value string // "" when the key copies nothing
		field session.CopyField
		label string
	}{
		{0, "c", "login-pw", session.CopyFieldPassword, "Password"},
		{0, "u", "user", session.CopyFieldUsername, "Username"},
		{0, "v", "", 0, ""},
		{1, "c", "4242424242424242", session.CopyFieldCardNumber, "Card number"},
		{1, "u", "", 0, ""},
		{1, "v", "123", session.CopyFieldCVV, "Security code"},
		{2, "c", "123-45-6789", session.CopyFieldSSN, "SSN"},
		{2, "u", "id@example.com", session.CopyFieldEmail, "Email"},
		{2, "v", "", 0, ""},
		{3, "c", "PRIVATE KEY", session.CopyFieldPrivateKey, "SSH private key"},
		{3, "u", "ssh-ed25519 PUBLIC", session.CopyFieldPublicKey, "Public key"},
		{3, "v", "", 0, ""},
		{4, "c", "", 0, ""},
		{4, "u", "", 0, ""},
		{4, "v", "", 0, ""},
	}
	for _, tt := range tests {
		m := newTestVault(copyTableItems())
		m.list.Select(tt.item)
		m.syncSelection()
		name := fmt.Sprintf("%s on %s", tt.key, m.selectedItem().Name)

		// No item has re-prompt, so one key press copies.
		updated, cmd := m.Update(keyMsg(tt.key))
		m = updated.(VaultModel)
		values, fields := clipboardWrites(t, cmd, candidates...)
		if tt.value == "" {
			if len(values) != 0 || len(fields) != 0 {
				t.Errorf("%s: copied %q (fields %v), want no copy", name, values, fields)
			}
			continue
		}
		if len(values) != 1 || values[0] != tt.value {
			t.Errorf("%s: copied %q, want %q", name, values, tt.value)
		}
		if len(fields) != 1 || fields[0] != tt.field {
			t.Errorf("%s: CopiedMsg fields = %v, want %v", name, fields, tt.field)
			continue
		}
		updated, _ = m.Update(session.CopiedMsg{Field: fields[0]})
		m = updated.(VaultModel)
		_, status := m.FooterContent()
		if want := tt.label + " sent to clipboard"; !strings.Contains(ansi.Strip(status), want) {
			t.Errorf("%s: footer status = %q, want %q", name, ansi.Strip(status), want)
		}
	}
}

func TestCopyEmptyUsernameWritesNothing(t *testing.T) {
	items := []bwcmd.Item{{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Site", Login: &bwcmd.Login{Password: "pw"}}}
	m := newTestVault(items)
	updated, cmd := m.Update(keyMsg("u"))
	m = updated.(VaultModel)
	for _, msg := range runMsgs(cmd) {
		if _, ok := msg.(session.CopiedMsg); ok || msg == tea.SetClipboard("")() {
			t.Fatalf("u on a login with no username sent %#v, want no clipboard write", msg)
		}
	}
	if !strings.Contains(m.toast, "No username") || m.toastIsError {
		t.Errorf("toast = %q, want an info toast about the missing username", m.toast)
	}
}

func TestToastExpiresAndFooterShowsSyncAge(t *testing.T) {
	orig := toastDuration
	t.Cleanup(func() { toastDuration = orig })
	toastDuration = time.Millisecond

	m := newTestVault(testItems())
	m.sess.LastSync = time.Now().Add(-3*time.Minute - 30*time.Second)
	updated, first := m.Update(keyMsg("d"))
	m = updated.(VaultModel)
	updated, second := m.Update(keyMsg("d"))
	m = updated.(VaultModel)
	if _, status := m.FooterContent(); !strings.Contains(status, "Drawer: inline") || strings.Contains(status, "synced") {
		t.Fatalf("status = %q, want the toast and no sync age", status)
	}
	if first == nil || second == nil {
		t.Fatal("a toast must return the command that ends it")
	}
	// The timer of the first toast must not end the second toast.
	updated, _ = m.Update(first())
	m = updated.(VaultModel)
	if _, status := m.FooterContent(); !strings.Contains(status, "Drawer: inline") {
		t.Errorf("status = %q: an older timer ended a newer toast", status)
	}
	expired, ok := second().(toastExpiredMsg)
	if !ok || expired.gen != m.toastGen {
		t.Fatalf("the toast command sent %#v, want the toastExpiredMsg of the latest toast", expired)
	}
	updated, _ = m.Update(expired)
	m = updated.(VaultModel)
	if _, status := m.FooterContent(); !strings.Contains(status, "synced 3m ago") {
		t.Errorf("status = %q, want the sync age after the toast ends", status)
	}
}

func TestGeneratorErrorIsNotCopied(t *testing.T) {
	m := newTestVault(testItems())
	updated, _ := m.openGenerator()
	m = updated.(VaultModel)
	updated, _ = m.Update(bwcmd.GenerateResult{Err: errors.New("boom"), Seq: m.genSeq})
	m = updated.(VaultModel)

	_, cmd := m.Update(keyMsg("c"))
	for _, msg := range runMsgs(cmd) {
		if _, ok := msg.(session.CopiedMsg); ok {
			t.Fatal("c after a failed generate must not report a copy")
		}
		for _, text := range []string{"boom", "Error: boom"} {
			if msg == tea.SetClipboard(text)() {
				t.Fatalf("c after a failed generate copied %q", text)
			}
		}
	}
	if cmd != nil {
		t.Error("c after a failed generate must return no command")
	}
	// The card shows the error in the error style. The overlay writes the
	// styles again, so the check uses the card itself.
	card := m.renderGeneratorCard(80)
	if !strings.Contains(card, ui.StyleError.Render("boom")) {
		t.Errorf("the generator card must show the error in the error style:\n%s", ansi.Strip(card))
	}
	if !strings.Contains(ansi.Strip(m.ViewContent(80, 24-ui.ChromeHeight)), "boom") {
		t.Error("the generator overlay must show the error")
	}
}

func TestRepromptCopyOpensPasswordPrompt(t *testing.T) {
	items := []bwcmd.Item{{ID: "1", Type: bwcmd.ItemTypeLogin, Name: "Bank", Reprompt: bwcmd.RepromptPassword,
		Login: &bwcmd.Login{Username: "me", Password: "bank-pw"}}}
	m := newTestVault(items)
	updated, cmd := m.Update(keyMsg("c"))
	m = updated.(VaultModel)
	for _, msg := range runMsgs(cmd) {
		if _, ok := msg.(session.CopiedMsg); ok || msg == tea.SetClipboard("bank-pw")() {
			t.Fatalf("c on a re-prompt item sent %#v before the password check", msg)
		}
	}
	if !m.reprompt.open {
		t.Fatal("c on a re-prompt item must open the password prompt")
	}
	plain := ansi.Strip(m.ViewContent(80, 24-ui.ChromeHeight))
	if !strings.Contains(plain, "Master password re-prompt") || !strings.Contains(plain, "Master password") {
		t.Errorf("the password prompt must show:\n%s", plain)
	}
	if hints, _ := m.FooterContent(); !hasHintKey(hints, "esc") || !hasHintDesc(hints, "cancel") {
		t.Errorf("footer hints = %v, want esc cancel", hints)
	}
}

func awsItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "p", Type: bwcmd.ItemTypeLogin, Name: "AWS (Prod)", Login: &bwcmd.Login{Username: "prod"}},
		{ID: "d", Type: bwcmd.ItemTypeLogin, Name: "AWS (Dev)", Login: &bwcmd.Login{Username: "dev"}},
		{ID: "l", Type: bwcmd.ItemTypeLogin, Name: "Laws of Motion", Login: &bwcmd.Login{Username: "newton"}},
		{ID: "g", Type: bwcmd.ItemTypeLogin, Name: "Gmail", Login: &bwcmd.Login{Username: "me"}},
	}
}

func TestGroupingDuringFilterKeepsMatches(t *testing.T) {
	m := vaultAt(awsItems(), 80, 24)
	m = typeKeys(t, m, "/prod")
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.list.FilterState() != list.FilterApplied {
		t.Fatalf("filter state = %v, want applied", m.list.FilterState())
	}
	m = press(t, m, tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	if !m.groups.enabled {
		t.Fatal("ctrl+g must turn grouping on")
	}
	if got := rowNames(m); !reflect.DeepEqual(got, []string{"AWS (Prod)"}) {
		t.Errorf("rows = %q, want the match AWS (Prod)", got)
	}
	if plain := ansi.Strip(m.ViewContent(80, 24-ui.ChromeHeight)); !strings.Contains(plain, "AWS (Prod)") {
		t.Errorf("AWS (Prod) must stay visible:\n%s", plain)
	}
}

// filterItems returns items with three matches for the filter "gi".
func filterItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "a", Type: bwcmd.ItemTypeLogin, Name: "Alpha"},
		{ID: "h", Type: bwcmd.ItemTypeLogin, Name: "GitHub"},
		{ID: "t", Type: bwcmd.ItemTypeLogin, Name: "Gitea"},
		{ID: "b", Type: bwcmd.ItemTypeLogin, Name: "Bravo"},
		{ID: "l", Type: bwcmd.ItemTypeLogin, Name: "GitLab"},
	}
}

func TestFilterDownMovesCursorAndKeepsInput(t *testing.T) {
	m := vaultAt(filterItems(), 80, 24)
	m = typeKeys(t, m, "/gi")
	if n := len(m.list.VisibleItems()); n < 2 {
		t.Fatalf("the filter gi has %d matches, want at least 2", n)
	}
	second := m.list.VisibleItems()[1].(VaultItem).ID
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if !m.list.SettingFilter() {
		t.Fatalf("after down, filter state = %v, want the input to stay open", m.list.FilterState())
	}
	if item := m.selectedItem(); item == nil || item.ID != second {
		t.Errorf("after down, selected = %+v, want the second match %q", item, second)
	}
	// Up moves back to the first match.
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.list.Index() != 0 || !m.list.SettingFilter() {
		t.Errorf("after up, index = %d and input open = %v, want 0 and true", m.list.Index(), m.list.SettingFilter())
	}
}

func TestFilterKeysAfterDownGoToInput(t *testing.T) {
	m := vaultAt(filterItems(), 80, 24)
	m = typeKeys(t, m, "/gi")
	m = press(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	updated, cmd := m.Update(keyMsg("l"))
	m = updated.(VaultModel)
	for _, msg := range runMsgs(cmd) {
		if _, ok := msg.(LockMsg); ok {
			t.Fatal("l in the filter input must not lock the vault")
		}
	}
	if got := m.list.FilterValue(); got != "gil" {
		t.Errorf("filter text = %q, want %q", got, "gil")
	}
	if !m.list.SettingFilter() {
		t.Error("the filter input must stay open")
	}
}

func TestOnlyEnterAcceptsFilter(t *testing.T) {
	keys := map[string]tea.KeyPressMsg{
		"tab":       {Code: tea.KeyTab},
		"shift+tab": {Code: tea.KeyTab, Mod: tea.ModShift},
		"ctrl+j":    {Code: 'j', Mod: tea.ModCtrl},
		"ctrl+k":    {Code: 'k', Mod: tea.ModCtrl},
	}
	for name, k := range keys {
		m := typeKeys(t, vaultAt(filterItems(), 80, 24), "/gi")
		m = press(t, m, k)
		if !m.list.SettingFilter() {
			t.Errorf("%s must not accept the filter: filter state = %v", name, m.list.FilterState())
		}
	}
	m := typeKeys(t, vaultAt(filterItems(), 80, 24), "/gi")
	if m = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter}); m.list.FilterState() != list.FilterApplied {
		t.Errorf("enter must accept the filter: filter state = %v", m.list.FilterState())
	}
}
