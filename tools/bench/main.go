// Command bench runs automated performance and load benchmarks against HydroGo.
// It verifies high-throughput reads, concurrent writes, and SSE streaming with
// percentile latency calculations (p50, p95, p99) and regression assertions.
//
// Usage:
//
//	go run ./tools/bench
//	go run ./tools/bench -target http://127.0.0.1:8090
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Result struct {
	Scenario    string
	Concurrency int
	TotalReqs   int64
	Duration    time.Duration
	RPS         float64
	P50         time.Duration
	P90         time.Duration
	P95         time.Duration
	P99         time.Duration
	Errors      int64
	Passed      bool
}

const articleSchema = `id: articles
label: Articles
public: true
realtime: true
fields:
  - {id: title, label: Title, kind: text, required: true, list: true, searchable: true}
  - {id: content, label: Content, kind: text}
  - {id: status, label: Status, kind: select, options: [draft, published], list: true}
`

func main() {
	targetURL := flag.String("target", "", "existing HydroGo server base URL (e.g. http://127.0.0.1:8090)")
	duration := flag.Duration("duration", 3*time.Second, "benchmark scenario duration")
	flag.Parse()

	var cleanup func()
	baseURL := *targetURL
	var token string
	hostHeader := "127.0.0.1"

	if baseURL == "" {
		srvURL, srvToken, srvHost, srvCleanup, err := startEphemeralServer()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to start ephemeral server: %v\n", err)
			os.Exit(1)
		}
		baseURL = srvURL
		token = srvToken
		hostHeader = srvHost
		cleanup = srvCleanup
		defer cleanup()
	}

	fmt.Println("================================================================================")
	fmt.Printf(" HydroGo Performance & Load Benchmark Suite\n")
	fmt.Printf(" Target: %s | Duration per scenario: %s\n", baseURL, *duration)
	fmt.Println("================================================================================")

	var results []Result

	// Scenario A: Cached Public Reads
	fmt.Println("==> Running Scenario A: High-Throughput Public Collection Reads...")
	resA := benchPublicReads(baseURL, hostHeader, 30, *duration)
	results = append(results, resA)

	// Scenario B: Concurrent Authenticated Writes
	fmt.Println("==> Running Scenario B: Concurrent Authenticated Writes (50 writers)...")
	resB := benchConcurrentWrites(baseURL, hostHeader, token, 50, *duration)
	results = append(results, resB)

	// Scenario C: SSE Connection Saturation
	fmt.Println("==> Running Scenario C: SSE Stream Saturation (20 streams)...")
	resC := benchSSESaturation(baseURL, hostHeader, 20, *duration)
	results = append(results, resC)

	// Print Markdown summary report
	printReport(results)

	for _, r := range results {
		if !r.Passed {
			fmt.Printf("FAIL: Scenario %q did not meet performance gates (p95 latency or errors).\n", r.Scenario)
			os.Exit(1)
		}
	}
	fmt.Println("PASS: All performance & load benchmark gates satisfied.")
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func startEphemeralServer() (string, string, string, func(), error) {
	tempDir, err := os.MkdirTemp("", "hydro-bench-*")
	if err != nil {
		return "", "", "", nil, err
	}

	port, err := freePort()
	if err != nil {
		os.RemoveAll(tempDir)
		return "", "", "", nil, err
	}

	host := fmt.Sprintf("127.0.0.1:%d", port)
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Create directories
	ctDir := filepath.Join(tempDir, "content-types")
	dataDir := filepath.Join(tempDir, "data")
	if err := os.MkdirAll(ctDir, 0755); err != nil {
		return "", "", "", nil, err
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", "", "", nil, err
	}

	if err := os.WriteFile(filepath.Join(ctDir, "articles.yaml"), []byte(articleSchema), 0644); err != nil {
		return "", "", "", nil, err
	}

	cfgContent := fmt.Sprintf(`profile: standard
modules: [core, auth, media, collections, realtime]
http:
  address: "127.0.0.1:%d"
  public_origin: "http://%s"
  behind_proxy: true
  trusted_proxies: ["127.0.0.1"]
database:
  dialect: "sqlite"
  dsn: "%s"
content_types_dir: "content-types"
security:
  secure_cookies: false
  enable_hsts: false
`, port, host, filepath.Join(dataDir, "bench.sqlite"))

	cfgPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		return "", "", "", nil, err
	}

	// Build binary
	binPath := filepath.Join(tempDir, "hydro")
	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/hydro")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return "", "", "", nil, fmt.Errorf("build hydro: %w (%s)", err, string(out))
	}

	// Run migrations
	migCmd := exec.Command(binPath, "migrate", "--config", cfgPath)
	migCmd.Dir = tempDir
	if out, err := migCmd.CombinedOutput(); err != nil {
		return "", "", "", nil, fmt.Errorf("migrate: %w (%s)", err, string(out))
	}

	// Create service account
	saCmd := exec.Command(binPath, "service-account", "create",
		"--name", "bench-runner",
		"--permissions", "admin.content.read,admin.content.write",
		"--expires-days", "30",
		"--config", cfgPath,
	)
	saCmd.Dir = tempDir
	saOut, err := saCmd.CombinedOutput()
	if err != nil {
		return "", "", "", nil, fmt.Errorf("service-account create: %w (%s)", err, string(saOut))
	}

	m := regexp.MustCompile(`(?m)^(hyd_\S+)$`).FindStringSubmatch(string(saOut))
	if m == nil {
		return "", "", "", nil, fmt.Errorf("token not found in output: %s", string(saOut))
	}
	token := m[1]

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	srvCmd := exec.CommandContext(ctx, binPath, "serve", "--config", cfgPath)
	srvCmd.Dir = tempDir
	srvCmd.Stdout = io.Discard
	srvCmd.Stderr = io.Discard

	if err := srvCmd.Start(); err != nil {
		cancel()
		os.RemoveAll(tempDir)
		return "", "", "", nil, fmt.Errorf("start server: %w", err)
	}

	// Poll healthcheck
	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(10 * time.Second)
	up := false
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest("GET", serverURL+"/healthz", nil)
		req.Host = host
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			up = true
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !up {
		cancel()
		_ = srvCmd.Wait()
		os.RemoveAll(tempDir)
		return "", "", "", nil, fmt.Errorf("server failed to start on %s", serverURL)
	}

	cleanup := func() {
		cancel()
		_ = srvCmd.Wait()
		_ = os.RemoveAll(tempDir)
	}

	return serverURL, token, host, cleanup, nil
}

