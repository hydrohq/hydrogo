package linkcheck

import (
	"net/http"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/testhttp"
)

func TestManagerRunsBoundedTenantScopedScan(t *testing.T) {
	const origin = "http://manager-site.test"
	client := testhttp.Client(testhttp.Handlers{"manager-site.test": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<title>Home</title><a href="/missing">missing</a>`))
			return
		}
		http.NotFound(w, r)
	})})
	manager := NewManager(Options{MaxPages: 10, MaxTargets: 20, RequestDelay: time.Millisecond, Client: client})
	job, err := manager.Start("tenant-a", origin)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := manager.Get("tenant-b", job.ID); ok {
		t.Fatal("another tenant can view scan")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current, ok := manager.Get("tenant-a", job.ID)
		if !ok {
			t.Fatal("scan disappeared")
		}
		if !current.Running {
			if current.Failed || len(current.Findings) != 1 || current.Findings[0].AnchorText != "missing" || current.Findings[0].SourceTitle != "Home" {
				t.Fatalf("result=%#v", current)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("scan did not finish")
}
