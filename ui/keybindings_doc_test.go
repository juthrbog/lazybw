package ui

import (
	"os"
	"regexp"
	"testing"
)

// keybindingsDoc is the key list for users, relative to this package.
const keybindingsDoc = "../docs/KEYBINDINGS.md"

// inlineCode matches a Markdown code span in single backticks, for example
// `c` or `shift+down`. The first group is the text in the span.
var inlineCode = regexp.MustCompile("`([^`\n]+)`")

// TestKeybindingsDocListsKeys fails when a key of a binding of
// DefaultVaultKeyMap, DefaultGeneratorKeyMap, DefaultConfirmKeyMap,
// DefaultNewItemKeyMap or DefaultFormKeyMap is not in docs/KEYBINDINGS.md as inline code. The test walks the structs, so
// a new binding also fails until the doc lists its keys.
func TestKeybindingsDocListsKeys(t *testing.T) {
	doc, err := os.ReadFile(keybindingsDoc)
	if err != nil {
		t.Fatalf("read %s: %v", keybindingsDoc, err)
	}
	documented := make(map[string]bool)
	for _, m := range inlineCode.FindAllStringSubmatch(string(doc), -1) {
		documented[m[1]] = true
	}

	for _, km := range []any{DefaultVaultKeyMap(), DefaultGeneratorKeyMap(), DefaultConfirmKeyMap(), DefaultNewItemKeyMap(), DefaultFormKeyMap()} {
		names, bindings := bindingsOf(t, km)
		for i, b := range bindings {
			if len(b.Keys()) == 0 {
				t.Errorf("%T.%s has no keys", km, names[i])
			}
			for _, k := range b.Keys() {
				if !documented[k] {
					t.Errorf("%s does not list the key %q of %T.%s as inline code", keybindingsDoc, k, km, names[i])
				}
			}
		}
	}
}
