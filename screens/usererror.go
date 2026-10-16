package screens

import (
	"context"
	"errors"

	"github.com/juthrbog/lazybw/bwcmd"
)

// Texts for known errors.
const (
	unlockFailedText  = "Could not unlock the vault. Check your master password and try again."
	wrongPasswordText = "The master password is not correct. Try again."
	loginFailedText   = "Could not log in. Check your email and master password and try again."
	timeoutText       = "bw did not answer in time. Try again."
	notLoggedInText   = "You are not logged in. Log in again."
	newDeviceText     = `Bitwarden asked for a new-device code, which lazybw cannot pass to bw. Run "bw login" once in a terminal, then start lazybw again.`
)

// errAction is what the user tried to do when an error occurred.
type errAction int

const (
	actionOther    errAction = iota // load, sync, generate and all other actions
	actionUnlock                    // the unlock form
	actionLogin                     // the login form
	actionReprompt                  // the master password re-prompt
)

// userErrors maps known errors to text for each action. An empty text
// means that the action shows the error text itself.
var userErrors = []struct {
	target   error
	unlock   string
	login    string
	reprompt string
	other    string
}{
	{bwcmd.ErrInvalidPassword, unlockFailedText, loginFailedText, wrongPasswordText, ""},
	{context.DeadlineExceeded, timeoutText, timeoutText, timeoutText, timeoutText},
	{bwcmd.ErrNotLoggedIn, notLoggedInText, notLoggedInText, notLoggedInText, notLoggedInText},
}

// userError returns the text that the screens show for err. Known errors
// get a short, plain text from userErrors. For other errors it returns
// err.Error(), which for a bw error is the last line that bw printed. The
// bwcmd package writes the full bw output to the debug log.
func userError(err error, action errAction) string {
	if err == nil {
		return ""
	}
	for _, e := range userErrors {
		if !errors.Is(err, e.target) {
			continue
		}
		text := e.other
		switch action {
		case actionUnlock:
			text = e.unlock
		case actionLogin:
			text = e.login
		case actionReprompt:
			text = e.reprompt
		}
		if text != "" {
			return text
		}
	}
	return err.Error()
}
