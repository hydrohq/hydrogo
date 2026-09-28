package transport

import (
	"context"
	"errors"

	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

var errUnavailableMediaReference = errors.New("media reference is unavailable")

func (h Handler) validateNewsMedia(ctx context.Context, item news.Item) error {
	refs := media.ReferencedIDs(item.Body + item.CoverMediaID)
	if item.CoverMediaID != "" {
		refs[item.CoverMediaID] = struct{}{}
	}
	for id := range refs {
		if h.MediaLookup == nil {
			return errUnavailableMediaReference
		}
		asset, err := h.MediaLookup.Get(ctx, h.DB, id)
		if err != nil || asset.Status != "ready" {
			return errUnavailableMediaReference
		}
	}
	return nil
}
