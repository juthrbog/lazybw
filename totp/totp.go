// Package totp computes TOTP codes locally per RFC 6238.
//
// It accepts either a raw base32 secret or an otpauth:// URI and
// returns the current code for the active time step.
package totp

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // SHA1 is required by RFC 6238 (TOTP)
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"hash"
	"math"
	"net/url"
	"strings"
	"time"
)

// Params holds the parsed TOTP parameters.
type Params struct {
	Secret    []byte // decoded secret key
	Period    int    // time step in seconds (default 30)
	Digits    int    // code length (default 6)
	Algorithm string // "SHA1", "SHA256", or "SHA512"
}

// Parse extracts TOTP parameters from a raw base32 secret or an
// otpauth:// URI. It returns an error if the secret is empty or
// cannot be decoded.
func Parse(raw string) (Params, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Params{}, fmt.Errorf("totp: empty secret")
	}

	p := Params{
		Period:    30,
		Digits:    6,
		Algorithm: "SHA1",
	}

	if strings.HasPrefix(strings.ToLower(raw), "otpauth://") {
		return parseURI(raw, p)
	}

	secret, err := decodeBase32(raw)
	if err != nil {
		return Params{}, fmt.Errorf("totp: %w", err)
	}
	p.Secret = secret
	return p, nil
}

// Code returns the TOTP code for the current time.
func Code(p Params) string {
	return CodeAt(p, time.Now())
}

// CodeAt returns the TOTP code for the given time.
func CodeAt(p Params, t time.Time) string {
	counter := uint64(t.Unix()) / uint64(p.Period) //nolint:gosec // Unix timestamps are positive
	return generateCode(p.Secret, counter, p.Digits, p.Algorithm)
}

// SecsLeft returns the number of seconds remaining in the current
// time step.
func SecsLeft(period int) int {
	return period - int(time.Now().Unix()%int64(period))
}

func generateCode(secret []byte, counter uint64, digits int, algo string) string {
	// Encode counter as big-endian 8-byte value.
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)

	// HMAC with the configured algorithm.
	var h func() hash.Hash
	switch algo {
	case "SHA256":
		h = sha256.New
	case "SHA512":
		h = sha512.New
	default:
		h = sha1.New
	}
	mac := hmac.New(h, secret)
	mac.Write(buf[:])
	sum := mac.Sum(nil)

	// Dynamic truncation (RFC 4226 §5.4).
	offset := sum[len(sum)-1] & 0x0f
	code := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff

	// Modulo to get the desired number of digits.
	mod := uint32(math.Pow10(digits))
	return fmt.Sprintf("%0*d", digits, code%mod)
}

func parseURI(raw string, defaults Params) (Params, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Params{}, fmt.Errorf("totp: invalid URI: %w", err)
	}

	q := u.Query()

	secretStr := q.Get("secret")
	if secretStr == "" {
		return Params{}, fmt.Errorf("totp: URI missing secret parameter")
	}
	secret, err := decodeBase32(secretStr)
	if err != nil {
		return Params{}, fmt.Errorf("totp: %w", err)
	}
	defaults.Secret = secret

	if v := q.Get("period"); v != "" {
		var period int
		if _, err := fmt.Sscanf(v, "%d", &period); err == nil && period > 0 {
			defaults.Period = period
		}
	}
	if v := q.Get("digits"); v != "" {
		var digits int
		if _, err := fmt.Sscanf(v, "%d", &digits); err == nil && digits > 0 {
			defaults.Digits = digits
		}
	}
	if v := q.Get("algorithm"); v != "" {
		switch strings.ToUpper(v) {
		case "SHA1", "SHA256", "SHA512":
			defaults.Algorithm = strings.ToUpper(v)
		}
	}

	return defaults, nil
}

// decodeBase32 decodes a base32 string, normalizing for common
// variations: lowercase, spaces, hyphens, and missing padding.
func decodeBase32(s string) ([]byte, error) {
	s = strings.ToUpper(s)
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' {
			return -1 // strip
		}
		return r
	}, s)

	// Add padding if needed.
	if pad := len(s) % 8; pad != 0 {
		s += strings.Repeat("=", 8-pad)
	}

	b, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid base32 secret: %w", err)
	}
	return b, nil
}
