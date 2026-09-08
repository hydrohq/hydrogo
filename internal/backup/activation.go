package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ActivateStagingMedia atomically promotes a verified restore staging root to
// the active media root. The previous active root is retained for rollback.
// It requires the quarantine marker written by a staging restore and never
// overwrites an existing rollback root.
func ActivateStagingMedia(staging, active, rollback string) error {
	staging, active, rollback, err := absoluteActivationPaths(staging, active, rollback)
	if err != nil {
		return err
	}
	if staging == active || staging == rollback || active == rollback {
		return errors.New("staging, active and rollback roots must be distinct")
	}
	marker, err := readRestoreMarker(staging)
	if err != nil {
		return err
	}
	if !marker.Controls.SessionsRevoked || !marker.Controls.EgressQuarantined || !marker.Controls.OldWritesBlocked {
		return errors.New("restore controls are incomplete")
	}
	if info, err := os.Stat(staging); err != nil {
		return fmt.Errorf("restore staging: %w", err)
	} else if !info.IsDir() {
		return errors.New("restore staging is not a directory")
	}
	if _, err := os.Stat(rollback); err == nil {
		return fmt.Errorf("rollback root already exists: %s", rollback)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(active), 0o750); err != nil {
		return err
	}
	hadActive := false
	if _, err := os.Stat(active); err == nil {
		hadActive = true
		if err := os.Rename(active, rollback); err != nil {
			return fmt.Errorf("retain active media: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(staging, active); err != nil {
		if hadActive {
			_ = os.Rename(rollback, active)
		}
		return fmt.Errorf("activate restore media: %w", err)
	}
	activeMarker := struct {
		ManifestVersion int       `json:"manifest_version"`
		ActivatedAt     time.Time `json:"activated_at"`
	}{ManifestVersion: marker.ManifestVersion, ActivatedAt: time.Now().UTC()}
	data, err := json.MarshalIndent(activeMarker, "", "  ")
	if err != nil {
		_ = rollbackActivation(active, rollback, hadActive)
		return err
	}
	if err := os.WriteFile(filepath.Join(active, "RESTORE_ACTIVE.json"), append(data, '\n'), 0o640); err != nil {
		_ = rollbackActivation(active, rollback, hadActive)
		return err
	}
	return nil
}

func absoluteActivationPaths(staging, active, rollback string) (string, string, string, error) {
	if strings.TrimSpace(staging) == "" || strings.TrimSpace(active) == "" || strings.TrimSpace(rollback) == "" {
		return "", "", "", errors.New("staging, active and rollback roots are required")
	}
	paths := make([]string, 3)
	for i, path := range []string{staging, active, rollback} {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return "", "", "", err
		}
		paths[i] = filepath.Clean(absolute)
	}
	return paths[0], paths[1], paths[2], nil
}

func readRestoreMarker(root string) (RestoreMarker, error) {
	data, err := os.ReadFile(filepath.Join(root, "RESTORE_QUARANTINED.json"))
	if err != nil {
		return RestoreMarker{}, fmt.Errorf("restore quarantine marker: %w", err)
	}
	var marker RestoreMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return RestoreMarker{}, fmt.Errorf("restore quarantine marker: %w", err)
	}
	if marker.ManifestVersion != 1 {
		return RestoreMarker{}, errors.New("unsupported restore manifest marker")
	}
	return marker, nil
}

func rollbackActivation(active, rollback string, hadActive bool) error {
	_ = os.RemoveAll(active)
	if hadActive {
		return os.Rename(rollback, active)
	}
	return nil
}
