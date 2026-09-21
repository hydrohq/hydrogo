package observability

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeExporter struct {
	calls int
	err   error
	seen  []SpanRecord
}

func (f *fakeExporter) Export(_ context.Context, records []SpanRecord) error {
	f.calls++
	f.seen = append(f.seen, records...)
	return f.err
}

func TestExporterAdapterSamplesAndRetriesWithBound(t *testing.T) {
	f := &fakeExporter{err: errors.New("collector unavailable")}
	a := ExporterAdapter{Exporter: f, MaxAttempts: 3, Timeout: time.Second, Backoff: time.Millisecond, Sleep: func(context.Context, time.Duration) error { return nil }, Sample: func(r SpanRecord) bool { return r.Error }}
	err := a.Export(context.Background(), []SpanRecord{{Name: "ok", Path: "/public", Error: false}, {Name: "err", Path: "/admin", Error: true}})
	if err == nil || f.calls != 3 || len(f.seen) != 3 || f.seen[0].Name != "err" {
		t.Fatalf("export err=%v calls=%d seen=%+v", err, f.calls, f.seen)
	}
}

func TestExporterAdapterNoOpWithoutExporter(t *testing.T) {
	if err := (ExporterAdapter{}).Export(context.Background(), []SpanRecord{{Name: "safe"}}); err != nil {
		t.Fatal(err)
	}
}
