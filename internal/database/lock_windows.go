//go:build windows

package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

type fileLock struct {
	file *os.File
}

func acquireFileLock(ctx context.Context, lockPath string, timeout time.Duration) (*fileLock, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file %s: %w", lockPath, err)
	}
	h := windows.Handle(f.Fd())
	deadline := time.Now().Add(timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	var overlapped windows.Overlapped
	for {
		err := windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &overlapped)
		if err == nil {
			return &fileLock{file: f}, nil
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
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(l.file.Fd()), 0, 1, 0, &overlapped)
}
