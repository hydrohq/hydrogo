package audit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type changeRecorder struct{ before, after map[string]int }

func (r *changeRecorder) Record(context.Context, database.Executor, string, string, string) error {
	return nil
}
func (r *changeRecorder) RecordChange(_ context.Context, _ database.Executor, _, _, _ string, before, after []byte) error {
	if err := json.Unmarshal(before, &r.before); err != nil {
		return err
	}
	return json.Unmarshal(after, &r.after)
}

func TestRecordVersionWritesOnlyVersionMetadata(t *testing.T) {
	recorder := &changeRecorder{}
	if err := RecordVersion(context.Background(), nil, recorder, "id", "page.updated", "page", 2, 3); err != nil {
		t.Fatal(err)
	}
	if recorder.before["version"] != 2 || recorder.after["version"] != 3 || len(recorder.before) != 1 || len(recorder.after) != 1 {
		t.Fatalf("version metadata before=%v after=%v", recorder.before, recorder.after)
	}
}
