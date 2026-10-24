package main

import (
	"os"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/config"
)

func TestValidateProductionOriginRequiresPublicOriginInProduction(t *testing.T) {
	t.Setenv("HYDRO_ENV", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("GO_ENV", "")

	empty := config.Config{}
	if err := validateProductionOrigin(empty); err != nil {
		t.Fatalf("dev/unset environment with no public_origin must not fail: %v", err)
	}

	t.Setenv("HYDRO_ENV", "production")
	if err := validateProductionOrigin(empty); err == nil {
		t.Fatal("production with an empty public_origin must be flagged")
	}

	withOrigin := config.Config{}
	withOrigin.HTTP.PublicOrigin = "https://shop.example.test"
	if err := validateProductionOrigin(withOrigin); err != nil {
		t.Fatalf("production with public_origin set: %v", err)
	}

	t.Setenv("HYDRO_ENV", "")
	t.Setenv("APP_ENV", "prod")
	if err := validateProductionOrigin(empty); err == nil {
		t.Fatal("APP_ENV=prod must also be treated as production")
	}
	_ = os.Unsetenv("APP_ENV")
}
