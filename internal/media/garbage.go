package media

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/pathguard"
)

var ErrGarbageCollectorUnavailable = errors.New("media garbage collector is unavailable")

type GarbageAsset struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

type GarbageRepository interface {
	ListGarbageCandidates(context.Context, database.Executor, time.Time, int) ([]GarbageAsset, error)
	ClaimGarbageAsset(context.Context, database.Executor, string, time.Time) (bool, error)
	GarbageObjectKeys(context.Context, database.Executor, string) ([]string, error)
	RestoreGarbageAsset(context.Context, database.Executor, string) error
	DeleteGarbageAsset(context.Context, database.Executor, string) error
}

type MediaReferenceScanner interface {
	ReferencedMediaIDs(context.Context, database.Executor) (map[string]struct{}, error)
}

type GarbageCollector struct {
	DB         database.Executor
	Repository GarbageRepository
	References MediaReferenceScanner
	BlobStore  BlobStore
	StorageDir string
	Retention  time.Duration
	Now        func() time.Time
}

func (g GarbageCollector) Plan(ctx context.Context) ([]GarbageAsset, error) {
	if g.Repository == nil || g.References == nil || database.RequireExecutor(g.DB) != nil || g.Retention < 7*24*time.Hour {
		return nil, ErrGarbageCollectorUnavailable
	}
	refs, err := g.References.ReferencedMediaIDs(ctx, g.DB)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if g.Now != nil {
		now = g.Now().UTC()
	}
	items, err := g.Repository.ListGarbageCandidates(ctx, g.DB, now.Add(-g.Retention), 200)
	if err != nil {
		return nil, err
	}
	out := make([]GarbageAsset, 0, len(items))
	for _, item := range items {
		if _, referenced := refs[item.ID]; !referenced {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (g GarbageCollector) Delete(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return 0, ErrGarbageCollectorUnavailable
	}
	plan, err := g.Plan(ctx)
	if err != nil {
		return 0, err
	}
	allowed := make(map[string]GarbageAsset, len(plan))
	for _, item := range plan {
		allowed[item.ID] = item
	}
	refs, err := g.References.ReferencedMediaIDs(ctx, g.DB)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, id := range ids {
		item, ok := allowed[id]
		if !ok {
			continue
		}
		if _, referenced := refs[id]; referenced {
			if item.Status == "deleting" {
				if err := g.Repository.RestoreGarbageAsset(ctx, g.DB, id); err != nil {
					return deleted, err
				}
			}
			continue
		}
		if item.Status != "deleting" {
			claimed, claimErr := g.Repository.ClaimGarbageAsset(ctx, g.DB, id, item.CreatedAt)
			if claimErr != nil {
				return deleted, claimErr
			}
			if !claimed {
				continue
			}
		}
		// Claiming blocks the normal media delivery path; scan once more to
		// catch a reference saved after the initial preview.
		refs, scanErr := g.References.ReferencedMediaIDs(ctx, g.DB)
		if scanErr != nil {
			return deleted, scanErr
		}
		if _, referenced := refs[id]; referenced {
			if err := g.Repository.RestoreGarbageAsset(ctx, g.DB, id); err != nil {
				return deleted, err
			}
			continue
		}
		keys, keyErr := g.Repository.GarbageObjectKeys(ctx, g.DB, id)
		if keyErr != nil {
			return deleted, keyErr
		}
		if err := g.deleteObjects(ctx, keys); err != nil {
			// Keep the deleting state so a later confirmed pass can retry safely.
			return deleted, err
		}
		if err := g.Repository.DeleteGarbageAsset(ctx, g.DB, id); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func (g GarbageCollector) deleteObjects(ctx context.Context, keys []string) error {
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if g.BlobStore != nil {
			if err := g.BlobStore.Delete(ctx, key); err != nil {
				return err
			}
			continue
		}
		if g.StorageDir == "" || filepath.IsAbs(key) {
			return ErrGarbageCollectorUnavailable
		}
		root, err := filepath.Abs(g.StorageDir)
		if err != nil {
			return err
		}
		path := filepath.Join(root, filepath.Clean(filepath.FromSlash(key)))
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return ErrInvalidStorageKey
		}
		if err := pathguard.RejectSymlinkComponents(filepath.Dir(path)); err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return ErrInvalidStorageKey
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}
