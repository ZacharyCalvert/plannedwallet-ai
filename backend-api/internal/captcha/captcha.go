package captcha

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"time"

	"github.com/dchest/captcha"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

const (
	ImageWidth  = 240
	ImageHeight = 80
	DigitsLen   = 6
)

// RepoStore adapts repository.Repository to captcha.Store for multi-replica persistence.
type RepoStore struct {
	repo repository.Repository
}

// NewRepoStore creates a new captcha.Store backed by repository.Repository.
func NewRepoStore(repo repository.Repository) *RepoStore {
	return &RepoStore{repo: repo}
}

// Set stores digits for the given captcha ID in the backing store.
func (s *RepoStore) Set(id string, digits []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.repo.SetCaptcha(ctx, id, digits)
}

// Get retrieves digits for the given captcha ID from the backing store, optionally clearing them.
func (s *RepoStore) Get(id string, clear bool) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	digits, err := s.repo.GetCaptcha(ctx, id, clear)
	if err != nil {
		return nil
	}
	return digits
}

// RunStartupCleanup deletes expired CAPTCHA records older than 5 minutes on application startup.
func RunStartupCleanup(ctx context.Context, repo repository.Repository) (int64, error) {
	deleted, err := repo.DeleteExpiredCaptchas(ctx, 5*time.Minute)
	if err != nil {
		trace.LogError(ctx, err, "Failed to purge expired CAPTCHAs on startup")
		return 0, err
	}
	trace.LogInfo(ctx, "Startup CAPTCHA cleanup completed: %d expired challenges deleted", deleted)
	return deleted, nil
}

// Generate creates a new CAPTCHA and returns its ID and base64 data URI image.
func Generate() (*models.CaptchaResponse, error) {
	id := captcha.NewLen(DigitsLen)
	var buf bytes.Buffer
	if err := captcha.WriteImage(&buf, id, ImageWidth, ImageHeight); err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return &models.CaptchaResponse{
		ID:    id,
		Image: "data:image/png;base64," + encoded,
	}, nil
}

// WriteImage streams the PNG image bytes for a CAPTCHA challenge directly to a writer.
func WriteImage(w io.Writer, id string) error {
	return captcha.WriteImage(w, id, ImageWidth, ImageHeight)
}

// Verify checks whether the submitted digits match the CAPTCHA solution.
func Verify(id, solution string) bool {
	return captcha.VerifyString(id, solution)
}

// Reload refreshes digits for an existing CAPTCHA challenge ID.
func Reload(id string) bool {
	return captcha.Reload(id)
}

// SetCustomStore replaces the default in-memory CAPTCHA storage.
func SetCustomStore(s captcha.Store) {
	captcha.SetCustomStore(s)
}
