//go:build hydro_no_redis

package config

import (
	"strings"
	"testing"
)

func TestConfigRejectsRedisWhenClientIsNotLinked(t *testing.T) {
	_, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN": "file:test.sqlite",
		"HYDRO_REDIS_URL":    "redis://127.0.0.1:6379/0",
	}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), "redis is not included") {
		t.Fatalf("Redis configuration error = %v, want unavailable-provider error", err)
	}
}
