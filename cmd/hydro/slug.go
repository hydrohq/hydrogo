package main

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

var slugRegex = regexp.MustCompile(`[^a-z0-9-]+`)

func generateCleanSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, "ı", "i")
	s = strings.ReplaceAll(s, "ğ", "g")
	s = strings.ReplaceAll(s, "ü", "u")
	s = strings.ReplaceAll(s, "ş", "s")
	s = strings.ReplaceAll(s, "ö", "o")
	s = strings.ReplaceAll(s, "ç", "c")
	s = slugRegex.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		var b [4]byte
		_, _ = rand.Read(b[:])
		s = "magaza-" + hex.EncodeToString(b[:])
	}
	return s
}
