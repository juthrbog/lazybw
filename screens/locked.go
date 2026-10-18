package screens

import (
	"errors"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/juthrbog/lazybw/bwcmd"
	"github.com/juthrbog/lazybw/ui"
)

// quitCmd asks the root model to quit, so that it can clear the clipboard
// first.
func quitCmd() tea.Msg { return QuitMsg{} }

// The bw commands that the locked screen runs. Tests replace them.
var (
	unlockCmd = bwcmd.Unlock
	loginCmd  = bwcmd.LoginUser
)

// UnlockedMsg is sent to the root model when the unlock succeeds.
type UnlockedMsg struct {
	Token string
	Email string
}

type lockedState int

const (
	lockedInput     lockedState = iota // the unlock or login form shows
	lockedUnlocking                    // bw unlock or bw login runs
	lockedTwoStep                      // the two-step login form shows
)

// twoStepStage is the question that the two-step form asks.
type twoStepStage int

const (
	stageMethod twoStepStage = iota // which two-step login method to use
	stageCode                       // the two-step login code
)

// LockedModel is the unlock/login prompt screen.
type LockedModel struct {
	form    *huh.Form
	isLogin bool
	state   lockedState
	spinner spinner.Model
	errText string
	// lockErr is the error of a bw lock that failed. It shows below the
	// form, above errText, until the root model drops the screen.
	lockErr string
	width   int
	height  int

	email    *string
	password *string

	// Two-step login. Each stage runs bw login again, so loginPassword
	// keeps the master password from the submit of the login form until
	// the login succeeds, fails, or the user presses esc.
	loginPassword string
	step          bwcmd.TwoStep // options for the running or the next bw login
	stage         twoStepStage
	twoStepForm   *huh.Form
	method        *bwcmd.TwoStepMethod
	code          *string
	// codeAsked is true when bw asked for a code before the user chose a
	// method. bw then sent the email already if the method is email.
	codeAsked bool
}

// NewLockedModel constructs the unlock model.
func NewLockedModel(isLogin bool) LockedModel {
	email := ""
	password := ""
	s := spinner.New()
	s.Spinner = ui.SpinnerUnlock
	s.Style = s.Style.Foreground(ui.ColorHighlight)
	m := LockedModel{
		isLogin:  isLogin,
		email:    &email,
		password: &password,
		spinner:  s,
	}
	m.form = buildLockedForm(isLogin, m.email, m.password)
	return m
}

// WithLockError returns m with text as the error of a failed bw lock. The
// form and what the user typed stay.
func (m LockedModel) WithLockError(text string) LockedModel {
	m.lockErr = text
	return m
}

// lockedFormWidth is the width of the unlock form, in cells.
const lockedFormWidth = 40

func buildLockedForm(isLogin bool, email, password *string) *huh.Form {
	pwField := huh.NewInput().
		Key("password").
		Title("Master password").
		EchoMode(huh.EchoModePassword).
		Validate(validatePassword).
		Value(password)

	if isLogin {
		emailField := huh.NewInput().
			Key("email").
			Title("Email").
			Validate(validateEmail).
			Value(email)
		return styleForm(huh.NewForm(huh.NewGroup(emailField, pwField)))
	}
	return styleForm(huh.NewForm(huh.NewGroup(pwField)))
}

// buildMethodForm returns the form that asks for the two-step login method.
func buildMethodForm(method *bwcmd.TwoStepMethod) *huh.Form {
	field := huh.NewSelect[bwcmd.TwoStepMethod]().
		Title("Two-step login method").
		Description("Select how you get your login code.").
		Options(
			huh.NewOption("Authenticator app", bwcmd.MethodAuthenticator),
			huh.NewOption("Email", bwcmd.MethodEmail),
			huh.NewOption("YubiKey OTP", bwcmd.MethodYubiKey),
		).
		Value(method)
	// Three options need no filter. Without the filter, esc always goes
	// back to the login form.
	keys := huh.NewDefaultKeyMap()
	keys.Select.Filter.SetEnabled(false)
	return styleForm(huh.NewForm(huh.NewGroup(field)).WithKeyMap(keys))
}

// buildCodeForm returns the form that asks for the two-step login code.
func buildCodeForm(method bwcmd.TwoStepMethod, code *string) *huh.Form {
	field := huh.NewInput().
		Title("Two-step login code").
		Description(codePrompt(method)).
		Validate(validateCode).
		Value(code)
	return styleForm(huh.NewForm(huh.NewGroup(field)))
}

// styleForm gives a form of the locked screen its width and theme. The form
// shows no help, because the footer shows the keys.
func styleForm(form *huh.Form) *huh.Form {
	form = form.WithWidth(lockedFormWidth).WithShowHelp(false)
	if ui.HuhTheme != nil {
		form = form.WithTheme(ui.HuhTheme)
	}
	return form
}

