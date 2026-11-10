package auth

import (
	"testing"
	"time"
)

func TestTOTPGenerationAndValidation(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}
	if len(secret) == 0 {
		t.Fatal("secret should not be empty")
	}

	now := time.Now()
	code, err := GenerateCode(secret, now.Unix()/30)
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6-digit code, got %s", code)
	}

	if !ValidateCode(secret, code, now) {
		t.Errorf("expected code %s to be valid for secret", code)
	}

	if ValidateCode(secret, "000000", now) && code != "000000" {
		t.Error("arbitrary invalid code should fail")
	}
}

func TestRFC6238StandardVector(t *testing.T) {
	// Standard ASCII "12345678901234567890" in base32: GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	// At Unix timestamp 59s -> step = 1 -> code is 287082
	code, err := GenerateCode(secret, 1)
	if err != nil {
		t.Fatalf("failed generating code for step 1: %v", err)
	}
	if code != "287082" {
		t.Errorf("expected 287082, got %s", code)
	}

	url := BuildTOTPURL("PlannedWallet", "alice", secret)
	if len(url) == 0 {
		t.Error("url should not be empty")
	}
}
