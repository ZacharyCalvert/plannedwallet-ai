package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	RecoveryPhraseWordCount = 12
	argon2MemoryKiB         = 19 * 1024
	argon2TimeCost          = 2
	argon2Threads           = 1
	argon2KeyLen            = 32
	argon2SaltLen           = 16
)

// GenerateRecoveryPhrase produces a random 12-word recovery phrase from recoveryWordlist.
func GenerateRecoveryPhrase() (string, error) {
	maxIdx := big.NewInt(int64(len(recoveryWordlist)))
	words := make([]string, RecoveryPhraseWordCount)
	for i := 0; i < RecoveryPhraseWordCount; i++ {
		idx, err := rand.Int(rand.Reader, maxIdx)
		if err != nil {
			return "", err
		}
		words[i] = recoveryWordlist[idx.Int64()]
	}
	return strings.Join(words, " "), nil
}

// NormalizeRecoveryPhrase canonicalizes spacing and casing across a 12-word phrase.
func NormalizeRecoveryPhrase(raw string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(raw)))
	return strings.Join(fields, " ")
}

// BuildRecoveryKeyHint creates a short masked hint from the end of the phrase for UI recognition.
func BuildRecoveryKeyHint(phrase string) string {
	compact := strings.ReplaceAll(NormalizeRecoveryPhrase(phrase), " ", "")
	if len(compact) <= 4 {
		return "••••"
	}
	return "••••" + compact[len(compact)-4:]
}

// HashRecoveryPhrase derives a salted Argon2id PHC-formatted digest of a normalized recovery phrase.
func HashRecoveryPhrase(phrase string) (string, error) {
	normalized := NormalizeRecoveryPhrase(phrase)
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	derived := argon2.IDKey([]byte(normalized), salt, argon2TimeCost, argon2MemoryKiB, argon2Threads, argon2KeyLen)
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(derived)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argon2MemoryKiB, argon2TimeCost, argon2Threads, saltB64, hashB64), nil
}

func parseArgon2ParamMap(raw string) (map[string]uint32, bool) {
	vals := make(map[string]uint32, 3)
	for _, kv := range strings.Split(raw, ",") {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return nil, false
		}
		v, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil {
			return nil, false
		}
		vals[parts[0]] = uint32(v)
	}
	return vals, true
}

func parseArgon2Params(raw string) (mem, timeCost uint32, threads uint8, ok bool) {
	vals, valid := parseArgon2ParamMap(raw)
	if !valid {
		return 0, 0, 0, false
	}
	mem, timeCost, parsedThreads := vals["m"], vals["t"], vals["p"]
	if mem == 0 || timeCost == 0 || parsedThreads == 0 || parsedThreads > 255 {
		return 0, 0, 0, false
	}
	return mem, timeCost, uint8(parsedThreads), true
}

// VerifyRecoveryPhrase checks a candidate recovery phrase in constant time against an Argon2id PHC digest.
func VerifyRecoveryPhrase(encodedHash, candidate string) bool {
	parts := strings.Split(strings.TrimSpace(encodedHash), "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	mem, timeCost, threads, ok := parseArgon2Params(parts[3])
	if !ok {
		return false
	}
	salt, errSalt := base64.RawStdEncoding.DecodeString(parts[4])
	expected, errHash := base64.RawStdEncoding.DecodeString(parts[5])
	if errSalt != nil || errHash != nil || len(salt) == 0 || len(expected) == 0 {
		return false
	}
	actual := argon2.IDKey([]byte(NormalizeRecoveryPhrase(candidate)), salt, timeCost, mem, threads, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
