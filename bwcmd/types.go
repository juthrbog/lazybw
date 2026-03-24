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
	ItemTypeSSHKey     ItemType = 5
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

// Identity holds identity-specific fields.
type Identity struct {
	Title          string `json:"title"`
	FirstName      string `json:"firstName"`
	MiddleName     string `json:"middleName"`
	LastName       string `json:"lastName"`
	Username       string `json:"username"`
	Company        string `json:"company"`
	SSN            string `json:"ssn"`
	PassportNumber string `json:"passportNumber"`
	LicenseNumber  string `json:"licenseNumber"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Address1       string `json:"address1"`
	Address2       string `json:"address2"`
	Address3       string `json:"address3"`
	City           string `json:"city"`
	State          string `json:"state"`
	PostalCode     string `json:"postalCode"`
	Country        string `json:"country"`
}

// SSHKey holds SSH key-specific fields.
type SSHKey struct {
	PrivateKey     string `json:"privateKey"`
	PublicKey      string `json:"publicKey"`
	KeyFingerprint string `json:"keyFingerprint"`
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
	Identity       *Identity   `json:"identity,omitempty"`
	SSHKey         *SSHKey     `json:"sshKey,omitempty"`
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
	case ItemTypeIdentity:
		if i.Identity != nil {
			name := strings.TrimSpace(i.Identity.FirstName + " " + i.Identity.LastName)
			if name != "" {
				return name
			}
			if i.Identity.Email != "" {
				return i.Identity.Email
			}
		}
	case ItemTypeSSHKey:
		if i.SSHKey != nil && i.SSHKey.KeyFingerprint != "" {
			return i.SSHKey.KeyFingerprint
		}
	}
	return ""
}
