package screens

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
)

var (
	enterKey = tea.KeyPressMsg{Code: tea.KeyEnter}
	escKey   = tea.KeyPressMsg{Code: tea.KeyEscape}
	downKey  = tea.KeyPressMsg{Code: tea.KeyDown}

	codeRequired   = &bwcmd.Error{Msg: "Code is required."}
	methodRequired = &bwcmd.Error{Msg: "Login failed. No provider selected."}
)

// loginCall records one call of loginCmd.
type loginCall struct {
	email, password string
	step            bwcmd.TwoStep
}

// stubLogin replaces loginCmd for one test. Call n returns results[n].
func stubLogin(t *testing.T, results ...bwcmd.UnlockResult) *[]loginCall {
	t.Helper()
	calls := &[]loginCall{}
	orig := loginCmd
	t.Cleanup(func() { loginCmd = orig })
	loginCmd = func(email, password string, step bwcmd.TwoStep) tea.Cmd {
		*calls = append(*calls, loginCall{email, password, step})
		if len(*calls) > len(results) {
			t.Errorf("unexpected bw login number %d", len(*calls))
			return nil
		}
		res := results[len(*calls)-1]
		return func() tea.Msg { return res }
	}
	return calls
}

// stubUnlock replaces unlockCmd for one test. Every call succeeds.
func stubUnlock(t *testing.T) *[]string {
	t.Helper()
	calls := &[]string{}
	orig := unlockCmd
	t.Cleanup(func() { unlockCmd = orig })
	unlockCmd = func(password string) tea.Cmd {
		*calls = append(*calls, password)
		return func() tea.Msg { return bwcmd.UnlockResult{Token: "tok"} }
	}
	return calls
}

// drive feeds the messages of cmd back into m. Each command runs in its
// own goroutine, so that a timer, such as the cursor blink, does not hold
// up the others. drive stops when done returns true, when no command is
// left, or when no message arrives for 300ms. It returns m and the
// messages that it saw. Spinner ticks are not fed back, so that the
// spinner does not keep the loop busy.
func drive(t *testing.T, m LockedModel, cmd tea.Cmd, done func(LockedModel, []tea.Msg) bool) (LockedModel, []tea.Msg) {
	t.Helper()
	msgs := make(chan tea.Msg)
	stop := make(chan struct{})
	defer close(stop)
	running := 0
	start := func(c tea.Cmd) {
		if c == nil {
			return
		}
		running++
		go func() {
			select {
			case msgs <- c():
			case <-stop:
			}
		}()
	}
	start(cmd)

	var seen []tea.Msg
	deadline := time.After(5 * time.Second)
	for done == nil || !done(m, seen) {
		if running == 0 {
			return m, seen
		}
		select {
		case msg := <-msgs:
			running--
			if batch, ok := msg.(tea.BatchMsg); ok {
				for _, c := range batch {
					start(c)
				}
				continue
			}
			if msg == nil {
				continue
			}
			seen = append(seen, msg)
			if _, ok := msg.(spinner.TickMsg); ok {
				continue
			}
			updated, next := m.Update(msg)
			m = updated.(LockedModel)
			start(next)
		case <-time.After(300 * time.Millisecond):
			return m, seen
		case <-deadline:
			t.Fatal("drive did not end after 5s")
		}
	}
	return m, seen
}

func focusIs(key string) func(LockedModel, []tea.Msg) bool {
	return func(m LockedModel, _ []tea.Msg) bool {
		return m.state == lockedInput && m.form.GetFocusedField().GetKey() == key
	}
}

func stateIs(s lockedState) func(LockedModel, []tea.Msg) bool {
	return func(m LockedModel, _ []tea.Msg) bool { return m.state == s }
}

func stageIs(s twoStepStage) func(LockedModel, []tea.Msg) bool {
	return func(m LockedModel, _ []tea.Msg) bool { return m.state == lockedTwoStep && m.stage == s }
}

func unlocked(_ LockedModel, seen []tea.Msg) bool {
	for _, msg := range seen {
		if _, ok := msg.(UnlockedMsg); ok {
			return true
		}
	}
	return false
}