// codePrompt tells the user where to find the code for method.
func codePrompt(method bwcmd.TwoStepMethod) string {
	switch method {
	case bwcmd.MethodAuthenticator:
		return "Enter the code from your authenticator app."
	case bwcmd.MethodEmail:
		return "Enter the code from the email that Bitwarden sent you."
	case bwcmd.MethodYubiKey:
		return "Touch your YubiKey to enter the code."
	}
	return "Enter the code from your authenticator app, or from the email that Bitwarden sent you."
}

// validatePassword rejects an empty master password, so that the form
// cannot start bw without one.
func validatePassword(s string) error {
	if s == "" {
		return errors.New("enter your master password")
	}
	return nil
}

// validateEmail accepts an email that is not empty, contains "@" and does
// not start with "-".
func validateEmail(s string) error {
	s = strings.TrimSpace(s)
	switch {
	case s == "":
		return errors.New("enter your email")
	case !strings.Contains(s, "@"), strings.HasPrefix(s, "-"):
		return errors.New("enter a valid email")
	}
	return nil
}

// validateCode rejects an empty two-step login code.
func validateCode(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("enter the code")
	}
	return nil
}

func (m LockedModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m LockedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case ThemeChangedMsg:
		// The forms take their colors from ui.HuhTheme when they draw, but
		// the spinner keeps a copy of its color.
		m.spinner.Style = m.spinner.Style.Foreground(ui.ColorHighlight)
		return m, nil

	case bwcmd.UnlockResult:
		return m.handleResult(msg)
	}

	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "ctrl+q" {
		return m, quitCmd
	}

	switch m.state {
	case lockedUnlocking:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case lockedTwoStep:
		return m.updateTwoStep(msg)
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	switch m.form.State {
	case huh.StateCompleted:
		return m.submit()
	case huh.StateAborted:
		return m, quitCmd
	}

	return m, cmd
}

// submit starts bw unlock or bw login with the values of the form. The
// input field of the completed form still holds the master password, and
// each update of the field copies it back to m.password. So submit puts an
// empty form in its place. The new form does not show or get updates
// while bw runs.
func (m LockedModel) submit() (tea.Model, tea.Cmd) {
	m.errText = ""
	pw := *m.password
	*m.password = ""
	*m.email = strings.TrimSpace(*m.email)
	m.password = new(string)
	m.form = buildLockedForm(m.isLogin, m.email, m.password)
	if !m.isLogin {
		m.state = lockedUnlocking
		return m, tea.Batch(m.spinner.Tick, unlockCmd(pw))
	}
	m.loginPassword = pw
	m.step = bwcmd.TwoStep{}
	return m.runLogin()
}

// runLogin starts bw login with the email, the kept master password and
// the two-step options in m.step.
func (m LockedModel) runLogin() (tea.Model, tea.Cmd) {
	m.state = lockedUnlocking
	return m, tea.Batch(m.spinner.Tick, loginCmd(*m.email, m.loginPassword, m.step))
}

// handleResult handles the result of bw unlock or bw login.
func (m LockedModel) handleResult(msg bwcmd.UnlockResult) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		m.endTwoStep()
		// Stay in lockedUnlocking until the root model switches screens,
		// so that the empty form from submit does not show.
		email := *m.email
		return m, func() tea.Msg {
			return UnlockedMsg{Token: msg.Token, Email: email}
		}
	}
	if !m.isLogin {
		return m.backToForm(userError(msg.Err, actionUnlock))
	}
	switch {
	case errors.Is(msg.Err, bwcmd.ErrMethodRequired) && m.step.Method == bwcmd.MethodAuto:
		return m.showTwoStep(stageMethod)
	case errors.Is(msg.Err, bwcmd.ErrCodeRequired) && m.step.Method == bwcmd.MethodAuto:
		// bw chose the method itself: the account has one two-step
		// method, or this is the new-device check. If the method is email,
		// bw sent the email. Without --method, bw sends a code only in its
		// second request to the server. For an account with only the
		// email method, the first request makes the server send a new
		// code, so the code that the user typed can be old. So ask for
		// the method first.
		m.codeAsked = true
		return m.showTwoStep(stageMethod)
	case errors.Is(msg.Err, bwcmd.ErrCodeRequired) && m.step.Code == "":
		// bw needs a code. A run with the email method and no code made
		// Bitwarden send the email.
		return m.showTwoStep(stageCode)
	case errors.Is(msg.Err, bwcmd.ErrCodeRequired):
		// bw got a code and asked for one again. That is the new-device
		// check, and bw has no option for its code.
		return m.backToForm(newDeviceText)
	}
	return m.backToForm(userError(msg.Err, actionLogin))
}

// showTwoStep shows the two-step form at stage.
func (m LockedModel) showTwoStep(stage twoStepStage) (tea.Model, tea.Cmd) {
	m.state = lockedTwoStep
	m.stage = stage
	m.errText = ""
	if stage == stageMethod {
		method := bwcmd.MethodAuthenticator
		m.method = &method
		m.twoStepForm = buildMethodForm(m.method)
	} else {
		code := ""
		m.code = &code
		m.twoStepForm = buildCodeForm(m.step.Method, m.code)
	}
	return m, m.twoStepForm.Init()
}

