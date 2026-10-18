package screens

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/list"
	"github.com/juthrbog/lazybw/bwcmd"
)

// GroupHeaderItem represents a collapsible group in the vault list.
type GroupHeaderItem struct {
	BaseKey  string
	Name     string // the baseName of the first item, in the case of that item
	Count    int
	Expanded bool
}

func (g GroupHeaderItem) FilterValue() string { return g.BaseKey }

// groupState holds the current grouping toggle and per-group expansion state.
type groupState struct {
	enabled  bool
	expanded map[string]bool
}

func newGroupState() groupState {
	return groupState{expanded: make(map[string]bool)}
}

func (gs *groupState) toggle(baseKey string) {
	gs.expanded[baseKey] = !gs.expanded[baseKey]
}

func (gs *groupState) toggleGrouping() {
	gs.enabled = !gs.enabled
	if !gs.enabled {
		clear(gs.expanded)
	}
}

// baseName repeatedly strips trailing parenthetical " (qualifier)" and dash
// " - qualifier" suffixes from a name. It keeps the case of the name.
//
//	"AWS (Prod) - Recovery Codes" → "AWS (Prod)" → "AWS"
//	"GitHub (Work)" → "GitHub"
//	"Gmail" → "Gmail"
func baseName(name string) string {
	k := strings.TrimSpace(name)
	for {
		stripped := stripTrailingParen(k)
		stripped = stripTrailingDash(stripped)
		if stripped == k {
			break
		}
		k = stripped
	}
	return k
}

// baseNameKey derives a group key: the case-folded baseName.
//
//	"AWS (Prod) - Recovery Codes" → "aws"
//	"GitHub (Work)" → "github"
//	"Gmail" → "gmail"
func baseNameKey(name string) string {
	return strings.ToLower(baseName(name))
}

// stripTrailingParen removes a final " (…)" suffix.
func stripTrailingParen(s string) string {
	if idx := strings.LastIndex(s, " ("); idx > 0 && strings.HasSuffix(s, ")") {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

// stripTrailingDash removes a final " - …" suffix.
func stripTrailingDash(s string) string {
	if idx := strings.LastIndex(s, " - "); idx > 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

// containsWord reports whether key occurs in s as a whole word: the
// characters right before and right after the match are not letters or
// digits. So "aws" is a word in "AWS Access Key" and "aws.example.com",
// but not in "Laws of Motion".
func containsWord(s, key string) bool {
	if key == "" {
		return false
	}
	for i := 0; i < len(s); {
		j := strings.Index(s[i:], key)
		if j < 0 {
			return false
		}
		start, end := i+j, i+j+len(key)
		before, _ := utf8.DecodeLastRuneInString(s[:start])
		after, _ := utf8.DecodeRuneInString(s[end:])
		if !isWordRune(before) && !isWordRune(after) {
			return true
		}
		// Look for the next match after the first character of this one.
		_, size := utf8.DecodeRuneInString(s[start:])
		i = start + size
	}
	return false
}

// isWordRune reports whether r is a letter or a digit. utf8.RuneError,
// which the decode functions return at the ends of a string, is neither.
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// itemGroup holds the vault items that one row of a grouped list stands
// for: a single item, or a group of two or more items.
type itemGroup struct {
	key   string
	name  string // the baseName of the first item
	items []bwcmd.Item
}

// groupItems sorts the vault items into groups, in the order in which
// their keys first show up. It runs the costly part of grouping, so the
// vault keeps the result until the items change, and an expand or a
// collapse only flattens it again.
func groupItems(items []bwcmd.Item) []itemGroup {
	// Phase 1: Bucket items by normalized key, preserving first-seen order.
	seen := make(map[string]int) // baseKey → index in buckets
	var buckets []itemGroup

	for _, item := range items {
		k := baseNameKey(item.Name)
		if idx, ok := seen[k]; ok {
			buckets[idx].items = append(buckets[idx].items, item)
		} else {
			seen[k] = len(buckets)
			buckets = append(buckets, itemGroup{key: k, name: baseName(item.Name), items: []bwcmd.Item{item}})
		}
	}

	// Phase 2: Absorption — ungrouped items (singleton buckets) whose name
	// contains an existing multi-item group key as a whole word get
	// absorbed into that group. When multiple group keys match, the longest
	// key wins (most specific).
	groupKeys := make([]string, 0, len(buckets))
	for _, b := range buckets {
		if len(b.items) >= 2 {
			groupKeys = append(groupKeys, b.key)
		}
	}

	// Iterate singletons and try to absorb them.
	for i := range buckets {
		if len(buckets[i].items) != 1 {
			continue
		}
		nameLower := strings.ToLower(buckets[i].items[0].Name)
		bestKey := ""
		for _, gk := range groupKeys {
			if len(gk) > len(bestKey) && containsWord(nameLower, gk) {
				bestKey = gk
			}
		}
		if bestKey != "" {
			target := seen[bestKey]
			buckets[target].items = append(buckets[target].items, buckets[i].items[0])
			buckets[i].items = nil // mark as absorbed
		}
	}

	groups := buckets[:0]
	for _, b := range buckets {
		if len(b.items) > 0 {
			groups = append(groups, b)
		}
	}
	return groups
}

// flattenGroups builds the list rows for groups: a row for each single
// item, and a header for each group of two or more items, followed by its
// items when the group is expanded.
func flattenGroups(groups []itemGroup, gs groupState) []list.Item {
	result := make([]list.Item, 0, len(groups))
	for _, g := range groups {
		if len(g.items) < 2 {
			result = append(result, VaultItem{Item: g.items[0]})
			continue
		}
		expanded := gs.expanded[g.key]
		result = append(result, GroupHeaderItem{
			BaseKey:  g.key,
			Name:     g.name,
			Count:    len(g.items),
			Expanded: expanded,
		})
		if expanded {
			for _, item := range g.items {
				result = append(result, VaultItem{Item: item, Indent: true})
			}
		}
	}
	return result
}

// buildGroupedItems converts raw vault items into a mixed list of
// GroupHeaderItem and VaultItem entries based on the current group state.
func buildGroupedItems(items []bwcmd.Item, gs groupState) []list.Item {
	if !gs.enabled {
		return toListItems(items)
	}
	return flattenGroups(groupItems(items), gs)
}

// groupCache holds the groups of the vault items, so that an expand or a
// collapse does not group the items again.
type groupCache struct {
	groups []itemGroup
	keyOf  map[string]string // item ID → group key, for groups of two or more items
}

func newGroupCache(items []bwcmd.Item) *groupCache {
	c := &groupCache{groups: groupItems(items), keyOf: make(map[string]string)}
	for _, g := range c.groups {
		if len(g.items) < 2 {
			continue
		}
		for _, item := range g.items {
			c.keyOf[item.ID] = g.key
		}
	}
	return c
}

// firstID returns the ID of the first item of the group with the given key,
// or "" when there is no such group.
func (c *groupCache) firstID(key string) string {
	for _, g := range c.groups {
		if g.key == key && len(g.items) >= 2 {
			return g.items[0].ID
		}
	}
	return ""
}

// groupToastMessage returns the toast text for a grouping toggle.
func groupToastMessage(enabled bool) string {
	if enabled {
		return "Grouping enabled"
	}
	return "Grouping disabled"
}
