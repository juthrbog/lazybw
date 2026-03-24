package bwcmd

import "testing"

func TestItemDescription(t *testing.T) {
	tests := []struct {
		name string
		item Item
		want string
	}{
		{
			name: "login with username",
			item: Item{Type: ItemTypeLogin, Login: &Login{Username: "user@test.com"}},
			want: "user@test.com",
		},
		{
			name: "login nil",
			item: Item{Type: ItemTypeLogin},
			want: "",
		},
		{
			name: "card with full number",
			item: Item{Type: ItemTypeCard, Card: &Card{Number: "4242424242424242"}},
			want: "•••• 4242",
		},
		{
			name: "card short number",
			item: Item{Type: ItemTypeCard, Card: &Card{Number: "42"}},
			want: "",
		},
		{
			name: "card nil",
			item: Item{Type: ItemTypeCard},
			want: "",
		},
		{
			name: "note with content",
			item: Item{Type: ItemTypeSecureNote, Notes: "first line\nsecond line"},
			want: "first line",
		},
		{
			name: "note empty content",
			item: Item{Type: ItemTypeSecureNote},
			want: "(note)",
		},
		{
			name: "identity with name",
			item: Item{Type: ItemTypeIdentity, Identity: &Identity{FirstName: "John", LastName: "Doe"}},
			want: "John Doe",
		},
		{
			name: "identity email fallback",
			item: Item{Type: ItemTypeIdentity, Identity: &Identity{Email: "john@test.com"}},
			want: "john@test.com",
		},
		{
			name: "identity nil",
			item: Item{Type: ItemTypeIdentity},
			want: "",
		},
		{
			name: "unknown type",
			item: Item{Type: 99},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.Description()
			if got != tt.want {
				t.Errorf("Description() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestItemFilterValue(t *testing.T) {
	item := Item{Name: "Gmail"}
	if got := item.FilterValue(); got != "Gmail" {
		t.Errorf("FilterValue() = %q, want %q", got, "Gmail")
	}
}

func TestItemTitle(t *testing.T) {
	item := Item{Name: "GitHub"}
	if got := item.Title(); got != "GitHub" {
		t.Errorf("Title() = %q, want %q", got, "GitHub")
	}
}
