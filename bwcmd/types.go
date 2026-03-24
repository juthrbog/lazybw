package bwcmd

import "strings"

// VaultStatus mirrors the JSON shape returned by `bw status`.
type VaultStatus struct {
	Status    string `json:"status"`    // "unauthenticated" | "locked" | "unlocked"
	UserEmail string `json:"userEmail"`
	LastSync  string `json:"lastSync"` // RFC3339; empty when unauthenticated
}

// ItemType enumerates Bitwarden item type IDs.
type ItemType int

const (
	ItemTypeLogin      ItemType = 1
	ItemTypeSecureNote ItemType = 2
	ItemTypeCard       ItemType = 3
	ItemTypeIdentity   ItemType = 4
)

// URI represents a single URL entry on a Login item.
type URI struct {
	URI   string `json:"uri"`
	Match *int   `json:"match"` // nil means default match policy
}

// Login holds login-specific fields.
type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Totp     string `json:"totp"`
	URIs     []URI  `json:"uris"`
}

// Card holds card-specific fields.
type Card struct {
	CardholderName string `json:"cardholderName"`
	Number         string `json:"number"`
	ExpMonth       string `json:"expMonth"`
	ExpYear        string `json:"expYear"`
	Code           string `json:"code"`
}

// SecureNote holds secure-note-specific fields.
type SecureNote struct {
	Type int `json:"type"`
}

// Item is the top-level vault item returned by `bw list items`.
type Item struct {
	ID             string      `json:"id"`
	OrganizationID string      `json:"organizationId"`
	FolderID       string      `json:"folderId"`
	Type           ItemType    `json:"type"`
	Name           string      `json:"name"`
	Notes          string      `json:"notes"`
	Login          *Login      `json:"login,omitempty"`
	Card           *Card       `json:"card,omitempty"`
	SecureNote     *SecureNote `json:"secureNote,omitempty"`
}

// FilterValue implements bubbles/list.Item. Used for fuzzy filtering.
func (i Item) FilterValue() string { return i.Name }

// Title implements the default bubbles/list delegate.
func (i Item) Title() string { return i.Name }

// Description implements the default bubbles/list delegate.
func (i Item) Description() string {
	switch i.Type {
	case ItemTypeLogin:
		if i.Login != nil {
			return i.Login.Username
		}
	case ItemTypeCard:
		if i.Card != nil && len(i.Card.Number) >= 4 {
			return "•••• " + i.Card.Number[len(i.Card.Number)-4:]
		}
	case ItemTypeSecureNote:
		if i.Notes != "" {
			first, _, _ := strings.Cut(i.Notes, "\n")
			return first
		}
		return "(note)"
	}
	return ""
}
