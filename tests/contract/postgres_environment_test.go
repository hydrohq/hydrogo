package contract_test

import (
	"os"
	"testing"
)

func TestPostgresContractEnvironmentIsConfiguredInCI(t *testing.T) {
	if os.Getenv("CI") != "" && os.Getenv("HYDRO_TEST_POSTGRES_URL") == "" {
		t.Fatal("HYDRO_TEST_POSTGRES_URL must be configured in CI; PostgreSQL contract tests must not silently skip")
	}
}
