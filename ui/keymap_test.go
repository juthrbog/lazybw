package ui

import (
	"reflect"
	"slices"
	"testing"

	"charm.land/bubbles/v2/key"
)

func TestDefaultVaultKeyMap(t *testing.T) {
	want := map[string][]string{
		"Copy":           {"c"},
		"CopyTOTP":       {"t"},
		"CopyUsername":   {"u"},
		"CopyNote":       {"n"},
		"CopyCVV":        {"v"},
		"OpenURL":        {"o"},
		"ScrollDown":     {"J", "shift+down"},
		"ScrollUp":       {"K", "shift+up"},
		"Sync":           {"r"},
		"Lock":           {"l"},
		"Help":           {"?"},
		"Quit":           {"q", "ctrl+c"},
		"CycleTheme":     {"T"},
		"Generate":       {"p"},
		"ToggleGrouping": {"ctrl+g"},
		"ToggleExpand":   {"enter"},
		"DrawerGrow":     {"+"},
		"DrawerShrink":   {"-"},
		"DrawerReset":    {"="},
		"DrawerToggle":   {"d"},
	}

	// Walk the struct, so that a new binding fails this test until the
	// table lists it.
	v := reflect.ValueOf(DefaultVaultKeyMap())
	if v.NumField() != len(want) {
		t.Errorf("VaultKeyMap has %d bindings, the test lists %d", v.NumField(), len(want))
	}
	owner := make(map[string]string)
	for i := range v.NumField() {
		name := v.Type().Field(i).Name
		b, ok := v.Field(i).Interface().(key.Binding)
		if !ok {
			t.Errorf("%s is not a key.Binding", name)
			continue
		}
		if keys, ok := want[name]; !ok {
			t.Errorf("%s is not in the test table", name)
		} else if !slices.Equal(b.Keys(), keys) {
			t.Errorf("%s keys = %v, want %v", name, b.Keys(), keys)
		}
		if h := b.Help(); h.Key == "" || h.Desc == "" {
			t.Errorf("%s has no help text: %+v", name, h)
		}
		for _, k := range b.Keys() {
			if other, ok := owner[k]; ok {
				t.Errorf("%s and %s both use %q", other, name, k)
			}
			owner[k] = name
		}
	}
}

func TestCopyCVVKey(t *testing.T) {
	km := DefaultVaultKeyMap()
	if keys := km.CopyCVV.Keys(); len(keys) != 1 || keys[0] != "v" {
		t.Errorf("CopyCVV keys = %v, want [v]", keys)
	}
	found := false
	for _, b := range km.HelpBindings() {
		if b.Key == "v" && b.Category == "Copy" {
			found = true
		}
	}
	if !found {
		t.Error("the help must list v under Copy")
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
