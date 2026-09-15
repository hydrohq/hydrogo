package sdkdist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const MaxGrantRequestBytes = 16 << 10

// GrantRequest contains only entitlement coordinates. Credentials are sent in
// the Authorization header and are never part of the request body.
type GrantRequest struct {
	ProjectID   string   `json:"project_id"`
	CoreVersion string   `json:"core_version"`
	SDKVersion  string   `json:"sdk_version"`
	Target      string   `json:"target"`
	Edition     string   `json:"edition"`
	Features    []string `json:"features,omitempty"`
}

func validateGrantRequest(request GrantRequest) error {
	for name, value := range map[string]string{
		"project_id": request.ProjectID, "core_version": request.CoreVersion,
		"sdk_version": request.SDKVersion, "target": request.Target, "edition": request.Edition,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("broker grant request %s is required", name)
		}
	}
	seen := make(map[string]struct{}, len(request.Features))
	for _, feature := range request.Features {
		feature = strings.TrimSpace(feature)
		if feature == "" || strings.ContainsAny(feature, "\\/\r\n") {
			return errors.New("broker grant request contains an invalid feature")
		}
		if _, ok := seen[feature]; ok {
			return errors.New("broker grant request contains duplicate features")
		}
		seen[feature] = struct{}{}
	}
	return nil
}

// RequestGrant requests one short-lived artifact grant from a broker. The
// caller must still run ValidateGrant against the project lock context.
func RequestGrant(ctx context.Context, endpoint, accessToken string, request GrantRequest, client *http.Client, allowInsecureLocalhost bool) (ArtifactGrant, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(accessToken) == "" || len(accessToken) > 4096 {
		return ArtifactGrant{}, errors.New("broker access token is required")
	}
	if err := validateGrantRequest(request); err != nil {
		return ArtifactGrant{}, err
	}
	parsed, err := ValidateDownloadURL(endpoint, allowInsecureLocalhost)
	if err != nil {
		return ArtifactGrant{}, fmt.Errorf("broker endpoint: %w", err)
	}
	payload, err := json.Marshal(request)
	if err != nil || len(payload) > MaxGrantRequestBytes {
		return ArtifactGrant{}, errors.New("broker grant request exceeds size limit")
	}
	httpClient := client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	clone := *httpClient
	clone.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("broker redirects are not allowed") }
	requestURL := parsed.String()
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(payload))
	if err != nil {
		return ArtifactGrant{}, fmt.Errorf("create broker request: %w", err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+accessToken)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	response, err := clone.Do(httpRequest)
	if err != nil {
		return ArtifactGrant{}, fmt.Errorf("request artifact grant: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, MaxGrantResponseBytes+1))
	if err != nil {
		return ArtifactGrant{}, fmt.Errorf("read broker response: %w", err)
	}
	if len(body) > MaxGrantResponseBytes {
		return ArtifactGrant{}, errors.New("broker response exceeds size limit")
	}
	if response.StatusCode != http.StatusOK {
		return ArtifactGrant{}, fmt.Errorf("broker returned HTTP %d", response.StatusCode)
	}
	grant, err := ParseGrant(body)
	if err != nil {
		return ArtifactGrant{}, err
	}
	return grant, nil
}

// RequestValidatedGrant combines broker transport with entitlement-coordinate
// validation. Callers that need a lower-level parser may use RequestGrant, but
// runtime installation paths should prefer this fail-closed helper.
func RequestValidatedGrant(ctx context.Context, endpoint, accessToken string, request GrantRequest, expectedProject, expectedCore, expectedSDK string, now time.Time, client *http.Client, allowInsecureLocalhost bool) (ArtifactGrant, error) {
	grant, err := RequestGrant(ctx, endpoint, accessToken, request, client, allowInsecureLocalhost)
	if err != nil {
		return ArtifactGrant{}, err
	}
	if err := ValidateGrant(grant, expectedProject, expectedCore, expectedSDK, now, allowInsecureLocalhost); err != nil {
		return ArtifactGrant{}, err
	}
	return grant, nil
}
