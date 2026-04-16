package main

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
)

func TestPKCS7Pad(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		blockSize int
		wantLen   int
		wantPad   byte
	}{
		{"exact block", make([]byte, 16), 16, 32, 16},
		{"one short", make([]byte, 15), 16, 16, 1},
		{"one byte", make([]byte, 1), 16, 16, 15},
		{"empty", make([]byte, 0), 16, 16, 16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pkcs7Pad(tt.input, tt.blockSize)
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
			if got[len(got)-1] != tt.wantPad {
				t.Errorf("last byte = %d, want %d", got[len(got)-1], tt.wantPad)
			}
		})
	}
}

func TestMasterKeyDerivation(t *testing.T) {
	// Verify PBKDF2 derivation produces consistent output for known inputs.
	email := "test@example.com"
	password := "password123"
	iterations := 600_000

	salt := []byte(strings.ToLower(email))
	masterKey := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)

	if len(masterKey) != 32 {
		t.Fatalf("masterKey length = %d, want 32", len(masterKey))
	}

	// Derive again — must be deterministic.
	masterKey2 := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)
	if string(masterKey) != string(masterKey2) {
		t.Error("PBKDF2 derivation is not deterministic")
	}
}

func TestMasterPasswordHash(t *testing.T) {
	email := "test@example.com"
	password := "password123"
	iterations := 600_000

	salt := []byte(strings.ToLower(email))
	masterKey := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)
	pwHash := pbkdf2.Key(masterKey, []byte(password), 1, 32, sha256.New)

	b64 := base64.StdEncoding.EncodeToString(pwHash)
	if len(b64) == 0 {
		t.Fatal("empty password hash")
	}

	// Decode back and verify length.
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("decoded hash length = %d, want 32", len(decoded))
	}
}

func TestHKDFExpand(t *testing.T) {
	// Use a known 32-byte key to verify HKDF-Expand produces 32-byte outputs.
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	encKey := make([]byte, 32)
	if _, err := hkdf.Expand(sha256.New, masterKey, []byte("enc")).Read(encKey); err != nil {
		t.Fatalf("hkdf expand enc: %v", err)
	}

	macKey := make([]byte, 32)
	if _, err := hkdf.Expand(sha256.New, masterKey, []byte("mac")).Read(macKey); err != nil {
		t.Fatalf("hkdf expand mac: %v", err)
	}

	if len(encKey) != 32 || len(macKey) != 32 {
		t.Fatal("unexpected key lengths")
	}

	// enc and mac keys must differ.
	if string(encKey) == string(macKey) {
		t.Error("enc and mac keys should differ")
	}

	// Must be deterministic.
	encKey2 := make([]byte, 32)
	if _, err := hkdf.Expand(sha256.New, masterKey, []byte("enc")).Read(encKey2); err != nil {
		t.Fatalf("hkdf expand enc (2nd): %v", err)
	}
	if string(encKey) != string(encKey2) {
		t.Error("HKDF-Expand is not deterministic")
	}
}

func TestEncryptCBC(t *testing.T) {
	encKey := make([]byte, 32)
	macKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i)
		macKey[i] = byte(i + 32)
	}

	plaintext := []byte("hello world, this is a test of AES-CBC encryption")
	cs, err := encryptCBC(encKey, macKey, plaintext)
	if err != nil {
		t.Fatalf("encryptCBC: %v", err)
	}

	// Verify CipherString format: 2.<iv>|<ct>|<mac>
	if !strings.HasPrefix(cs, "2.") {
		t.Errorf("CipherString should start with '2.', got: %s", cs[:10])
	}

	parts := strings.SplitN(cs[2:], "|", 3)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts separated by |, got %d", len(parts))
	}

	// IV should be 16 bytes = 24 base64 chars.
	iv, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode iv: %v", err)
	}
	if len(iv) != 16 {
		t.Errorf("iv length = %d, want 16", len(iv))
	}

	// Ciphertext should be a multiple of 16 bytes.
	ct, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode ct: %v", err)
	}
	if len(ct)%16 != 0 {
		t.Errorf("ciphertext length %d is not a multiple of 16", len(ct))
	}

	// MAC should be 32 bytes (SHA-256).
	mac, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode mac: %v", err)
	}
	if len(mac) != 32 {
		t.Errorf("mac length = %d, want 32", len(mac))
	}
}

func TestDeriveRegKeys(t *testing.T) {
	keys, err := deriveRegKeys("test@lazybw.dev", "master-password-for-dev", 600_000)
	if err != nil {
		t.Fatalf("deriveRegKeys: %v", err)
	}

	if keys.MasterPasswordHash == "" {
		t.Error("empty MasterPasswordHash")
	}
	if !strings.HasPrefix(keys.EncryptedSymKey, "2.") {
		t.Error("EncryptedSymKey should be a CipherString")
	}
	if !strings.HasPrefix(keys.EncryptedPrivateKey, "2.") {
		t.Error("EncryptedPrivateKey should be a CipherString")
	}
	if keys.PublicKeyB64 == "" {
		t.Error("empty PublicKeyB64")
	}

	// Public key should decode to valid DER.
	der, err := base64.StdEncoding.DecodeString(keys.PublicKeyB64)
	if err != nil {
		t.Fatalf("decode public key: %v", err)
	}
	if len(der) < 200 {
		t.Errorf("public key DER seems too short: %d bytes", len(der))
	}
}
