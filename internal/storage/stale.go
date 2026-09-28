package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MinimumStaleAge = 24 * time.Hour

// SweepStaleFiles removes only old, regular Hydro-owned temporary files from
// the media root. It never traverses subdirectories or follows symlinks.
func SweepStaleFiles(ctx context.Context, root string, maxAge time.Duration, now time.Time) (int, error) {
	if strings.TrimSpace(root) == "" || maxAge < MinimumStaleAge {
		return 0, errors.New("a media root and stale age of at least 24 hours are required")
	}
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return removed, err
		}
		name := entry.Name()
		if !(strings.HasPrefix(name, "upload-") || strings.HasPrefix(name, "metadata-clean-") || (strings.HasSuffix(name, ".tmp") && strings.Contains(name, "-"))) {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			continue
		}
		path := filepath.Join(root, name)
		info, err := entry.Info()
		if err != nil {
			return removed, err
		}
		if !info.Mode().IsRegular() || now.Sub(info.ModTime()) < maxAge {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
