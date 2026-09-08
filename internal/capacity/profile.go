// Package capacity defines topology profiles and their bounded local budgets.
// The values are starting defaults for repeatable measurements, not hardware
// or throughput guarantees.
package capacity

import (
	"errors"
	"fmt"
	"time"
)

type Profile string

const (
	Compact     Profile = "compact"
	Standard    Profile = "standard"
	Distributed Profile = "distributed"
)

type Budget struct {
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	CacheEntries      int
	CacheTTL          time.Duration
	WorkerConcurrency int
	QueueEntries      int
	HTTPBodyBytes     int64
	MediaMaxBytes     int64
	ShutdownTimeout   time.Duration
}

func (p Profile) Validate() error {
	switch p {
	case Compact, Standard, Distributed:
		return nil
	default:
		return fmt.Errorf("unsupported capacity profile %q", p)
	}
}

func Defaults(p Profile) (Budget, error) {
	if err := p.Validate(); err != nil {
		return Budget{}, err
	}
	switch p {
	case Compact:
		return Budget{DBMaxOpenConns: 1, DBMaxIdleConns: 1, CacheEntries: 256, CacheTTL: 30 * time.Second, WorkerConcurrency: 1, QueueEntries: 10000, HTTPBodyBytes: 64 << 10, MediaMaxBytes: 10 << 20, ShutdownTimeout: 10 * time.Second}, nil
	case Standard:
		return Budget{DBMaxOpenConns: 10, DBMaxIdleConns: 5, CacheEntries: 1024, CacheTTL: 30 * time.Second, WorkerConcurrency: 4, QueueEntries: 100000, HTTPBodyBytes: 1 << 20, MediaMaxBytes: 50 << 20, ShutdownTimeout: 20 * time.Second}, nil
	case Distributed:
		return Budget{DBMaxOpenConns: 50, DBMaxIdleConns: 25, CacheEntries: 4096, CacheTTL: 30 * time.Second, WorkerConcurrency: 16, QueueEntries: 500000, HTTPBodyBytes: 4 << 20, MediaMaxBytes: 100 << 20, ShutdownTimeout: 30 * time.Second}, nil
	default:
		return Budget{}, errors.New("capacity profile defaults are unavailable")
	}
}

func (b Budget) Validate() error {
	if b.DBMaxOpenConns < 1 || b.DBMaxIdleConns < 0 || b.DBMaxIdleConns > b.DBMaxOpenConns || b.CacheEntries < 1 || b.CacheTTL <= 0 || b.WorkerConcurrency < 1 || b.QueueEntries < 1 || b.HTTPBodyBytes < 1 || b.MediaMaxBytes < 1 || b.ShutdownTimeout <= 0 {
		return errors.New("capacity budget is invalid")
	}
	return nil
}
