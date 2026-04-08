package ui

import "testing"

func TestDefaultVaultKeyMap(t *testing.T) {
	km := DefaultVaultKeyMap()

	// All fields should have at least one key binding.
	bindings := []struct {
		name string
		keys []string
	}{
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
		{"ToggleGrouping", km.ToggleGrouping.Keys()},
		{"ToggleExpand", km.ToggleExpand.Keys()},
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

func TestHelpBindings(t *testing.T) {
	km := DefaultVaultKeyMap()
	bindings := km.HelpBindings()

	if len(bindings) == 0 {
		t.Fatal("HelpBindings() returned no bindings")
	}

	// Check all expected categories are present.
	categories := make(map[string]bool)
	for _, b := range bindings {
		categories[b.Category] = true
	}
	for _, cat := range []string{"Copy", "Navigation", "Vault", "UI"} {
		if !categories[cat] {
			t.Errorf("missing category %q", cat)
		}
	}

	// Every binding should have a key and description.
	for _, b := range bindings {
		if b.Key == "" {
			t.Errorf("binding with empty key in category %q", b.Category)
		}
		if b.Desc == "" {
			t.Errorf("binding %q has empty description", b.Key)
		}
	}
}
