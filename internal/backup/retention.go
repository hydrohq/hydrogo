package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SnapshotRetention controls local snapshot pruning. Pinned snapshot
// directory names are always retained, even when they exceed Keep.
type SnapshotRetention struct {
	Keep   int
	Pinned map[string]struct{}
}

// PlanSnapshotPrune returns snapshot directory names that are safe to remove.
// It never mutates the filesystem.
func PlanSnapshotPrune(root string, policy SnapshotRetention) ([]string, error) {
	if policy.Keep < 1 {
		return nil, errors.New("snapshot retention keep must be positive")
	}
	root, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("snapshot retention root cannot be a symlink: %s", root)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("snapshot retention root is not a directory: %s", root)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	type candidate struct {
		name      string
		createdAt int64
	}
	candidates := make([]candidate, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		manifest, err := ReadManifest(filepath.Join(root, name, "manifest.json"))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("snapshot %s: %w", name, err)
		}
		if _, pinned := policy.Pinned[name]; pinned {
			continue
		}
		candidates = append(candidates, candidate{name: name, createdAt: manifest.CreatedAt.UnixNano()})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].createdAt == candidates[j].createdAt {
			return candidates[i].name < candidates[j].name
		}
		return candidates[i].createdAt > candidates[j].createdAt
	})
	keepUnpinned := policy.Keep
	if keepUnpinned > len(candidates) {
		keepUnpinned = len(candidates)
	}
	planned := make([]string, 0)
	for _, item := range candidates[keepUnpinned:] {
		planned = append(planned, item.name)
	}
	sort.Strings(planned)
	return planned, nil
}

// PruneSnapshots removes only directories containing a valid manifest.json.
// The newest Keep snapshots and all pinned snapshots remain available. A
// malformed or partially written snapshot fails closed instead of being
// treated as disposable data.
func PruneSnapshots(root string, policy SnapshotRetention) ([]string, error) {
	root, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return nil, err
	}
	planned, err := PlanSnapshotPrune(root, policy)
	if err != nil {
		return nil, err
	}
	removed := make([]string, 0, len(planned))
	for _, name := range planned {
		if err := os.RemoveAll(filepath.Join(root, name)); err != nil {
			return removed, fmt.Errorf("remove snapshot %s: %w", name, err)
		}
		removed = append(removed, name)
	}
	return removed, nil
}
