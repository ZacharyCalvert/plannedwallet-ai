package captcha

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/plannedwallet/internal/repository"
)

type customStore struct {
	data map[string][]byte
}

func (s *customStore) Set(id string, digits []byte) {
	s.data[id] = digits
}

func (s *customStore) Get(id string, clear bool) []byte {
	val, ok := s.data[id]
	if !ok {
		return nil
	}
	if clear {
		delete(s.data, id)
	}
	return val
}

func TestGenerate(t *testing.T) {
	res, err := Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}
	if res.ID == "" {
		t.Error("expected non-empty captcha ID")
	}
	if !strings.HasPrefix(res.Image, "data:image/png;base64,") {
		t.Errorf("expected data URI prefix, got: %s", res.Image[:30])
	}
}

func TestWriteImage(t *testing.T) {
	res, err := Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}
	var buf bytes.Buffer
	if err := WriteImage(&buf, res.ID); err != nil {
		t.Fatalf("WriteImage() failed: %v", err)
	}
	pngHeader := []byte("\x89PNG\r\n\x1a\n")
	if !bytes.HasPrefix(buf.Bytes(), pngHeader) {
		t.Errorf("expected PNG magic bytes header")
	}
}

func TestVerifyWithCustomStore(t *testing.T) {
	store := &customStore{data: make(map[string][]byte)}
	SetCustomStore(store)

	store.Set("test-id", []byte{1, 2, 3, 4, 5, 6})

	if Verify("test-id", "000000") {
		t.Error("expected failure with wrong solution")
	}

	store.Set("test-id", []byte{1, 2, 3, 4, 5, 6})
	if !Verify("test-id", "123456") {
		t.Error("expected success with correct solution")
	}

	// Should be consumed
	if Verify("test-id", "123456") {
		t.Error("expected single-use failure on repeat verification")
	}
}

func TestReload(t *testing.T) {
	store := &customStore{data: make(map[string][]byte)}
	SetCustomStore(store)

	store.Set("reload-id", []byte{1, 2, 3, 4, 5, 6})
	if !Reload("reload-id") {
		t.Error("expected Reload to succeed")
	}
	if Reload("nonexistent-id") {
		t.Error("expected Reload to fail for nonexistent ID")
	}
}

func TestRepoStoreAndStartupCleanup(t *testing.T) {
	repo := repository.NewMemoryRepository()
	store := NewRepoStore(repo)
	SetCustomStore(store)
	ctx := context.Background()

	// 1. Generate captcha through the package (uses store.Set)
	res, err := Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// 2. Verification through the package (uses store.Get with clear=true)
	if Verify(res.ID, "999999") {
		t.Error("expected wrong solution to fail")
	}

	// 3. Test RunStartupCleanup
	deleted, err := RunStartupCleanup(ctx, repo)
	if err != nil {
		t.Fatalf("RunStartupCleanup failed: %v", err)
	}
	if deleted < 0 {
		t.Errorf("unexpected negative deleted count: %d", deleted)
	}
}
