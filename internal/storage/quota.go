// Package storage contains bounded filesystem protection primitives.
package storage

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

var ErrQuotaExceeded = errors.New("storage capacity is below the configured reserve")

type SpaceProbe func(path string) (uint64, error)

type QuotaGuard struct {
	MinFreeBytes uint64
	Probe        SpaceProbe
}

// Check reserves MinFreeBytes for the host and additionalBytes for the pending
// write. A zero reserve disables the guard.
func (q QuotaGuard) Check(path string, additionalBytes int64) error {
	if q.MinFreeBytes == 0 {
		return nil
	}
	if strings.TrimSpace(path) == "" || additionalBytes < 0 {
		return errors.New("storage path and non-negative write estimate are required")
	}
	probe := q.Probe
	if probe == nil {
		probe = AvailableBytes
	}
	free, err := probe(path)
	if err != nil {
		return fmt.Errorf("inspect available storage: %w", err)
	}
	additional := uint64(additionalBytes)
	if additional > math.MaxUint64-q.MinFreeBytes || free < q.MinFreeBytes+additional {
		return ErrQuotaExceeded
	}
	return nil
}
