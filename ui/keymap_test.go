package ui

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"github.com/charmbracelet/x/ansi"
)

// bindingsOf returns the bindings of the key map km, a struct of
// key.Binding fields, by field name. It fails the test for a field that is
// not a key.Binding.
func bindingsOf(t *testing.T, km any) (names []string, bindings []key.Binding) {
	t.Helper()
	v := reflect.ValueOf(km)
	for i := range v.NumField() {
		name := v.Type().Field(i).Name
		b, ok := v.Field(i).Interface().(key.Binding)
		if !ok {
			t.Errorf("%s is not a key.Binding", name)
			continue
		}
		names = append(names, name)
		bindings = append(bindings, b)
	}
	return names, bindings
}

// checkKeyMap checks the keys of each binding of km against want, which
// must list each field. Each binding must have help text, and no key may
// belong to two enabled bindings. A disabled binding can share a key, because
// the screen that turns it on takes the key from the other binding.
func checkKeyMap(t *testing.T, km any, want map[string][]string) {
	t.Helper()
	names, bindings := bindingsOf(t, km)
	if len(names) != len(want) {
		t.Errorf("%T has %d bindings, the test lists %d", km, len(names), len(want))
	}
	owner := make(map[string]string)
	for i, b := range bindings {
		name := names[i]
		if keys, ok := want[name]; !ok {
			t.Errorf("%s is not in the test table", name)
		} else if !slices.Equal(b.Keys(), keys) {
			t.Errorf("%s keys = %v, want %v", name, b.Keys(), keys)
		}
		if h := b.Help(); h.Key == "" || h.Desc == "" {
			t.Errorf("%s has no help text: %+v", name, h)
		}
		if !b.Enabled() {
			continue
		}
		for _, k := range b.Keys() {
			if other, ok := owner[k]; ok {
				t.Errorf("%s and %s both use %q", other, name, k)
			}
			owner[k] = name
		}
	}
}

func TestDefaultVaultKeyMap(t *testing.T) {
	// The struct walk makes a new binding fail this test until the table
	// lists it.
	checkKeyMap(t, DefaultVaultKeyMap(), map[string][]string{
		"Copy":           {"c"},
		"CopyTOTP":       {"t"},
		"CopyUsername":   {"u"},
		"CopyNote":       {"m"},
		"CopyCVV":        {"v"},
		"OpenURL":        {"o"},
		"Navigate":       {"j", "k", "down", "up"},
		"Jump":           {"g", "G", "home", "end"},
		"Filter":         {"/"},
		"ScrollDown":     {"J", "shift+down"},
		"ScrollUp":       {"K", "shift+up"},
		"DrawerGrow":     {"+"},
		"DrawerShrink":   {"-"},
		"DrawerReset":    {"="},
		"DrawerToggle":   {"z"},
		"New":            {"n"},
		"Generate":       {"p"},
		"Sync":           {"r"},
		"Lock":           {"l"},
		"ToggleGrouping": {"ctrl+g"},
		"ToggleExpand":   {"enter"},
		"Help":           {"?"},
		"ThemePicker":    {"T"},
		"Quit":           {"q"},
	})
}

func TestDefaultGeneratorKeyMap(t *testing.T) {
	checkKeyMap(t, DefaultGeneratorKeyMap(), map[string][]string{
		"Regenerate": {"enter", "r"},
		"Mode":       {"m", "tab"},
		"Longer":     {"+", "="},
		"Shorter":    {"-"},
		"Option1":    {"1"},
		"Option2":    {"2"},
		"Option3":    {"3"},
		"Option4":    {"4"},
		"Copy":       {"c"},
		"Use":        {"enter"},
		"Close":      {"esc"},
	})
}

func TestDefaultNewItemKeyMap(t *testing.T) {
	checkKeyMap(t, DefaultNewItemKeyMap(), map[string][]string{
		"Login":  {"l"},
		"Note":   {"s"},
		"Cancel": {"esc"},
	})
}

func TestDefaultFormKeyMap(t *testing.T) {
	checkKeyMap(t, DefaultFormKeyMap(), map[string][]string{
		"Next":     {"tab"},
		"Prev":     {"shift+tab"},
		"Save":     {"ctrl+s"},
		"Generate": {"ctrl+g"},
		"Reveal":   {"ctrl+r"},
		"Cancel":   {"esc"},
	})
}

func TestDefaultConfirmKeyMap(t *testing.T) {
	checkKeyMap(t, DefaultConfirmKeyMap(), map[string][]string{
		"Yes": {"y"},
		"No":  {"n", "esc"},
	})
}

// TestHelpOverlayShowsEachVaultBinding opens the help overlay with all
// vault bindings. Each binding of VaultKeyMap must show on its own line,
// with the key label and the description of its help.
func TestHelpOverlayShowsEachVaultBinding(t *testing.T) {
	km := DefaultVaultKeyMap()
	h := NewHelpOverlay()
	// A tall area, so that the viewport shows all lines.
	h.Show(km.HelpBindings(), 80, 200)
	lines := strings.Split(ansi.Strip(h.View()), "\n")

	names, bindings := bindingsOf(t, km)
	if n := len(km.HelpBindings()); n != len(bindings) {
		t.Errorf("the help has %d entries, VaultKeyMap has %d bindings", n, len(bindings))
	}
	for i, b := range bindings {
		help := b.Help()
		found := false
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) > 0 && fields[0] == help.Key && strings.Join(fields[1:], " ") == help.Desc {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("the help overlay has no line %q for %s:\n%s", help.Key+" "+help.Desc, names[i], strings.Join(lines, "\n"))
		}
	}
}

func TestHelpBindingsCategories(t *testing.T) {
	bindings := DefaultVaultKeyMap().HelpBindings()
	got := make(map[string]bool)
	for _, b := range bindings {
		if !slices.Contains(categoryOrder, b.Category) {
			t.Errorf("binding %q has the category %q, which the help does not order", b.Key, b.Category)
		}
		got[b.Category] = true
	}
	for _, cat := range categoryOrder {
		if !got[cat] {
			t.Errorf("no binding has the category %q", cat)
		}
	}
	for _, want := range []HelpBinding{
		{Key: "v", Desc: "copy card security code", Category: CategoryCopy},
		{Key: "T", Desc: "theme picker", Category: CategoryUI},
	} {
		if !slices.Contains(bindings, want) {
			t.Errorf("the help has no entry %+v", want)
		}
	}
}

func TestHintTakesKeyLabel(t *testing.T) {
	km := DefaultGeneratorKeyMap()
	if got := Hint(km.Copy, "copy"); got != (HintBinding{Key: "c", Desc: "copy"}) {
		t.Errorf("Hint = %+v, want c copy", got)
	}
	if got := HintPair(km.Option1, km.Option4, "-", "toggles"); got != (HintBinding{Key: "1-4", Desc: "toggles"}) {
		t.Errorf("HintPair = %+v, want 1-4 toggles", got)
	}
}
