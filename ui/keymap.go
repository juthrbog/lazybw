package ui

import "charm.land/bubbles/v2/key"

// VaultKeyMap defines app-specific keybindings for the main vault screen.
// Navigation (j/k, g/G, /, pgup/pgdn) is handled by the embedded bubbles/list.
type VaultKeyMap struct {
	Copy           key.Binding
	CopyTOTP       key.Binding
	CopyUsername   key.Binding
	OpenURL        key.Binding
	ScrollDown     key.Binding
	ScrollUp       key.Binding
	Sync           key.Binding
	Lock           key.Binding
	Help           key.Binding
	Quit           key.Binding
	CycleTheme     key.Binding
	Generate       key.Binding
	ToggleGrouping key.Binding
	ToggleExpand   key.Binding
}

// ShortHelp implements help.KeyMap.
func (k VaultKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Copy, k.CopyTOTP, k.CopyUsername, k.ToggleGrouping, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k VaultKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Copy, k.CopyTOTP, k.CopyUsername, k.OpenURL},
		{k.Generate, k.Sync, k.Lock},
		{k.ScrollDown, k.ScrollUp},
		{k.ToggleGrouping, k.ToggleExpand},
		{k.Help, k.CycleTheme, k.Quit},
	}
}

// HelpBindings returns categorized bindings for the help overlay.
func (k VaultKeyMap) HelpBindings() []HelpBinding {
	return []HelpBinding{
		// Copy
		{Key: "c", Desc: "copy password", Category: "Copy"},
		{Key: "t", Desc: "copy TOTP", Category: "Copy"},
		{Key: "u", Desc: "copy username", Category: "Copy"},
		{Key: "o", Desc: "open URL", Category: "Copy"},
		// Navigation
		{Key: "j/k", Desc: "move up/down", Category: "Navigation"},
		{Key: "g/G", Desc: "jump to top/bottom", Category: "Navigation"},
		{Key: "/", Desc: "filter", Category: "Navigation"},
		{Key: "J/K", Desc: "scroll drawer", Category: "Navigation"},
		// Vault
		{Key: "p", Desc: "generate password", Category: "Vault"},
		{Key: "r", Desc: "sync", Category: "Vault"},
		{Key: "l", Desc: "lock", Category: "Vault"},
		{Key: "ctrl+g", Desc: "toggle grouping", Category: "Vault"},
		{Key: "enter", Desc: "expand/collapse", Category: "Vault"},
		// UI
		{Key: "?", Desc: "help", Category: "UI"},
		{Key: "T", Desc: "cycle theme", Category: "UI"},
		{Key: "q", Desc: "quit", Category: "UI"},
	}
}

// DefaultVaultKeyMap returns the standard bindings for the vault screen.
func DefaultVaultKeyMap() VaultKeyMap {
	return VaultKeyMap{
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy pwd"),
		),
		CopyTOTP: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "copy totp"),
		),
		CopyUsername: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "copy user"),
		),
		OpenURL: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open url"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("J", "shift+down"),
			key.WithHelp("J", "scroll drawer ↓"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("K", "shift+up"),
			key.WithHelp("K", "scroll drawer ↑"),
		),
		Sync: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "sync"),
		),
		Lock: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "lock"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		CycleTheme: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "theme"),
		),
		Generate: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "generate"),
		),
		ToggleGrouping: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "group"),
		),
		ToggleExpand: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "expand/collapse"),
		),
	}
}
