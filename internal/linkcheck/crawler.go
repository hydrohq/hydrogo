// Package linkcheck crawls one public HTTP origin and reports broken links and assets.
package linkcheck

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const defaultBodyLimit = 2 << 20

type Finding struct {
	SourcePage  string `json:"source_page"`
	SourceTitle string `json:"source_title,omitempty"`
	URL         string `json:"url"`
	Status      int    `json:"status"`
	AnchorText  string `json:"anchor_text,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

type Options struct {
	StartURL            string
	MaxPages            int
	MaxTargets          int
	MaxFindings         int
	Concurrency         int
	RequestDelay        time.Duration
	Retention           time.Duration
	Client              *http.Client
	AllowPrivateTargets bool
}

func Crawl(ctx context.Context, options Options) ([]Finding, error) {
	start, err := url.Parse(strings.TrimSpace(options.StartURL))
	if err != nil || (start.Scheme != "http" && start.Scheme != "https") || start.Host == "" || start.User != nil {
		return nil, fmt.Errorf("start URL must be an absolute HTTP(S) URL")
	}
	start.Fragment = ""
	start.RawQuery = ""
	origin := strings.ToLower(start.Scheme + "://" + start.Host)
	if options.MaxPages <= 0 {
		options.MaxPages = 500
	}
	if options.MaxPages > 5000 {
		options.MaxPages = 5000
	}
	if options.MaxTargets <= 0 {
		options.MaxTargets = 5000
	}
	if options.MaxTargets > 50000 {
		options.MaxTargets = 50000
	}
	if options.MaxFindings <= 0 {
		options.MaxFindings = 1000
	}
	if options.MaxFindings > 5000 {
		options.MaxFindings = 5000
	}
	if options.Concurrency <= 0 {
		options.Concurrency = 4
	}
	if options.Concurrency > 16 {
		options.Concurrency = 16
	}
	if options.RequestDelay <= 0 {
		options.RequestDelay = 100 * time.Millisecond
	}
	client := options.Client
	if client == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if !options.AllowPrivateTargets {
			dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
			transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				host, port, splitErr := net.SplitHostPort(address)
				if splitErr != nil {
					return nil, splitErr
				}
				ips, lookupErr := net.DefaultResolver.LookupIP(ctx, "ip", host)
				if lookupErr != nil {
					return nil, lookupErr
				}
				var lastErr error
				for _, ip := range ips {
					addr, ok := netip.AddrFromSlice(ip)
					if !ok || !publicAddress(addr.Unmap()) {
						continue
					}
					conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
					if dialErr == nil {
						return conn, nil
					}
					lastErr = dialErr
				}
				if lastErr != nil {
					return nil, lastErr
				}
				return nil, fmt.Errorf("link target resolves to a non-public address")
			}
		}
		client = &http.Client{Transport: transport, Timeout: 8 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		}}
	}

	type crawlTarget struct{ url, source, sourceTitle, anchor string }
	queue := []crawlTarget{{url: start.String(), source: start.String()}}
	queued := map[string]bool{start.String(): true}
	checked := map[string]bool{start.String(): true}
	visited := make(map[string]bool)
	findings := make([]Finding, 0)
	var lock sync.Mutex
	var nextRequest time.Time
	addFinding := func(finding Finding) {
		lock.Lock()
		if len(findings) < options.MaxFindings {
			findings = append(findings, finding)
		}
		lock.Unlock()
	}
	fetch := func(raw, source, sourceTitle, anchor string, readHTML bool) (*http.Response, []byte, error) {
		lock.Lock()
		now := time.Now()
		wait := time.Duration(0)
		if nextRequest.After(now) {
			wait = nextRequest.Sub(now)
		}
		nextRequest = now.Add(wait + options.RequestDelay)
		lock.Unlock()
		if wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-timer.C:
			}
		}
		var response *http.Response
		var reqErr error
		for attempt := 0; attempt < 2; attempt++ {
			request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
			if requestErr != nil {
				return nil, nil, requestErr
			}
			request.Header.Set("User-Agent", "HydroGo-LinkCheck/1.0 (+operator initiated)")
			response, reqErr = client.Do(request)
			transientResponse := response != nil && (response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500)
			if response != nil && transientResponse {
				_ = response.Body.Close()
			}
			if reqErr == nil && !transientResponse {
				break
			}
			if attempt == 0 {
				timer := time.NewTimer(250 * time.Millisecond)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, nil, ctx.Err()
				case <-timer.C:
				}
			}
		}
		if reqErr != nil {
			addFinding(Finding{SourcePage: source, SourceTitle: sourceTitle, URL: raw, Status: 0, AnchorText: strings.TrimSpace(anchor), Suggestion: "Retry later or check that the destination site is available."})
			return nil, nil, reqErr
		}
		if response == nil {
			return nil, nil, fmt.Errorf("link check returned no response")
		}
		if response.Body == nil {
			return response, nil, nil
		}
		if response.StatusCode >= 500 || response.StatusCode == http.StatusTooManyRequests {
			addFinding(Finding{SourcePage: source, SourceTitle: sourceTitle, URL: raw, Status: response.StatusCode, AnchorText: strings.TrimSpace(anchor), Suggestion: "Retry later; the destination may be temporarily unavailable."})
			_ = response.Body.Close()
			return response, nil, nil
		}
		if response.StatusCode >= 400 {
			_ = response.Body.Close()
			addFinding(Finding{SourcePage: source, SourceTitle: sourceTitle, URL: raw, Status: response.StatusCode, AnchorText: strings.TrimSpace(anchor), Suggestion: suggestionForStatus(response.StatusCode)})
			return response, nil, nil
		}
		if !readHTML || !strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "html") {
			_ = response.Body.Close()
			return response, nil, nil
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, defaultBodyLimit+1))
		_ = response.Body.Close()
		if len(body) > defaultBodyLimit {
			body = body[:defaultBodyLimit]
		}
		if readErr != nil {
			return response, body, readErr
		}
		return response, body, nil
	}

	for len(queue) > 0 && len(visited) < options.MaxPages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch := append([]crawlTarget(nil), queue...)
		queue = nil
		jobs := make(chan crawlTarget)
		var workers sync.WaitGroup
		worker := func() {
			defer workers.Done()
			for item := range jobs {
				pageURL := item.url
				response, body, fetchErr := fetch(pageURL, item.source, item.sourceTitle, item.anchor, true)
				if fetchErr != nil {
					continue
				}
				if response.StatusCode < 200 || response.StatusCode >= 300 || !strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "html") {
					continue
				}
				sourceTitle, links := parseDocument(body)
				for _, link := range links {
					target, parseErr := url.Parse(link.href)
					if parseErr != nil {
						continue
					}
					target = response.Request.URL.ResolveReference(target)
					if target.Scheme != "http" && target.Scheme != "https" {
						continue
					}
					target.Fragment = ""
					if target.Host == "" {
						continue
					}
					if link.page && strings.EqualFold(target.Scheme+"://"+target.Host, origin) {
						target.RawQuery = ""
						canonical := target.String()
						lock.Lock()
						if !queued[canonical] && !visited[canonical] && !checked[canonical] && len(checked) < options.MaxTargets && len(queued) < options.MaxPages {
							queued[canonical] = true
							checked[canonical] = true
							queue = append(queue, crawlTarget{url: canonical, source: pageURL, sourceTitle: sourceTitle, anchor: link.text})
						}
						lock.Unlock()
					} else {
						key := target.String()
						lock.Lock()
						shouldCheck := !checked[key] && len(checked) < options.MaxTargets
						if shouldCheck {
							checked[key] = true
						}
						lock.Unlock()
						if shouldCheck {
							_, _, _ = fetch(key, pageURL, sourceTitle, link.text, false)
						}
					}
				}
			}
		}
		count := options.Concurrency
		if count > len(batch) {
			count = len(batch)
		}
		for range count {
			workers.Add(1)
			go worker()
		}
		for _, item := range batch {
			lock.Lock()
			visited[item.url] = true
			lock.Unlock()
			jobs <- item
		}
		close(jobs)
		workers.Wait()
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].SourcePage != findings[j].SourcePage {
			return findings[i].SourcePage < findings[j].SourcePage
		}
		return findings[i].URL < findings[j].URL
	})
	return findings, nil
}

func publicAddress(address netip.Addr) bool {
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	for _, prefix := range []netip.Prefix{netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("198.18.0.0/15")} {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

type pageLink struct {
	href, text string
	page       bool
}

func parseLinks(body []byte) []pageLink {
	_, links := parseDocument(body)
	return links
}

func parseDocument(body []byte) (string, []pageLink) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return "", nil
	}
	var result []pageLink
	title := ""
	var textContent func(*html.Node) string
	textContent = func(node *html.Node) string {
		if node.Type == html.TextNode {
			return node.Data
		}
		var text strings.Builder
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			text.WriteString(textContent(child))
		}
		return text.String()
	}
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if node.Data == "title" && title == "" {
				title = strings.TrimSpace(textContent(node))
			}
			attrs := make(map[string]string, len(node.Attr))
			for _, attr := range node.Attr {
				attrs[strings.ToLower(attr.Key)] = strings.TrimSpace(attr.Val)
			}
			switch node.Data {
			case "a", "area":
				if attrs["href"] != "" {
					result = append(result, pageLink{href: attrs["href"], text: strings.TrimSpace(textContent(node)), page: true})
				}
			case "img", "script", "iframe", "source":
				if attrs["src"] != "" {
					result = append(result, pageLink{href: attrs["src"], text: strings.TrimSpace(attrs["alt"])})
				}
				if node.Data == "img" || node.Data == "source" {
					for _, candidate := range strings.Split(attrs["srcset"], ",") {
						fields := strings.Fields(strings.TrimSpace(candidate))
						if len(fields) > 0 && !strings.HasPrefix(strings.ToLower(fields[0]), "data:") {
							result = append(result, pageLink{href: fields[0], text: strings.TrimSpace(attrs["alt"])})
						}
					}
				}
			case "link":
				rel := strings.ToLower(attrs["rel"])
				if attrs["href"] != "" && (strings.Contains(rel, "stylesheet") || strings.Contains(rel, "icon") || strings.Contains(rel, "preload") || strings.Contains(rel, "manifest")) {
					result = append(result, pageLink{href: attrs["href"]})
				}
			case "video":
				if attrs["poster"] != "" {
					result = append(result, pageLink{href: attrs["poster"], text: attrs["alt"]})
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(doc)
	return title, result
}

func suggestionForStatus(status int) string {
	switch status {
	case http.StatusNotFound, http.StatusGone:
		return "Update the address to the destination's current page or remove the link."
	case http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
		return "Retry later; the destination may be temporarily unavailable."
	default:
		return "Check the destination response and replace the link if it has moved."
	}
}
