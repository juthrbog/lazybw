package screens

import (
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/session"
	"github.com/juthrbog/lazybw/ui"
)

// repromptUnlockCmd is the bw command that checks the master password for
// an item with master password re-prompt. Tests replace it.
var repromptUnlockCmd = bwcmd.Unlock

// errNoSessionKey is the error for a password check that returns neither
// a token nor an error.
var errNoSessionKey = errors.New("bw did not return a session key")

// secretAction is a copy of a protected field. For an item with master
// password re-prompt, the vault runs it only after a password check.
type secretAction int

const (
	secretMain secretAction = iota // c: the password, card number, SSN or SSH private key
	secretTOTP                     // t: the TOTP code
	secretCVV                      // v: the card security code
	secretNote                     // n: the note
)

// repromptState is the master password prompt of the vault screen. The
// prompt shows while the overlay of the vault is overlayReprompt.
type repromptState struct {
	itemID   string       // the item that the prompt is for
	action   secretAction // the copy to run after the check passes
	form     *huh.Form
	password *string
	seq      uint64 // the number of the check that runs, or 0 when none runs
	// queued is a submitted password that waits for the check of a closed
	// prompt to end. It is never empty, because the form does not accept an
	// empty password.
	queued  string
	errText string
}

// busy reports whether the prompt waits for a password check.
func (r repromptState) busy() bool {
	return r.seq != 0 || r.queued != ""
}

// repromptResultMsg is the result of a password check. It is not a
// bwcmd.UnlockResult, so that the unlock screen cannot take a late result
// for the result of its own unlock.
type repromptResultMsg struct {
	res   bwcmd.UnlockResult
	epoch uint64 // the session epoch when the check started
	seq   uint64
}

// checkMasterPassword runs bw unlock with password and returns its result
// as a repromptResultMsg.
func checkMasterPassword(password string, epoch, seq uint64) tea.Cmd {
	unlock := repromptUnlockCmd(password)
	return func() tea.Msg {
		res, _ := unlock().(bwcmd.UnlockResult)
		return repromptResultMsg{res: res, epoch: epoch, seq: seq}
	}
}

// buildRepromptForm returns the form that asks for the master password,
// for a content area of the given width. The form shows no help and no
// errors: the footer shows the keys, and renderReprompt shows the error
// above the input, where a short area does not cut it.
func buildRepromptForm(password *string, width int) *huh.Form {
	field := huh.NewInput().
		Title("Master password").
		EchoMode(huh.EchoModePassword).
		Validate(validatePassword).
		Value(password)
	return styleForm(huh.NewForm(huh.NewGroup(field))).
		WithShowHelp(false).
		WithShowErrors(false).
		WithWidth(repromptFormWidth(width))
}

// repromptFormWidth returns the width of the password form for a content
// area of the given width: the width of the unlock form, or the width
// inside the card when that is less.
func repromptFormWidth(width int) int {
	if w := width - ui.OverlayEdgeWidth; w > 0 && w < lockedFormWidth {
		return w
	}
	return lockedFormWidth
}

// secretFor returns the value that act copies from the selected item, and
// its field. It reports false when the item has no such value.
func (m *VaultModel) secretFor(act secretAction) (string, session.CopyField, bool) {
	item := m.selectedItem()
	if item == nil {
		return "", 0, false
	}
	switch act {
	case secretMain:
		switch {
		case item.Type == bwcmd.ItemTypeIdentity && item.Identity != nil && item.Identity.SSN != "":
			return item.Identity.SSN, session.CopyFieldSSN, true
		case item.Type == bwcmd.ItemTypeSSHKey && item.SSHKey != nil && item.SSHKey.PrivateKey != "":
			return item.SSHKey.PrivateKey, session.CopyFieldPrivateKey, true
		case item.Login != nil && item.Login.Password != "":
			return item.Login.Password, session.CopyFieldPassword, true
		case item.Card != nil && item.Card.Number != "":
			return item.Card.Number, session.CopyFieldCardNumber, true
		}
	case secretTOTP:
		// Compute the code now. The cached code can be up to a second old
		// at a step change.
		if m.totpParams != nil {
			m.computeTOTP()
			return m.totpCode, session.CopyFieldTOTP, true
		}
	case secretCVV:
		if item.Card != nil && item.Card.Code != "" {
			return item.Card.Code, session.CopyFieldCVV, true
		}
	case secretNote:
		// Copy the stored note, not the sanitized text that the drawer shows.
		if item.Notes != "" {
			return item.Notes, session.CopyFieldNote, true
		}
	}
	return "", 0, false
}

