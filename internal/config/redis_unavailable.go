//go:build hydro_no_redis

package config

import (
	"errors"
	"strings"
)

func validateRedisBuild(rawURL string) error {
	if strings.TrimSpace(rawURL) != "" {
		return errors.New("redis is not included in this Edition build")
	}
	return nil
}
