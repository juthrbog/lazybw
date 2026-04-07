package screens

import (
	"testing"

	"charm.land/bubbles/v2/list"
	"github.com/juthrbog/lazybw/bwcmd"
)

func TestBaseNameKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GitHub", "github"},
		{"GitHub (Work)", "github"},
		{"GitHub (Personal)", "github"},
		{"Gmail", "gmail"},
		{"Something (a) (b)", "something"},       // strips both suffixes
		{"(starts with paren)", "(starts with paren)"},
		{"trailing space (x) ", "trailing space"}, // trimmed then stripped
		{"", ""},
		{"No qualifier", "no qualifier"},
		{"A (B)", "a"},
		{"GIThub (Org)", "github"},                          // case-folded + stripped
		{"AWS (Prod) - Recovery Codes", "aws"},              // dash then paren stripped
		{"AWS Access Key - deploy", "aws access key"},
		{"foo - bar - baz", "foo"},                          // repeated dash stripping
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := baseNameKey(tt.input)
			if got != tt.want {
				t.Errorf("baseNameKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func githubItems() []bwcmd.Item {
	return []bwcmd.Item{
		{ID: "1", Name: "GitHub", Type: bwcmd.ItemTypeLogin},
		{ID: "2", Name: "GitHub (Work)", Type: bwcmd.ItemTypeLogin},
		{ID: "3", Name: "GitHub (Personal)", Type: bwcmd.ItemTypeLogin},
		{ID: "4", Name: "Gmail", Type: bwcmd.ItemTypeLogin},
		{ID: "5", Name: "Slack", Type: bwcmd.ItemTypeLogin},
	}
}

func TestBuildGroupedItemsDisabled(t *testing.T) {
	gs := newGroupState()
	items := githubItems()
	result := buildGroupedItems(items, gs)

	if len(result) != len(items) {
		t.Fatalf("expected %d items, got %d", len(items), len(result))
	}
	for i, li := range result {
		vi, ok := li.(VaultItem)
		if !ok {
			t.Fatalf("item %d: expected VaultItem, got %T", i, li)
		}
		if vi.Name != items[i].Name {
			t.Errorf("item %d: expected %q, got %q", i, items[i].Name, vi.Name)
		}
		if vi.Indent {
			t.Errorf("item %d: should not be indented when grouping disabled", i)
		}
	}
}

func TestBuildGroupedItemsCollapsed(t *testing.T) {
	gs := newGroupState()
	gs.enabled = true
	result := buildGroupedItems(githubItems(), gs)

	// Expect: GroupHeader("github"), VaultItem("Gmail"), VaultItem("Slack")
	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}

	gh, ok := result[0].(GroupHeaderItem)
	if !ok {
		t.Fatalf("item 0: expected GroupHeaderItem, got %T", result[0])
	}
	if gh.BaseKey != "github" || gh.Count != 3 || gh.Expanded {
		t.Errorf("header: got %+v", gh)
	}

	for i, name := range []string{"Gmail", "Slack"} {
		vi, ok := result[i+1].(VaultItem)
		if !ok {
			t.Fatalf("item %d: expected VaultItem, got %T", i+1, result[i+1])
		}
		if vi.Name != name {
			t.Errorf("item %d: expected %q, got %q", i+1, name, vi.Name)
		}
	}
}

func TestBuildGroupedItemsExpanded(t *testing.T) {
	gs := newGroupState()
	gs.enabled = true
	gs.expanded["github"] = true
	result := buildGroupedItems(githubItems(), gs)

	// Expect: Header, GitHub, GitHub (Work), GitHub (Personal), Gmail, Slack
	if len(result) != 6 {
		t.Fatalf("expected 6 items, got %d", len(result))
	}

	gh := result[0].(GroupHeaderItem)
	if !gh.Expanded {
		t.Error("header should be expanded")
	}

	// Children should be indented.
	for i := 1; i <= 3; i++ {
		vi := result[i].(VaultItem)
		if !vi.Indent {
			t.Errorf("child %d should be indented", i)
		}
	}

	// Ungrouped items should not be indented.
	for i := 4; i <= 5; i++ {
		vi := result[i].(VaultItem)
		if vi.Indent {
			t.Errorf("ungrouped item %d should not be indented", i)
		}
	}
}

