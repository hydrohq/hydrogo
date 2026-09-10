package transport

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

type mediaReaderStub struct{ asset media.Asset }

func (s mediaReaderStub) Get(context.Context, database.Executor, string) (media.Asset, error) {
	return s.asset, nil
}

func TestWithPublicMediaAddsOnlyControlledMediaURL(t *testing.T) {
	h := Handler{Media: mediaReaderStub{asset: media.Asset{ID: "asset-1", Status: "ready", Visibility: media.VisibilityPublic}}}
	block := pages.Block{Type: "hero", Payload: json.RawMessage(`{"title":"Hero","media_asset_id":"asset-1"}`)}
	got, err := h.withPublicMedia(context.Background(), block)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	if err := json.Unmarshal(got.Payload, &payload); err != nil || payload["media_url"] != "/media/asset-1" {
		t.Fatalf("payload = %s, err = %v", got.Payload, err)
	}
}

func TestWithPublicMediaDoesNotExposePrivateAsset(t *testing.T) {
	h := Handler{Media: mediaReaderStub{asset: media.Asset{ID: "asset-1", Status: "ready", Visibility: media.VisibilityPrivate}}}
	block := pages.Block{Type: "hero", Payload: json.RawMessage(`{"title":"Hero","media_asset_id":"asset-1"}`)}
	got, err := h.withPublicMedia(context.Background(), block)
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Payload) != string(block.Payload) {
		t.Fatalf("private asset was exposed: %s", got.Payload)
	}
}
