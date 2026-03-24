package ui

import "testing"

func TestDefaultVaultKeyMap(t *testing.T) {
	km := DefaultVaultKeyMap()

	// All fields should have at least one key binding.
	bindings := []struct {
		name string
		keys []string
	}{
		{"Up", km.Up.Keys()},
		{"Down", km.Down.Keys()},
		{"Top", km.Top.Keys()},
		{"Bottom", km.Bottom.Keys()},
		{"Filter", km.Filter.Keys()},
		{"Copy", km.Copy.Keys()},
		{"CopyTOTP", km.CopyTOTP.Keys()},
		{"CopyUsername", km.CopyUsername.Keys()},
		{"OpenURL", km.OpenURL.Keys()},
		{"ScrollDown", km.ScrollDown.Keys()},
		{"ScrollUp", km.ScrollUp.Keys()},
		{"Sync", km.Sync.Keys()},
		{"Lock", km.Lock.Keys()},
		{"Help", km.Help.Keys()},
		{"Quit", km.Quit.Keys()},
		{"CycleTheme", km.CycleTheme.Keys()},
		{"Generate", km.Generate.Keys()},
	}

	for _, b := range bindings {
		if len(b.keys) == 0 {
			t.Errorf("%s has no key bindings", b.name)
		}
	}
}

func TestShortHelp(t *testing.T) {
	km := DefaultVaultKeyMap()
	help := km.ShortHelp()
	if len(help) != 6 {
		t.Errorf("ShortHelp() returned %d bindings, want 6", len(help))
	}
}

func TestFullHelp(t *testing.T) {
	km := DefaultVaultKeyMap()
	groups := km.FullHelp()
	if len(groups) != 5 {
		t.Errorf("FullHelp() returned %d groups, want 5", len(groups))
	}
	// Every group should have at least one binding.
	for i, g := range groups {
		if len(g) == 0 {
			t.Errorf("group %d is empty", i)
		}
	}
}
