package screens

import (
	"errors"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/ui"
)

func hasHintDesc(hints []ui.HintBinding, desc string) bool {
	for _, h := range hints {
		if h.Desc == desc {
			return true
		}
	}
	return false
}

func hasHintKey(hints []ui.HintBinding, key string) bool {
	for _, h := range hints {
		if h.Key == key {
			return true
		}
	}
	return false
}

func TestLockedFooterContentUnlock(t *testing.T) {
	m := NewLockedModel(false)
	hints, status := m.FooterContent()
	if !hasHintDesc(hints, "unlock") {
		t.Errorf("unlock mode hints should contain 'unlock', got %v", hints)
	}
	if status != "" {
		t.Errorf("status should be empty, got %q", status)
	}
}

func TestLockedFooterContentLogin(t *testing.T) {
	m := NewLockedModel(true)
	hints, _ := m.FooterContent()
	if !hasHintDesc(hints, "submit") {
		t.Errorf("login mode hints should contain 'submit', got %v", hints)
	}
}

func TestLockedFooterContentUnlocking(t *testing.T) {
	m := NewLockedModel(false)
	m.state = lockedUnlocking
	hints, status := m.FooterContent()
	if hints != nil {
		t.Errorf("unlocking hints should be nil, got %v", hints)
	}
	if status != "" {
		t.Errorf("unlocking status should be empty, got %q", status)
	}
}

func TestLockedViewContentUnlock(t *testing.T) {
	m := NewLockedModel(false)
	out := m.ViewContent(80, 24)
	if !strings.Contains(out, "Vault is locked") {
		t.Error("unlock view should contain 'Vault is locked'")
	}
}

func TestLockedViewContentLogin(t *testing.T) {
	m := NewLockedModel(true)
	out := m.ViewContent(80, 24)
	if !strings.Contains(out, "Log in to Bitwarden") {
		t.Error("login view should contain 'Log in to Bitwarden'")
	}
}

func TestLockedCtrlQSendsQuitMsg(t *testing.T) {
	m := NewLockedModel(false)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("ctrl+q must return a command")
	}
	if _, ok := cmd().(QuitMsg); !ok {
		t.Errorf("ctrl+q returned %T, want QuitMsg", cmd())
	}
}

func TestLockedSuccessStaysUnlocking(t *testing.T) {
	m := NewLockedModel(false)
	m.state = lockedUnlocking
	updated, cmd := m.Update(bwcmd.UnlockResult{Token: "tok"})
	m = updated.(LockedModel)
	if m.state != lockedUnlocking {
		t.Fatal("a successful unlock must keep the unlocking state")
	}
	if msg, ok := cmd().(UnlockedMsg); !ok || msg.Token != "tok" {
		t.Fatalf("cmd returned %T, want UnlockedMsg with the token", cmd())
	}

	// A message that arrives before UnlockedMsg must not send the unlock
	// again with the cleared password.
	updated, _ = m.Update(spinner.TickMsg{})
	m = updated.(LockedModel)
	if m.state != lockedUnlocking {
		t.Error("a stray message must not leave the unlocking state")
	}
}

// submitLockedForm feeds the messages of cmd back into m until the form
// submits. huh completes a form over several internal messages.
func submitLockedForm(m LockedModel, cmd tea.Cmd) LockedModel {
	for i := 0; i < 10 && cmd != nil && m.state != lockedUnlocking; i++ {
		ch := make(chan tea.Msg, 1)
		go func(c tea.Cmd) { ch <- c() }(cmd)
		var msg tea.Msg
		select {
		case msg = <-ch:
		case <-time.After(100 * time.Millisecond):
			return m // a timer, such as the cursor blink
		}
		cmd = nil
		msgs := []tea.Msg{msg}
		if batch, ok := msg.(tea.BatchMsg); ok {
			msgs = nil
			for _, c := range batch {
				if c != nil {
					msgs = append(msgs, c())
				}
			}
		}
		for _, sub := range msgs {
			var updated tea.Model
			var next tea.Cmd
			updated, next = m.Update(sub)
			m = updated.(LockedModel)
			if next != nil && m.state != lockedUnlocking {
				cmd = next
			}
		}
	}
	return m
}

