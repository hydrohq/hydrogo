//go:build !windows

package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

type fileLock struct {
	file *os.File
}

func acquireFileLock(ctx context.Context, lockPath string, timeout time.Duration) (*fileLock, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file %s: %w", lockPath, err)
	}
	deadline := time.Now().Add(timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return &fileLock{file: f}, nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			_ = f.Close()
			return nil, fmt.Errorf("flock %s: %w", lockPath, err)
		}
		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, fmt.Errorf("timeout waiting for sqlite migration lock on %s", lockPath)
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (l *fileLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	defer l.file.Close()
	_ = unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	return nil
}
