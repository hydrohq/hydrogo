package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
)

func TestSyncHookCallerSignsAndReadsTheAnswer(t *testing.T) {
	var gotTopic, gotSignature, gotBody string
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotTopic, gotSignature, gotBody = r.Header.Get("X-Hydro-Event"), r.Header.Get("X-Hydro-Signature"), string(body)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"message":"not today"}`))
	}))
	defer server.Close()
	call := syncHookCaller(config.Config{Hooks: []config.SyncHook{{ID: "ledger", URL: server.URL, Secret: "0123456789abcdef"}}})
	event := contenttypes.HookEvent{Type: "invoices", On: "transition", ID: "r1", From: "draft", To: "posted"}

	got, message, err := call(context.Background(), "ledger", event)
	if err != nil || got != http.StatusOK || gotTopic != "hook.invoices.transition" || !strings.HasPrefix(gotSignature, "sha256=") || !strings.Contains(gotBody, `"to":"posted"`) {
		t.Fatalf("call: status=%d msg=%q err=%v topic=%q sig=%q body=%q", got, message, err, gotTopic, gotSignature, gotBody)
	}
	status = http.StatusUnprocessableEntity
	if got, message, _ := call(context.Background(), "ledger", event); got != status || message != "not today" {
		t.Fatalf("rejection: %d %q", got, message)
	}
	status = http.StatusFound
	if got, _, _ := call(context.Background(), "ledger", event); got != http.StatusFound {
		t.Fatalf("redirects must not be followed, got %d", got)
	}
	if _, _, err := call(context.Background(), "other", event); err == nil {
		t.Fatal("unknown hook accepted")
	}
	if err := checkHookServices(nil, []contenttypes.Definition{{ID: "x", Hooks: []contenttypes.Hook{{On: "create", Service: "ledger"}}}}); err == nil {
		t.Fatal("undefined hook service accepted at startup")
	}
}
