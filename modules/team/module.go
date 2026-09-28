package team

import (
	"github.com/ardaozyildirim/hydrogo/extension"
	modulecontract "github.com/ardaozyildirim/hydrogo/internal/modules"
)

type Extension struct{}

func (Extension) Name() string                    { return "team" }
func (Extension) Routes() []extension.ModuleRoute { return nil }
func (Extension) ModuleManifest() modulecontract.Manifest {
	return modulecontract.Manifest{ID: "team", Migrations: []string{"202609280009_sectoral_team"}}
}
