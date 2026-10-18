package ui

import (
	"os"
	"reflect"
	"regexp"
	"testing"

	"charm.land/bubbles/v2/key"
)

// keybindingsDoc is the key list for users, relative to this package.
const keybindingsDoc = "../docs/KEYBINDINGS.md"

// inlineCode matches a Markdown code span in single backticks, for example
// `c` or `shift+down`. The first group is the text in the span.
var inlineCode = regexp.MustCompile("`([^`\n]+)`")

// TestKeybindingsDocListsVaultKeys fails when a key of a DefaultVaultKeyMap
// binding is not in docs/KEYBINDINGS.md as inline code. The test walks the
// struct, so a new binding also fails until the doc lists its keys.
func TestKeybindingsDocListsVaultKeys(t *testing.T) {
	doc, err := os.ReadFile(keybindingsDoc)
	if err != nil {
		t.Fatalf("read %s: %v", keybindingsDoc, err)
	}
	documented := make(map[string]bool)
	for _, m := range inlineCode.FindAllStringSubmatch(string(doc), -1) {
		documented[m[1]] = true
	}

	v := reflect.ValueOf(DefaultVaultKeyMap())
	for i := range v.NumField() {
		name := v.Type().Field(i).Name
		b, ok := v.Field(i).Interface().(key.Binding)
		if !ok {
			t.Errorf("%s is not a key.Binding", name)
			continue
		}
		if len(b.Keys()) == 0 {
			t.Errorf("%s has no keys", name)
		}
		for _, k := range b.Keys() {
			if !documented[k] {
				t.Errorf("%s does not list the key %q of %s as inline code", keybindingsDoc, k, name)
			}
		}
	}
}
