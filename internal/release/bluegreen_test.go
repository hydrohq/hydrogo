package release

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/testhttp"
)

func TestTrafficPublishesOnlyGatedSlotAndRollsBack(t *testing.T) {
	traffic, err := NewTraffic(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("blue")) }), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("green")) }))
	if err != nil {
		t.Fatal(err)
	}
	deployment, err := NewDeployment(SlotBlue)
	if err != nil {
		t.Fatal(err)
	}
	if err := traffic.Switch(deployment); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("ungated traffic switch error = %v", err)
	}
	for _, step := range []func() error{deployment.PrepareGreen, deployment.MarkReady, func() error { return deployment.Warmup(true) }, func() error { return deployment.Switch(true) }} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if err := traffic.Switch(deployment); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	traffic.ServeHTTP(res, req)
	if res.Body.String() != "green" {
		t.Fatalf("active response = %q", res.Body.String())
	}
	if err := deployment.DrainBlue(); err != nil {
		t.Fatal(err)
	}
	if err := deployment.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := traffic.Rollback(deployment); err != nil {
		t.Fatal(err)
	}
	res = httptest.NewRecorder()
	traffic.ServeHTTP(res, req)
	if res.Body.String() != "blue" {
		t.Fatalf("rollback response = %q", res.Body.String())
	}
}

func TestTrafficSwitchRemainsBoundedUnderConcurrentRequests(t *testing.T) {
	traffic, err := NewTraffic(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("blue")) }), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("green")) }))
	if err != nil {
		t.Fatal(err)
	}
	deployment, _ := NewDeployment(SlotBlue)
	_ = deployment.PrepareGreen()
	_ = deployment.MarkReady()
	_ = deployment.Warmup(true)
	_ = deployment.Switch(true)
	if err := traffic.Switch(deployment); err != nil {
		t.Fatal(err)
	}
	var failures atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := httptest.NewRecorder()
			traffic.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
			if res.Code != http.StatusOK || res.Body.String() != "green" {
				failures.Add(1)
			}
		}()
	}
	wg.Wait()
	if failures.Load() != 0 {
		t.Fatalf("concurrent green requests failed: %d", failures.Load())
	}
}

func TestTrafficWorksBehindReverseProxyHandlers(t *testing.T) {
	const blueURL, greenURL = "http://blue.test", "http://green.test"
	client := testhttp.Client(testhttp.Handlers{
		"blue.test":  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("blue-proxy")) }),
		"green.test": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("green-proxy")) }),
	})
	proxyFor := func(raw string) http.Handler {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		proxy := httputil.NewSingleHostReverseProxy(u)
		proxy.Transport = client.Transport
		return proxy
	}
	traffic, err := NewTraffic(proxyFor(blueURL), proxyFor(greenURL))
	if err != nil {
		t.Fatal(err)
	}
	deployment, _ := NewDeployment(SlotBlue)
	_ = deployment.PrepareGreen()
	_ = deployment.MarkReady()
	_ = deployment.Warmup(true)
	_ = deployment.Switch(true)
	if err := traffic.Switch(deployment); err != nil {
		t.Fatal(err)
	}
	res := httptest.NewRecorder()
	traffic.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "http://proxy/health", nil))
	if res.Code != http.StatusOK || res.Body.String() != "green-proxy" {
		t.Fatalf("proxied green response = %d %q", res.Code, res.Body.String())
	}
}

func TestBlueGreenRequiresReadinessWarmupAndRollbackSafety(t *testing.T) {
	deployment, err := NewDeployment(SlotBlue)
	if err != nil {
		t.Fatal(err)
	}
	if err := deployment.Switch(true); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("switch before readiness error = %v", err)
	}
	for _, step := range []func() error{deployment.PrepareGreen, deployment.MarkReady} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if err := deployment.Warmup(false); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("warmup without retained assets error = %v", err)
	}
	if err := deployment.Warmup(true); err != nil {
		t.Fatal(err)
	}
	if err := deployment.Switch(false); !errors.Is(err, ErrNotRollbackSafe) {
		t.Fatalf("unsafe switch error = %v", err)
	}
	if err := deployment.Switch(true); err != nil {
		t.Fatal(err)
	}
	if err := deployment.DrainBlue(); err != nil {
		t.Fatal(err)
	}
	if err := deployment.Rollback(); err != nil {
		t.Fatal(err)
	}
	if deployment.ActiveSlot != SlotBlue || deployment.Phase != PhaseStable {
		t.Fatalf("rollback state = %#v", deployment)
	}
}

func TestBlueGreenRejectsInvalidStartingSlot(t *testing.T) {
	if _, err := NewDeployment(Slot("unknown")); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("invalid slot error = %v", err)
	}
}
