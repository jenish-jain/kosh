package share

import (
	"strings"
	"testing"
	"time"
)

// RFC 6238 Appendix B test secret ("12345678901234567890", SHA-1).
var rfcSecret = b32.EncodeToString([]byte("12345678901234567890"))

func TestVerifyTOTP_RFC6238Vectors(t *testing.T) {
	// Last 6 digits of the RFC's 8-digit SHA-1 vectors.
	cases := []struct {
		unix int64
		code string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1234567890, "005924"},
		{2000000000, "279037"},
	}
	for _, c := range cases {
		if _, ok := VerifyTOTP(rfcSecret, c.code, time.Unix(c.unix, 0), 0); !ok {
			t.Errorf("t=%d: expected %s to verify", c.unix, c.code)
		}
	}
}

func TestVerifyTOTP_AcceptsOneStepDrift(t *testing.T) {
	now := time.Unix(1234567890, 0)
	if _, ok := VerifyTOTP(rfcSecret, "005924", now.Add(30*time.Second), 0); !ok {
		t.Error("code from previous step should be accepted")
	}
	if _, ok := VerifyTOTP(rfcSecret, "005924", now.Add(-30*time.Second), 0); !ok {
		t.Error("code from next step should be accepted")
	}
	if _, ok := VerifyTOTP(rfcSecret, "005924", now.Add(90*time.Second), 0); ok {
		t.Error("code three steps old should be rejected")
	}
}

func TestVerifyTOTP_RejectsReplay(t *testing.T) {
	now := time.Unix(1234567890, 0)
	step, ok := VerifyTOTP(rfcSecret, "005924", now, 0)
	if !ok {
		t.Fatal("first use should verify")
	}
	if _, ok := VerifyTOTP(rfcSecret, "005924", now, step); ok {
		t.Error("reusing a code at or before lastStep must be rejected")
	}
}

func TestVerifyTOTP_RejectsMalformed(t *testing.T) {
	now := time.Unix(59, 0)
	for _, code := range []string{"", "28708", "2870820", "abcdef"} {
		if _, ok := VerifyTOTP(rfcSecret, code, now, 0); ok {
			t.Errorf("%q should not verify", code)
		}
	}
}

func TestProvisioningURI(t *testing.T) {
	uri := ProvisioningURI("Kosh", "Mom", "ABC")
	for _, want := range []string{"otpauth://totp/Kosh:Mom?", "secret=ABC", "issuer=Kosh"} {
		if !strings.Contains(uri, want) {
			t.Errorf("%s missing %q", uri, want)
		}
	}
}

func TestSealer_RoundTripAndTamper(t *testing.T) {
	s, err := NewSealer(strings.Repeat("k", MinKeyLength))
	if err != nil {
		t.Fatal(err)
	}
	enc, err := s.Seal("JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(enc, "JBSWY3DPEHPK3PXP") {
		t.Fatal("ciphertext leaks plaintext")
	}
	got, err := s.Open(enc)
	if err != nil || got != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("round trip: got %q, %v", got, err)
	}
	other, _ := NewSealer(strings.Repeat("x", MinKeyLength))
	if _, err := other.Open(enc); err == nil {
		t.Error("opening with a different key should fail")
	}
}

func TestNewSealer_RejectsShortKey(t *testing.T) {
	if _, err := NewSealer("short"); err == nil {
		t.Error("expected short key to be rejected")
	}
}
