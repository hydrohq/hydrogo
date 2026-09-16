package search

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
)

const DefaultMemoryMaxDocuments = 10000

var ErrMemoryIndexFull = errors.New("memory search index is full")

// Document is the provider-neutral searchable projection owned by a module.
// Callers must provide already-published, already-authorized content only.
type Document struct {
	ID     string
	Locale string
	Title  string
	Body   string
}

// MemoryProvider is a bounded local search adapter for Compact and prototype
// profiles. It is intentionally not wired as a Distributed production index.
type MemoryProvider struct {
	mu        sync.RWMutex
	max       int
	documents map[string]Document
}

func NewMemoryProvider(maxDocuments int) *MemoryProvider {
	if maxDocuments <= 0 {
		maxDocuments = DefaultMemoryMaxDocuments
	}
	return &MemoryProvider{max: maxDocuments, documents: make(map[string]Document)}
}

func (p *MemoryProvider) Upsert(ctx context.Context, document Document) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := validateDocument(document); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.documents[document.ID]; !exists && len(p.documents) >= p.max {
		return ErrMemoryIndexFull
	}
	p.documents[document.ID] = normalizeDocument(document)
	return nil
}

func (p *MemoryProvider) Delete(ctx context.Context, id string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("search document id is required")
	}
	p.mu.Lock()
	delete(p.documents, id)
	p.mu.Unlock()
	return nil
}

// Rebuild replaces the complete bounded index, preventing stale documents from
// surviving a module reindex operation.
func (p *MemoryProvider) Rebuild(ctx context.Context, documents []Document) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if len(documents) > p.max {
		return ErrMemoryIndexFull
	}
	next := make(map[string]Document, len(documents))
	for _, document := range documents {
		if err := validateDocument(document); err != nil {
			return err
		}
		document = normalizeDocument(document)
		if _, exists := next[document.ID]; exists {
			return errors.New("duplicate search document id")
		}
		next[document.ID] = document
	}
	p.mu.Lock()
	p.documents = next
	p.mu.Unlock()
	return nil
}

func (p *MemoryProvider) Search(ctx context.Context, query Query) (Response, error) {
	if err := contextErr(ctx); err != nil {
		return Response{}, err
	}
	if err := query.Validate(); err != nil {
		return Response{}, err
	}
	text := strings.ToLower(strings.TrimSpace(query.Text))
	terms := tokenize(text)
	p.mu.RLock()
	documents := make([]Document, 0, len(p.documents))
	for _, document := range p.documents {
		documents = append(documents, document)
	}
	p.mu.RUnlock()
	results := make([]Result, 0)
	for _, document := range documents {
		if query.Locale != "" && canonical.Locale(document.Locale) != canonical.Locale(query.Locale) {
			continue
		}
		title := strings.ToLower(document.Title)
		body := strings.ToLower(document.Body)
		score := 0.0
		for _, term := range terms {
			if strings.Contains(title, term) {
				score += 2
			}
			if strings.Contains(body, term) {
				score++
			}
		}
		if score == 0 {
			continue
		}
		results = append(results, Result{ID: document.ID, Score: score, Title: document.Title, Snippet: snippet(document.Body, terms)})
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].ID < results[j].ID
	})
	total := len(results)
	start := query.Offset
	if start > total {
		start = total
	}
	limit := query.Limit
	if limit <= 0 || limit > MaxLimit {
		limit = MaxLimit
	}
	end := start + limit
	if end > total {
		end = total
	}
	return Response{Items: results[start:end], Total: total}, nil
}

func validateDocument(document Document) error {
	if strings.TrimSpace(document.ID) == "" || len(document.ID) > 128 || canonical.Locale(document.Locale) == "" || strings.TrimSpace(document.Title) == "" {
		return errors.New("invalid search document")
	}
	return nil
}

func normalizeDocument(document Document) Document {
	document.ID = strings.TrimSpace(document.ID)
	document.Locale = canonical.Locale(document.Locale)
	document.Title = strings.TrimSpace(document.Title)
	document.Body = strings.TrimSpace(document.Body)
	return document
}

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func tokenize(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsPunct(r) })
}

func snippet(body string, terms []string) string {
	body = strings.Join(strings.Fields(body), " ")
	if len([]rune(body)) > 180 {
		return string([]rune(body)[:180]) + "…"
	}
	return body
}