// typeText types s into the focused field of m.
func typeText(m LockedModel, s string) LockedModel {
	for _, r := range s {
		updated, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = updated.(LockedModel)
	}
	return m
}

func pressLocked(m LockedModel, k tea.KeyPressMsg) (LockedModel, tea.Cmd) {
	updated, cmd := m.Update(k)
	return updated.(LockedModel), cmd
}

// loginUntilTwoStep fills the login form and submits it. It returns m when
// the two-step form shows.
func loginUntilTwoStep(t *testing.T, m LockedModel) LockedModel {
	t.Helper()
	m.form.Init()
	m = typeText(m, "u@t.com")
	m, cmd := pressLocked(m, enterKey)
	m, _ = drive(t, m, cmd, focusIs("password"))
	m = typeText(m, "pw")
	m, cmd = pressLocked(m, enterKey)
	m, _ = drive(t, m, cmd, stateIs(lockedTwoStep))
	if m.state != lockedTwoStep {
		t.Fatalf("state = %d, want lockedTwoStep", m.state)
	}
	if *m.password != "" {
		t.Error("the login form must drop its reference to the password")
	}
	return m
}

// chooseMethod picks option i of the method form, submits it and drives m
// until done returns true.
func chooseMethod(t *testing.T, m LockedModel, i int, done func(LockedModel, []tea.Msg) bool) LockedModel {
	t.Helper()
	if m.state != lockedTwoStep || m.stage != stageMethod {
		t.Fatalf("state = %d, stage = %d, want the method form", m.state, m.stage)
	}
	for range i {
		m, _ = pressLocked(m, downKey)
	}
	m, cmd := pressLocked(m, enterKey)
	m, _ = drive(t, m, cmd, done)
	return m
}

// submitCode types code into the code form and submits it.
func submitCode(t *testing.T, m LockedModel, code string, done func(LockedModel, []tea.Msg) bool) (LockedModel, []tea.Msg) {
	t.Helper()
	if m.state != lockedTwoStep || m.stage != stageCode {
		t.Fatalf("state = %d, stage = %d, want the code form", m.state, m.stage)
	}
	m = typeText(m, code)
	m, cmd := pressLocked(m, enterKey)
	return drive(t, m, cmd, done)
}

func TestLockedUnlockSubmitsPassword(t *testing.T) {
	unlocks := stubUnlock(t)
	m := NewLockedModel(false)
	m.form.Init()
	m = typeText(m, "pw")
	m, cmd := pressLocked(m, enterKey)
	m, _ = drive(t, m, cmd, stateIs(lockedUnlocking))
	if m.state != lockedUnlocking || len(*unlocks) != 1 || (*unlocks)[0] != "pw" {
		t.Errorf("state = %d, unlock calls = %q, want one unlock with the password", m.state, *unlocks)
	}
}

func TestLoginFormRejectsBadEmail(t *testing.T) {
	for _, email := range []string{"user", "--help", "-u@t.com"} {
		logins := stubLogin(t)
		m := NewLockedModel(true)
		m.form.Init()
		m = typeText(m, email)
		m, cmd := pressLocked(m, enterKey)
		m, _ = drive(t, m, cmd, nil)
		if got := m.form.GetFocusedField().GetKey(); got != "email" {
			t.Errorf("%q: focus moved to %q, want it to stay on email", email, got)
		}
		if len(*logins) != 0 {
			t.Errorf("%q: the form ran bw login", email)
		}
	}
}

func TestLockedFormValidators(t *testing.T) {
	emails := map[string]bool{
		"": false, "   ": false, "user": false, "--help": false, "-u@t.com": false,
		"u@t.com": true, " u@t.com ": true,
	}
	for in, ok := range emails {
		if err := validateEmail(in); (err == nil) != ok {
			t.Errorf("validateEmail(%q) = %v, want ok=%v", in, err, ok)
		}
	}
	if validatePassword("") == nil || validatePassword("pw") != nil {
		t.Error("validatePassword must reject only an empty password")
	}
	if validateCode("") == nil || validateCode("  ") == nil || validateCode("123456") != nil {
		t.Error("validateCode must reject an empty code")
	}
}

