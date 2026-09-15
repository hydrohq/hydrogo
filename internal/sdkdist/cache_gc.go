package sdkdist

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CacheEntry describes one artifact file found directly below a versioned
// runtime cache directory.
type CacheEntry struct {
	Version string
	Path    string
	Bytes   int64
}

// ListCache returns regular artifact files in root/<version> directories. It
// never follows symlinks and ignores unrelated directories/files.
func ListCache(root string) ([]CacheEntry, error) {
	rootAbs, err := validatedCacheRoot(root)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(rootAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return []CacheEntry{}, nil
		}
		return nil, err
	}
	var result []CacheEntry
	for _, versionDir := range entries {
		if !versionDir.IsDir() || !versionPattern.MatchString(versionDir.Name()) || strings.Contains(versionDir.Name(), "..") {
			continue
		}
		files, err := os.ReadDir(filepath.Join(rootAbs, versionDir.Name()))
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			info, err := os.Lstat(filepath.Join(rootAbs, versionDir.Name(), file.Name()))
			if err != nil {
				return nil, err
			}
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				return nil, fmt.Errorf("unsafe cache entry %q", file.Name())
			}
			result = append(result, CacheEntry{Version: versionDir.Name(), Path: filepath.Join(rootAbs, versionDir.Name(), file.Name()), Bytes: info.Size()})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

// PruneCache removes artifacts whose version is not in keepVersions. Set
// dryRun to true to obtain the exact removal plan without changing the disk.
// It refuses symlinks and removes only empty version directories.
func PruneCache(root string, keepVersions map[string]bool, dryRun bool) ([]CacheEntry, error) {
	entries, err := ListCache(root)
	if err != nil {
		return nil, err
	}
	var removed []CacheEntry
	for _, entry := range entries {
		if keepVersions[entry.Version] {
			continue
		}
		removed = append(removed, entry)
		if !dryRun {
			if err := os.Remove(entry.Path); err != nil {
				return nil, err
			}
		}
	}
	if !dryRun {
		rootAbs, err := validatedCacheRoot(root)
		if err != nil {
			return nil, err
		}
		for _, entry := range removed {
			_ = os.Remove(filepath.Join(rootAbs, entry.Version))
		}
	}
	return removed, nil
}

func validatedCacheRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("runtime cache root is required")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if info, err := os.Lstat(rootAbs); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("runtime cache root cannot be a symlink")
	}
	return rootAbs, nil
}
