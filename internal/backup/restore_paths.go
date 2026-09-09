package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validateRestorePaths(source, destination string) error {
	if strings.TrimSpace(source) == "" || strings.TrimSpace(destination) == "" {
		return errors.New("restore source and destination are required")
	}
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}
	for label, path := range map[string]string{"source": source, "destination": destination} {
		info, statErr := os.Lstat(path)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("restore %s root cannot be a symlink: %s", label, path)
			}
			if label == "source" && !info.IsDir() {
				return fmt.Errorf("restore source is not a directory: %s", path)
			}
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
	}
	relative, err := filepath.Rel(source, destination)
	if err != nil {
		return err
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return fmt.Errorf("restore destination must not be inside source: %s", destination)
	}
	return nil
}
