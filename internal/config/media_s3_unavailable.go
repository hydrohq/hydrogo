//go:build hydro_no_s3

package config

import "errors"

func validateMediaStorageBuild(driver string) error {
	if driver == "s3" {
		return errors.New("s3 media provider unavailable in this Edition build")
	}
	return nil
}
