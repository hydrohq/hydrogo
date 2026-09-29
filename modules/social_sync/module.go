package socialsync

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/extension"
	modulecontract "github.com/ardaozyildirim/hydrogo/internal/modules"
)

type Extension struct {
	Handler *Handler
}

func (e Extension) Name() string {
	return "social_sync"
}

func (e Extension) Routes() []extension.ModuleRoute {
	if e.Handler == nil {
		return nil
	}
	return []extension.ModuleRoute{
		{
			Pattern: "GET /api/v1/social/posts",
			Handler: http.HandlerFunc(e.Handler.HandleGetPosts),
		},
		{
			Pattern: "GET /api/v1/social/reviews",
			Handler: http.HandlerFunc(e.Handler.HandleGetReviews),
		},
	}
}

func (e Extension) ModuleManifest() modulecontract.Manifest {
	return modulecontract.Manifest{
		ID:         "social_sync",
		Migrations: []string{},
	}
}
