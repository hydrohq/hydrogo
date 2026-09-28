package media

import (
	"regexp"
	"sort"
	"strings"
)

var mediaPathReference = regexp.MustCompile(`/media/([A-Za-z0-9_-]{8,128})|(?:media|before_image|after_image)_asset_id["']?\s*:\s*["']?([A-Za-z0-9_-]{8,128})|cover_media_id["']?\s*:\s*["']?([A-Za-z0-9_-]{8,128})`)

// MissingReferences reports content-addressed media IDs referenced by content
// but absent from the known asset set. Results are stable and deduplicated.
func MissingReferences(content string, known map[string]struct{}) []string {
	missing := map[string]struct{}{}
	for id := range ReferencedIDs(content) {
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

func ReferencedIDs(content string) map[string]struct{} {
	ids := make(map[string]struct{})
	for _, match := range mediaPathReference.FindAllStringSubmatch(content, -1) {
		id := strings.TrimSpace(match[1])
		if id == "" && len(match) > 2 {
			id = strings.TrimSpace(match[2])
		}
		if id == "" && len(match) > 3 {
			id = strings.TrimSpace(match[3])
		}
		if id != "" {
			ids[id] = struct{}{}
		}
	}
	return ids
}
