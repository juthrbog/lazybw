package screens

import (
	"strings"

	"charm.land/bubbles/v2/list"
	"github.com/juthrbog/lazybw/bwcmd"
)

// GroupHeaderItem represents a collapsible group in the vault list.
type GroupHeaderItem struct {
	BaseKey  string
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

// baseNameKey derives a group key by case-folding and repeatedly stripping
// trailing parenthetical " (qualifier)" and dash " - qualifier" suffixes.
//
//	"AWS (Prod) - Recovery Codes" → "aws (prod)" → "aws"
//	"GitHub (Work)" → "github"
//	"Gmail" → "gmail"
func baseNameKey(name string) string {
	k := strings.ToLower(strings.TrimSpace(name))
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

// buildGroupedItems converts raw vault items into a mixed list of
// GroupHeaderItem and VaultItem entries based on the current group state.
func buildGroupedItems(items []bwcmd.Item, gs groupState) []list.Item {
	if !gs.enabled {
		result := make([]list.Item, len(items))
		for i, item := range items {
			result[i] = VaultItem{Item: item}
		}
		return result
	}

	// Phase 1: Bucket items by normalized key, preserving first-seen order.
	type bucket struct {
		key   string
		items []bwcmd.Item
	}
	seen := make(map[string]int) // baseKey → index in buckets
	var buckets []bucket

	for _, item := range items {
		k := baseNameKey(item.Name)
		if idx, ok := seen[k]; ok {
			buckets[idx].items = append(buckets[idx].items, item)
		} else {
			seen[k] = len(buckets)
			buckets = append(buckets, bucket{key: k, items: []bwcmd.Item{item}})
		}
	}

	// Phase 2: Absorption — ungrouped items (singleton buckets) whose name
	// contains an existing multi-item group key get absorbed into that group.
	// When multiple group keys match, the longest key wins (most specific).
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
			if strings.Contains(nameLower, gk) && len(gk) > len(bestKey) {
				bestKey = gk
			}
		}
		if bestKey != "" {
			target := seen[bestKey]
			buckets[target].items = append(buckets[target].items, buckets[i].items[0])
			buckets[i].items = nil // mark as absorbed
		}
	}

	// Phase 3: Build the final list.
	var result []list.Item
	for _, b := range buckets {
		if len(b.items) == 0 {
			continue // absorbed
		}
		if len(b.items) < 2 {
			result = append(result, VaultItem{Item: b.items[0]})
			continue
		}
		expanded := gs.expanded[b.key]
		result = append(result, GroupHeaderItem{
			BaseKey:  b.key,
			Count:    len(b.items),
			Expanded: expanded,
		})
		if expanded {
			for _, item := range b.items {
				result = append(result, VaultItem{Item: item, Indent: true})
			}
		}
	}
	return result
}

// groupToastMessage returns the toast text for a grouping toggle.
func groupToastMessage(enabled bool) string {
	if enabled {
		return "Grouping enabled"
	}
	return "Grouping disabled"
}
