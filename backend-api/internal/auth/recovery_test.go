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

func TestVerifyRecoveryPhrase_RejectsElevenOfTwelveAndTranspositions(t *testing.T) {
	phrase, err := auth.GenerateRecoveryPhrase()
	if err != nil {
		t.Fatalf("GenerateRecoveryPhrase failed: %v", err)
	}
	hash, err := auth.HashRecoveryPhrase(phrase)
	if err != nil {
		t.Fatalf("HashRecoveryPhrase failed: %v", err)
	}
	words := strings.Fields(phrase)

	// Mutating any 1 of the 12 words (leaving 11 of 12 exact) must fail verification
	for idx := range words {
		candidate := append([]string(nil), words...)
		replacement := "zebra"
		if candidate[idx] == replacement {
			replacement = "quart"
		}
		candidate[idx] = replacement
		if auth.VerifyRecoveryPhrase(hash, strings.Join(candidate, " ")) {
			t.Fatalf("expected 11-of-12 match with word #%d altered to fail", idx+1)
		}
	}

	// Swapping two adjacent words (all 12 words present, wrong order) must fail
	swapped := append([]string(nil), words...)
	swapped[0], swapped[1] = swapped[1], swapped[0]
	if auth.VerifyRecoveryPhrase(hash, strings.Join(swapped, " ")) {
		t.Fatal("expected transposed 12-word phrase to fail")
	}

	// Missing 12th word (11 words) and extra 13th word must fail
	if auth.VerifyRecoveryPhrase(hash, strings.Join(words[:11], " ")) {
		t.Fatal("expected truncated 11-word phrase to fail")
	}
	if auth.VerifyRecoveryPhrase(hash, phrase+" extra") {
		t.Fatal("expected 13-word phrase to fail")
	}
}
