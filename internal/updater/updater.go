// Package updater implements signed, runtime-aware self-update checks,
// atomic binary replacement, and rollback for HydroGo.
package updater

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hydrohq/hydrogo/internal/pathguard"
)

const (
	DefaultManifestURL = "https://github.com/hydrohq/hydrogo/releases/latest/download/hydro-update.json"
	maxManifestBytes   = 1 << 20   // 1 MiB
	maxBinaryBytes     = 256 << 20 // 256 MiB
)

var (
	ErrInvalidManifest       = errors.New("invalid update manifest")
	ErrInvalidSignature      = errors.New("update manifest signature verification failed")
	ErrChecksumMismatch      = errors.New("downloaded binary sha256 mismatch")
	ErrSelfUpdateUnsupported = errors.New("in-place self-update is not supported in this runtime")
	ErrEditionMismatch       = errors.New("update manifest edition does not match running binary")
	ErrNoRollbackAvailable   = errors.New("no previous binary (.prev) found for rollback")
)

type RuntimeMode string

const (
	RuntimeStandalone RuntimeMode = "standalone"
	RuntimeCGI        RuntimeMode = "cgi"
	RuntimeDocker     RuntimeMode = "docker"
)

// DetectCurrentRuntime inspects the live process environment to classify the host runtime.
func DetectCurrentRuntime(isCGI bool) RuntimeMode {
	return DetectRuntime(isCGI, os.Getenv, func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	})
}

// DetectRuntime classifies standalone/VPS, CGI/FastCGI, or Docker container runtimes.
func DetectRuntime(isCGI bool, envLookup func(string) string, fileExists func(string) bool) RuntimeMode {
	if isCGI {
		return RuntimeCGI
	}
	if envLookup != nil {
		if envLookup("GATEWAY_INTERFACE") != "" || envLookup("FCGI_ROLE") != "" {
			return RuntimeCGI
		}
		if envLookup("HYDRO_CONTAINER") == "1" {
			return RuntimeDocker
		}
	}
	if fileExists != nil && fileExists("/.dockerenv") {
		return RuntimeDocker
	}
	return RuntimeStandalone
}

type Asset struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type UpdateManifest struct {
	FormatVersion int              `json:"format_version"`
	Version       string           `json:"version"`
	Edition       string           `json:"edition"`
	PublishedAt   string           `json:"published_at,omitempty"`
	Notes         string           `json:"notes,omitempty"`
	Assets        map[string]Asset `json:"assets"`
}

type SignedUpdateManifest struct {
	Manifest  json.RawMessage `json:"manifest"`
	PublicKey string          `json:"public_key"`
	Signature string          `json:"signature"`
}

type CheckOptions struct {
	CurrentVersion   string
	Edition          string
	Runtime          RuntimeMode
	ManifestURL      string
	TrustedPublicKey ed25519.PublicKey
}

type UpdateStatus struct {
	CurrentVersion  string      `json:"current_version"`
	LatestVersion   string      `json:"latest_version"`
	UpdateAvailable bool        `json:"update_available"`
	Edition         string      `json:"edition"`
	RuntimeMode     RuntimeMode `json:"runtime_mode"`
	CanSelfUpdate   bool        `json:"can_self_update"`
	Reason          string      `json:"reason,omitempty"`
	Notes           string      `json:"notes,omitempty"`
	DownloadURL     string      `json:"download_url,omitempty"`
	SHA256          string      `json:"sha256,omitempty"`
	SizeBytes       int64       `json:"size_bytes,omitempty"`
	CheckedAt       string      `json:"checked_at"`
}

type ApplyOptions struct {
	CheckOptions
	ExecutablePath string
}

type ApplyResult struct {
	Version    string      `json:"version"`
	BackupPath string      `json:"backup_path"`
	Runtime    RuntimeMode `json:"runtime"`
}

