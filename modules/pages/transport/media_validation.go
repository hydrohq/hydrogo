package transport

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

var errUnavailableMediaReference = errors.New("media reference is unavailable")

func (h Handler) validatePageMedia(ctx context.Context, page pages.Page) error {
	blocks, err := json.Marshal(page.Blocks)
	if err != nil {
		return errUnavailableMediaReference
	}
	for id := range media.ReferencedIDs(page.Body + string(blocks)) {
		if h.Media == nil {
			return errUnavailableMediaReference
		}
		asset, err := h.Media.Get(ctx, h.DB, id)
		if err != nil || asset.Status != "ready" {
			return errUnavailableMediaReference
		}
	}
	return nil
}
