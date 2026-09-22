//go:build !hydro_no_s3

package config

import (
	"strings"
	"testing"
)

func TestLoadMediaS3RequiresEndpointAndBucket(t *testing.T) {
	path := writeConfig(t, "media:\n  storage_driver: s3\n")
	_, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), "s3 media storage requires") {
		t.Fatalf("error = %v", err)
	}
	path = writeConfig(t, "media:\n  storage_driver: s3\n  s3_endpoint: http://127.0.0.1:9000\n  s3_bucket: hydro\n")
	cfg, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Media.StorageDriver != "s3" || cfg.Media.S3Bucket != "hydro" {
		t.Fatalf("media config = %+v", cfg.Media)
	}
	path = writeConfig(t, "media:\n  storage_driver: s3\n  s3_endpoint: http://127.0.0.1:9000\n  s3_bucket: hydro\n")
	cfg, err = Load(path, lookup(map[string]string{
		"HYDRO_DATABASE_DSN":              "file:test.sqlite",
		"HYDRO_MEDIA_S3_FORCE_PATH_STYLE": "true",
		"HYDRO_MEDIA_CDN_BASE_URL":        "https://cdn.example.test/media",
	}), Overrides{})
	if err != nil || !cfg.Media.S3ForcePathStyle || cfg.Media.CDNBaseURL != "https://cdn.example.test/media" {
		t.Fatalf("extended media config = %+v, err=%v", cfg.Media, err)
	}
	for _, base := range []string{"https://cdn.example.test/media?cache=1", "https://user:secret@cdn.example.test/media"} {
		path = writeConfig(t, "media:\n  storage_driver: s3\n  s3_endpoint: http://127.0.0.1:9000\n  s3_bucket: hydro\n  cdn_base_url: "+base+"\n")
		if _, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{}); err == nil {
			t.Fatalf("unsafe CDN base accepted: %q", base)
		}
	}
}
