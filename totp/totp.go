// Package totp computes TOTP codes locally per RFC 6238.
//
// It accepts a raw base32 secret, an otpauth:// URI or a steam:// key, and
// returns the current code for the active time step. Parsing follows the
// Bitwarden clients, so that a key that works there works here.
package totp

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // SHA1 is required by RFC 6238 (TOTP)
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPeriod = 30
	defaultDigits = 6
	// maxDigits is the largest code length. Bitwarden clamps digits to 10,
	// and 10 digits is also the most that a 31-bit RFC 4226 value can fill.
	// For 10 digits, Bitwarden computes the modulus in 32 bits, which wraps
	// to 1410065408. So its 10-digit codes differ from the RFC value on
	// about a third of the time steps. lazybw keeps the RFC value.
	maxDigits = 10
	// steamDigits is the length of a Steam Guard code.
	steamDigits = 5
)

// AlgorithmSteam marks a Steam Guard key. A Steam code uses HMAC-SHA1 and
// takes its characters from steamChars instead of decimal digits.
const AlgorithmSteam = "STEAM"

const steamChars = "23456789BCDFGHJKMNPQRTVWXY"

// pow10 holds the code modulus for each length from 0 to maxDigits.
var pow10 = [maxDigits + 1]uint64{
	1, 10, 100, 1_000, 10_000, 100_000, 1_000_000,
	10_000_000, 100_000_000, 1_000_000_000, 10_000_000_000,
}

// Params holds the parsed TOTP parameters.
type Params struct {
	Secret    []byte // decoded secret key
	Period    int    // time step in seconds (default 30)
	Digits    int    // code length (default 6)
	Algorithm string // "SHA1", "SHA256", "SHA512", or AlgorithmSteam
}

// Parse extracts TOTP parameters from a raw base32 secret, an otpauth://
// URI or a steam:// key. It returns an error if the secret is empty or
// cannot be decoded.
func Parse(raw string) (Params, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Params{}, errors.New("totp: empty secret")
	}

	p := Params{
		Period:    defaultPeriod,
		Digits:    defaultDigits,
		Algorithm: "SHA1",
	}

	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "otpauth://"):
		return parseURI(raw, p)
	case strings.HasPrefix(lower, "steam://"):
		secret, err := decodeBase32(raw[len("steam://"):])
		if err != nil {
			return Params{}, fmt.Errorf("totp: %w", err)
		}
		p.Secret = secret
		p.Digits = steamDigits
		p.Algorithm = AlgorithmSteam
		return p, nil
	}

	secret, err := decodeBase32(raw)
	if err != nil {
		return Params{}, fmt.Errorf("totp: %w", err)
	}
	p.Secret = secret
	return p, nil
}

// Clear zeros the secret bytes in-place and nils the slice, preventing
// the decoded key material from lingering in memory after use.
func (p *Params) Clear() {
	for i := range p.Secret {
		p.Secret[i] = 0
	}
	p.Secret = nil
}

// Code returns the TOTP code for the current time.
func Code(p Params) string {
	return CodeAt(p, time.Now())
}

// CodeAt returns the TOTP code for the given time.
func CodeAt(p Params, t time.Time) string {
	counter := uint64(t.Unix()) / uint64(validPeriod(p.Period)) //nolint:gosec // Unix timestamps are positive, and validPeriod is at least 1
	return generateCode(p.Secret, counter, p.Digits, p.Algorithm)
}

// SecsLeft returns the number of seconds remaining in the current
// time step.
func SecsLeft(period int) int {
	period = validPeriod(period)
	return period - int(time.Now().Unix()%int64(period))
}

// validPeriod returns period, or the default period when period is not
// positive. Params is exported, so a caller can build it without Parse.
func validPeriod(period int) int {
	if period <= 0 {
		return defaultPeriod
	}
	return period
}

// validDigits returns digits limited to 1..maxDigits. A value below 1
// gives the default length.
func validDigits(digits int) int {
	switch {
	case digits < 1:
		return defaultDigits
	case digits > maxDigits:
		return maxDigits
	}
	return digits
}

func generateCode(secret []byte, counter uint64, digits int, algo string) string {
	// Encode counter as big-endian 8-byte value.
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)

	// HMAC with the configured algorithm. Steam uses SHA1.
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

	if algo == AlgorithmSteam {
		return steamCode(code)
	}

	// Modulo to get the desired number of digits. Integer math keeps the
	// result exact for every length up to maxDigits.
	digits = validDigits(digits)
	return fmt.Sprintf("%0*d", digits, uint64(code)%pow10[digits])
}

