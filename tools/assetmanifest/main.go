// Command assetmanifest emits a deterministic fingerprint manifest for theme assets.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	themev1 "github.com/ardaozyildirim/hydrogo/extension/v1"
)

type asset struct {
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type themeEntry struct {
	Version string           `json:"version"`
	Assets  map[string]asset `json:"assets"`
}

type manifest struct {
	Version int                   `json:"version"`
	Themes  map[string]themeEntry `json:"themes"`
}

func main() {
	root := flag.String("root", "themes", "theme root directory")
	output := flag.String("output", "", "output JSON path; stdout when empty")
	flag.Parse()

	entries, err := build(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, err := json.MarshalIndent(manifest{Version: 1, Themes: entries}, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if *output == "" {
		_, _ = os.Stdout.Write(data)
		return
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func build(root string) (map[string]themeEntry, error) {
	items, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read theme root: %w", err)
	}
	result := make(map[string]themeEntry)
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		name := item.Name()
		bundle, err := themev1.LoadTheme(root, name)
		if err != nil {
			return nil, fmt.Errorf("load theme %q: %w", name, err)
		}
		assets := make(map[string]asset, len(bundle.Manifest.Assets))
		for _, declared := range bundle.Manifest.Assets {
			path := filepath.Join(bundle.Root, filepath.FromSlash(declared))
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", path, err)
			}
			digest := sha256.Sum256(data)
			assets[strings.TrimPrefix(filepath.ToSlash(declared), "static/")] = asset{Bytes: len(data), SHA256: hex.EncodeToString(digest[:])}
		}
		result[name] = themeEntry{Version: bundle.Manifest.Version, Assets: assets}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no themes found in %s", root)
	}
	return result, nil
}
