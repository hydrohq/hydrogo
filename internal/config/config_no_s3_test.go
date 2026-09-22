//go:build hydro_no_s3

package config

import (
	"strings"
	"testing"
)

func TestConfigRejectsS3WhenProviderIsNotLinked(t *testing.T) {
	path := writeConfig(t, "media:\n  storage_driver: s3\n  s3_endpoint: https://s3.example.test\n  s3_bucket: assets\n")
	_, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), "s3 media provider unavailable") {
		t.Fatalf("S3 configuration error = %v, want unavailable-provider error", err)
	}
}