// VerifySignedUpdateManifest verifies the Ed25519 signature against trustedKey
// and strictly decodes the UpdateManifest payload.
func VerifySignedUpdateManifest(signed SignedUpdateManifest, trustedKey ed25519.PublicKey) (UpdateManifest, error) {
	if len(trustedKey) != ed25519.PublicKeySize {
		return UpdateManifest{}, ErrInvalidSignature
	}
	pubBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signed.PublicKey))
	if err != nil || len(pubBytes) != ed25519.PublicKeySize || !bytes.Equal(pubBytes, trustedKey) {
		return UpdateManifest{}, ErrInvalidSignature
	}
	sigBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signed.Signature))
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return UpdateManifest{}, ErrInvalidSignature
	}
	if !ed25519.Verify(trustedKey, signed.Manifest, sigBytes) {
		return UpdateManifest{}, ErrInvalidSignature
	}

	var manifest UpdateManifest
	dec := json.NewDecoder(bytes.NewReader(signed.Manifest))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return UpdateManifest{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}
	if manifest.FormatVersion != 1 || strings.TrimSpace(manifest.Version) == "" || strings.TrimSpace(manifest.Edition) == "" {
		return UpdateManifest{}, ErrInvalidManifest
	}
	return manifest, nil
}

// Check fetches and verifies the signed update manifest and evaluates runtime eligibility.
func Check(ctx context.Context, client *http.Client, opts CheckOptions) (UpdateStatus, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	manifestURL := strings.TrimSpace(opts.ManifestURL)
	if manifestURL == "" {
		manifestURL = DefaultManifestURL
	}
	edition := strings.TrimSpace(opts.Edition)
	if edition == "" {
		edition = "core"
	}
	rt := opts.Runtime
	if rt == "" {
		rt = DetectCurrentRuntime(false)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return UpdateStatus{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req) // #nosec G107 -- operator-configured manifest URL
	if err != nil {
		return UpdateStatus{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return UpdateStatus{}, fmt.Errorf("update manifest HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestBytes+1))
	if err != nil {
		return UpdateStatus{}, err
	}
	if len(body) > maxManifestBytes {
		return UpdateStatus{}, ErrInvalidManifest
	}

	var signed SignedUpdateManifest
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&signed); err != nil {
		return UpdateStatus{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}
	manifest, err := VerifySignedUpdateManifest(signed, opts.TrustedPublicKey)
	if err != nil {
		return UpdateStatus{}, err
	}

	platformKey := runtime.GOOS + "/" + runtime.GOARCH
	asset, hasAsset := manifest.Assets[platformKey]
	updateAvail := CompareSemver(opts.CurrentVersion, manifest.Version) < 0

	status := UpdateStatus{
		CurrentVersion:  opts.CurrentVersion,
		LatestVersion:   manifest.Version,
		UpdateAvailable: updateAvail,
		Edition:         edition,
		RuntimeMode:     rt,
		CanSelfUpdate:   true,
		Notes:           manifest.Notes,
		DownloadURL:     asset.URL,
		SHA256:          asset.SHA256,
		SizeBytes:       asset.Size,
		CheckedAt:       time.Now().UTC().Format(time.RFC3339),
	}

	switch {
	case rt == RuntimeDocker:
		status.CanSelfUpdate = false
		status.Reason = "In-place updates are disabled inside containers; pull the new container image instead."
	case manifest.Edition != edition:
		status.CanSelfUpdate = false
		status.Reason = fmt.Sprintf("Edition mismatch (%s vs %s); custom editions require their own manifest.", edition, manifest.Edition)
	case !hasAsset || strings.TrimSpace(asset.URL) == "":
		status.CanSelfUpdate = false
		status.Reason = fmt.Sprintf("No binary asset published for %s.", platformKey)
	}
	return status, nil
}

// Apply downloads, checksum-verifies, and atomically swaps the running binary,
// keeping a .prev backup for Rollback.
func Apply(ctx context.Context, client *http.Client, opts ApplyOptions) (ApplyResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	status, err := Check(ctx, client, opts.CheckOptions)
	if err != nil {
		return ApplyResult{}, err
	}
	if status.RuntimeMode == RuntimeDocker {
		return ApplyResult{}, fmt.Errorf("%w: %s", ErrSelfUpdateUnsupported, status.Reason)
	}
	if !status.CanSelfUpdate {
		if strings.Contains(status.Reason, "Edition mismatch") {
			return ApplyResult{}, fmt.Errorf("%w: %s", ErrEditionMismatch, status.Reason)
		}
		return ApplyResult{}, fmt.Errorf("%w: %s", ErrSelfUpdateUnsupported, status.Reason)
	}
	exePath := strings.TrimSpace(opts.ExecutablePath)
	if exePath == "" {
		resolved, err := os.Executable()
		if err != nil {
			return ApplyResult{}, err
		}
		exePath = resolved
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}
	if err := pathguard.RejectSymlinkComponents(filepath.Dir(exePath)); err != nil {
		return ApplyResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, status.DownloadURL, nil)
	if err != nil {
		return ApplyResult{}, err
	}
	resp, err := client.Do(req) // #nosec G107 -- verified URL from signed update manifest
	if err != nil {
		return ApplyResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ApplyResult{}, fmt.Errorf("download binary HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(exePath), ".hydro-update-*.tmp")
	if err != nil {
		return ApplyResult{}, err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	hasher := sha256.New()
	limited := io.LimitReader(resp.Body, maxBinaryBytes+1)
	written, err := io.Copy(io.MultiWriter(tmpFile, hasher), limited)
	if err != nil {
		_ = tmpFile.Close()
		return ApplyResult{}, err
	}
	if written > maxBinaryBytes {
		_ = tmpFile.Close()
		return ApplyResult{}, fmt.Errorf("%w: binary exceeds max size", ErrChecksumMismatch)
	}
	if err := tmpFile.Chmod(0o755); err != nil { // #nosec G302 -- executable binary requires 0755
		_ = tmpFile.Close()
		return ApplyResult{}, err
	}
	if err := tmpFile.Close(); err != nil {
		return ApplyResult{}, err
	}

	actualHex := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actualHex, strings.TrimSpace(status.SHA256)) {
		return ApplyResult{}, fmt.Errorf("%w: got %s want %s", ErrChecksumMismatch, actualHex, status.SHA256)
	}

	backupPath := exePath + ".prev"
	if err := copyFileAtomic(exePath, backupPath, 0o755); err != nil {
		return ApplyResult{}, fmt.Errorf("backup current binary: %w", err)
	}
	if err := os.Rename(tmpPath, exePath); err != nil {
		return ApplyResult{}, fmt.Errorf("replace binary: %w", err)
	}
	return ApplyResult{
		Version:    status.LatestVersion,
		BackupPath: backupPath,
		Runtime:    status.RuntimeMode,
	}, nil
}

// Rollback restores the .prev binary created by Apply.
func Rollback(executablePath string) error {
	exePath := strings.TrimSpace(executablePath)
	if exePath == "" {
		resolved, err := os.Executable()
		if err != nil {
			return err
		}
		exePath = resolved
	}
	backupPath := exePath + ".prev"
	if _, err := os.Stat(backupPath); errors.Is(err, os.ErrNotExist) {
		return ErrNoRollbackAvailable
	} else if err != nil {
		return err
	}
	return copyFileAtomic(backupPath, exePath, 0o755)
}

func copyFileAtomic(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src) // #nosec G304 -- internal executable path
	if err != nil {
		return err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".hydro-copy-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, dst)
}

// CompareSemver compares two vX.Y.Z strings, returning -1, 0, or 1.
func CompareSemver(a, b string) int {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	if idx := strings.IndexAny(v, "-+"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err == nil && n >= 0 {
			out[i] = n
		}
	}
	return out
}