// updateTwoStep handles a message while the two-step form shows. Esc goes
// back to the login form.
func (m LockedModel) updateTwoStep(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "esc" {
		return m.backToForm("")
	}

	form, cmd := m.twoStepForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.twoStepForm = f
	}

	switch m.twoStepForm.State {
	case huh.StateCompleted:
		return m.submitTwoStep()
	case huh.StateAborted:
		return m, quitCmd
	}
	return m, cmd
}

// submitTwoStep goes to the next step after the two-step form completes.
func (m LockedModel) submitTwoStep() (tea.Model, tea.Cmd) {
	if m.stage == stageMethod {
		m.step.Method = *m.method
		if m.step.Method == bwcmd.MethodEmail && !m.codeAsked {
			// Without a code, bw makes Bitwarden send the code by email.
			return m.runLogin()
		}
		return m.showTwoStep(stageCode)
	}
	// A second Enter in one input read can complete the new code form
	// before the user types. Ask again instead of sending an empty code.
	if validateCode(*m.code) != nil {
		return m.showTwoStep(stageCode)
	}
	m.step.Code = strings.TrimSpace(*m.code)
	*m.code = ""
	return m.runLogin()
}

// backToForm ends a two-step login, drops the kept master password and
// shows the unlock or login form again, with text as the error.
func (m LockedModel) backToForm(text string) (tea.Model, tea.Cmd) {
	m.endTwoStep()
	m.state = lockedInput
	m.errText = text
	pw := ""
	m.password = &pw
	m.form = buildLockedForm(m.isLogin, m.email, m.password)
	return m, m.form.Init()
}

// endTwoStep drops the kept master password and all two-step state.
func (m *LockedModel) endTwoStep() {
	m.loginPassword = ""
	m.step = bwcmd.TwoStep{}
	m.codeAsked = false
	m.twoStepForm = nil
	m.method = nil
	if m.code != nil {
		*m.code = ""
		m.code = nil
	}
}

// ViewContent returns the screen content for the root frame to center.
func (m LockedModel) ViewContent(width, contentHeight int) string {
	if m.state == lockedUnlocking {
		busy := "Unlocking…"
		if m.isLogin {
			busy = "Logging in…"
		}
		return ui.CenterInArea(m.spinner.View()+" "+busy, width, contentHeight)
	}

	title := "Vault is locked."
	form := m.form
	switch {
	case m.state == lockedTwoStep:
		title = "Two-step login"
		form = m.twoStepForm
	case m.isLogin:
		title = "Log in to Bitwarden"
	}

	// The lock error comes before the unlock error. CenterInArea cuts the
	// lines at the bottom when the content is too tall, and the lock error
	// is a security warning. Each error wraps to the form width. The root
	// centers the widest line, so a wider error moves the form off center.
	parts := []string{title, form.View()}
	for _, text := range []string{m.lockErr, m.errText} {
		if text != "" {
			parts = append(parts, errorBlock(text, width, lockedFormWidth))
		}
	}
	content := strings.Join(parts, "\n\n")
	if strings.Count(content, "\n") >= contentHeight {
		// On a short terminal, leave out the blank lines between the parts.
		content = strings.Join(parts, "\n")
	}

	return ui.CenterInArea(content, width, contentHeight)
}

// FooterContent returns hints and status for the footer bar.
func (m LockedModel) FooterContent() ([]ui.HintBinding, string) {
	switch {
	case m.state == lockedUnlocking:
		return nil, ""
	case m.state == lockedTwoStep && m.stage == stageMethod:
		return []ui.HintBinding{
			{Key: "enter", Desc: "submit"},
			{Key: "↑/↓", Desc: "navigate"},
			{Key: "esc", Desc: "back"},
			{Key: "ctrl+q", Desc: "quit"},
		}, ""
	case m.state == lockedTwoStep:
		return []ui.HintBinding{
			{Key: "enter", Desc: "submit"},
			{Key: "esc", Desc: "back"},
			{Key: "ctrl+q", Desc: "quit"},
		}, ""
	case m.isLogin && m.form != nil && m.form.GetFocusedField().GetKey() == "password":
		// The form shows no help, so the footer shows the key that goes back
		// to the email field.
		return []ui.HintBinding{
			{Key: "enter", Desc: "submit"},
			{Key: "shift+tab", Desc: "back"},
			{Key: "ctrl+q", Desc: "quit"},
		}, ""
	case m.isLogin:
		return []ui.HintBinding{
			{Key: "enter", Desc: "submit"},
			{Key: "ctrl+q", Desc: "quit"},
		}, ""
	}
	return []ui.HintBinding{
		{Key: "enter", Desc: "unlock"},
		{Key: "ctrl+q", Desc: "quit"},
	}, ""
}

// View implements tea.Model (kept for interface compliance).
func (m LockedModel) View() tea.View {
	return tea.NewView(m.ViewContent(m.width, m.height))
}