// steamCode turns a truncated HMAC value into a Steam Guard code, as the
// Bitwarden clients do.
func steamCode(code uint32) string {
	var b [steamDigits]byte
	for i := range b {
		b[i] = steamChars[code%uint32(len(steamChars))]
		code /= uint32(len(steamChars))
	}
	return string(b[:])
}

func parseURI(raw string, defaults Params) (Params, error) {
	q := queryParams(raw)

	secretStr := q["secret"]
	if secretStr == "" {
		return Params{}, errors.New("totp: URI missing secret parameter")
	}
	secret, err := decodeBase32(secretStr)
	if err != nil {
		return Params{}, fmt.Errorf("totp: %w", err)
	}
	defaults.Secret = secret

	if v := q["period"]; v != "" {
		if period, err := strconv.Atoi(v); err == nil && period > 0 {
			defaults.Period = period
		}
	}
	if v := q["digits"]; v != "" {
		if digits, err := strconv.Atoi(v); err == nil && digits > 0 {
			// Bitwarden treats a larger value as 10.
			defaults.Digits = min(digits, maxDigits)
		}
	}
	if v := q["algorithm"]; v != "" {
		switch strings.ToUpper(v) {
		case "SHA1", "SHA256", "SHA512":
			defaults.Algorithm = strings.ToUpper(v)
		}
	}

	return defaults, nil
}

// queryParams reads the query of an otpauth URI the way the Bitwarden
// clients do. It does not parse the label, so a stray "%" there does no
// harm. Tabs and line breaks are dropped. Keys match without regard to
// case, and a later key replaces an earlier one.
func queryParams(raw string) map[string]string {
	q := make(map[string]string)
	if i := strings.IndexByte(raw, '#'); i >= 0 {
		raw = raw[:i]
	}
	_, query, ok := strings.Cut(raw, "?")
	if !ok {
		return q
	}
	query = strings.Map(func(r rune) rune {
		if r == '\t' || r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, query)
	for _, pair := range strings.Split(query, "&") {
		k, v, _ := strings.Cut(pair, "=")
		q[strings.ToLower(unescapeLenient(k))] = unescapeLenient(v)
	}
	return q
}

// unescapeLenient decodes "+" as a space and each valid %XX escape. Any
// other "%" stays as it is, where net/url would reject the whole value.
func unescapeLenient(s string) string {
	if !strings.ContainsAny(s, "%+") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '+':
			b.WriteByte(' ')
		case c == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]):
			b.WriteByte(unhex(s[i+1])<<4 | unhex(s[i+2]))
			i += 2
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func isHex(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

func unhex(c byte) byte {
	switch {
	case c >= 'a':
		return c - 'a' + 10
	case c >= 'A':
		return c - 'A' + 10
	}
	return c - '0'
}

// base32Values maps each byte to its value in the RFC 4648 base32
// alphabet, or to 0xff for a byte outside the alphabet.
var base32Values = func() [256]byte {
	var t [256]byte
	for i := range t {
		t[i] = 0xff
	}
	for i, c := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567" {
		t[c] = byte(i)
		if c >= 'A' && c <= 'Z' {
			t[c+'a'-'A'] = byte(i)
		}
	}
	return t
}()

// decodeBase32 decodes a base32 secret the way the Bitwarden clients do.
// It ignores case and every byte outside the base32 alphabet, such as
// spaces, hyphens, dots and padding. Trailing bits that do not fill a byte
// are dropped. The output buffer is sized up front, so that no partial
// copy of the secret is left behind by a reallocation.
func decodeBase32(s string) ([]byte, error) {
	out := make([]byte, 0, len(s)*5/8+1)
	var acc uint32
	var bits uint
	for i := 0; i < len(s); i++ {
		v := base32Values[s[i]]
		if v == 0xff {
			continue
		}
		acc = acc<<5 | uint32(v)
		bits += 5
		if bits >= 8 {
			bits -= 8
			out = append(out, byte(acc>>bits)) //nolint:gosec // keeps the low 8 bits on purpose
			acc &= 1<<bits - 1
		}
	}
	if len(out) == 0 {
		return nil, errors.New("invalid base32 secret")
	}
	return out, nil
}
