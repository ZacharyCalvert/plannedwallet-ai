package version

import (
	"strings"

	"github.com/plannedwallet/internal/models"
)

// Build provenance metadata injected at link time via -ldflags.
var (
	GitHubRun = "local"
	BuildDate = "unknown"
	GitSHA    = "unknown"
	Tag       = "untagged"
)

func valueOrDefault(val, fallback string) string {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

// Current returns the stamped build version metadata for the running binary.
func Current() models.VersionResponse {
	return models.VersionResponse{
		GitHubRun: valueOrDefault(GitHubRun, "local"),
		BuildDate: valueOrDefault(BuildDate, "unknown"),
		GitSHA:    valueOrDefault(GitSHA, "unknown"),
		Tag:       valueOrDefault(Tag, "untagged"),
	}
}
