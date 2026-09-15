package sdkdist

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/release"
)

const DefaultMaxArtifactBytes int64 = 512 << 20

type DownloadOptions struct {
	Client                 *http.Client
	MaxBytes               int64
	AllowInsecureLocalhost bool
}

// DownloadGrantAndInstall validates a broker grant before starting any
// network request, then delegates to the bounded download/atomic install path.
func DownloadGrantAndInstall(ctx context.Context, grant ArtifactGrant, expectedProject, expectedCore, expectedSDK string, now time.Time, expected release.Artifact, destination string, options DownloadOptions) error {
	if err := ValidateGrant(grant, expectedProject, expectedCore, expectedSDK, now, options.AllowInsecureLocalhost); err != nil {
		return err
	}
	return DownloadAndInstall(ctx, grant.ArtifactURL, expected, destination, options)
}

// DownloadAndInstall downloads one broker-authorized artifact into a temporary
// file and installs it atomically after release metadata verification. The
// function never follows redirects and never logs the URL/token.
func DownloadAndInstall(ctx context.Context, rawURL string, expected release.Artifact, destination string, options DownloadOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := ValidateDownloadURL(rawURL, options.AllowInsecureLocalhost); err != nil {
		return err
	}
	maxBytes := options.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxArtifactBytes
	}
	if expected.Size < 0 || expected.Size > maxBytes {
		return fmt.Errorf("artifact size exceeds configured download limit")
	}
	client := options.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	} else {
		clone := *client
		clone.CheckRedirect = func(*http.Request, []*http.Request) error {
			return errors.New("artifact download redirects are not allowed")
		}
		client = &clone
	}
	if client.CheckRedirect == nil {
		clone := *client
		clone.CheckRedirect = func(*http.Request, []*http.Request) error {
			return errors.New("artifact download redirects are not allowed")
		}
		client = &clone
	}
	parsed, _ := ValidateDownloadURL(rawURL, options.AllowInsecureLocalhost)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return fmt.Errorf("create artifact request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download artifact: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("artifact download returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength >= 0 && response.ContentLength > maxBytes {
		return fmt.Errorf("artifact response exceeds configured download limit")
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(parent, ".hydro-download-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	limited := io.LimitReader(response.Body, maxBytes+1)
	if _, err := io.Copy(temporary, limited); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write downloaded artifact: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := InstallArtifact(temporaryPath, destination, expected); err != nil {
		return err
	}
	return nil
}
