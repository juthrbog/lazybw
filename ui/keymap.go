package ui

import "github.com/charmbracelet/bubbles/key"

// VaultKeyMap defines all keybindings for the main vault screen.
type VaultKeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Top          key.Binding
	Bottom       key.Binding
	Filter       key.Binding
	Copy         key.Binding
	CopyTOTP     key.Binding
	CopyUsername key.Binding
	OpenURL      key.Binding
	ScrollDown   key.Binding
	ScrollUp     key.Binding
	Sync         key.Binding
	Lock         key.Binding
	Help         key.Binding
	Quit         key.Binding
	CycleTheme   key.Binding
	Generate     key.Binding
}

// ShortHelp implements help.KeyMap.
func (k VaultKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Filter, k.Copy, k.CopyTOTP, k.CopyUsername, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k VaultKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom},
		{k.Copy, k.CopyTOTP, k.CopyUsername, k.OpenURL},
		{k.Filter, k.Generate, k.Sync, k.Lock},
		{k.ScrollDown, k.ScrollUp},
		{k.Help, k.CycleTheme, k.Quit},
	}
}

// DefaultVaultKeyMap returns the standard bindings for the vault screen.
func DefaultVaultKeyMap() VaultKeyMap {
	return VaultKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "bottom"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
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
	}
}
