package transport

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestSummarizePagesDoesNotExposeDetailFields(t *testing.T) {
	summaries := summarizePages([]pages.Page{{ID: "page-1", Locale: "tr", Slug: "about", Title: "About", Body: "private draft body", Blocks: []pages.Block{{Type: "hero"}}, Revision: 2, PublicationGeneration: 3, UpdatedAt: time.Unix(1, 0)}})
	data, err := json.Marshal(summaries)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private draft body", "blocks", "body", "status", "created_at"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("summary leaked %q: %s", forbidden, data)
		}
	}
	if !strings.Contains(string(data), `"publication_generation":3`) {
		t.Fatalf("summary missing generation: %s", data)
	}
}
