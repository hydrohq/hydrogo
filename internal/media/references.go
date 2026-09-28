package media

import (
	"regexp"
	"sort"
	"strings"
)

var mediaPathReference = regexp.MustCompile(`(?:/media/|media_id["']?\s*[:=]\s*["']?)([A-Za-z0-9_-]{8,128})`)

// MissingReferences reports content-addressed media IDs referenced by content
// but absent from the known asset set. Results are stable and deduplicated.
func MissingReferences(content string, known map[string]struct{}) []string {
	missing := map[string]struct{}{}
	for _, match := range mediaPathReference.FindAllStringSubmatch(content, -1) {
		id := strings.TrimSpace(match[1])
		if _, ok := known[id]; !ok {
			missing[id] = struct{}{}
		}
	}
	result := make([]string, 0, len(missing))
	for id := range missing {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}
