package site

import (
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type compiledRedirect struct {
	rule  RedirectRule
	exact string
	re    *regexp.Regexp
	hits  atomic.Uint64
}

// RedirectRegistry is an immutable per-tenant lookup snapshot with atomic hit
// counters. Replacing a tenant snapshot makes content and rule edits visible
// without database access on the public request path.
type RedirectRegistry struct {
	updateMu           sync.Mutex
	mu                 sync.RWMutex
	byTenant           map[string][]*compiledRedirect
	RequireTenantScope bool
}

func (r *RedirectRegistry) Replace(tenantID string, rules []RedirectRule) error {
	r.updateMu.Lock()
	defer r.updateMu.Unlock()
	compiled, err := compileRedirects(tenantID, rules)
	if err != nil {
		return err
	}
	r.publish(tenantID, compiled)
	return nil
}

func compileRedirects(tenantID string, rules []RedirectRule) ([]*compiledRedirect, error) {
	scopedRules := make([]RedirectRule, 0, len(rules))
	for _, rule := range rules {
		if rule.TenantID == "" || rule.TenantID == tenantID {
			scopedRules = append(scopedRules, rule)
		}
	}
	if redirectRulesCycle(scopedRules) {
		return nil, ErrInvalidRedirect
	}
	compiled := make([]*compiledRedirect, 0, len(scopedRules))
	for _, rule := range scopedRules {
		if err := ValidateRedirectRule(rule); err != nil {
			return nil, err
		}
		entry := &compiledRedirect{rule: rule}
		switch rule.Kind {
		case "exact":
			entry.exact = rule.Pattern
		case "wildcard":
			parts := strings.SplitN(regexp.QuoteMeta(rule.Pattern), `\*`, 2)
			entry.re = regexp.MustCompile("^" + parts[0] + "(.*?)" + parts[1] + "$")
		case "regex":
			entry.re = regexp.MustCompile("^(?:" + rule.Pattern + ")$")
		}
		entry.hits.Store(rule.Hits)
		compiled = append(compiled, entry)
	}
	sort.SliceStable(compiled, func(i, j int) bool {
		if (compiled[i].exact != "") != (compiled[j].exact != "") {
			return compiled[i].exact != ""
		}
		return len(compiled[i].rule.Pattern) > len(compiled[j].rule.Pattern)
	})
	return compiled, nil
}

func (r *RedirectRegistry) publish(tenantID string, compiled []*compiledRedirect) {
	r.mu.Lock()
	if r.byTenant == nil {
		r.byTenant = make(map[string][]*compiledRedirect)
	}
	r.byTenant[tenantID] = compiled
	r.mu.Unlock()
}

func redirectRulesCycle(rules []RedirectRule) bool {
	probes := make([]string, 0, len(rules))
	for _, rule := range rules {
		switch rule.Kind {
		case "exact":
			probes = append(probes, rule.Pattern)
		case "wildcard":
			probes = append(probes, strings.Replace(rule.Pattern, "*", "__hydro_probe__", 1))
		}
	}
	for _, start := range probes {
		seen, path := map[string]bool{}, start
		for range len(rules) + 1 {
			if seen[path] {
				return true
			}
			seen[path] = true
			next, ok := redirectTarget(rules, path)
			if !ok {
				break
			}
			path = next
		}
	}
	return false
}

func redirectTarget(rules []RedirectRule, path string) (string, bool) {
	for _, rule := range rules {
		switch rule.Kind {
		case "exact":
			if path == rule.Pattern {
				return rule.TargetPath, true
			}
		case "wildcard":
			parts := strings.SplitN(rule.Pattern, "*", 2)
			if len(parts) == 2 && strings.HasPrefix(path, parts[0]) && strings.HasSuffix(path, parts[1]) {
				value := strings.TrimSuffix(strings.TrimPrefix(path, parts[0]), parts[1])
				return strings.Replace(rule.TargetPath, "*", value, 1), true
			}
		case "regex":
			if re, err := regexp.Compile("^(?:" + rule.Pattern + ")$"); err == nil && re.MatchString(path) {
				return re.ReplaceAllString(path, rule.TargetPath), true
			}
		}
	}
	return "", false
}

func (r *RedirectRegistry) Upsert(rule RedirectRule) error {
	return r.UpsertWith(rule, nil)
}

// UpsertWith validates and serializes the next tenant snapshot, applies the
// durable write, then publishes the already-validated snapshot to readers.
func (r *RedirectRegistry) UpsertWith(rule RedirectRule, persist func() error) error {
	return r.UpsertManyWith([]RedirectRule{rule}, persist)
}

// UpsertManyWith applies a batch as one validated registry snapshot and only
// publishes it after persist succeeds. Concurrent writers cannot validate
// against stale registry state and then overwrite each other.
func (r *RedirectRegistry) UpsertManyWith(upserts []RedirectRule, persist func() error) error {
	if r == nil || len(upserts) == 0 || len(upserts) > 1000 {
		return ErrInvalidRedirect
	}
	tenantID := upserts[0].TenantID
	for _, rule := range upserts {
		if rule.TenantID != tenantID {
			return ErrInvalidRedirect
		}
	}
	r.updateMu.Lock()
	defer r.updateMu.Unlock()
	r.mu.RLock()
	current := append([]*compiledRedirect(nil), r.byTenant[tenantID]...)
	r.mu.RUnlock()
	rules := make([]RedirectRule, 0, len(current)+len(upserts))
	for _, existing := range current {
		ruleCopy := existing.rule
		ruleCopy.Hits = existing.hits.Load()
		rules = append(rules, ruleCopy)
	}
	for _, rule := range upserts {
		next := make([]RedirectRule, 0, len(rules)+1)
		replaced := false
		for _, existing := range rules {
			if existing.ID == rule.ID || (existing.Pattern == rule.Pattern && existing.Kind == rule.Kind) {
				if !replaced {
					next = append(next, rule)
					replaced = true
				}
				continue
			}
			next = append(next, existing)
		}
		if !replaced {
			next = append(next, rule)
		}
		rules = next
	}
	compiled, err := compileRedirects(tenantID, rules)
	if err != nil {
		return err
	}
	if persist != nil {
		if err := persist(); err != nil {
			return err
		}
	}
	r.publish(tenantID, compiled)
	return nil
}

func (r *RedirectRegistry) Remove(tenantID, id string) {
	_ = r.RemoveWith(tenantID, id, nil)
}

func (r *RedirectRegistry) RemoveWith(tenantID, id string, persist func() error) error {
	if r == nil {
		return ErrInvalidRedirect
	}
	r.updateMu.Lock()
	defer r.updateMu.Unlock()
	r.mu.RLock()
	current := append([]*compiledRedirect(nil), r.byTenant[tenantID]...)
	r.mu.RUnlock()
	rules := make([]RedirectRule, 0, len(current))
	for _, rule := range current {
		if rule.rule.ID != id {
			copy := rule.rule
			copy.Hits = rule.hits.Load()
			rules = append(rules, copy)
		}
	}
	compiled, err := compileRedirects(tenantID, rules)
	if err != nil {
		return err
	}
	if persist != nil {
		if err := persist(); err != nil {
			return err
		}
	}
	r.publish(tenantID, compiled)
	return nil
}

func (r *RedirectRegistry) Match(scope tenant.Scope, requestPath string) (RedirectRule, bool) {
	if r == nil {
		return RedirectRule{}, false
	}
	tenantID := scope.TenantID
	if !r.RequireTenantScope {
		tenantID = ""
	}
	r.mu.RLock()
	rules := r.byTenant[tenantID]
	for _, rule := range rules {
		var matched []string
		if rule.exact != "" {
			if requestPath != rule.exact {
				continue
			}
		} else {
			matched = rule.re.FindStringSubmatch(requestPath)
			if matched == nil {
				continue
			}
		}
		result := rule.rule
		if len(matched) > 1 {
			if result.Kind == "wildcard" {
				result.TargetPath = strings.Replace(result.TargetPath, "*", matched[1], 1)
			} else {
				result.TargetPath = rule.re.ReplaceAllString(requestPath, result.TargetPath)
			}
		}
		rule.hits.Add(1)
		r.mu.RUnlock()
		return result, true
	}
	r.mu.RUnlock()
	return RedirectRule{}, false
}

func (r *RedirectRegistry) List(tenantID string) []RedirectRule {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]RedirectRule, 0, len(r.byTenant[tenantID]))
	for _, item := range r.byTenant[tenantID] {
		rule := item.rule
		rule.Hits = item.hits.Load()
		result = append(result, rule)
	}
	return result
}
