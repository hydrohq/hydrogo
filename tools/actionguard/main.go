// Command actionguard rejects mutable GitHub Actions references in repository
// workflows. External actions must use a full 40-character commit SHA.
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
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".yml" && filepath.Ext(entry.Name()) != ".yaml") {
			continue
		}
		if err := checkFile(filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func checkFile(path string) error {
	file, err := os.Open(path)
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
		if comment := strings.IndexByte(value, '#'); comment >= 0 {
			value = strings.TrimSpace(value[:comment])
		}
		at := strings.LastIndexByte(value, '@')
		if at <= 0 || !isFullSHA(value[at+1:]) {
			return fmt.Errorf("%s:%d: external action must use a full 40-character commit SHA", path, lineNumber)
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
