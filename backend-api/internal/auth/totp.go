package auth

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

// GenerateTOTPSecret creates a 160-bit (20-byte) cryptographically random base32 encoded secret.
func GenerateTOTPSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}

// GenerateCode computes a 6-digit TOTP code for a given secret at counter t.
func GenerateCode(secret string, t int64) (string, error) {
	cleanSecret := strings.ToUpper(strings.TrimSpace(secret))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(cleanSecret)
	if err != nil {
		return "", err
	}

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(t))

	h := hmac.New(sha1.New, key)
	h.Write(buf)
	hash := h.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	binaryCode := (int(hash[offset]&0x7f) << 24) |
		(int(hash[offset+1]&0xff) << 16) |
		(int(hash[offset+2]&0xff) << 8) |
		(int(hash[offset+3] & 0xff))

	code := binaryCode % 1000000
	return fmt.Sprintf("%06d", code), nil
}

// ValidateCode checks if the provided 6-digit code matches the expected TOTP at the given time,
// allowing a +/- 1 step (30s) drift window.
func ValidateCode(secret, code string, at time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	t0 := at.Unix() / 30
	for _, step := range []int64{0, -1, 1} {
		expected, err := GenerateCode(secret, t0+step)
		if err == nil && hmac.Equal([]byte(expected), []byte(code)) {
			return true
		}
	}
	return false
}

// BuildTOTPURL constructs the standard otpauth:// URL for authenticator QR codes.
func BuildTOTPURL(issuer, accountName, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		url.PathEscape(issuer), url.PathEscape(accountName), secret, url.QueryEscape(issuer))
}
