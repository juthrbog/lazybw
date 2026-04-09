package totp

import (
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
