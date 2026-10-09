package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/linkcheck"
)

func scanLinksCommand(args []string, stdout, stderr io.Writer) error {
	return scanLinksCommandWithClient(args, stdout, stderr, nil)
}

func scanLinksCommandWithClient(args []string, stdout, stderr io.Writer, client *http.Client) error {
	flags := flag.NewFlagSet("scan-links", flag.ContinueOnError)
	flags.SetOutput(stderr)
	startURL := flags.String("url", "", "public site URL to scan")
	maxPages := flags.Int("max-pages", 500, "maximum same-origin pages to crawl (max 5000)")
	concurrency := flags.Int("concurrency", 4, "parallel page requests (max 16)")
	requestDelay := flags.Duration("delay", 100*time.Millisecond, "minimum delay between requests")
	allowPrivate := flags.Bool("allow-private-targets", false, "allow scanning private network targets (only for trusted local sites)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if *startURL == "" {
		return fmt.Errorf("scan-links requires --url")
	}
	findings, err := linkcheck.Crawl(context.Background(), linkcheck.Options{StartURL: *startURL, MaxPages: *maxPages, Concurrency: *concurrency, RequestDelay: *requestDelay, AllowPrivateTargets: *allowPrivate, Client: client})
	if err != nil {
		return err
	}
	writer := csv.NewWriter(stdout)
	if err := writer.Write([]string{"source_page", "source_title", "broken_url", "http_status", "anchor_text", "suggested_action"}); err != nil {
		return err
	}
	for _, finding := range findings {
		status := strconv.Itoa(finding.Status)
		if finding.Status == 0 {
			status = "unreachable"
		}
		if err := writer.Write([]string{finding.SourcePage, finding.SourceTitle, finding.URL, status, finding.AnchorText, finding.Suggestion}); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return nil
}
