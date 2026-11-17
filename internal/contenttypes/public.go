package contenttypes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hydrohq/hydrogo/internal/api"
	"github.com/hydrohq/hydrogo/internal/collections"
	"github.com/hydrohq/hydrogo/internal/realtime"
	"github.com/hydrohq/hydrogo/internal/tenant"
	"golang.org/x/sync/singleflight"
)

// PublicAPI serves read-only JSON for content types marked `public: true`:
//
//	GET /api/v1/content/{type}?limit=&offset=&after=&sort=&search=&filter[field]=&expand=
//	    -> {"data": [...], "metadata": {"limit": n, "offset": n, "has_more": b, "locale": "tr"}}
//	GET /api/v1/content/{type}/{id} -> {"data": record}
//
// sort takes a field ID (or created_at, updated_at), prefixed with "-" for
// descending order; filter[field] matches fields marked filterable exactly;
// search matches fields marked searchable. See parsePublicQuery. expand=a,b
// adds the records that relation fields a and b point to under "expanded"
// (see expansion); only relations to public types can be expanded.
//
// Both accept ?locale= (a configured site locale, default: the default
// locale). Localized fields then carry that locale's values, falling back
// to the default locale's value field by field; search, filters and sorting
// always use the default locale. Records and lists include "locale". An
// unknown locale answers 400.
//
// Types that are not public answer 404, exactly like unknown types. For
// types with drafts only published records exist here: lists filter on the
// status and a draft's ID answers 404. Rich text fields are returned as
// HTML when RenderRichText is set.
type PublicAPI struct {
	Service        collections.Service
	RenderRichText func(stored string) string
	Types          []Definition
	// TypesFunc optionally supplies the definitions dynamically.
	TypesFunc func() []Definition
	// TypesForContext optionally supplies tenant-scoped definitions per request.
	TypesForContext  func(context.Context) []Definition
	Locales          Locales
	Hub              *realtime.Hub
	ResolvePrincipal func(r *http.Request) Principal
	FlightGroup      *singleflight.Group
}

var defaultPublicFlightGroup = &singleflight.Group{}

func (p PublicAPI) singleflight() *singleflight.Group {
	if p.FlightGroup != nil {
		return p.FlightGroup
	}
	return defaultPublicFlightGroup
}

func requestTenantID(ctx context.Context) string {
	if scope, err := tenant.FromContext(ctx); err == nil {
		return strings.TrimSpace(scope.TenantID)
	}
	return ""
}

func isAnonymousPublicRule(def Definition, rule string, r *http.Request) bool {
	if !def.Public {
		return false
	}
	trimmed := strings.TrimSpace(rule)
	if trimmed != "" && !strings.EqualFold(trimmed, "public") && !strings.EqualFold(trimmed, "true") {
		return false
	}
	return r == nil || strings.TrimSpace(r.Header.Get("Authorization")) == ""
}

func etagMatches(header, etag string) bool {
	if header == "" || etag == "" {
		return false
	}
	for _, part := range strings.Split(header, ",") {
		candidate := strings.TrimSpace(part)
		candidate = strings.TrimPrefix(candidate, "W/")
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}

func hashJSONValue(h interface{ Write([]byte) (int, error) }, label string, v any) {
	_, _ = fmt.Fprintf(h, "%s:", label)
	if v == nil {
		_, _ = h.Write([]byte("null\n"))
		return
	}
	if b, err := json.Marshal(v); err == nil {
		_, _ = h.Write(b)
	}
	_, _ = h.Write([]byte("\n"))
}

func computeListETag(tenantID, typeID, locale, rawQuery string, hasMore bool, nextCursor string, records []collections.Record, expanded map[string]map[string]any, aggregates any) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "tenant=%s\ntype=%s\nlocale=%s\nquery=%s\nmore=%t\ncursor=%s\n", tenantID, typeID, locale, rawQuery, hasMore, nextCursor)
	for _, rec := range records {
		_, _ = fmt.Fprintf(h, "rec=%s:%d:%d\n", rec.ID, rec.Version, rec.UpdatedAt.UTC().UnixNano())
		hashJSONValue(h, "data", rec.Data)
		if rel, ok := expanded[rec.ID]; ok {
			keys := make([]string, 0, len(rel))
			for k := range rel {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				hashJSONValue(h, "exp:"+k, rel[k])
			}
		}
	}
	if aggregates != nil {
		hashJSONValue(h, "agg", aggregates)
	}
	sum := h.Sum(nil)
	return fmt.Sprintf("%q", "list_"+hex.EncodeToString(sum[:8]))
}

func computeRecordETag(tenantID, typeID, locale string, record collections.Record, related map[string]any) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "tenant=%s\ntype=%s\nlocale=%s\nrec=%s:%d:%d\n", tenantID, typeID, locale, record.ID, record.Version, record.UpdatedAt.UTC().UnixNano())
	hashJSONValue(h, "data", record.Data)
	if len(related) > 0 {
		keys := make([]string, 0, len(related))
		for k := range related {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			hashJSONValue(h, "exp:"+k, related[k])
		}
	}
	sum := h.Sum(nil)
	return fmt.Sprintf("%q", "rec_"+hex.EncodeToString(sum[:8]))
}

