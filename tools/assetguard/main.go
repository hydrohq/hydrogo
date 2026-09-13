// Command assetguard rejects mixed cache-buster versions for theme assets.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var assetPattern = regexp.MustCompile(`theme\.(?:css|js)\?v=([A-Za-z0-9_-]+)`)

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	versions := map[string][]string{}
	err := filepath.WalkDir(*root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || (filepath.Ext(path) != ".go" && filepath.Ext(path) != ".html") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range assetPattern.FindAllStringSubmatch(string(data), -1) {
			versions[match[1]] = append(versions[match[1]], path)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	keys := make([]string, 0, len(versions))
	for key := range versions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > 1 {
		fmt.Fprintln(os.Stderr, "assetguard: mixed theme asset versions:")
		for _, key := range keys {
			fmt.Fprintf(os.Stderr, " - %s: %s\n", key, strings.Join(versions[key], ", "))
		}
		os.Exit(1)
	}
	if len(keys) == 0 {
		fmt.Fprintln(os.Stderr, "assetguard: no theme assets found")
		os.Exit(1)
	}
	fmt.Printf("assetguard: OK version=%s references=%d\n", keys[0], len(versions[keys[0]]))
}
