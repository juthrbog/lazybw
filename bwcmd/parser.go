package bwcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// utf8BOM is the byte order mark that can start UTF-8 text.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// trimOutput removes white space and a leading UTF-8 byte order mark from
// bw output.
func trimOutput(data []byte) []byte {
	data = bytes.TrimSpace(data)
	data = bytes.TrimPrefix(data, utf8BOM)
	return bytes.TrimSpace(data)
}

// ParseStatus decodes the JSON blob from `bw status` into a VaultStatus.
// It returns an error for "null" and for a status other than
// "unauthenticated", "locked" or "unlocked".
func ParseStatus(data []byte) (VaultStatus, error) {
	var v *VaultStatus
	if err := json.Unmarshal(trimOutput(data), &v); err != nil {
		return VaultStatus{}, fmt.Errorf("parse bw status output: %w", err)
	}
	if v == nil {
		return VaultStatus{}, errors.New("parse bw status output: got null")
	}
	switch v.Status {
	case "unauthenticated", "locked", "unlocked":
		return *v, nil
	}
	return VaultStatus{}, fmt.Errorf("parse bw status output: unknown status %q", v.Status)
}

// ParseItems decodes the JSON array from `bw list items`.
func ParseItems(data []byte) ([]Item, error) {
	var items []Item
	if err := json.Unmarshal(trimOutput(data), &items); err != nil {
		return nil, fmt.Errorf("parse bw list items output: %w", err)
	}
	return items, nil
}