func (p PublicAPI) forRequest(r *http.Request) PublicAPI {
	if p.TypesForContext != nil && r != nil {
		cp := p
		cp.Types = p.TypesForContext(r.Context())
		cp.TypesFunc = nil
		cp.TypesForContext = nil
		return cp
	}
	return p
}

func (p PublicAPI) principal(r *http.Request) Principal {
	if p.ResolvePrincipal != nil {
		return p.ResolvePrincipal(r)
	}
	return Principal{Anonymous: true}
}

func (p PublicAPI) types() []Definition {
	if p.TypesFunc != nil {
		return p.TypesFunc()
	}
	return p.Types
}

// publicPage is the list metadata: the page plus the locale served.
type publicPage struct {
	api.PaginationMeta
	Locale     string `json:"locale,omitempty"`
	Aggregates any    `json:"aggregates,omitempty"`
}

func (p PublicAPI) present(typeID, locale string, record collections.Record) map[string]any {
	result := Flatten(record)
	if locale != "" {
		result["locale"] = locale
	}
	for _, definition := range p.types() {
		if definition.ID != typeID {
			continue
		}
		for _, field := range definition.Fields {
			if field.Private {
				delete(result, field.ID)
				continue
			}
			if p.RenderRichText == nil {
				continue
			}
			if text, ok := result[field.ID].(string); ok && field.Kind == "richtext" {
				result[field.ID] = p.RenderRichText(text)
			}
		}
	}
	return result
}

func (p PublicAPI) definition(id string) (Definition, bool) {
	for _, definition := range p.types() {
		if definition.ID == id && (definition.Public || definition.AccessRules != nil) {
			return definition, true
		}
	}
	return Definition{}, false
}

// expansion loads related records for ?expand=; only public types qualify.
func (p PublicAPI) expansion() expansion {
	return expansion{
		service:       p.Service,
		target:        p.definition,
		publishedOnly: true,
		present: func(definition Definition, record collections.Record) map[string]any {
			return p.present(definition.ID, "", record)
		},
	}
}

