package totp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // SHA1 is required by RFC 6238 (TOTP)
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"testing"
	"time"
)

// RFC 6238 §B.  Test Vectors
// The test token shared secret uses the ASCII string:
//   "12345678901234567890" (20 bytes for SHA1)
//   "12345678901234567890123456789012" (32 bytes for SHA256)
//   "1234567890123456789012345678901234567890123456789012345678901234" (64 bytes for SHA512)
//
// We use CodeAt with raw secret bytes (bypassing base32) to match the spec.

func TestGenerateCode_RFC6238(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		algo   string
		digits int
		time   int64
		want   string
	}{
		// SHA1 test vectors from RFC 6238 appendix B.
		{"SHA1/59", "12345678901234567890", "SHA1", 8, 59, "94287082"},
		{"SHA1/1111111109", "12345678901234567890", "SHA1", 8, 1111111109, "07081804"},
		{"SHA1/1111111111", "12345678901234567890", "SHA1", 8, 1111111111, "14050471"},
		{"SHA1/1234567890", "12345678901234567890", "SHA1", 8, 1234567890, "89005924"},
		{"SHA1/2000000000", "12345678901234567890", "SHA1", 8, 2000000000, "69279037"},
		{"SHA1/20000000000", "12345678901234567890", "SHA1", 8, 20000000000, "65353130"},

		// SHA256 test vectors.
		{"SHA256/59", "12345678901234567890123456789012", "SHA256", 8, 59, "46119246"},
		{"SHA256/1111111109", "12345678901234567890123456789012", "SHA256", 8, 1111111109, "68084774"},
		{"SHA256/1111111111", "12345678901234567890123456789012", "SHA256", 8, 1111111111, "67062674"},
		{"SHA256/1234567890", "12345678901234567890123456789012", "SHA256", 8, 1234567890, "91819424"},
		{"SHA256/2000000000", "12345678901234567890123456789012", "SHA256", 8, 2000000000, "90698825"},
		{"SHA256/20000000000", "12345678901234567890123456789012", "SHA256", 8, 20000000000, "77737706"},

		// SHA512 test vectors.
		{"SHA512/59", "1234567890123456789012345678901234567890123456789012345678901234", "SHA512", 8, 59, "90693936"},
		{"SHA512/1111111109", "1234567890123456789012345678901234567890123456789012345678901234", "SHA512", 8, 1111111109, "25091201"},
		{"SHA512/1111111111", "1234567890123456789012345678901234567890123456789012345678901234", "SHA512", 8, 1111111111, "99943326"},
		{"SHA512/1234567890", "1234567890123456789012345678901234567890123456789012345678901234", "SHA512", 8, 1234567890, "93441116"},
		{"SHA512/2000000000", "1234567890123456789012345678901234567890123456789012345678901234", "SHA512", 8, 2000000000, "38618901"},
		{"SHA512/20000000000", "1234567890123456789012345678901234567890123456789012345678901234", "SHA512", 8, 20000000000, "47863826"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateCode([]byte(tt.secret), uint64(tt.time)/30, tt.digits, tt.algo) //nolint:gosec // test constants
			if got != tt.want {
				t.Errorf("generateCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParse_RawBase32(t *testing.T) {
	// "JBSWY3DPEHPK3PXP" decodes to "Hello!ÞĐ÷" (just verifying it parses)
	p, err := Parse("JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	if p.Period != 30 {
		t.Errorf("Period = %d, want 30", p.Period)
	}
	if p.Digits != 6 {
		t.Errorf("Digits = %d, want 6", p.Digits)
	}
	if p.Algorithm != "SHA1" {
		t.Errorf("Algorithm = %q, want SHA1", p.Algorithm)
	}
}

func TestParse_RawBase32Lowercase(t *testing.T) {
	p, err := Parse("jbswy3dpehpk3pxp")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Secret) == 0 {
		t.Error("expected non-empty secret")
	}
}

func TestParse_RawBase32WithSpaces(t *testing.T) {
	p, err := Parse("JBSW Y3DP EHPK 3PXP")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Secret) == 0 {
		t.Error("expected non-empty secret")
	}
}

func TestParse_OTPAuthURI(t *testing.T) {
	uri := "otpauth://totp/Example:alice@example.com?secret=JBSWY3DPEHPK3PXP&period=60&digits=8&algorithm=SHA256"
	p, err := Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
	if p.Period != 60 {
		t.Errorf("Period = %d, want 60", p.Period)
	}
	if p.Digits != 8 {
		t.Errorf("Digits = %d, want 8", p.Digits)
	}
	if p.Algorithm != "SHA256" {
		t.Errorf("Algorithm = %q, want SHA256", p.Algorithm)
	}
}

func TestParse_OTPAuthDefaults(t *testing.T) {
	uri := "otpauth://totp/Test?secret=JBSWY3DPEHPK3PXP"
	p, err := Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
	if p.Period != 30 || p.Digits != 6 || p.Algorithm != "SHA1" {
		t.Errorf("unexpected defaults: period=%d digits=%d algo=%s", p.Period, p.Digits, p.Algorithm)
	}
}

func TestParse_Empty(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Error("expected error for empty secret")
	}
}

