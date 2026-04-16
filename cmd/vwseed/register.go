package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type registerRequest struct {
	Email              string      `json:"email"`
	Name               string      `json:"name"`
	MasterPasswordHash string      `json:"masterPasswordHash"`
	MasterPasswordHint string      `json:"masterPasswordHint"`
	Key                string      `json:"key"`
	Keys               registerRSA `json:"keys"`
	Kdf                int         `json:"kdf"`
	KdfIterations      int         `json:"kdfIterations"`
}

type registerRSA struct {
	PublicKey           string `json:"publicKey"`
	EncryptedPrivateKey string `json:"encryptedPrivateKey"`
}

// registerAccount creates a Bitwarden account on the Vaultwarden instance.
// Returns nil if the account was created or already exists (HTTP 409).
func registerAccount(serverURL, email, password string, iterations int) error {
	keys, err := deriveRegKeys(email, password, iterations)
	if err != nil {
		return fmt.Errorf("derive keys: %w", err)
	}

	body := registerRequest{
		Email:              email,
		Name:               "Test User",
		MasterPasswordHash: keys.MasterPasswordHash,
		Key:                keys.EncryptedSymKey,
		Keys: registerRSA{
			PublicKey:           keys.PublicKeyB64,
			EncryptedPrivateKey: keys.EncryptedPrivateKey,
		},
		Kdf:           0, // PBKDF2
		KdfIterations: iterations,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	resp, err := httpClient.Post(serverURL+"/identity/accounts/register", "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("POST register: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest, http.StatusConflict:
		// Account already exists — Vaultwarden returns 400 with
		// "user already exists", not 409. Both are safe to ignore.
		return nil
	default:
		var errBody bytes.Buffer
		if _, err := errBody.ReadFrom(resp.Body); err != nil {
			return fmt.Errorf("register returned HTTP %d (failed to read body: %w)", resp.StatusCode, err)
		}
		return fmt.Errorf("register returned HTTP %d: %s", resp.StatusCode, errBody.String())
	}
}