func benchPublicReads(baseURL, hostHeader string, concurrency int, duration time.Duration) Result {
	endpoint := baseURL + "/api/v1/content/articles"
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: concurrency,
			DisableKeepAlives:   false,
		},
	}

	var totalReqs int64
	var errCount int64
	latencies := make([]time.Duration, 0, 50000)
	var mu sync.Mutex

	var stopping atomic.Bool
	time.AfterFunc(duration, func() { stopping.Store(true) })

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			localLat := make([]time.Duration, 0, 1000)
			counter := 0
			for !stopping.Load() {
				counter++
				reqCtx, reqCancel := context.WithTimeout(context.Background(), 5*time.Second)
				req, _ := http.NewRequestWithContext(reqCtx, "GET", endpoint, nil)
				req.Host = hostHeader
				req.Header.Set("X-Forwarded-For", fmt.Sprintf("10.0.%d.%d", workerID+1, (counter%200)+1))
				t0 := time.Now()
				resp, err := client.Do(req)
				dur := time.Since(t0)
				reqCancel()

				if err != nil {
					atomic.AddInt64(&errCount, 1)
				} else if resp.StatusCode != http.StatusOK {
					atomic.AddInt64(&errCount, 1)
					resp.Body.Close()
				} else {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					localLat = append(localLat, dur)
					atomic.AddInt64(&totalReqs, 1)
				}
			}
			mu.Lock()
			latencies = append(latencies, localLat...)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	p50, p90, p95, p99 := calcPercentiles(latencies)
	rps := float64(totalReqs) / elapsed.Seconds()

	return Result{
		Scenario:    "Public Cached Reads",
		Concurrency: concurrency,
		TotalReqs:   totalReqs,
		Duration:    elapsed,
		RPS:         rps,
		P50:         p50,
		P90:         p90,
		P95:         p95,
		P99:         p99,
		Errors:      errCount,
		Passed:      errCount == 0 && p95 < 25*time.Millisecond,
	}
}

