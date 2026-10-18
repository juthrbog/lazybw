package screens

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/juthrbog/lazybw/bwcmd"
)

// benchItems returns n items. Two of every three are in groups of two,
// and the others have names of their own.
func benchItems(n int) []bwcmd.Item {
	items := make([]bwcmd.Item, n)
	for i := range items {
		name := fmt.Sprintf("Site %d", i)
		switch i % 3 {
		case 0:
			name = fmt.Sprintf("Service %d (Prod)", i)
		case 1:
			name = fmt.Sprintf("Service %d (Dev)", i-1)
		}
		items[i] = bwcmd.Item{ID: fmt.Sprint(i), Type: bwcmd.ItemTypeLogin, Name: name}
	}
	return items
}

// BenchmarkToggleExpand measures enter on a group header of a grouped
// vault with 10,000 items. Each run expands or collapses the group.
func BenchmarkToggleExpand(b *testing.B) {
	m := newTestVault(benchItems(10_000))
	updated, _ := m.Update(ctrlG)
	m = updated.(VaultModel)
	if _, ok := m.list.SelectedItem().(GroupHeaderItem); !ok {
		b.Fatalf("selected = %T, want a group header", m.list.SelectedItem())
	}
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}
	b.ResetTimer()
	for range b.N {
		updated, _ = m.Update(enter)
		m = updated.(VaultModel)
	}
}

// BenchmarkGroupItems measures the grouping of 10,000 items, which the
// first ctrl+g after a load or a sync runs.
func BenchmarkGroupItems(b *testing.B) {
	items := benchItems(10_000)
	for range b.N {
		groupItems(items)
	}
}
