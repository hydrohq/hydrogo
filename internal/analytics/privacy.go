package analytics

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type NoCookieVisit struct {
	Day, TenantID, Page, Referrer, Device, Browser, OS string
	VisitorKey                                         string
}

type FormDailyMetric struct {
	Day         string
	TenantID    string
	FormID      string
	Views       int64
	Submissions int64
}

type FormMetrics interface {
	RecordFormView(context.Context, string, string, string, time.Time) (bool, error)
	RecordFormSubmission(context.Context, database.Executor, string, string, time.Time) error
	ListFormDaily(context.Context, string, string, string, string) ([]FormDailyMetric, error)
}

type NoCookieSink interface {
	RecordNoCookieVisit(context.Context, NoCookieVisit) (bool, error)
	PurgeNoCookieBefore(context.Context, string, string) error
}

// VisitorHasher creates daily rotating pseudonyms. Every instance in a
// deployment must use the same operator-provided key for stable deduplication.
type VisitorHasher struct{ key []byte }

func NewVisitorHasher(key []byte) (*VisitorHasher, error) {
	if len(key) < 32 {
		return nil, errors.New("analytics hash key must contain at least 32 bytes")
	}
	return &VisitorHasher{key: append([]byte(nil), key...)}, nil
}

func (h *VisitorHasher) Hash(day, tenantID, address, userAgent string) string {
	if h == nil || day == "" {
		return ""
	}
	ip := net.ParseIP(requestmetaAddress(address))
	if ip == nil {
		return ""
	}
	if v4 := ip.To4(); v4 != nil {
		v4[3] = 0 // retain only the directly connected /24 for IPv4
		ip = v4
	} else {
		v6 := ip.To16()
		for i := 7; i < len(v6); i++ {
			v6[i] = 0
		} // retain only the /56 prefix
		ip = v6
	}
	ua := strings.TrimSpace(userAgent)
	if len(ua) > 512 {
		ua = ua[:512]
	}
	daily := hmac.New(sha256.New, h.key)
	_, _ = daily.Write([]byte("hydro-analytics-day:" + day))
	visitor := hmac.New(sha256.New, daily.Sum(nil))
	_, _ = visitor.Write([]byte(tenantID))
	_, _ = visitor.Write([]byte{0})
	_, _ = visitor.Write(ip)
	_, _ = visitor.Write([]byte{0})
	_, _ = visitor.Write([]byte(ua))
	return hex.EncodeToString(visitor.Sum(nil))
}

func requestmetaAddress(value string) string {
	if host, _, err := net.SplitHostPort(strings.TrimSpace(value)); err == nil {
		return host
	}
	return strings.TrimSpace(value)
}

type NoCookieQueue struct {
	sink         NoCookieSink
	jobs         chan NoCookieVisit
	close        chan struct{}
	wg           sync.WaitGroup
	failed       atomic.Uint64
	dropped      atomic.Uint64
	lastPurgeDay string
}

func NewNoCookieQueue(sink NoCookieSink, capacity int) *NoCookieQueue {
	if capacity < 1 {
		capacity = 128
	}
	queue := &NoCookieQueue{sink: sink, jobs: make(chan NoCookieVisit, capacity), close: make(chan struct{})}
	if sink != nil {
		queue.wg.Add(1)
		go queue.worker()
	}
	return queue
}

func (q *NoCookieQueue) Enqueue(event NoCookieVisit) bool {
	if q == nil || q.sink == nil || event.VisitorKey == "" {
		return false
	}
	select {
	case <-q.close:
		return false
	default:
	}
	select {
	case q.jobs <- event:
		return true
	default:
		q.dropped.Add(1)
		return false
	}
}

func (q *NoCookieQueue) Failed() uint64 {
	if q == nil {
		return 0
	}
	return q.failed.Load()
}
func (q *NoCookieQueue) Dropped() uint64 {
	if q == nil {
		return 0
	}
	return q.dropped.Load()
}
func (q *NoCookieQueue) Close() {
	if q == nil {
		return
	}
	select {
	case <-q.close:
		return
	default:
		close(q.close)
	}
	q.wg.Wait()
}

