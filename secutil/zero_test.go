package secutil

import (
	"testing"

	"github.com/juthrbog/lazybw/bwcmd"
)

func TestZeroBytes(t *testing.T) {
	b := []byte{0x41, 0x42, 0x43}
	ZeroBytes(b)
	for i, v := range b {
		if v != 0 {
			t.Errorf("b[%d] = %d, want 0", i, v)
		}
	}
}

func TestZeroBytesNil(t *testing.T) {
	ZeroBytes(nil) // should not panic
}

func TestZeroItem(t *testing.T) {
	item := bwcmd.Item{
		Notes: "secret note",
		Login: &bwcmd.Login{
			Username: "user",
			Password: "hunter2",
			Totp:     "JBSWY3DPEHPK3PXP",
		},
		Card: &bwcmd.Card{
			CardholderName: "John",
			Number:         "4111111111111111",
			Code:           "123",
		},
		Identity: &bwcmd.Identity{
			FirstName:      "John",
			SSN:            "123-45-6789",
			PassportNumber: "AB1234567",
			LicenseNumber:  "DL999",
		},
		SSHKey: &bwcmd.SSHKey{
			PrivateKey:     "-----BEGIN OPENSSH PRIVATE KEY-----",
			PublicKey:      "ssh-ed25519 AAAA...",
			KeyFingerprint: "SHA256:abc",
		},
	}

	ZeroItem(&item)

	// Sensitive fields should be cleared.
	if item.Notes != "" {
		t.Error("Notes not cleared")
	}
	if item.Login.Password != "" {
		t.Error("Login.Password not cleared")
	}
	if item.Login.Totp != "" {
		t.Error("Login.Totp not cleared")
	}
	if item.Card.Number != "" {
		t.Error("Card.Number not cleared")
	}
	if item.Card.Code != "" {
		t.Error("Card.Code not cleared")
	}
	if item.Identity.SSN != "" {
		t.Error("Identity.SSN not cleared")
	}
	if item.Identity.PassportNumber != "" {
		t.Error("Identity.PassportNumber not cleared")
	}
	if item.Identity.LicenseNumber != "" {
		t.Error("Identity.LicenseNumber not cleared")
	}
	if item.SSHKey.PrivateKey != "" {
		t.Error("SSHKey.PrivateKey not cleared")
	}

	// Non-sensitive fields should be preserved.
	if item.Login.Username != "user" {
		t.Error("Login.Username should be preserved")
	}
	if item.Card.CardholderName != "John" {
		t.Error("Card.CardholderName should be preserved")
	}
	if item.Identity.FirstName != "John" {
		t.Error("Identity.FirstName should be preserved")
	}
	if item.SSHKey.PublicKey != "ssh-ed25519 AAAA..." {
		t.Error("SSHKey.PublicKey should be preserved")
	}
}

func TestZeroItemNilSubstructs(t *testing.T) {
	item := bwcmd.Item{Notes: "note"}
	ZeroItem(&item) // should not panic with nil Login/Card/Identity/SSHKey
	if item.Notes != "" {
		t.Error("Notes not cleared")
	}
}

func TestZeroItems(t *testing.T) {
	items := []bwcmd.Item{
		{Login: &bwcmd.Login{Password: "pw1"}},
		{Login: &bwcmd.Login{Password: "pw2"}},
	}
	ZeroItems(items)
	for i, item := range items {
		if item.Login.Password != "" {
			t.Errorf("items[%d].Login.Password not cleared", i)
		}
	}
}

func TestZeroItemsEmpty(t *testing.T) {
	ZeroItems(nil)          // should not panic
	ZeroItems([]bwcmd.Item{}) // should not panic
}