func TestTwoStepSingleProvider(t *testing.T) {
	// bw sends a code without --method only in its second request to the
	// server, so the form asks for the method before the code.
	calls := stubLogin(t,
		bwcmd.UnlockResult{Err: codeRequired},
		bwcmd.UnlockResult{Token: "tok"},
	)
	m := loginUntilTwoStep(t, NewLockedModel(true))
	if m.stage != stageMethod {
		t.Fatalf("stage = %d, want the method form", m.stage)
	}
	if !strings.Contains(m.ViewContent(80, 24), "Two-step login") {
		t.Error("the view must show the two-step form")
	}

	m = chooseMethod(t, m, 0, stageIs(stageCode)) // authenticator app
	if len(*calls) != 1 {
		t.Fatalf("bw login ran %d times, want once before the code", len(*calls))
	}
	m, seen := submitCode(t, m, "123456", unlocked)
	want := []loginCall{
		{"u@t.com", "pw", bwcmd.TwoStep{}},
		{"u@t.com", "pw", bwcmd.TwoStep{Method: bwcmd.MethodAuthenticator, Code: "123456"}},
	}
	if !equalCalls(*calls, want) {
		t.Errorf("login calls = %+v, want %+v", *calls, want)
	}
	if !hasUnlocked(seen, "tok", "u@t.com") {
		t.Errorf("messages = %v, want UnlockedMsg with the token and email", seen)
	}
	if m.loginPassword != "" {
		t.Error("a successful login must clear the kept password")
	}
}

func TestTwoStepMethodSelect(t *testing.T) {
	calls := stubLogin(t,
		bwcmd.UnlockResult{Err: methodRequired},
		bwcmd.UnlockResult{Token: "tok"},
	)
	m := loginUntilTwoStep(t, NewLockedModel(true))
	if m.stage != stageMethod {
		t.Fatalf("stage = %d, want the method form", m.stage)
	}
	view := m.ViewContent(80, 24)
	for _, opt := range []string{"Authenticator app", "Email", "YubiKey OTP"} {
		if !strings.Contains(view, opt) {
			t.Errorf("the method form must offer %q", opt)
		}
	}
	m, _ = pressLocked(m, tea.KeyPressMsg{Code: '/', Text: "/"})
	if strings.Contains(view, "filter") || m.ViewContent(80, 24) != view {
		t.Error("the method form must have no filter")
	}

	m, cmd := pressLocked(m, enterKey) // the first option: authenticator app
	m, _ = drive(t, m, cmd, stageIs(stageCode))
	if len(*calls) != 1 {
		t.Fatalf("bw login ran %d times, want once before the code", len(*calls))
	}
	_, seen := submitCode(t, m, "654321", unlocked)
	want := bwcmd.TwoStep{Method: bwcmd.MethodAuthenticator, Code: "654321"}
	if len(*calls) != 2 || (*calls)[1].step != want {
		t.Errorf("login calls = %+v, want a second call with %+v", *calls, want)
	}
	if !hasUnlocked(seen, "tok", "u@t.com") {
		t.Error("the login must finish")
	}
}

func TestTwoStepEmailSendsCodeFirst(t *testing.T) {
	calls := stubLogin(t,
		bwcmd.UnlockResult{Err: methodRequired},
		bwcmd.UnlockResult{Err: codeRequired}, // bw sent the email
		bwcmd.UnlockResult{Token: "tok"},
	)
	m := loginUntilTwoStep(t, NewLockedModel(true))
	m, _ = pressLocked(m, downKey)
	m, cmd := pressLocked(m, enterKey)
	m, _ = drive(t, m, cmd, stageIs(stageCode))
	if len(*calls) != 2 || (*calls)[1].step != (bwcmd.TwoStep{Method: bwcmd.MethodEmail}) {
		t.Fatalf("login calls = %+v, want a call with the email method and no code", *calls)
	}
	if !strings.Contains(m.ViewContent(80, 24), "email") {
		t.Error("the code form must point to the email")
	}

	_, seen := submitCode(t, m, "111222", unlocked)
	want := bwcmd.TwoStep{Method: bwcmd.MethodEmail, Code: "111222"}
	if len(*calls) != 3 || (*calls)[2].step != want {
		t.Errorf("login calls = %+v, want a third call with %+v", *calls, want)
	}
	if !hasUnlocked(seen, "tok", "u@t.com") {
		t.Error("the login must finish")
	}
}

