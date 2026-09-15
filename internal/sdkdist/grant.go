package sdkdist

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const MaxGrantLifetime = 24 * time.Hour
const MaxGrantResponseBytes = 64 << 10

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

// ParseGrant decodes a bounded broker response. It intentionally performs no
// network access and does not validate entitlement; callers must run
// ValidateGrant with the project lock context afterwards.
func ParseGrant(data []byte) (ArtifactGrant, error) {
	if len(data) == 0 || len(data) > MaxGrantResponseBytes {
		return ArtifactGrant{}, errors.New("artifact grant response exceeds size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var grant ArtifactGrant
	if err := decoder.Decode(&grant); err != nil {
		return ArtifactGrant{}, fmt.Errorf("invalid artifact grant JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ArtifactGrant{}, errors.New("artifact grant response contains trailing data")
	}
	return grant, nil
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