// copySecret copies the field that act names from the selected item. For
// an item with master password re-prompt, it first opens the password
// prompt, unless the user entered the password for the item already. It
// reports false when the item has no such field, so that the list gets the
// key.
func (m VaultModel) copySecret(act secretAction) (tea.Model, tea.Cmd, bool) {
	if act == secretTOTP && m.totpInvalid {
		return m, m.setErrorToast("Invalid TOTP secret"), true
	}
	value, field, ok := m.secretFor(act)
	if !ok {
		return m, nil, false
	}
	if m.needsReprompt() {
		return m, m.openReprompt(act), true
	}
	return m, session.CopyToClipboard(value, field), true
}

// repromptVerified reports whether the user entered the master password
// for item while it stayed selected.
func (m VaultModel) repromptVerified(item *bwcmd.Item) bool {
	return item != nil && item.ID != "" && item.ID == m.verifiedID
}

// needsReprompt reports whether the selected item has master password
// re-prompt and the user did not enter the password for it yet.
func (m VaultModel) needsReprompt() bool {
	item := m.selectedItem()
	return item != nil && item.Reprompt == bwcmd.RepromptPassword && !m.repromptVerified(item)
}

// openReprompt opens the password prompt for the selected item. act is
// the copy to run after the check passes.
func (m *VaultModel) openReprompt(act secretAction) tea.Cmd {
	m.reprompt = repromptState{
		itemID:   m.selectedItem().ID,
		action:   act,
		password: new(string),
	}
	m.reprompt.form = buildRepromptForm(m.reprompt.password, m.width)
	m.overlay = overlayReprompt
	return m.reprompt.form.Init()
}

// closeReprompt closes the password prompt and drops what the user typed,
// also a password that waits for a check. A check that runs goes on, but
// its result only stores the new token.
func (m *VaultModel) closeReprompt() {
	if m.reprompt.password != nil {
		*m.reprompt.password = ""
	}
	m.reprompt = repromptState{}
	if m.overlay == overlayReprompt {
		m.overlay = overlayNone
	}
}

// updateReprompt handles a message while the password prompt shows. Esc
// closes the prompt. While a check runs, the form gets no keys.
func (m VaultModel) updateReprompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if key.String() == "esc" {
			m.closeReprompt()
			return m, nil
		}
		if m.reprompt.busy() {
			return m, nil
		}
	}

	form, cmd := m.reprompt.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.reprompt.form = f
	}

	switch m.reprompt.form.State {
	case huh.StateCompleted:
		return m.submitReprompt()
	case huh.StateAborted:
		m.closeReprompt()
		return m, nil
	}
	return m, cmd
}

// submitReprompt starts the check of the typed password. As in the unlock
// screen, the completed form still holds the password, so the prompt gets
// an empty form in its place.
func (m VaultModel) submitReprompt() (tea.Model, tea.Cmd) {
	pw := *m.reprompt.password
	*m.reprompt.password = ""
	m.reprompt.password = new(string)
	m.reprompt.form = buildRepromptForm(m.reprompt.password, m.width)
	m.reprompt.errText = ""
	if m.checking {
		// A check from a prompt that the user closed still runs. A second
		// bw unlock at the same time could store a token that does not
		// work, so this check starts when that check ends.
		m.reprompt.queued = pw
		return m, nil
	}
	return m, m.startCheck(pw)
}

// startCheck starts the check of password for the open prompt.
func (m *VaultModel) startCheck(password string) tea.Cmd {
	m.checking = true
	m.checkSeq++
	m.reprompt.seq = m.checkSeq
	return checkMasterPassword(password, m.sess.Epoch, m.checkSeq)
}

