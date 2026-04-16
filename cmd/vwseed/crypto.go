package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
)

// registrationKeys holds all derived and generated keys needed for
// the Bitwarden /api/accounts/register endpoint.
type registrationKeys struct {
	MasterPasswordHash  string // base64-encoded hash sent to server
	EncryptedSymKey     string // CipherString: encrypted 64-byte symmetric key
	PublicKeyB64        string // base64 DER-encoded RSA public key
	EncryptedPrivateKey string // CipherString: RSA private key encrypted with symmetric key
}

// deriveRegKeys performs the full Bitwarden registration key derivation:
//
//  1. Master key = PBKDF2-SHA256(password, lowercase(email), iterations, 32)
//  2. Password hash = base64(PBKDF2-SHA256(masterKey, password, 1, 32))
//  3. Stretched key = HKDF-Expand(masterKey, "enc", 32) || HKDF-Expand(masterKey, "mac", 32)
//  4. Symmetric key = random 64 bytes, AES-256-CBC encrypted with stretched key
//  5. RSA 2048 keypair, private key encrypted with symmetric key
func deriveRegKeys(email, password string, iterations int) (*registrationKeys, error) {
	salt := []byte(strings.ToLower(email))

	// Step 1: master key.
	masterKey := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)

	// Step 2: master password hash (proves knowledge of password to server).
	pwHash := pbkdf2.Key(masterKey, []byte(password), 1, 32, sha256.New)
	pwHashB64 := base64.StdEncoding.EncodeToString(pwHash)

	// Step 3: stretch master key via HKDF-Expand (NOT Extract+Expand).
	encKey := make([]byte, 32)
	if _, err := hkdf.Expand(sha256.New, masterKey, []byte("enc")).Read(encKey); err != nil {
		return nil, fmt.Errorf("hkdf expand enc: %w", err)
	}
	macKey := make([]byte, 32)
	if _, err := hkdf.Expand(sha256.New, masterKey, []byte("mac")).Read(macKey); err != nil {
		return nil, fmt.Errorf("hkdf expand mac: %w", err)
	}

	// Step 4: generate random 64-byte symmetric key and encrypt it.
	symKey := make([]byte, 64)
	if _, err := rand.Read(symKey); err != nil {
		return nil, fmt.Errorf("generate symmetric key: %w", err)
	}
	encSymKey, err := encryptCBC(encKey, macKey, symKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt symmetric key: %w", err)
	}

	// Step 5: generate RSA 2048 keypair.
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}

	pubDER, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}

	// Encrypt private key with the symmetric key (first 32 = enc, last 32 = mac).
	encPriv, err := encryptCBC(symKey[:32], symKey[32:], privDER)
	if err != nil {
		return nil, fmt.Errorf("encrypt private key: %w", err)
	}

	return &registrationKeys{
		MasterPasswordHash:  pwHashB64,
		EncryptedSymKey:     encSymKey,
		PublicKeyB64:        base64.StdEncoding.EncodeToString(pubDER),
		EncryptedPrivateKey: encPriv,
	}, nil
}

// encryptCBC encrypts plaintext with AES-256-CBC and returns a Bitwarden
// CipherString in the format: 2.<iv>|<ciphertext>|<mac>
// where type 2 = AesCbc256_HmacSha256_B64.
func encryptCBC(encKey, macKey, plaintext []byte) (string, error) {
	padded := pkcs7Pad(plaintext, aes.BlockSize)

	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return "", err
	}

	ct := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, padded)

	// HMAC-SHA256 over iv || ciphertext.
	mac := hmac.New(sha256.New, macKey)
	mac.Write(iv)
	mac.Write(ct)
	macBytes := mac.Sum(nil)

	return fmt.Sprintf("2.%s|%s|%s",
		base64.StdEncoding.EncodeToString(iv),
		base64.StdEncoding.EncodeToString(ct),
		base64.StdEncoding.EncodeToString(macBytes),
	), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - (len(data) % blockSize)
	padded := make([]byte, len(data)+padLen)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padLen) //nolint:gosec // padLen is always 1-16 for AES
	}
	return padded
}
