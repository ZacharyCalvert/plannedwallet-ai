package auth_test

import (
	"strings"
	"testing"

	"github.com/plannedwallet/internal/auth"
)

func TestGenerateAndVerifyRecoveryPhrase_Argon2id(t *testing.T) {
	phrase1, err := auth.GenerateRecoveryPhrase()
	if err != nil {
		t.Fatalf("GenerateRecoveryPhrase failed: %v", err)
	}
	words := strings.Fields(phrase1)
	if len(words) != auth.RecoveryPhraseWordCount {
		t.Fatalf("expected %d words, got %d (%q)", auth.RecoveryPhraseWordCount, len(words), phrase1)
	}
	phrase2, err := auth.GenerateRecoveryPhrase()
	if err != nil {
		t.Fatalf("GenerateRecoveryPhrase 2 failed: %v", err)
	}
	if phrase1 == phrase2 {
		t.Fatal("expected distinct recovery phrases across calls")
	}

	hint := auth.BuildRecoveryKeyHint(phrase1)
	if !strings.HasPrefix(hint, "••••") || len([]rune(hint)) != 8 {
		t.Fatalf("unexpected recovery hint format: %q", hint)
	}

	hash, err := auth.HashRecoveryPhrase(phrase1)
	if err != nil {
		t.Fatalf("HashRecoveryPhrase failed: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("expected Argon2id PHC hash, got %q", hash)
	}

	noisyInput := "  " + strings.ToUpper(strings.ReplaceAll(phrase1, " ", "   \n ")) + "  "
	if !auth.VerifyRecoveryPhrase(hash, noisyInput) {
		t.Fatal("expected normalized noisy input to verify against Argon2id hash")
	}
	if auth.VerifyRecoveryPhrase(hash, phrase2) {
		t.Fatal("expected different phrase to fail verification")
	}
	if auth.VerifyRecoveryPhrase("invalid-hash", phrase1) {
		t.Fatal("expected malformed hash to fail verification cleanly")
	}
}
