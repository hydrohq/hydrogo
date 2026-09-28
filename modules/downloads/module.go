package downloads

import (
	"github.com/ardaozyildirim/hydrogo/extension"
	modulecontract "github.com/ardaozyildirim/hydrogo/internal/modules"
)

type Extension struct{}

func (Extension) Name() string                    { return "downloads" }
func (Extension) Routes() []extension.ModuleRoute { return nil }
func (Extension) ModuleManifest() modulecontract.Manifest {
	return modulecontract.Manifest{ID: "downloads", Migrations: []string{"202609280016_sectoral_downloads"}}
}
