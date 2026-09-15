package sdkdist

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const MaxGrantLifetime = 24 * time.Hour

// ArtifactGrant is the broker-to-CLI hand-off. AccessToken is never persisted
// by this package; callers should keep it in memory only for the request.
type ArtifactGrant struct {
	AccessToken string    `json:"access_token"`
	ProjectID   string    `json:"project_id"`
	CoreVersion string    `json:"core_version"`
	SDKVersion  string    `json:"sdk_version"`
	ArtifactURL string    `json:"artifact_url"`
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func ValidateGrant(grant ArtifactGrant, expectedProject, expectedCore, expectedSDK string, now time.Time, allowInsecureLocalhost bool) error {
	if strings.TrimSpace(grant.AccessToken) == "" || len(grant.AccessToken) > 4096 {
		return errors.New("artifact grant token is required")
	}
	if strings.TrimSpace(grant.ProjectID) == "" || grant.ProjectID != expectedProject {
		return errors.New("artifact grant project does not match")
	}
	if strings.TrimSpace(grant.CoreVersion) == "" || grant.CoreVersion != expectedCore {
		return errors.New("artifact grant Core version does not match")
	}
	if strings.TrimSpace(grant.SDKVersion) == "" || grant.SDKVersion != expectedSDK {
		return errors.New("artifact grant SDK version does not match")
	}
	if grant.IssuedAt.IsZero() || grant.ExpiresAt.IsZero() || !grant.ExpiresAt.After(grant.IssuedAt) {
		return errors.New("artifact grant lifetime is invalid")
	}
	if grant.ExpiresAt.Sub(grant.IssuedAt) > MaxGrantLifetime {
		return errors.New("artifact grant lifetime is too long")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if now.Before(grant.IssuedAt.Add(-5*time.Minute)) || !now.Before(grant.ExpiresAt) {
		return errors.New("artifact grant is expired or not yet valid")
	}
	if _, err := ValidateDownloadURL(grant.ArtifactURL, allowInsecureLocalhost); err != nil {
		return fmt.Errorf("artifact grant URL: %w", err)
	}
	return nil
}

func RedactToken(token string) string {
	if len(token) <= 8 {
		return "[redacted]"
	}
	return token[:4] + "…" + token[len(token)-4:]
}
