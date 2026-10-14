package main

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type fileChange struct {
	path    string
	rebuild bool
}

func watch(ctx context.Context, root string, interval time.Duration, changed func(fileChange)) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		// Keep the development command usable on unusual filesystems where
		// native notifications are unavailable.
		pollWatch(ctx, root, interval, changed)
		return
	}
	defer watcher.Close()

	if err := addWatchDirectories(watcher, root); err != nil {
		pollWatch(ctx, root, interval, changed)
		return
	}

	pending := make(map[string]struct{})
	var debounce *time.Timer
	var debounceC <-chan time.Time
	flush := func() {
		if len(pending) == 0 {
			return
		}
		paths := make([]string, 0, len(pending))
		for path := range pending {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		assetOnly := true
		for _, path := range paths {
			if !isAsset(path) {
				assetOnly = false
				break
			}
		}
		pending = make(map[string]struct{})
		changed(fileChange{path: paths[0], rebuild: !assetOnly})
	}

	for {
		select {
		case <-ctx.Done():
			if debounce != nil {
				debounce.Stop()
			}
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			if event.Op&fsnotify.Create != 0 {
				if info, statErr := os.Stat(event.Name); statErr == nil && info.IsDir() && shouldWatchDirectory(event.Name, root) {
					_ = addWatchDirectory(watcher, event.Name)
					continue
				}
			}
			if !watched(event.Name) {
				continue
			}
			pending[event.Name] = struct{}{}
			if debounce == nil {
				debounce = time.NewTimer(150 * time.Millisecond)
			} else {
				if !debounce.Stop() {
					select {
					case <-debounce.C:
					default:
					}
				}
				debounce.Reset(150 * time.Millisecond)
			}
			debounceC = debounce.C
		case <-debounceC:
			debounceC = nil
			flush()
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
			// The next editor save will still be observed. Avoid falling back
			// to a full repository scan on every transient watcher warning.
		}
	}
}

func pollWatch(ctx context.Context, root string, interval time.Duration, changed func(fileChange)) {
	known := scan(root)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := scan(root)
			paths := changedPaths(known, current)
			known = current
			if len(paths) == 0 {
				continue
			}
			assetOnly := true
			for _, path := range paths {
				if !isAsset(path) {
					assetOnly = false
					break
				}
			}
			changed(fileChange{path: paths[0], rebuild: !assetOnly})
		}
	}
}

func addWatchDirectories(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if path != root && !shouldWatchDirectory(path, root) {
			return filepath.SkipDir
		}
		return addWatchDirectory(watcher, path)
	})
}

func addWatchDirectory(watcher *fsnotify.Watcher, path string) error {
	return watcher.Add(path)
}

func shouldWatchDirectory(path, root string) bool {
	if path == root {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		switch part {
		case ".git", ".tmp", "data", "node_modules", "vendor":
			return false
		}
	}
	return true
}

type fileStamp struct {
	modified time.Time
	size     int64
}

func scan(root string) map[string]fileStamp {
	files := make(map[string]fileStamp)
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
		if !watched(path) {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr == nil {
			files[path] = fileStamp{modified: info.ModTime(), size: info.Size()}
		}
		return nil
	})
	return files
}

func watched(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".html", ".css", ".js", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func isAsset(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	if extension != ".css" && extension != ".js" {
		return false
	}
	rel := "/" + filepath.ToSlash(path) + "/"
	// Edition theme files are served from disk and can be reloaded without
	// compiling the Go binary. Core module assets use go:embed and need rebuilds.
	return strings.Contains(rel, "/themes/") && strings.Contains(rel, "/static/")
}

func changedPaths(before, after map[string]fileStamp) []string {
	paths := make([]string, 0)
	for path, stamp := range after {
		if old, ok := before[path]; !ok || old != stamp {
			paths = append(paths, path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}
