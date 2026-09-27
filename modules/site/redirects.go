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
	mu                 sync.RWMutex
	byTenant           map[string][]*compiledRedirect
	RequireTenantScope bool
}

func (r *RedirectRegistry) Replace(tenantID string, rules []RedirectRule) error {
	compiled := make([]*compiledRedirect, 0, len(rules))
	for _, rule := range rules {
		if rule.TenantID != "" && rule.TenantID != tenantID {
			continue
		}
		if err := ValidateRedirectRule(rule); err != nil {
			return err
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
	r.mu.Lock()
	if r.byTenant == nil {
		r.byTenant = make(map[string][]*compiledRedirect)
	}
	r.byTenant[tenantID] = compiled
	r.mu.Unlock()
	return nil
}

func (r *RedirectRegistry) Upsert(rule RedirectRule) error {
	if err := ValidateRedirectRule(rule); err != nil {
		return err
	}
	tenantID := rule.TenantID
	r.mu.RLock()
	current := append([]*compiledRedirect(nil), r.byTenant[tenantID]...)
	r.mu.RUnlock()
	rules := make([]RedirectRule, 0, len(current)+1)
	replaced := false
	for _, existing := range current {
		if existing.rule.ID == rule.ID || (existing.rule.Pattern == rule.Pattern && existing.rule.Kind == rule.Kind) {
			if !replaced {
				rules = append(rules, rule)
				replaced = true
			}
			continue
		}
		ruleCopy := existing.rule
		ruleCopy.Hits = existing.hits.Load()
		rules = append(rules, ruleCopy)
	}
	if !replaced {
		rules = append(rules, rule)
	}
	return r.Replace(tenantID, rules)
}

func (r *RedirectRegistry) Remove(tenantID, id string) {
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
	_ = r.Replace(tenantID, rules)
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
