package announcements

import (
	"github.com/ardaozyildirim/hydrogo/extension"
	modulecontract "github.com/ardaozyildirim/hydrogo/internal/modules"
)

type Extension struct{}

func (Extension) Name() string                    { return "announcements" }
func (Extension) Routes() []extension.ModuleRoute { return nil }
func (Extension) ModuleManifest() modulecontract.Manifest {
	return modulecontract.Manifest{ID: "announcements", Migrations: []string{"202609280011_sectoral_announcements"}}
}
