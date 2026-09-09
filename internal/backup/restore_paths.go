package backup

import (
	"errors"
	"fmt"
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
	relative, err := filepath.Rel(source, destination)
	if err != nil {
		return err
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return fmt.Errorf("restore destination must not be inside source: %s", destination)
	}
	return nil
}