func TestTwoStepSingleEmailProvider(t *testing.T) {
	// bw sent the email in the first run, so the choice of email must not
	// run bw again without a code, which sends one more email.
	calls := stubLogin(t,
		bwcmd.UnlockResult{Err: codeRequired},
		bwcmd.UnlockResult{Token: "tok"},
	)
	m := loginUntilTwoStep(t, NewLockedModel(true))
	m = chooseMethod(t, m, 1, stageIs(stageCode)) // email
	if len(*calls) != 1 {
		t.Fatalf("login calls = %+v, want no run between the method and the code", *calls)
	}
	if !strings.Contains(m.ViewContent(80, 24), "email") {
		t.Error("the code form must point to the email")
	}

	_, seen := submitCode(t, m, "111222", unlocked)
	want := bwcmd.TwoStep{Method: bwcmd.MethodEmail, Code: "111222"}
	if len(*calls) != 2 || (*calls)[1].step != want {
		t.Errorf("login calls = %+v, want a second call with %+v", *calls, want)
	}
	if !hasUnlocked(seen, "tok", "u@t.com") {
		t.Error("the login must finish")
	}
}

func TestTwoStepNewDeviceHint(t *testing.T) {
	stubLogin(t,
		bwcmd.UnlockResult{Err: codeRequired},
		bwcmd.UnlockResult{Err: codeRequired},
	)
	m := loginUntilTwoStep(t, NewLockedModel(true))
	m = chooseMethod(t, m, 0, stageIs(stageCode))
	m, _ = submitCode(t, m, "123456", stateIs(lockedInput))
	if m.state != lockedInput {
		t.Fatalf("state = %d, want lockedInput", m.state)
	}
	if m.errText != newDeviceText {
		t.Errorf("errText = %q, want the new-device hint", m.errText)
	}
	if !strings.Contains(m.ViewContent(80, 24), "Bitwarden asked") {
		t.Error("the view must show the new-device hint")
	}
	if m.loginPassword != "" {
		t.Error("the hint must clear the kept password")
	}
}

func TestTwoStepEscGoesBackToLogin(t *testing.T) {
	calls := stubLogin(t, bwcmd.UnlockResult{Err: codeRequired})
	m := loginUntilTwoStep(t, NewLockedModel(true))
	m = chooseMethod(t, m, 0, stageIs(stageCode))
	m = typeText(m, "12")
	m, _ = pressLocked(m, escKey)
	if m.state != lockedInput {
		t.Fatalf("state = %d, want lockedInput", m.state)
	}
	if m.loginPassword != "" || m.code != nil || m.step != (bwcmd.TwoStep{}) || m.codeAsked {
		t.Error("esc must clear the kept password and the two-step state")
	}
	if m.errText != "" {
		t.Errorf("errText = %q, want none", m.errText)
	}
	if *m.email != "u@t.com" || !strings.Contains(m.ViewContent(80, 24), "Log in to Bitwarden") {
		t.Error("esc must show the login form with the email")
	}
	if len(*calls) != 1 {
		t.Errorf("bw login ran %d times, want once", len(*calls))
	}
}