func (q *NoCookieQueue) worker() {
	defer q.wg.Done()
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	q.purgeExpired(context.Background(), time.Now().UTC())
	for {
		select {
		case <-q.close:
			return
		case now := <-ticker.C:
			q.purgeExpired(context.Background(), now.UTC())
		case event := <-q.jobs:
			ctx, span := observability.StartComponentSpan(context.Background(), "analytics", "no_cookie.persist")
			if _, err := q.sink.RecordNoCookieVisit(ctx, event); err != nil {
				q.failed.Add(1)
			}
			span.End()
		}
	}
}

func (q *NoCookieQueue) purgeExpired(ctx context.Context, now time.Time) {
	day := now.Format("2006-01-02")
	if day == q.lastPurgeDay {
		return
	}
	if err := q.sink.PurgeNoCookieBefore(ctx, now.AddDate(0, 0, -90).Format("2006-01-02"), now.AddDate(0, 0, -1).Format("2006-01-02")); err != nil {
		q.failed.Add(1)
		return
	}
	q.lastPurgeDay = day
}

// NoCookieMiddleware observes successful public GETs without reading or
// setting cookies. DNT and Global Privacy Control are respected as opt-outs.
func NoCookieMiddleware(next http.Handler, queue *NoCookieQueue, hasher *VisitorHasher) http.Handler {
	if next == nil || queue == nil || hasher == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		if r.Method != http.MethodGet || recorder.status < 200 || recorder.status >= 300 || r.Header.Get("DNT") == "1" || r.Header.Get("Sec-GPC") == "1" {
			return
		}
		day := time.Now().UTC().Format("2006-01-02")
		tenantID := ""
		if scope, err := tenant.FromContext(r.Context()); err == nil {
			tenantID = scope.TenantID
		}
		visitorKey := hasher.Hash(day, tenantID, requestmeta.ClientIP(r), r.UserAgent())
		if visitorKey == "" {
			return
		}
		page := r.URL.EscapedPath()
		if page == "" || len(page) > 1024 {
			return
		}
		device, browser, osName := classifyUserAgent(r.UserAgent())
		queue.Enqueue(NoCookieVisit{Day: day, TenantID: tenantID, Page: page, Referrer: referrerHost(r.Referer(), r.Host), Device: device, Browser: browser, OS: osName, VisitorKey: visitorKey})
	})
}

func referrerHost(value, requestHost string) string {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "(direct)"
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" || strings.EqualFold(host, strings.Split(requestHost, ":")[0]) {
		return "(direct)"
	}
	if len(host) > 253 {
		return "(direct)"
	}
	return host
}

func classifyUserAgent(value string) (device, browser, osName string) {
	ua := strings.ToLower(value)
	device = "desktop"
	if strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet") {
		device = "tablet"
	} else if strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || strings.Contains(ua, "android") {
		device = "mobile"
	}
	switch {
	case strings.Contains(ua, "edg/"):
		browser = "Edge"
	case strings.Contains(ua, "firefox/"):
		browser = "Firefox"
	case strings.Contains(ua, "chrome/") && !strings.Contains(ua, "chromium"):
		browser = "Chrome"
	case strings.Contains(ua, "safari/"):
		browser = "Safari"
	default:
		browser = "Other"
	}
	switch {
	case strings.Contains(ua, "windows"):
		osName = "Windows"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ios"):
		osName = "iOS"
	case strings.Contains(ua, "android"):
		osName = "Android"
	case strings.Contains(ua, "mac os") || strings.Contains(ua, "macintosh"):
		osName = "macOS"
	case strings.Contains(ua, "linux"):
		osName = "Linux"
	default:
		osName = "Other"
	}
	return
}
