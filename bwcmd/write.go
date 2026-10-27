package bwcmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// errNoItemID is the error for a bw create that exits 0 but prints no item.
var errNoItemID = errors.New("bw did not return the new item")

// CreateResult is returned by CreateItem. ID is the ID of the new item.
// Epoch is the session epoch that was passed to CreateItem.
type CreateResult struct {
	ID    string
	Err   error
	Epoch uint64
}

// GetItemResult is returned by GetItem. Item is the item JSON as bw prints
// it, with each number as a json.Number, so that bw edit item gets back
// each key that lazybw does not change.
type GetItemResult struct {
	ID    string
	Item  map[string]any
	Err   error
	Epoch uint64
}

// EditResult is returned by EditItem. ID is the ID of the item.
type EditResult struct {
	ID    string
	Err   error
	Epoch uint64
}

// NewItemJSON returns the item JSON for a new item of type t, as bw create
// item takes it: no organization, no folder, no favorite and no master
// password re-prompt. The form sets the name and the fields of the type.
func NewItemJSON(t ItemType) map[string]any {
	item := map[string]any{
		"organizationId": nil,
		"collectionIds":  nil,
		"folderId":       nil,
		"type":           int(t),
		"name":           "",
		"notes":          nil,
		"favorite":       false,
		"fields":         []any{},
		"login":          nil,
		"secureNote":     nil,
		"card":           nil,
		"identity":       nil,
		"reprompt":       RepromptNone,
	}
	switch t {
	case ItemTypeLogin:
		item["login"] = map[string]any{
			"uris":     []any{},
			"username": nil,
			"password": nil,
			"totp":     nil,
		}
	case ItemTypeSecureNote:
		// 0 is the only type of secure note, "generic".
		item["secureNote"] = map[string]any{"type": 0}
	}
	return item
}

// CreateItem runs `bw create item` and returns a CreateResult tagged with
// epoch. The item JSON goes to bw base64-encoded on stdin, never in args,
// which ps and /proc/PID/cmdline show to other users. The debug log hides
// each value of private, so the caller passes the field values there.
func CreateItem(token string, epoch uint64, item map[string]any, private []string) tea.Cmd {
	return func() tea.Msg {
		input, err := encodeItem(item)
		if err != nil {
			return CreateResult{Err: err, Epoch: epoch}
		}
		defer clear(input)
		out, err := execBwInput(longTimeout, token, "", input, private, "create", "item")
		if err != nil {
			return CreateResult{Err: err, Epoch: epoch}
		}
		id, err := itemID(out)
		return CreateResult{ID: id, Err: err, Epoch: epoch}
	}
}

// encodeItem returns item as base64-encoded JSON, the input format of bw
// create and bw edit. It zeros the JSON after it encodes it.
func encodeItem(item map[string]any) ([]byte, error) {
	raw, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("encode the item: %w", err)
	}
	defer clear(raw)
	out := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
	base64.StdEncoding.Encode(out, raw)
	return out, nil
}

// itemID returns the ID of the item JSON that bw create prints, and then
// zeros out. The JSON holds the decrypted item.
func itemID(out []byte) (string, error) {
	defer clear(out)
	var v struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return "", fmt.Errorf("parse bw create output: %w", err)
	}
	if v.ID == "" {
		return "", errNoItemID
	}
	return v.ID, nil
}

// GetItem runs `bw get item <id>` and returns a GetItemResult tagged with
// epoch. The edit form starts from this fresh copy of the item, not from
// the list, so that it keeps each key of the item, also the keys that
// bwcmd.Item does not model.
func GetItem(token string, epoch uint64, id string) tea.Cmd {
	return func() tea.Msg {
		out, err := execBw(longTimeout, token, "", "get", "item", id)
		if err != nil {
			return GetItemResult{ID: id, Err: err, Epoch: epoch}
		}
		item, err := decodeItem(out)
		return GetItemResult{ID: id, Item: item, Err: err, Epoch: epoch}
	}
}

// decodeItem decodes the item JSON that bw get item prints, and then zeros
// out. Numbers stay json.Number values, so that they encode as bw printed
// them.
func decodeItem(out []byte) (map[string]any, error) {
	defer clear(out)
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.UseNumber()
	var item map[string]any
	if err := dec.Decode(&item); err != nil {
		return nil, fmt.Errorf("parse bw get item output: %w", err)
	}
	if item == nil {
		return nil, errors.New("parse bw get item output: got null")
	}
	return item, nil
}

// EditItem runs `bw edit item <id>` and returns an EditResult tagged with
// epoch. As in CreateItem, the item JSON goes to bw on stdin, and the debug
// log hides each value of private.
func EditItem(token string, epoch uint64, id string, item map[string]any, private []string) tea.Cmd {
	return func() tea.Msg {
		input, err := encodeItem(item)
		if err != nil {
			return EditResult{ID: id, Err: err, Epoch: epoch}
		}
		defer clear(input)
		out, err := execBwInput(longTimeout, token, "", input, private, "edit", "item", id)
		clear(out)
		return EditResult{ID: id, Err: err, Epoch: epoch}
	}
}