func TestTwoStepFailureClearsPassword(t *testing.T) {
	stubLogin(t,
		bwcmd.UnlockResult{Err: codeRequired},
		bwcmd.UnlockResult{Err: &bwcmd.Error{Msg: "Login failed."}},
	)
	m := loginUntilTwoStep(t, NewLockedModel(true))
	m = chooseMethod(t, m, 0, stageIs(stageCode))
	m, _ = submitCode(t, m, "000000", stateIs(lockedInput))
	if m.state != lockedInput || m.errText != "Login failed." {
		t.Errorf("state = %d, errText = %q, want the login form with the bw message", m.state, m.errText)
	}
	if m.loginPassword != "" {
		t.Error("a failed login must clear the kept password")
	}
}

func TestLockedWrongPasswordShowsFriendlyText(t *testing.T) {
	tests := []struct {
		isLogin bool
		err     *bwcmd.Error
		want    string
		hidden  []string
	}{
		{false, &bwcmd.Error{Msg: "The provided key is not the expected type", Stderr: "ERROR bitwarden_crypto: error=The decryption operation failed"},
			unlockFailedText, []string{"provided key", "bitwarden_crypto"}},
		{true, &bwcmd.Error{Msg: "Invalid master password. Confirm your email is correct and your account was created on vault.example."},
			loginFailedText, []string{"Invalid master password", "vault.example"}},
	}
	for _, tt := range tests {
		m := NewLockedModel(tt.isLogin)
		m.state = lockedUnlocking
		m.loginPassword = "pw"
		updated, _ := m.Update(bwcmd.UnlockResult{Err: tt.err})
		m = updated.(LockedModel)
		if m.errText != tt.want {
			t.Errorf("login=%v: errText = %q, want %q", tt.isLogin, m.errText, tt.want)
		}
		view := m.ViewContent(200, 24)
		for _, s := range tt.hidden {
			if strings.Contains(view, s) {
				t.Errorf("login=%v: the view shows the bw text %q", tt.isLogin, s)
			}
		}
		if m.loginPassword != "" {
			t.Errorf("login=%v: a failure must clear the kept password", tt.isLogin)
		}
	}
}

// noopMsg is a message that no field handles.
type noopMsg struct{}

func TestSubmitDropsPasswordFromForm(t *testing.T) {
	stubUnlock(t)
	stubLogin(t, bwcmd.UnlockResult{Err: codeRequired}, bwcmd.UnlockResult{Token: "tok"})
	unlock := NewLockedModel(false)
	unlock.form.Init()
	unlock = typeText(unlock, "unlock-secret")
	unlock, cmd := pressLocked(unlock, enterKey)
	unlock, _ = drive(t, unlock, cmd, unlocked)

	login := loginUntilTwoStep(t, NewLockedModel(true))
	login = chooseMethod(t, login, 0, stageIs(stageCode))
	login, _ = submitCode(t, login, "123456", unlocked)

	for name, m := range map[string]LockedModel{"unlock": unlock, "login": login} {
		// The root model keeps the locked model after the unlock. An update
		// of the form field copies the field text to m.password.
		m.form.GetFocusedField().Update(noopMsg{})
		if *m.password != "" || m.form.GetString("password") != "" {
			t.Errorf("%s: the form holds the password after the unlock", name)
		}
		if m.loginPassword != "" {
			t.Errorf("%s: loginPassword = %q, want empty", name, m.loginPassword)
		}
	}
}

func equalCalls(got, want []loginCall) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func hasUnlocked(seen []tea.Msg, token, email string) bool {
	for _, msg := range seen {
		if u, ok := msg.(UnlockedMsg); ok && u.Token == token && u.Email == email {
			return true
		}
	}
	return false
}

func TestTwoStepEmptyCodeIsNotSent(t *testing.T) {
	calls := stubLogin(t)
	m := NewLockedModel(true)
	updated, _ := m.showTwoStep(stageCode)
	m = updated.(LockedModel)
	// The form completed with an empty code, for example after a second
	// Enter in one input read.
	updated, _ = m.submitTwoStep()
	m = updated.(LockedModel)
	if len(*calls) != 0 {
		t.Errorf("bw login ran %d times with an empty code", len(*calls))
	}
	if m.state != lockedTwoStep || m.stage != stageCode {
		t.Errorf("state = %d stage = %d, want the code form again", m.state, m.stage)
	}
}
