// Package share lets the owner give family members read-only access to the
// family report via a per-recipient secret link plus an authenticator-app
// (TOTP) code — no passwords, no Google account, no SMS provider.
package share

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// RFC 6238 defaults — the only parameters every authenticator app supports.
const (
	totpStep   = 30 // seconds
	totpDigits = 6
	// totpSkew is how many steps either side of "now" are accepted, to absorb
	// clock drift on the recipient's phone.
	totpSkew = 1
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewTOTPSecret returns a fresh 160-bit secret, base32-encoded as
// authenticator apps expect.
func NewTOTPSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return b32.EncodeToString(b), nil
}

// ProvisioningURI builds the otpauth:// URI encoded into the enrolment QR.
func ProvisioningURI(issuer, account, secret string) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("digits", fmt.Sprint(totpDigits))
	q.Set("period", fmt.Sprint(totpStep))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// hotp computes the RFC 4226 code for counter.
func hotp(key []byte, counter int64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(counter))
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", bin%1_000_000)
}

// VerifyTOTP checks code against secret at time now, accepting ±totpSkew
// steps. Any step at or before lastStep is rejected so an observed code can't
// be replayed. Returns the matched step so the caller can persist it.
func VerifyTOTP(secret, code string, now time.Time, lastStep int64) (int64, bool) {
	if len(code) != totpDigits {
		return 0, false
	}
	key, err := b32.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return 0, false
	}
	cur := now.Unix() / totpStep
	for s := cur - totpSkew; s <= cur+totpSkew; s++ {
		if s <= lastStep {
			continue
		}
		if hmac.Equal([]byte(hotp(key, s)), []byte(code)) {
			return s, true
		}
	}
	return 0, false
}