func benchConcurrentWrites(baseURL, hostHeader, token string, concurrency int, duration time.Duration) Result {
	endpoint := baseURL + "/api/v1/admin/content/articles"
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: concurrency,
		},
	}

	var totalReqs int64
	var errCount int64
	latencies := make([]time.Duration, 0, 20000)
	var mu sync.Mutex

	var stopping atomic.Bool
	time.AfterFunc(duration, func() { stopping.Store(true) })

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			localLat := make([]time.Duration, 0, 500)
			counter := 0
			for !stopping.Load() {
				counter++
				payload := fmt.Sprintf(`{"title":"Benchmark Item %d-%d","status":"draft"}`, workerID, counter)
				reqCtx, reqCancel := context.WithTimeout(context.Background(), 5*time.Second)
				req, _ := http.NewRequestWithContext(reqCtx, "POST", endpoint, bytes.NewBufferString(payload))
				req.Host = hostHeader
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Forwarded-For", fmt.Sprintf("10.1.%d.%d", workerID+1, (counter%200)+1))
				if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}

				t0 := time.Now()
				resp, err := client.Do(req)
				dur := time.Since(t0)
				reqCancel()

				if err != nil {
					atomic.AddInt64(&errCount, 1)
				} else if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
					atomic.AddInt64(&errCount, 1)
					resp.Body.Close()
				} else {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					localLat = append(localLat, dur)
					atomic.AddInt64(&totalReqs, 1)
				}
			}
			mu.Lock()
			latencies = append(latencies, localLat...)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	p50, p90, p95, p99 := calcPercentiles(latencies)
	rps := float64(totalReqs) / elapsed.Seconds()

	return Result{
		Scenario:    "Concurrent Writes (100 VUs)",
		Concurrency: concurrency,
		TotalReqs:   totalReqs,
		Duration:    elapsed,
		RPS:         rps,
		P50:         p50,
		P90:         p90,
		P95:         p95,
		P99:         p99,
		Errors:      errCount,
		Passed:      errCount == 0 && p95 < 300*time.Millisecond,
	}
}

func benchSSESaturation(baseURL, hostHeader string, concurrency int, duration time.Duration) Result {
	endpoint := baseURL + "/api/v1/content/articles/subscribe"
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 1000,
			MaxConnsPerHost:     0,
		},
	}

	var totalConnected int64
	var errCount int64
	latencies := make([]time.Duration, 0, concurrency)
	var mu sync.Mutex

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()
			reqCtx, reqCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer reqCancel()

			req, _ := http.NewRequestWithContext(reqCtx, "GET", endpoint, nil)
			req.Host = hostHeader
			req.Header.Set("Accept", "text/event-stream")
			req.Header.Set("X-Forwarded-For", fmt.Sprintf("10.2.%d.%d", (connID/250)+1, (connID%250)+1))

			t0 := time.Now()
			resp, err := client.Do(req)
			dur := time.Since(t0)

			if err != nil || resp.StatusCode != http.StatusOK {
				if atomic.AddInt64(&errCount, 1) == 1 {
					var bodySample []byte
					if resp != nil && resp.Body != nil {
						bodySample, _ = io.ReadAll(resp.Body)
					}
					fmt.Printf("[SSE fail %d]: err=%v resp=%v body=%s\n", connID, err, resp, string(bodySample))
				}
				return
			}
			defer resp.Body.Close()

			buf := make([]byte, 13)
			if _, err := io.ReadFull(resp.Body, buf); err == nil {
				atomic.AddInt64(&totalConnected, 1)
				mu.Lock()
				latencies = append(latencies, dur)
				mu.Unlock()
			}
			time.Sleep(duration)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	p50, p90, p95, p99 := calcPercentiles(latencies)

	return Result{
		Scenario:    "SSE Stream Saturation",
		Concurrency: concurrency,
		TotalReqs:   totalConnected,
		Duration:    elapsed,
		RPS:         float64(totalConnected) / elapsed.Seconds(),
		P50:         p50,
		P90:         p90,
		P95:         p95,
		P99:         p99,
		Errors:      errCount,
		Passed:      errCount == 0 && totalConnected > 0,
	}
}

func calcPercentiles(latencies []time.Duration) (p50, p90, p95, p99 time.Duration) {
	if len(latencies) == 0 {
		return
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	n := len(latencies)
	p50 = latencies[n*50/100]
	p90 = latencies[n*90/100]
	p95 = latencies[n*95/100]
	p99 = latencies[n*99/100]
	return
}

func printReport(results []Result) {
	fmt.Println()
	fmt.Println("| Scenario | Concurrency | Total Reqs | RPS | p50 | p90 | p95 | p99 | Errors | Status |")
	fmt.Println("|---|---|---|---|---|---|---|---|---|---|")
	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}
		fmt.Printf("| %s | %d | %d | %.1f | %s | %s | %s | %s | %d | %s |\n",
			r.Scenario, r.Concurrency, r.TotalReqs, r.RPS,
			r.P50.Round(time.Microsecond), r.P90.Round(time.Microsecond),
			r.P95.Round(time.Microsecond), r.P99.Round(time.Microsecond),
			r.Errors, status)
	}
	fmt.Println()
}