func TestMinGroupSize(t *testing.T) {
	items := []bwcmd.Item{
		{ID: "1", Name: "Unique", Type: bwcmd.ItemTypeLogin},
		{ID: "2", Name: "Also Unique", Type: bwcmd.ItemTypeLogin},
	}
	gs := newGroupState()
	gs.enabled = true
	result := buildGroupedItems(items, gs)

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	for i, li := range result {
		if _, ok := li.(GroupHeaderItem); ok {
			t.Errorf("item %d should not be a group header", i)
		}
	}
}

func TestToggleExpandCollapse(t *testing.T) {
	gs := newGroupState()
	gs.enabled = true

	gs.toggle("github")
	if !gs.expanded["github"] {
		t.Error("expected expanded after first toggle")
	}

	gs.toggle("github")
	if gs.expanded["github"] {
		t.Error("expected collapsed after second toggle")
	}
}

func TestToggleGroupingClearsExpanded(t *testing.T) {
	gs := newGroupState()
	gs.enabled = true
	gs.expanded["github"] = true

	gs.toggleGrouping()
	if gs.enabled {
		t.Error("expected disabled")
	}
	if len(gs.expanded) != 0 {
		t.Error("expected expanded map to be cleared")
	}
}

func TestGroupHeaderFilterValue(t *testing.T) {
	gh := GroupHeaderItem{BaseKey: "github", Count: 3}
	if gh.FilterValue() != "github" {
		t.Errorf("FilterValue() = %q, want %q", gh.FilterValue(), "github")
	}
}

func TestBuildGroupedItemsPreservesOrder(t *testing.T) {
	items := []bwcmd.Item{
		{ID: "1", Name: "Bravo", Type: bwcmd.ItemTypeLogin},
		{ID: "2", Name: "Alpha (1)", Type: bwcmd.ItemTypeLogin},
		{ID: "3", Name: "Alpha (2)", Type: bwcmd.ItemTypeLogin},
		{ID: "4", Name: "Charlie", Type: bwcmd.ItemTypeLogin},
	}
	gs := newGroupState()
	gs.enabled = true
	result := buildGroupedItems(items, gs)

	// Expect: Bravo, Header(alpha), Charlie
	if len(result) != 3 {
		var types []string
		for _, r := range result {
			switch v := r.(type) {
			case GroupHeaderItem:
				types = append(types, "Header("+v.BaseKey+")")
			case VaultItem:
				types = append(types, "Item("+v.Name+")")
			}
		}
		t.Fatalf("expected 3 items, got %d: %v", len(result), types)
	}

	assertVaultItem(t, result[0], "Bravo")
	if gh, ok := result[1].(GroupHeaderItem); !ok || gh.BaseKey != "alpha" {
		t.Errorf("item 1: expected alpha header, got %+v", result[1])
	}
	assertVaultItem(t, result[2], "Charlie")
}

func TestCaseInsensitiveGrouping(t *testing.T) {
	items := []bwcmd.Item{
		{ID: "1", Name: "AWS", Type: bwcmd.ItemTypeLogin},
		{ID: "2", Name: "Aws (Staging)", Type: bwcmd.ItemTypeLogin},
		{ID: "3", Name: "aws (Dev)", Type: bwcmd.ItemTypeLogin},
	}
	gs := newGroupState()
	gs.enabled = true
	result := buildGroupedItems(items, gs)

	// All three normalize to "aws" → single group.
	if len(result) != 1 {
		dumpItems(t, result)
		t.Fatalf("expected 1 group header, got %d items", len(result))
	}
	gh := result[0].(GroupHeaderItem)
	if gh.BaseKey != "aws" || gh.Count != 3 {
		t.Errorf("expected aws group with 3 items, got %+v", gh)
	}
}

func TestDashSuffixStripping(t *testing.T) {
	items := []bwcmd.Item{
		{ID: "1", Name: "AWS", Type: bwcmd.ItemTypeLogin},
		{ID: "2", Name: "AWS (Prod)", Type: bwcmd.ItemTypeLogin},
		{ID: "3", Name: "AWS (Prod) - Recovery Codes", Type: bwcmd.ItemTypeLogin},
	}
	gs := newGroupState()
	gs.enabled = true
	result := buildGroupedItems(items, gs)

	// All normalize to "aws".
	if len(result) != 1 {
		dumpItems(t, result)
		t.Fatalf("expected 1 group header, got %d items", len(result))
	}
	gh := result[0].(GroupHeaderItem)
	if gh.Count != 3 {
		t.Errorf("expected 3 items in group, got %d", gh.Count)
	}
}

