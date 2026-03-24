package bwcmd

import "encoding/json"

// ParseStatus decodes the JSON blob from `bw status` into a VaultStatus.
func ParseStatus(data []byte) (VaultStatus, error) {
	var v VaultStatus
	if err := json.Unmarshal(data, &v); err != nil {
		return VaultStatus{}, err
	}
	return v, nil
}

// ParseItems decodes the JSON array from `bw list items`.
func ParseItems(data []byte) ([]Item, error) {
	var items []Item
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}