// handleRepromptResult handles the result of a password check. A passed
// check runs the copy that the user asked for. A failed check shows an
// error in the prompt and copies nothing.
func (m VaultModel) handleRepromptResult(msg repromptResultMsg) (tea.Model, tea.Cmd) {
	if msg.epoch != m.sess.Epoch {
		// The session ended while bw ran. Drop the result.
		return m, nil
	}
	m.checking = false
	err := msg.res.Err
	if err == nil && msg.res.Token == "" {
		err = errNoSessionKey
	}
	if err == nil {
		// bw unlock made a new session key, and the old one stopped
		// working. Store it even when the user closed the prompt, so that
		// the next bw command works.
		m.sess.ReplaceToken(msg.res.Token)
	}
	if m.overlay != overlayReprompt || m.reprompt.seq != msg.seq {
		// The user closed the prompt of this check. A password that the
		// user submitted in a new prompt waited for this check to end.
		if m.overlay == overlayReprompt && m.reprompt.queued != "" {
			pw := m.reprompt.queued
			m.reprompt.queued = ""
			return m, m.startCheck(pw)
		}
		return m, nil
	}
	if err != nil {
		m.reprompt.seq = 0
		m.reprompt.errText = userError(err, actionReprompt)
		return m, m.reprompt.form.Init()
	}
	itemID, act := m.reprompt.itemID, m.reprompt.action
	m.closeReprompt()
	item := m.selectedItem()
	if item == nil || item.ID != itemID {
		return m, nil
	}
	m.verifiedID = itemID
	value, field, ok := m.secretFor(act)
	if !ok {
		return m, nil
	}
	return m, session.CopyToClipboard(value, field)
}

// repromptText is the message of the password prompt while it shows no
// error.
const repromptText = "Enter your master password to continue."

// renderReprompt renders the password prompt card for a content area of
// the given size. An error shows in place of the description, above the
// input, so that a short area does not cut it.
func (m VaultModel) renderReprompt(width, contentHeight int) string {
	textW := max(1, width-ui.OverlayEdgeWidth)
	title := ui.StyleTitle.Render(ansi.Wrap("Master password re-prompt", textW, ""))

	errText := m.reprompt.errText
	body := ui.StyleFaint.Render(ansi.Wrap("Checking the master password"+ui.Glyphs.Ellipsis, textW, ""))
	if !m.reprompt.busy() {
		body = m.reprompt.form.View()
		// The form does not show its own error. That error is newer than
		// errText, because each submit gives the prompt a new form.
		if errs := m.reprompt.form.Errors(); len(errs) > 0 {
			errText = errs[0].Error()
		}
	}

	message := ansi.Wrap(repromptText, min(textW, lockedFormWidth), "")
	if errText != "" {
		// errorBlock keeps a 2-cell margin on each side. Give it the card
		// width plus the margins, so that the error fits in the card.
		message = errorBlock(errText, textW+4, lockedFormWidth)
	}
	return fitReprompt(contentHeight-ui.OverlayEdgeHeight, title, message, body, errText != "")
}

// fitReprompt stacks the title, the message and the body of the password
// prompt in at most rows rows. When they do not fit, it drops the blank
// rows first. Then, for an error, it drops the title and then the last
// rows of the error. For a description, it drops the description and then
// the title. When the body alone does not fit, it keeps the last rows of
// the body, where the input is. rows <= 0 means no limit, as in
// ui.RenderOverlay.
func fitReprompt(rows int, title, message, body string, isError bool) string {
	stacks := []string{
		title + "\n\n" + message + "\n\n" + body,
		title + "\n" + message + "\n" + body,
	}
	if isError {
		lines := strings.Split(message, "\n")
		for n := len(lines); n > 0; n-- {
			stacks = append(stacks, strings.Join(lines[:n], "\n")+"\n"+body)
		}
	} else {
		stacks = append(stacks, title+"\n"+body)
	}
	for _, s := range stacks {
		if rows <= 0 || lipgloss.Height(s) <= rows {
			return s
		}
	}
	lines := strings.Split(body, "\n")
	return strings.Join(lines[max(0, len(lines)-rows):], "\n")
}

// repromptHints returns the footer hints while the password prompt shows.
func (m VaultModel) repromptHints() []ui.HintBinding {
	if m.reprompt.busy() {
		return []ui.HintBinding{{Key: "esc", Desc: "cancel"}}
	}
	return []ui.HintBinding{
		{Key: "enter", Desc: "submit"},
		{Key: "esc", Desc: "cancel"},
	}
}