func TestLockedFormSubmitThenSuccessDoesNotResubmit(t *testing.T) {
	m := NewLockedModel(false)
	m.form.Init()
	for _, k := range "pw" {
		updated, _ := m.Update(tea.KeyPressMsg{Code: k, Text: string(k)})
		m = updated.(LockedModel)
	}
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = submitLockedForm(updated.(LockedModel), cmd)
	if m.state != lockedUnlocking {
		t.Fatalf("state = %d, want lockedUnlocking after submit", m.state)
	}

	updated, _ = m.Update(bwcmd.UnlockResult{Token: "tok"})
	m = updated.(LockedModel)
	updated, cmd = m.Update(spinner.TickMsg{})
	if updated.(LockedModel).state == lockedInput {
		t.Error("after a successful unlock, a stray message must not return to the input state")
	}

	// PATH is empty in these tests, so a new unlock fails at once with an
	// UnlockResult. The spinner tick returns later, or not at all.
	if cmd == nil {
		return
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		msgs := []tea.Msg{msg}
		if batch, ok := msg.(tea.BatchMsg); ok {
			msgs = nil
			for _, c := range batch {
				if c != nil {
					msgs = append(msgs, c())
				}
			}
		}
		for _, sub := range msgs {
			if _, ok := sub.(bwcmd.UnlockResult); ok {
				t.Error("after a successful unlock, a stray message must not start bw unlock again")
			}
		}
	case <-time.After(500 * time.Millisecond):
	}
}

func TestLockedErrorWrapsToFormWidth(t *testing.T) {
	msg := "Invalid master password. The request to the server failed because it timed out after 30 seconds"
	m := NewLockedModel(false)
	m.state = lockedUnlocking
	updated, _ := m.Update(bwcmd.UnlockResult{Err: errors.New(msg + "\n    at stack line\x1b[31m")})
	m = updated.(LockedModel)

	out := m.ViewContent(80, 24-ui.ChromeHeight)
	// The root centers the widest line. The error must not be wider than
	// the form, so that the form stays in the middle.
	formLeft := (80 - lockedFormWidth) / 2
	for _, line := range strings.Split(out, "\n") {
		plain := ansi.Strip(line)
		if w := lipgloss.Width(strings.TrimRight(plain, " ")); w > formLeft+lockedFormWidth {
			t.Errorf("line ends at column %d, past the form at %d: %q", w, formLeft+lockedFormWidth, plain)
		}
		if strings.Contains(plain, "Vault is locked.") && !strings.HasPrefix(plain, strings.Repeat(" ", formLeft)+"V") {
			t.Errorf("the title must start at column %d: %q", formLeft, plain)
		}
		if strings.ContainsAny(plain, "\x1b\r") {
			t.Errorf("line has control characters: %q", line)
		}
	}
	plain := strings.Join(strings.Fields(ansi.Strip(out)), " ")
	if !strings.Contains(plain, msg+" at stack line") {
		t.Errorf("the wrapped error lost text:\n%s", ansi.Strip(out))
	}

	// On a narrow screen, the error fits the width.
	for _, line := range strings.Split(m.ViewContent(40, 12-ui.ChromeHeight), "\n") {
		if w := lipgloss.Width(line); w > 40 {
			t.Errorf("at 40 columns, line is %d cells wide: %q", w, ansi.Strip(line))
		}
	}
}

func TestLockedErrorReturnsToInput(t *testing.T) {
	m := NewLockedModel(false)
	m.state = lockedUnlocking
	updated, _ := m.Update(bwcmd.UnlockResult{Err: errors.New("bad password")})
	m = updated.(LockedModel)
	if m.state != lockedInput {
		t.Error("a failed unlock must return to the input state")
	}
}

func TestLockedEnterOnEmptyFormStartsNoBw(t *testing.T) {
	for _, isLogin := range []bool{false, true} {
		unlocks := stubUnlock(t)
		logins := stubLogin(t)
		m := NewLockedModel(isLogin)
		m.form.Init()
		m, cmd := pressLocked(m, enterKey)
		m, _ = drive(t, m, cmd, nil)
		if m.state != lockedInput {
			t.Errorf("login=%v: state = %d, want lockedInput", isLogin, m.state)
		}
		if len(*unlocks) != 0 || len(*logins) != 0 {
			t.Errorf("login=%v: Enter on an empty form ran bw: unlock %d, login %d", isLogin, len(*unlocks), len(*logins))
		}
		if len(m.form.Errors()) == 0 {
			t.Errorf("login=%v: the form must show a validation error", isLogin)
		}
	}
}
