package sdkdist

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ardaozyildirim/hydrogo/internal/pathguard"
	"github.com/ardaozyildirim/hydrogo/internal/release"
)

// InstallArtifact verifies source and atomically installs it at destination.
// Existing identical files are reused; a different existing file is never
// overwritten. Callers should derive destination with CachePath.
func InstallArtifact(source, destination string, expected release.Artifact) error {
	if source == "" || destination == "" {
		return errors.New("source and destination are required")
	}
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	if err := rejectSource(sourceAbs); err != nil {
		return err
	}
	if err := pathguard.RejectSymlinkComponents(filepath.Dir(destinationAbs)); err != nil {
		return fmt.Errorf("destination path: %w", err)
	}
	if err := release.VerifyArtifactFile(sourceAbs, expected); err != nil {
		return fmt.Errorf("verify downloaded artifact: %w", err)
	}
	if info, err := os.Lstat(destinationAbs); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("destination artifact is not a regular file")
		}
		if err := release.VerifyArtifactFile(destinationAbs, expected); err == nil {
			return nil
		}
		return errors.New("destination artifact differs and will not be overwritten")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	parent := filepath.Dir(destinationAbs)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(parent, ".hydro-runtime-*-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o750); err != nil {
		_ = temporary.Close()
		return err
	}
	input, err := os.Open(sourceAbs)
	if err != nil {
		_ = temporary.Close()
		return err
	}
	_, copyErr := io.Copy(temporary, input)
	closeInputErr := input.Close()
	if copyErr != nil {
		_ = temporary.Close()
		return copyErr
	}
	if closeInputErr != nil {
		_ = temporary.Close()
		return closeInputErr
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := release.VerifyArtifactFile(temporaryPath, expected); err != nil {
		return fmt.Errorf("verify staged artifact: %w", err)
	}
	if err := os.Rename(temporaryPath, destinationAbs); err != nil {
		return fmt.Errorf("activate runtime artifact: %w", err)
	}
	return nil
}

func rejectSource(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("source artifact is not a regular file")
	}
	return nil
}
