package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/juthrbog/lazybw/bwcmd"
)

func seedItems() ([]bwcmd.Item, error) {
	sshItem, err := sshKeyItem()
	if err != nil {
		return nil, err
	}

	return []bwcmd.Item{
		{
			Type:  bwcmd.ItemTypeLogin,
			Name:  "GitHub (Personal)",
			Notes: "Personal account",
			Login: &bwcmd.Login{
				Username: "dev@lazybw.dev",
				Password: "S3cureP@ssw0rd!",
				Totp:     "otpauth://totp/GitHub:dev@lazybw.dev?secret=JBSWY3DPEHPK3PXP&issuer=GitHub",
				URIs:     []bwcmd.URI{{URI: "https://github.com/login"}},
			},
		},
		{
			Type: bwcmd.ItemTypeLogin,
			Name: "AWS (Production)",
			Login: &bwcmd.Login{ //nolint:gosec // test seed data, not a real credential
				Username: "admin@lazybw.dev",
				Password: "Kj8#mP2!vR9$xL4w",
				URIs:     []bwcmd.URI{{URI: "https://console.aws.amazon.com"}},
			},
		},
		{
			Type: bwcmd.ItemTypeLogin,
			Name: "GitHub (Work)",
			Login: &bwcmd.Login{
				Username: "jakob@work.dev",
				Password: "W0rkP@ss!2024",
				Totp:     "otpauth://totp/GitHub:jakob@work.dev?secret=HXDMVJECJJWSRB3HWIZR4IFUGFTMXBOZ&issuer=GitHub",
				URIs:     []bwcmd.URI{{URI: "https://github.com/login"}},
			},
		},
		{
			Type: bwcmd.ItemTypeLogin,
			Name: "AWS (Staging)",
			Login: &bwcmd.Login{ //nolint:gosec // test seed data, not a real credential
				Username: "staging@lazybw.dev",
				Password: "StgP@ss#2024",
				URIs:     []bwcmd.URI{{URI: "https://staging.console.aws.amazon.com"}},
			},
		},
		{ //nolint:gosec // test seed data using AWS example key format, not real credentials
			Type:  bwcmd.ItemTypeSecureNote,
			Name:  "AWS Access Key",
			Notes: "AKIAIOSFODNN7EXAMPLE\nwJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\n\nRotate every 90 days.",
			SecureNote: &bwcmd.SecureNote{
				Type: 0,
			},
		},
		{
			Type:  bwcmd.ItemTypeSecureNote,
			Name:  "API Keys",
			Notes: "ANTHROPIC_KEY=sk-ant-api03-test\nOPENAI_KEY=sk-proj-test\nSTRIPE_KEY=sk_test_abc123\n\nRotate quarterly.",
			SecureNote: &bwcmd.SecureNote{
				Type: 0,
			},
		},
		{
			Type: bwcmd.ItemTypeCard,
			Name: "Visa Debit",
			Card: &bwcmd.Card{
				CardholderName: "Test User",
				Number:         "4111111111111111",
				ExpMonth:       "12",
				ExpYear:        "2028",
				Code:           "123",
			},
		},
		{
			Type: bwcmd.ItemTypeIdentity,
			Name: "Personal Identity",
			Identity: &bwcmd.Identity{
				Title:          "Mr",
				FirstName:      "Test",
				MiddleName:     "M",
				LastName:       "User",
				Username:       "testuser",
				Company:        "lazybw Inc",
				SSN:            "123-45-6789",
				PassportNumber: "X12345678",
				LicenseNumber:  "D1234567",
				Email:          "test@lazybw.dev",
				Phone:          "+1-555-0100",
				Address1:       "123 Main St",
				Address2:       "Apt 4B",
				City:           "Portland",
				State:          "OR",
				PostalCode:     "97201",
				Country:        "US",
			},
		},
		sshItem,
	}, nil
}

func sshKeyItem() (bwcmd.Item, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return bwcmd.Item{}, fmt.Errorf("generate ed25519 key: %w", err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return bwcmd.Item{}, fmt.Errorf("ssh public key: %w", err)
	}

	privPEM, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return bwcmd.Item{}, fmt.Errorf("marshal private key: %w", err)
	}

	return bwcmd.Item{
		Type: bwcmd.ItemTypeSSHKey,
		Name: "Dev SSH Key",
		SSHKey: &bwcmd.SSHKey{
			PrivateKey:     string(pem.EncodeToMemory(privPEM)),
			PublicKey:      string(ssh.MarshalAuthorizedKey(sshPub)),
			KeyFingerprint: ssh.FingerprintSHA256(sshPub),
		},
	}, nil
}