func (p PublicAPI) List(w http.ResponseWriter, r *http.Request) {
	p = p.forRequest(r)
	typeID := r.PathValue("type")
	definition, ok := p.definition(typeID)
	if !ok {
		api.WriteError(w, r, http.StatusNotFound, "not found", nil)
		return
	}
	principal := p.principal(r)
	if definition.AccessRules != nil && definition.AccessRules.List != "" {
		listRule := definition.AccessRules.List
		allowed, err := EvaluateRule(listRule, principal, nil, true)
		if err == nil && !allowed && (listRule == "false" || listRule == "admin" || (principal.Anonymous && (strings.Contains(listRule, "@auth.") || listRule == "member"))) {
			if principal.Anonymous {
				api.WriteError(w, r, http.StatusUnauthorized, "authentication required", nil)
			} else {
				api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
			}
			return
		}
	}
	locale, ok := p.publicLocale(r)
	if !ok {
		api.WriteError(w, r, http.StatusBadRequest, "unsupported locale", nil)
		return
	}
	query, err := parsePublicQuery(r.URL.Query(), definition)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query", nil)
		return
	}
	aggQuery, hasAgg, err := collections.ParseAggregateQuery(r.URL.Query())
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query", nil)
		return
	}
	expand, err := p.expansion().parse(r.URL.Query(), definition)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query", nil)
		return
	}
	// ?mine=1: a signed-in member's own records of a drafts type, drafts included.
	mine := definition.Drafts && definition.OwnerField != "" && r.URL.Query().Get("mine") == "1"
	if mine && (!principal.IsMember || principal.Email == "") {
		api.WriteError(w, r, http.StatusUnauthorized, "authentication required", nil)
		return
	}
	switch {
	case mine:
		query.Where = append(query.Where, collections.Condition{Field: definition.OwnerField, Op: collections.Equals, Value: principal.Email})
	case definition.Drafts:
		query.Where = append(query.Where, publishedCondition())
	}
	if !mine && definition.AccessRules != nil && definition.AccessRules.List != "" {
		listRule := strings.TrimSpace(definition.AccessRules.List)
		if !principal.IsAdmin && !strings.EqualFold(listRule, "public") && !strings.EqualFold(listRule, "true") {
			query.AccessRule = listRule
			query.AuthContext = collections.AuthContext{
				ID:       principal.ID,
				Email:    principal.Email,
				Role:     principal.Role,
				IsAdmin:  principal.IsAdmin,
				IsMember: principal.IsMember,
			}
		}
	}
	limit := query.Limit
	if limit < 100 {
		query.Limit = limit + 1
	}
	schema, err := p.Service.GetSchema(r.Context(), typeID)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	checkedQuery, err := schema.ValidateQuery(query)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query", nil)
		return
	}
	records, err := p.Service.QueryRecords(r.Context(), typeID, checkedQuery)
	if err == nil {
		err = p.translate(r, definition, locale, records)
	}
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	if limit >= 100 && len(records) == limit {
		hasMore = true // see AdminAPI.page
	}
	expanded, err := p.expansion().load(r.Context(), expand, records)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	items := make([]map[string]any, 0, len(records))
	for _, record := range records {
		item := p.present(typeID, locale, record)
		if related, ok := expanded[record.ID]; ok {
			item["expanded"] = related
		}
		items = append(items, item)
	}
	page := api.PaginationMeta{Limit: limit, Offset: query.Offset, HasMore: hasMore}
	if hasMore && len(records) > 0 {
		page.NextCursor = collections.NextCursor(query, records[len(records)-1])
	}
	var aggregates any
	if hasAgg {
		aggQuery.Where = checkedQuery.Where
		aggQuery.AnyOf = checkedQuery.AnyOf
		aggQuery.AccessRule = checkedQuery.AccessRule
		aggQuery.AuthContext = checkedQuery.AuthContext
		aggregates, err = p.Service.AggregateRecords(r.Context(), typeID, aggQuery)
		if err != nil {
			writeServiceError(w, r, err, collections.Schema{})
			return
		}
	}
	accessListRule := ""
	if definition.AccessRules != nil {
		accessListRule = definition.AccessRules.List
	}
	if mine || !isAnonymousPublicRule(definition, accessListRule, r) {
		w.Header().Set("Cache-Control", "private, no-cache")
	} else {
		etag := computeListETag(requestTenantID(r.Context()), typeID, locale, r.URL.RawQuery, hasMore, page.NextCursor, records, expanded, aggregates)
		w.Header().Set("Cache-Control", "public, max-age=30, stale-while-revalidate=120")
		w.Header().Set("ETag", etag)
		if etagMatches(r.Header.Get("If-None-Match"), etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	if hasAgg {
		api.WriteData(w, http.StatusOK, map[string]any{
			"items":      items,
			"aggregates": aggregates,
		}, publicPage{PaginationMeta: page, Locale: locale, Aggregates: aggregates})
		return
	}
	api.WriteData(w, http.StatusOK, items, publicPage{PaginationMeta: page, Locale: locale})
}

func (p PublicAPI) Get(w http.ResponseWriter, r *http.Request) {
	p = p.forRequest(r)
	typeID := r.PathValue("type")
	definition, ok := p.definition(typeID)
	if !ok {
		api.WriteError(w, r, http.StatusNotFound, "not found", nil)
		return
	}
	locale, ok := p.publicLocale(r)
	if !ok {
		api.WriteError(w, r, http.StatusBadRequest, "unsupported locale", nil)
		return
	}
	expand, err := p.expansion().parse(r.URL.Query(), definition)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid query", nil)
		return
	}
	id := r.PathValue("id")
	tenantID := requestTenantID(r.Context())
	flightKey := fmt.Sprintf("get:%s:%s:%s:%s", tenantID, typeID, id, locale)
	v, err, _ := p.singleflight().Do(flightKey, func() (any, error) {
		sfCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
		defer cancel()
		record, err := p.Service.GetRecord(sfCtx, typeID, id)
		if err == nil && definition.Drafts && recordStatus(record.Data) != StatusPublished {
			err = collections.ErrRecordNotFound
		}
		if err == nil {
			records := []collections.Record{record}
			err = p.translate(r.WithContext(sfCtx), definition, locale, records)
			record = records[0]
		}
		return record, err
	})
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	record := v.(collections.Record)
	principal := p.principal(r)
	if definition.AccessRules != nil && definition.AccessRules.Read != "" {
		allowed, err := EvaluateRule(definition.AccessRules.Read, principal, record.Data, true)
		if err != nil || !allowed {
			if principal.Anonymous {
				api.WriteError(w, r, http.StatusUnauthorized, "authentication required", nil)
			} else {
				api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
			}
			return
		}
	}
	expanded, err := p.expansion().load(r.Context(), expand, []collections.Record{record})
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	related := expanded[record.ID]
	accessReadRule := ""
	if definition.AccessRules != nil {
		accessReadRule = definition.AccessRules.Read
	}
	if !isAnonymousPublicRule(definition, accessReadRule, r) {
		w.Header().Set("Cache-Control", "private, no-cache")
	} else {
		etag := computeRecordETag(tenantID, typeID, locale, record, related)
		w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
		w.Header().Set("ETag", etag)
		if etagMatches(r.Header.Get("If-None-Match"), etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	item := p.present(typeID, locale, record)
	if related != nil {
		item["expanded"] = related
	}
	api.WriteData(w, http.StatusOK, item, nil)
}
