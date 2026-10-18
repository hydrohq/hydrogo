package transport

import (
	"context"
	"errors"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

var errUnavailableMediaReference = errors.New("media reference is unavailable")

func (h Handler) validateNewsMedia(ctx context.Context, item news.Item) error {
	refs := media.ReferencedIDs(item.Body)
	if item.CoverMediaID != "" && !isExternalMedia(item.CoverMediaID) {
		refs[item.CoverMediaID] = struct{}{}
	}
	for id := range refs {
		if isExternalMedia(id) {
			continue
		}
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

func isExternalMedia(val string) bool {
	return strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") || strings.HasPrefix(val, "demo-")
}
