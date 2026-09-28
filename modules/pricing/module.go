package pricing

import (
	"github.com/ardaozyildirim/hydrogo/extension"
	modulecontract "github.com/ardaozyildirim/hydrogo/internal/modules"
)

type Extension struct{}

func (Extension) Name() string                    { return "pricing" }
func (Extension) Routes() []extension.ModuleRoute { return nil }
func (Extension) ModuleManifest() modulecontract.Manifest {
	return modulecontract.Manifest{ID: "pricing", Migrations: []string{"202609280014_sectoral_pricing"}}
}
