package ui

import "charm.land/bubbles/v2/key"

// Help categories, in the order of the help overlay.
const (
	CategoryCopy       = "Copy"
	CategoryNavigation = "Navigation"
	CategoryVault      = "Vault"
	CategoryUI         = "UI"
)

// VaultKeyMap defines the keys of the vault screen. The help of each
// binding gives the key label and the description that the help overlay
// shows. The footer and the drawer take the key label from the help too.
type VaultKeyMap struct {
	Copy         key.Binding
	CopyTOTP     key.Binding
	CopyUsername key.Binding
	CopyNote     key.Binding
	CopyCVV      key.Binding
	OpenURL      key.Binding
	// The bubbles list handles the keys of Navigate, Jump and Filter.
	// They are in this map for the help overlay and the footer.
	Navigate       key.Binding
	Jump           key.Binding
	Filter         key.Binding
	ScrollDown     key.Binding
	ScrollUp       key.Binding
	DrawerGrow     key.Binding
	DrawerShrink   key.Binding
	DrawerReset    key.Binding
	DrawerToggle   key.Binding
	New            key.Binding
	Generate       key.Binding
	Sync           key.Binding
	Lock           key.Binding
	ToggleGrouping key.Binding
	ToggleExpand   key.Binding
	Help           key.Binding
	ThemePicker    key.Binding
	Quit           key.Binding
}

// categorizedBinding is a binding and its help category.
type categorizedBinding struct {
	binding  key.Binding
	category string
}

// categorized returns each binding of k next to its help category, in the
// order of the help overlay.
func (k VaultKeyMap) categorized() []categorizedBinding {
	return []categorizedBinding{
		{k.Copy, CategoryCopy},
		{k.CopyTOTP, CategoryCopy},
		{k.CopyUsername, CategoryCopy},
		{k.CopyNote, CategoryCopy},
		{k.CopyCVV, CategoryCopy},
		{k.OpenURL, CategoryCopy},
		{k.Navigate, CategoryNavigation},
		{k.Jump, CategoryNavigation},
		{k.Filter, CategoryNavigation},
		{k.ScrollDown, CategoryNavigation},
		{k.ScrollUp, CategoryNavigation},
		{k.DrawerGrow, CategoryNavigation},
		{k.DrawerShrink, CategoryNavigation},
		{k.DrawerReset, CategoryNavigation},
		{k.DrawerToggle, CategoryNavigation},
		{k.New, CategoryVault},
		{k.Generate, CategoryVault},
		{k.Sync, CategoryVault},
		{k.Lock, CategoryVault},
		{k.ToggleGrouping, CategoryVault},
		{k.ToggleExpand, CategoryVault},
		{k.Help, CategoryUI},
		{k.ThemePicker, CategoryUI},
		{k.Quit, CategoryUI},
	}
}

// HelpBindings returns the entries of the help overlay: the key label and
// the description of the help of each binding, with its category.
func (k VaultKeyMap) HelpBindings() []HelpBinding {
	cb := k.categorized()
	out := make([]HelpBinding, len(cb))
	for i, c := range cb {
		h := c.binding.Help()
		out[i] = HelpBinding{Key: h.Key, Desc: h.Desc, Category: c.category}
	}
	return out
}

