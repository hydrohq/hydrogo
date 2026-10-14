package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func hasTemplSources(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".tmp", "data", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".templ") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func resolveTemplBinary(ctx context.Context, root string) (string, error) {
	binDir := filepath.Join(root, ".tmp", "dev", "bin")
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		return "", err
	}
	name := "templ"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(binDir, name)
	if info, err := os.Stat(binary); err == nil && info.Mode().IsRegular() {
		return binary, nil
	}
	command := exec.CommandContext(ctx, "go", "install", templCLI)
	command.Dir = root
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	command.Env = append(os.Environ(), "GOBIN="+binDir)
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("install pinned development tool %s: %w", templCLI, err)
	}
	return binary, nil
}

func runTemplGenerate(ctx context.Context, root, binary string, stdout, stderr io.Writer) error {
	command := exec.CommandContext(ctx, binary, "generate")
	command.Dir = root
	command.Stdout, command.Stderr = stdout, stderr
	return command.Run()
}

func startTemplWatcher(ctx context.Context, root, binary string) error {
	command := exec.CommandContext(ctx, binary, "generate", "-watch")
	command.Dir = root
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Start(); err != nil {
		return err
	}
	go func() {
		if err := command.Wait(); err != nil && ctx.Err() == nil {
			_, _ = fmt.Fprintf(os.Stderr, "templ watcher stopped: %v\n", err)
		}
	}()
	return nil
}
