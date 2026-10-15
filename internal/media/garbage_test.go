package media

import (
	"context"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type garbageRepoStub struct {
	items    []GarbageAsset
	keys     []string
	claimed  map[string]bool
	deleted  []string
	restored []string
}

func (r *garbageRepoStub) ListGarbageCandidates(_ context.Context, _ database.Executor, before time.Time, _ int) ([]GarbageAsset, error) {
	var out []GarbageAsset
	for _, item := range r.items {
		if item.Status == "deleting" || item.CreatedAt.Before(before) {
			out = append(out, item)
		}
	}
	return out, nil
}
func (r *garbageRepoStub) ClaimGarbageAsset(_ context.Context, _ database.Executor, id string, _ time.Time) (bool, error) {
	if r.claimed == nil {
		r.claimed = map[string]bool{}
	}
	r.claimed[id] = true
	return true, nil
}
func (r *garbageRepoStub) GarbageObjectKeys(context.Context, database.Executor, string) ([]string, error) {
	return r.keys, nil
}
func (r *garbageRepoStub) RestoreGarbageAsset(_ context.Context, _ database.Executor, id string) error {
	r.restored = append(r.restored, id)
	return nil
}
func (r *garbageRepoStub) DeleteGarbageAsset(_ context.Context, _ database.Executor, id string) error {
	r.deleted = append(r.deleted, id)
	return nil
}

type referencesStub struct {
	sets  []map[string]struct{}
	calls int
}

func (r *referencesStub) ReferencedMediaIDs(context.Context, database.Executor) (map[string]struct{}, error) {
	index := r.calls
	r.calls++
	if index >= len(r.sets) {
		index = len(r.sets) - 1
	}
	return r.sets[index], nil
}

type garbageBlobStub struct{ deleted []string }

func (*garbageBlobStub) Put(context.Context, string, io.Reader, int64, string) error { return nil }
func (*garbageBlobStub) Get(context.Context, string) (io.ReadCloser, error)          { return nil, nil }
func (b *garbageBlobStub) Delete(_ context.Context, key string) error {
	b.deleted = append(b.deleted, key)
	return nil
}

func TestGarbagePlanRequiresSevenDaysAndReferenceAbsence(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "gc.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	refs := &referencesStub{sets: []map[string]struct{}{{"used": {}}}}
	repo := &garbageRepoStub{items: []GarbageAsset{{ID: "unused", CreatedAt: now.Add(-10 * 24 * time.Hour), Status: "ready"}, {ID: "used", CreatedAt: now.Add(-10 * 24 * time.Hour), Status: "ready"}, {ID: "young", CreatedAt: now.Add(-time.Hour), Status: "ready"}}}
	collector := GarbageCollector{DB: db, Repository: repo, References: refs, Retention: 7 * 24 * time.Hour, Now: func() time.Time { return now }}
	items, err := collector.Plan(context.Background())
	if err != nil || len(items) != 1 || items[0].ID != "unused" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestGarbageDeleteRescansReferencesAfterClaimAndDeletesDerivatives(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "gc-delete.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC()
	refs := &referencesStub{sets: []map[string]struct{}{{}, {}, {"orphan": {}}}}
	repo := &garbageRepoStub{items: []GarbageAsset{{ID: "orphan", CreatedAt: now.Add(-8 * 24 * time.Hour), Status: "ready"}}, keys: []string{"master.png", "thumb.png"}}
	blobs := &garbageBlobStub{}
	collector := GarbageCollector{DB: db, Repository: repo, References: refs, BlobStore: blobs, Retention: 7 * 24 * time.Hour}
	deleted, err := collector.Delete(context.Background(), []string{"orphan"})
	if err != nil || deleted != 0 || len(repo.restored) != 1 || len(repo.deleted) != 0 || len(blobs.deleted) != 0 {
		t.Fatalf("deleted=%d restored=%v metadata=%v blobs=%v err=%v", deleted, repo.restored, repo.deleted, blobs.deleted, err)
	}
	refs = &referencesStub{sets: []map[string]struct{}{{}, {}, {}}}
	collector.References = refs
	deleted, err = collector.Delete(context.Background(), []string{"orphan"})
	if err != nil || deleted != 1 || len(blobs.deleted) != 2 || len(repo.deleted) != 1 {
		t.Fatalf("deleted=%d metadata=%v blobs=%v err=%v", deleted, repo.deleted, blobs.deleted, err)
	}
}