// DefaultVaultKeyMap returns the standard bindings for the vault screen.
func DefaultVaultKeyMap() VaultKeyMap {
	return VaultKeyMap{
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy password"),
		),
		CopyTOTP: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "copy TOTP"),
		),
		CopyUsername: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "copy username"),
		),
		CopyNote: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "copy note"),
		),
		CopyCVV: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "copy card security code"),
		),
		OpenURL: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open URL"),
		),
		Navigate: key.NewBinding(
			key.WithKeys("j", "k", "down", "up"),
			key.WithHelp("j/k", "move up/down"),
		),
		Jump: key.NewBinding(
			key.WithKeys("g", "G", "home", "end"),
			key.WithHelp("g/G", "jump to top/bottom"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("J", "shift+down"),
			key.WithHelp("J", "scroll drawer down"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("K", "shift+up"),
			key.WithHelp("K", "scroll drawer up"),
		),
		DrawerGrow: key.NewBinding(
			key.WithKeys("+"),
			key.WithHelp("+", "grow drawer"),
		),
		DrawerShrink: key.NewBinding(
			key.WithKeys("-"),
			key.WithHelp("-", "shrink drawer"),
		),
		DrawerReset: key.NewBinding(
			key.WithKeys("="),
			key.WithHelp("=", "reset drawer height"),
		),
		DrawerToggle: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "toggle drawer overlay"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new item"),
		),
		Generate: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "generate password"),
		),
		Sync: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "sync"),
		),
		Lock: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "lock"),
		),
		ToggleGrouping: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "toggle grouping"),
		),
		ToggleExpand: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "expand/collapse"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		ThemePicker: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "theme picker"),
		),
		// ctrl+c is not here. RootModel quits on it before any screen gets
		// the key.
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ConfirmKeyMap defines the keys of a yes or no dialog, such as the quit
// confirmation.
type ConfirmKeyMap struct {
	Yes key.Binding
	No  key.Binding
}

// DefaultConfirmKeyMap returns the standard bindings for a yes or no
// dialog.
func DefaultConfirmKeyMap() ConfirmKeyMap {
	return ConfirmKeyMap{
		Yes: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yes"),
		),
		No: key.NewBinding(
			key.WithKeys("n", "esc"),
			key.WithHelp("n", "no"),
		),
	}
}

// GeneratorKeyMap defines the keys of the password generator card.
type GeneratorKeyMap struct {
	Regenerate key.Binding
	Mode       key.Binding
	Longer     key.Binding
	Shorter    key.Binding
	// Option1 to Option4 toggle the options of the mode. In passphrase
	// mode, Option3 moves to the next separator, and Option4 does nothing.
	Option1 key.Binding
	Option2 key.Binding
	Option3 key.Binding
	Option4 key.Binding
	Copy    key.Binding
	// Use puts the value in the password field of the item form. The
	// generator turns it on only when the form opened it, and then takes
	// enter from Regenerate.
	Use   key.Binding
	Close key.Binding
}

// DefaultGeneratorKeyMap returns the standard bindings for the password
// generator.
func DefaultGeneratorKeyMap() GeneratorKeyMap {
	return GeneratorKeyMap{
		Regenerate: key.NewBinding(
			key.WithKeys("enter", "r"),
			key.WithHelp("enter", "regen"),
		),
		Mode: key.NewBinding(
			key.WithKeys("m", "tab"),
			key.WithHelp("m", "mode"),
		),
		Longer: key.NewBinding(
			key.WithKeys("+", "="),
			key.WithHelp("+", "longer"),
		),
		Shorter: key.NewBinding(
			key.WithKeys("-"),
			key.WithHelp("-", "shorter"),
		),
		Option1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "option 1"),
		),
		Option2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "option 2"),
		),
		Option3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "option 3"),
		),
		Option4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "option 4"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy"),
		),
		Use: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "use"),
			key.WithDisabled(),
		),
		Close: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),
	}
}

// NewItemKeyMap defines the keys of the card that picks the type of a new
// item.
type NewItemKeyMap struct {
	Login  key.Binding
	Note   key.Binding
	Cancel key.Binding
}

// DefaultNewItemKeyMap returns the standard bindings for the new item card.
func DefaultNewItemKeyMap() NewItemKeyMap {
	return NewItemKeyMap{
		Login: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "login"),
		),
		Note: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "secure note"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// FormKeyMap defines the keys of the item form. Letters type text, so the
// commands use tab, ctrl and esc.
type FormKeyMap struct {
	Next     key.Binding
	Prev     key.Binding
	Save     key.Binding
	Generate key.Binding
	Reveal   key.Binding
	Cancel   key.Binding
}

// DefaultFormKeyMap returns the standard bindings for the item form.
func DefaultFormKeyMap() FormKeyMap {
	return FormKeyMap{
		Next: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		Prev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous field"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		Generate: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "generate password"),
		),
		Reveal: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "show or hide password"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}
