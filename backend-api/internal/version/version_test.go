package version_test

import (
	"testing"

	"github.com/plannedwallet/internal/version"
)

func TestCurrentDefaultsAndOverrides(t *testing.T) {
	origRun, origDate, origSHA := version.GitHubRun, version.BuildDate, version.GitSHA
	defer func() {
		version.GitHubRun = origRun
		version.BuildDate = origDate
		version.GitSHA = origSHA
	}()

	version.GitHubRun = ""
	version.BuildDate = "   "
	version.GitSHA = ""
	def := version.Current()
	if def.GitHubRun != "local" || def.BuildDate != "unknown" || def.GitSHA != "unknown" {
		t.Fatalf("unexpected fallback values: %+v", def)
	}

	version.GitHubRun = "42-999999"
	version.BuildDate = "2026-11-18T19:00:00Z"
	version.GitSHA = "0123456789abcdef0123456789abcdef01234567"
	stamped := version.Current()
	if stamped.GitHubRun != "42-999999" || stamped.BuildDate != "2026-11-18T19:00:00Z" || stamped.GitSHA != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("unexpected stamped values: %+v", stamped)
	}
}