func TestParse_OTPAuthMissingSecret(t *testing.T) {
	_, err := Parse("otpauth://totp/Test?period=30")
	if err == nil {
		t.Error("expected error for missing secret")
	}
}

func TestCodeAt_Deterministic(t *testing.T) {
	p, err := Parse("JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	ts := time.Unix(1234567890, 0)
	a := CodeAt(p, ts)
	b := CodeAt(p, ts)
	if a != b {
		t.Errorf("CodeAt not deterministic: %q != %q", a, b)
	}
	if len(a) != 6 {
		t.Errorf("expected 6 digits, got %d", len(a))
	}
}

func TestParams_Clear(t *testing.T) {
	p, err := Parse("JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Secret) == 0 {
		t.Fatal("Secret should be non-empty before Clear")
	}

	// Keep a reference to the original backing array so we can verify
	// the bytes were actually zeroed in-place.
	orig := p.Secret

	p.Clear()

	if p.Secret != nil {
		t.Error("Secret should be nil after Clear")
	}
	for i, b := range orig {
		if b != 0 {
			t.Errorf("orig[%d] = %d, want 0 — backing array not zeroed", i, b)
		}
	}
}

func TestSecsLeft(t *testing.T) {
	s := SecsLeft(30)
	if s < 1 || s > 30 {
		t.Errorf("SecsLeft(30) = %d, want 1-30", s)
	}
}

func TestParse_Steam(t *testing.T) {
	// Test vector from the Bitwarden SDK (crates/bitwarden-vault/src/totp.rs).
	p, err := Parse("steam://HXDMVJECJJWSRB3HWIZR4IFUGFTMXBOZ")
	if err != nil {
		t.Fatal(err)
	}
	if p.Algorithm != AlgorithmSteam || p.Digits != 5 || p.Period != 30 {
		t.Errorf("Params = %+v, want Steam, 5 digits, 30s", p)
	}
	at := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := CodeAt(p, at); got != "7W6CJ" {
		t.Errorf("CodeAt = %q, want 7W6CJ", got)
	}
}

func TestParse_LenientBase32(t *testing.T) {
	clean, err := Parse("JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		"JBSW\tY3DP\tEHPK\t3PXP",
		"JBSW.Y3DP.EHPK.3PXP",
		"JBSWY3DPEHPK3PXP========",
		"jbsw_y3dp_ehpk_3pxp",
		"otpauth://totp/x?SECRET=JBSWY3DPEHPK3PXP",
		"otpauth://totp/x?Secret=JBSWY3DPEHPK3PXP&Digits=8",
	} {
		p, err := Parse(raw)
		if err != nil {
			t.Errorf("Parse(%q): %v", raw, err)
			continue
		}
		if !bytes.Equal(p.Secret, clean.Secret) {
			t.Errorf("Parse(%q) secret = %x, want %x", raw, p.Secret, clean.Secret)
		}
	}
}

func TestParse_InvalidSecret(t *testing.T) {
	for _, raw := range []string{"!!!!", "otpauth://totp/x?secret=0189", "steam://"} {
		if _, err := Parse(raw); err == nil {
			t.Errorf("Parse(%q) returned no error", raw)
		}
	}
}

func TestParse_DigitsClamped(t *testing.T) {
	p, err := Parse("otpauth://totp/x?secret=JBSWY3DPEHPK3PXP&digits=20")
	if err != nil {
		t.Fatal(err)
	}
	if p.Digits != 10 {
		t.Errorf("Digits = %d, want 10", p.Digits)
	}
}

func TestCodeAt_NoPanicForBadParams(t *testing.T) {
	secret := []byte("12345678901234567890")
	for _, d := range []int{-1, 0, 19, 20, 100} {
		for _, period := range []int{-30, 0, 30} {
			p := Params{Secret: secret, Period: period, Digits: d, Algorithm: "SHA1"}
			if got := CodeAt(p, time.Unix(59, 0)); got == "" {
				t.Errorf("CodeAt(digits=%d, period=%d) = empty", d, period)
			}
		}
	}
	if s := SecsLeft(0); s < 1 || s > 30 {
		t.Errorf("SecsLeft(0) = %d, want 1-30", s)
	}
}

// referenceCode computes an RFC 4226 code with uint64 math only.
func referenceCode(secret []byte, counter uint64, digits int) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, secret)
	mac.Write(buf[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	v := uint64(binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff)
	mod := uint64(1)
	for i := 0; i < digits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", digits, v%mod)
}

func TestGenerateCode_MatchesReferenceForAllLengths(t *testing.T) {
	secret := []byte("12345678901234567890")
	for d := 1; d <= 10; d++ {
		for c := uint64(0); c < 2000; c++ {
			if got, want := generateCode(secret, c, d, "SHA1"), referenceCode(secret, c, d); got != want {
				t.Fatalf("digits=%d counter=%d: got %q, want %q", d, c, got, want)
			}
		}
	}
}

func TestParse_RFC6238VectorsThroughURI(t *testing.T) {
	secret := base32.StdEncoding.EncodeToString([]byte("12345678901234567890"))
	p, err := Parse("otpauth://totp/x?secret=" + secret + "&digits=8")
	if err != nil {
		t.Fatal(err)
	}
	for ts, want := range map[int64]string{
		59:          "94287082",
		1111111109:  "07081804",
		1111111111:  "14050471",
		1234567890:  "89005924",
		2000000000:  "69279037",
		20000000000: "65353130",
	} {
		if got := CodeAt(p, time.Unix(ts, 0)); got != want {
			t.Errorf("CodeAt(%d) = %q, want %q", ts, got, want)
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"JBSWY3DPEHPK3PXP",
		"steam://HXDMVJECJJWSRB3HWIZR4IFUGFTMXBOZ",
		"otpauth://totp/x?secret=JBSWY3DPEHPK3PXP&digits=20&period=0",
		"otpauth://totp/x?secret=JBSWY3DPEHPK3PXP&algorithm=sha512&digits=-3",
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		p, err := Parse(raw)
		if err != nil {
			return
		}
		code := CodeAt(p, time.Unix(1234567890, 0))
		if code == "" {
			t.Errorf("Parse(%q) gave an empty code", raw)
		}
	})
}

func TestParse_DuplicateKeysLastWins(t *testing.T) {
	// Bitwarden reads the query in order, and a later key wins.
	for i := 0; i < 50; i++ {
		p, err := Parse("otpauth://totp/x?secret=JBSWY3DPEHPK3PXP&digits=8&Digits=6")
		if err != nil {
			t.Fatal(err)
		}
		if p.Digits != 6 {
			t.Fatalf("Digits = %d, want 6 from the last key", p.Digits)
		}
	}
	p, err := Parse("otpauth://totp/x?SECRET=GEZDGNBVGY3TQOJQ&secret=JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	clean, _ := Parse("JBSWY3DPEHPK3PXP")
	if !bytes.Equal(p.Secret, clean.Secret) {
		t.Error("the last secret key must win")
	}
}

func TestParse_LenientURI(t *testing.T) {
	clean, err := Parse("JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		"otpauth://totp/100%?secret=JBSWY3DPEHPK3PXP",
		"otpauth://totp/a%zz?secret=JBSWY3DPEHPK3PXP",
		"otpauth://totp/x?secret=JBSW%Y3DPEHPK3PXP",
		"otpauth://totp/x?secret=JBSW;Y3DPEHPK3PXP",
		"otpauth://totp/x?secret=JBSW%59%33DPEHPK3PXP",
		"otpauth://totp/x?secret=JBSWY3DP\nEHPK3PXP",
		"otpauth://totp/x?issuer=a+b&secret=JBSWY3DPEHPK3PXP#frag",
	} {
		p, err := Parse(raw)
		if err != nil {
			t.Errorf("Parse(%q): %v", raw, err)
			continue
		}
		if !bytes.Equal(p.Secret, clean.Secret) {
			t.Errorf("Parse(%q) secret = %x, want %x", raw, p.Secret, clean.Secret)
		}
	}
}