func TestAbsorptionBySubstring(t *testing.T) {
	items := []bwcmd.Item{
		{ID: "1", Name: "AWS", Type: bwcmd.ItemTypeLogin},
		{ID: "2", Name: "AWS (Prod)", Type: bwcmd.ItemTypeLogin},
		{ID: "3", Name: "AWS Access Key - deploy", Type: bwcmd.ItemTypeLogin},
		{ID: "4", Name: "aws.console.example.com", Type: bwcmd.ItemTypeLogin},
		{ID: "5", Name: "Gmail", Type: bwcmd.ItemTypeLogin},
	}
	gs := newGroupState()
	gs.enabled = true
	result := buildGroupedItems(items, gs)

	// Items 1+2 form "aws" group via key matching.
	// Items 3+4 get absorbed (names contain "aws").
	// Item 5 stays flat.
	if len(result) != 2 {
		dumpItems(t, result)
		t.Fatalf("expected 2 items (1 header + 1 flat), got %d", len(result))
	}

	gh := result[0].(GroupHeaderItem)
	if gh.BaseKey != "aws" || gh.Count != 4 {
		t.Errorf("expected aws group with 4 items, got %+v", gh)
	}
	assertVaultItem(t, result[1], "Gmail")
}

func TestAbsorptionLongestKeyWins(t *testing.T) {
	items := []bwcmd.Item{
		{ID: "1", Name: "Git", Type: bwcmd.ItemTypeLogin},
		{ID: "2", Name: "Git (work)", Type: bwcmd.ItemTypeLogin},
		{ID: "3", Name: "GitHub", Type: bwcmd.ItemTypeLogin},
		{ID: "4", Name: "GitHub (Org)", Type: bwcmd.ItemTypeLogin},
		{ID: "5", Name: "GitHub Token", Type: bwcmd.ItemTypeLogin},
	}
	gs := newGroupState()
	gs.enabled = true
	result := buildGroupedItems(items, gs)

	// "git" group: items 1+2. "github" group: items 3+4.
	// Item 5 ("GitHub Token") contains both "git" and "github" — longest wins → "github".
	var gitGroup, githubGroup GroupHeaderItem
	for _, r := range result {
		if gh, ok := r.(GroupHeaderItem); ok {
			switch gh.BaseKey {
			case "git":
				gitGroup = gh
			case "github":
				githubGroup = gh
			}
		}
	}
	if gitGroup.Count != 2 {
		t.Errorf("git group: expected 2 items, got %d", gitGroup.Count)
	}
	if githubGroup.Count != 3 {
		t.Errorf("github group: expected 3 items (2 + absorbed), got %d", githubGroup.Count)
	}
}

func TestAbsorptionDoesNotCreateGroups(t *testing.T) {
	// If there's only one "aws" item, no group forms, so nothing can be absorbed.
	items := []bwcmd.Item{
		{ID: "1", Name: "AWS", Type: bwcmd.ItemTypeLogin},
		{ID: "2", Name: "AWS Access Key", Type: bwcmd.ItemTypeLogin},
		{ID: "3", Name: "Gmail", Type: bwcmd.ItemTypeLogin},
	}
	gs := newGroupState()
	gs.enabled = true
	result := buildGroupedItems(items, gs)

	// No group has 2+ members from strict matching, so all stay flat.
	if len(result) != 3 {
		dumpItems(t, result)
		t.Fatalf("expected 3 flat items, got %d", len(result))
	}
	for i, r := range result {
		if _, ok := r.(GroupHeaderItem); ok {
			t.Errorf("item %d should not be a group header", i)
		}
	}
}

func assertVaultItem(t *testing.T, item list.Item, name string) {
	t.Helper()
	vi, ok := item.(VaultItem)
	if !ok {
		t.Fatalf("expected VaultItem, got %T", item)
	}
	if vi.Name != name {
		t.Errorf("expected %q, got %q", name, vi.Name)
	}
}

func dumpItems(t *testing.T, items []list.Item) {
	t.Helper()
	for i, r := range items {
		switch v := r.(type) {
		case GroupHeaderItem:
			t.Logf("  [%d] Header(%s, count=%d)", i, v.BaseKey, v.Count)
		case VaultItem:
			t.Logf("  [%d] Item(%s)", i, v.Name)
		}
	}
}
