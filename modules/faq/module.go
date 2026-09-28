package faq

import (
	"github.com/ardaozyildirim/hydrogo/extension"
	modulecontract "github.com/ardaozyildirim/hydrogo/internal/modules"
)

type Extension struct{}

func (Extension) Name() string                    { return "faq" }
func (Extension) Routes() []extension.ModuleRoute { return nil }
func (Extension) ModuleManifest() modulecontract.Manifest {
	return modulecontract.Manifest{ID: "faq", Migrations: []string{"202609280018_sectoral_faq"}}
}
