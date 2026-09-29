package events

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/extension"
	modulecontract "github.com/ardaozyildirim/hydrogo/internal/modules"
)

type Extension struct{ Calendar http.Handler }

func (Extension) Name() string { return "events" }
func (e Extension) Routes() []extension.ModuleRoute {
	if e.Calendar == nil {
		return nil
	}
	return []extension.ModuleRoute{{Pattern: "GET /events/{slug}", Handler: e.Calendar, Public: true}}
}
func (e Extension) ModuleManifest() modulecontract.Manifest {
	manifest := modulecontract.Manifest{ID: "events", Migrations: []string{"202609280015_sectoral_events"}}
	if e.Calendar != nil {
		manifest.Routes = []modulecontract.RouteSpec{{Pattern: "GET /events/{slug}", Public: true}}
	}
	return manifest
}
