package media

import (
	"errors"
	"strings"
	"unicode"
)

func isQuarantineKey(parts []string) bool {
	if len(parts) >= 2 && parts[0] == "quarantine" {
		return true
	}
	return len(parts) >= 4 && parts[0] == "tenants" && parts[1] != "" && parts[2] == "quarantine"
}

func validateObjectKey(raw string) (string, []string, error) {
	key := strings.TrimSpace(raw)
	if key == "" || len(key) > 1024 {
		return "", nil, errors.New("invalid object key")
	}
	for _, r := range key {
		if unicode.IsControl(r) || r == '\\' {
			return "", nil, errors.New("invalid object key")
		}
	}
	parts := strings.Split(key, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", nil, errors.New("invalid object key")
		}
	}
	return key, parts, nil
}
