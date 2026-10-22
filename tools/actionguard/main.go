// Command actionguard rejects mutable GitHub Actions references in repository
// workflows. External actions must use a full 40-character commit SHA, and the
// same action at the same version comment must use the same SHA everywhere (a
// one-character typo in a pin otherwise only shows when that workflow runs).
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := ".github/workflows"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	if err := checkWorkflows(root); err != nil {
		fmt.Fprintln(os.Stderr, "actionguard:", err)
		os.Exit(1)
	}
}

func checkWorkflows(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	pins := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".yml" && filepath.Ext(entry.Name()) != ".yaml") {
			continue
		}
		if err := checkFile(filepath.Join(root, entry.Name()), pins); err != nil {
			return err
		}
	}
	return nil
}

// checkFile validates one workflow. pins remembers, per "action # version
// comment", the SHA and place first seen, so a differing SHA elsewhere is caught.
func checkFile(path string, pins map[string]string) error {
	file, err := os.Open(path) // #nosec G304 ,G703-- developer tool reading the workflow files it walked
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var value string
		switch {
		case strings.HasPrefix(line, "- uses:"):
			value = strings.TrimSpace(strings.TrimPrefix(line, "- uses:"))
		case strings.HasPrefix(line, "uses:"):
			value = strings.TrimSpace(strings.TrimPrefix(line, "uses:"))
		default:
			continue
		}
		if strings.HasPrefix(value, "./") {
			continue
		}
		version := ""
		if comment := strings.IndexByte(value, '#'); comment >= 0 {
			version = strings.TrimSpace(value[comment+1:])
			value = strings.TrimSpace(value[:comment])
		}
		at := strings.LastIndexByte(value, '@')
		if at <= 0 || !isFullSHA(value[at+1:]) {
			return fmt.Errorf("%s:%d: external action must use a full 40-character commit SHA", path, lineNumber)
		}
		if version != "" {
			key := value[:at] + " # " + version
			here := fmt.Sprintf("%s@%s (%s:%d)", value[:at], value[at+1:], path, lineNumber)
			if first, seen := pins[key]; !seen {
				pins[key] = here
			} else if !strings.HasPrefix(first, value[:at]+"@"+value[at+1:]+" ") {
				return fmt.Errorf("%s:%d: %s is pinned to a different commit than %s", path, lineNumber, key, first)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func isFullSHA(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
