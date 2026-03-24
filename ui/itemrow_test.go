package ui

import (
	"strings"
	"testing"

	"github.com/juthrbog/lazybw/bwcmd"
)

func TestRenderItemRowLoginSelected(t *testing.T) {
	item := bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Gmail",
		Login: &bwcmd.Login{Username: "user@gmail.com"},
	}
	out := RenderItemRow(item, true, 60)
	if !strings.Contains(out, "▶") {
		t.Error("selected row should contain cursor")
	}
	if !strings.Contains(out, "Gmail") {
		t.Error("row should contain item name")
	}
	if !strings.Contains(out, "user@gmail.com") {
		t.Error("row should contain username")
	}
}

func TestRenderItemRowLoginUnselected(t *testing.T) {
	item := bwcmd.Item{
		Type:  bwcmd.ItemTypeLogin,
		Name:  "Gmail",
		Login: &bwcmd.Login{Username: "user@gmail.com"},
	}
	out := RenderItemRow(item, false, 60)
	if strings.Contains(out, "▶") {
		t.Error("unselected row should not contain cursor")
	}
	if !strings.Contains(out, "Gmail") {
		t.Error("row should contain item name")
	}
}

func TestRenderItemRowCard(t *testing.T) {
	item := bwcmd.Item{
		Type: bwcmd.ItemTypeCard,
		Name: "Visa Debit",
		Card: &bwcmd.Card{Number: "4242424242424242"},
	}
	out := RenderItemRow(item, false, 60)
	if !strings.Contains(out, "Visa Debit") {
		t.Error("row should contain card name")
	}
	if !strings.Contains(out, "4242") {
		t.Error("row should contain last 4 digits")
	}
}

func TestRenderItemRowNote(t *testing.T) {
	item := bwcmd.Item{
		Type:  bwcmd.ItemTypeSecureNote,
		Name:  "API Keys",
		Notes: "some secret",
	}
	out := RenderItemRow(item, false, 60)
	if !strings.Contains(out, "API Keys") {
		t.Error("row should contain note name")
	}
}

func TestRenderItemRowTruncation(t *testing.T) {
	item := bwcmd.Item{
		Type: bwcmd.ItemTypeLogin,
		Name: "This is a very long item name that should be truncated",
	}
	out := RenderItemRow(item, false, 30)
	if !strings.Contains(out, "…") {
		t.Error("long name should be truncated with ellipsis")
	}
}
