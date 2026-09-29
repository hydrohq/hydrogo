package audit

import (
	"context"
	"encoding/json"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type Writer interface {
	Record(context.Context, database.Executor, string, string, string) error
}

type ChangeWriter interface {
	RecordChange(context.Context, database.Executor, string, string, string, []byte, []byte) error
}

// RecordVersion stores a minimal, non-sensitive before/after version diff when
// supported, while preserving compatibility with existing audit writers.
func RecordVersion(ctx context.Context, tx database.Executor, writer Writer, id, action, target string, before, after int) error {
	if detailed, ok := writer.(ChangeWriter); ok {
		oldState, err := json.Marshal(struct {
			Version int `json:"version"`
		}{before})
		if err != nil {
			return err
		}
		newState, err := json.Marshal(struct {
			Version int `json:"version"`
		}{after})
		if err != nil {
			return err
		}
		return detailed.RecordChange(ctx, tx, id, action, target, oldState, newState)
	}
	return writer.Record(ctx, tx, id, action, target)
}
