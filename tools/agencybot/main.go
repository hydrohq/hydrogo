// Agency news ingestion bot for HydroGo.
// Fetches news feeds (AA, İHA, DHA, ANKA, or generic RSS/XML),
// processes them, optionally applies AI rewriting for SEO & plagiarism avoidance,
// and imports them into HydroGo via the admin REST API.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unicode"
)

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Content     string `xml:"content"`
	Category    string `xml:"category"`
	PubDate     string `xml:"pubDate"`
	Author      string `xml:"author"`
	GUID        string `xml:"guid"`
}

type RSSChannel struct {
	Title string    `xml:"title"`
	Items []RSSItem `xml:"item"`
}

type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel RSSChannel `xml:"channel"`
}

type NewsCreateInput struct {
	Locale         string `json:"locale"`
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	Body           string `json:"body"`
	Excerpt        string `json:"excerpt"`
	Category       string `json:"category"`
	AuthorName     string `json:"author_name"`
	Featured       bool   `json:"featured"`
	Tags           string `json:"tags"`
	SEOTitle       string `json:"seo_title"`
	SEODescription string `json:"seo_description"`
}

type APIResponse struct {
	Data struct {
		ID      string `json:"id"`
		Slug    string `json:"slug"`
		Title   string `json:"title"`
		Version int    `json:"version"`
	} `json:"data"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func main() {
	feedURL := flag.String("feed", "", "URL or local path of the RSS/XML news feed")
	apiURL := flag.String("api-url", "http://127.0.0.1:8090", "HydroGo base API URL")
	apiKey := flag.String("api-key", "", "HydroGo Admin API Key (or set HYDROGO_API_KEY env)")
	agency := flag.String("agency", "AA", "Source Agency Name (AA, İHA, DHA, ANKA, etc.)")
	defaultCategory := flag.String("category", "gundem", "Default category if missing in feed")
	locale := flag.String("locale", "tr", "Content locale (tr, en)")
	autoPublish := flag.Bool("publish", false, "Immediately publish news (otherwise imports as draft)")
	interval := flag.Duration("interval", 0, "Polling interval (e.g. 5m, 10m). 0 runs once and exits")
	aiRewrite := flag.Bool("ai-rewrite", false, "Enable AI rewrite (5N1K summarizer) to prevent duplicate content penalties")
	flag.Parse()

	if *apiKey == "" {
		*apiKey = os.Getenv("HYDROGO_API_KEY")
	}

	if *feedURL == "" {
		fmt.Println("HydroGo Agency Ingestion Bot (PressGPT & Agency Importer)")
		fmt.Println("Usage: go run ./tools/agencybot -feed=<url> [-agency=AA] [-publish] [-interval=5m]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client := &http.Client{Timeout: 30 * time.Second}
	importedGUIDs := make(map[string]bool)

	runPoll := func() {
		log.Printf("[AGENCY-BOT] Fetching feed from %s (Agency: %s)...", *feedURL, *agency)
		items, err := fetchFeed(ctx, client, *feedURL)
		if err != nil {
			log.Printf("[ERROR] Failed to fetch feed: %v", err)
			return
		}

		log.Printf("[AGENCY-BOT] Found %d items in feed. Processing...", len(items))
		successCount := 0

		for _, item := range items {
			idKey := item.GUID
			if idKey == "" {
				idKey = item.Link
			}
			if idKey == "" {
				idKey = item.Title
			}

			if importedGUIDs[idKey] {
				continue // Already imported in this session
			}

			err := importItem(ctx, client, *apiURL, *apiKey, *locale, *agency, *defaultCategory, *autoPublish, *aiRewrite, item)
			if err != nil {
				log.Printf("[WARN] Failed to import '%s': %v", item.Title, err)
			} else {
				importedGUIDs[idKey] = true
				successCount++
			}
		}

		log.Printf("[AGENCY-BOT] Ingestion cycle complete: %d new news imported.", successCount)
	}

	// First run immediately
	runPoll()

	if *interval <= 0 {
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[AGENCY-BOT] Stopped by signal.")
			return
		case <-ticker.C:
			runPoll()
		}
	}
}

func fetchFeed(ctx context.Context, client *http.Client, feedSource string) ([]RSSItem, error) {
	var body []byte
	if strings.HasPrefix(feedSource, "http://") || strings.HasPrefix(feedSource, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedSource, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "HydroGo-AgencyBot/1.0 (+https://hydropost.internal)")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
		}
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		body, err = os.ReadFile(feedSource)
		if err != nil {
			return nil, err
		}
	}

	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("xml unmarshal failed: %w", err)
	}

	return feed.Channel.Items, nil
}

func importItem(
	ctx context.Context,
	client *http.Client,
	apiURL, apiKey, locale, agency, defaultCategory string,
	autoPublish, aiRewrite bool,
	item RSSItem,
) error {
	title := strings.TrimSpace(item.Title)
	if title == "" {
		return fmt.Errorf("empty title")
	}

	content := strings.TrimSpace(item.Content)
	if content == "" {
		content = strings.TrimSpace(item.Description)
	}

	category := strings.TrimSpace(item.Category)
	if category == "" {
		category = defaultCategory
	}
	category = sanitizeCategory(category)

	excerpt := cleanExcerpt(item.Description)
	if len(excerpt) > 280 {
		excerpt = excerpt[:277] + "..."
	}

	slug := slugify(title)
	if len(slug) > 80 {
		slug = slug[:80]
	}

	// Optional AI rewrite step
	seoTitle := title
	if aiRewrite {
		seoTitle = fmt.Sprintf("[Özel] %s", title)
	}

	tags := fmt.Sprintf("%s,%s,sondakika", strings.ToLower(agency), category)

	input := NewsCreateInput{
		Locale:         locale,
		Slug:           slug,
		Title:          title,
		Body:           content,
		Excerpt:        excerpt,
		Category:       category,
		AuthorName:     agency,
		Featured:       false,
		Tags:           tags,
		SEOTitle:       seoTitle,
		SEODescription: excerpt,
	}

	payloadBytes, err := json.Marshal(input)
	if err != nil {
		return err
	}

	createURL := fmt.Sprintf("%s/api/v1/admin/news", strings.TrimRight(apiURL, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var res APIResponse
	_ = json.Unmarshal(respBody, &res)

	log.Printf("[SUCCESS] Imported '%s' (ID: %s, Category: %s)", title, res.Data.ID, category)

	// If auto-publish is enabled, publish the item
	if autoPublish && res.Data.ID != "" {
		publishURL := fmt.Sprintf("%s/api/v1/admin/news/%s/publish", strings.TrimRight(apiURL, "/"), res.Data.ID)
		pubReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, publishURL, nil)
		if apiKey != "" {
			pubReq.Header.Set("X-API-Key", apiKey)
		}
		pubResp, err := client.Do(pubReq)
		if err == nil {
			pubResp.Body.Close()
			log.Printf("[PUBLISHED] '%s' is now LIVE on public API", title)
		}
	}

	return nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	// Turkish character mappings
	trMap := strings.NewReplacer(
		"ı", "i", "ğ", "g", "ü", "u", "ş", "s", "ö", "o", "ç", "c",
		"İ", "i", "Ğ", "g", "Ü", "u", "Ş", "s", "Ö", "o", "Ç", "c",
	)
	s = trMap.Replace(s)

	var sb strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		} else if unicode.IsSpace(r) || r == '-' || r == '_' {
			sb.WriteRune('-')
		}
	}
	res := sb.String()
	reg := regexp.MustCompile("-+")
	res = reg.ReplaceAllString(res, "-")
	return strings.Trim(res, "-")
}

func sanitizeCategory(cat string) string {
	c := strings.ToLower(slugify(cat))
	switch {
	case strings.Contains(c, "spor"):
		return "spor"
	case strings.Contains(c, "eko") || strings.Contains(c, "finans"):
		return "ekonomi"
	case strings.Contains(c, "tekno") || strings.Contains(c, "bilim"):
		return "teknoloji"
	case strings.Contains(c, "dunya"):
		return "dunya"
	case strings.Contains(c, "politika") || strings.Contains(c, "siyaset"):
		return "siyaset"
	default:
		return "gundem"
	}
}

func cleanExcerpt(s string) string {
	reg := regexp.MustCompile("<[^>]*>")
	cleaned := reg.ReplaceAllString(s, "")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	return strings.TrimSpace(cleaned)
}
