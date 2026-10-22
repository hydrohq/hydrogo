package contract_test

import (
	"os"
	"runtime"
	"testing"
)

func TestPostgresContractEnvironmentIsConfiguredInCI(t *testing.T) {
	// The Windows job proves the SQLite path and has no PostgreSQL service; the
	// Linux jobs, which do, are the ones this guard keeps from skipping silently.
	if runtime.GOOS == "windows" {
		return
	}
	if os.Getenv("CI") != "" && os.Getenv("HYDRO_TEST_POSTGRES_URL") == "" {
		t.Fatal("HYDRO_TEST_POSTGRES_URL must be configured in CI; PostgreSQL contract tests must not silently skip")
	}
}
