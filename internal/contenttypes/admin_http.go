package contenttypes

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// AdminAPIPath is where the content-type admin API is mounted; a type's
// own path is AdminAPIPath + "/" + id.
const AdminAPIPath = "/api/v1/admin/content"

// Routes lists the admin content API. Every route is protected; reads need
// Authorize(r, false) and mutations Authorize(r, true).
//
//	GET    /api/v1/admin/content/{type}?limit&offset&sort&search&filter[f]=&status=&locale=&expand=
//	POST   /api/v1/admin/content/{type}                     -> 201 record
//	GET    /api/v1/admin/content/{type}/{id}?locale=
//	PUT    /api/v1/admin/content/{type}/{id}   {version, fields...}
//	DELETE /api/v1/admin/content/{type}/{id}                -> 204
//	POST   .../{id}/workflow                   {target, version}
//	GET    .../{id}/revisions?limit=
//	GET    .../{id}/revisions/{version}
//	POST   .../{id}/revisions/{version}/restore {version}
//	GET    .../{id}/locales
//	GET    .../{id}/locale/{locale}
//	PUT    .../{id}/locale/{locale}            {fields, version}
//	POST   .../{id}/locale/{locale}/copy       {from, version}
//	POST   .../{id}/presence                   heartbeat
//	GET    .../{id}/presence                   active editors
//	DELETE .../{id}/presence                   leave
func (a AdminAPI) Routes() []router.Route {
	base := AdminAPIPath + "/{type}"
	record := base + "/{id}"
	handlers := []struct {
		pattern string
		handler http.HandlerFunc
	}{
		{"GET " + base, a.list},
		{"POST " + base, a.create},
		{"GET " + record, a.get},
		{"PUT " + record, a.update},
		{"DELETE " + record, a.delete},
		{"POST " + record + "/workflow", a.workflow},
		{"GET " + record + "/revisions", a.revisions},
		{"GET " + record + "/revisions/{version}", a.revision},
		{"POST " + record + "/revisions/{version}/restore", a.restore},
		{"GET " + record + "/locales", a.locales},
		{"GET " + record + "/locale/{locale}", a.locale},
		{"PUT " + record + "/locale/{locale}", a.saveLocale},
		{"POST " + record + "/locale/{locale}/copy", a.copyLocale},
	}
	routes := make([]router.Route, 0, len(handlers)+3)
	for _, entry := range handlers {
		routes = append(routes, router.Route{Pattern: entry.pattern, Access: router.Protected, Handler: entry.handler})
	}
	if a.Presence != nil {
		routes = append(routes,
			router.Route{Pattern: "POST " + record + "/presence", Access: router.Protected, Handler: http.HandlerFunc(a.presenceHeartbeat)},
			router.Route{Pattern: "GET " + record + "/presence", Access: router.Protected, Handler: http.HandlerFunc(a.presenceList)},
			router.Route{Pattern: "POST " + record + "/presence/leave", Access: router.Protected, Handler: http.HandlerFunc(a.presenceLeave)},
		)
	}
	return routes
}

// resolve authorizes the request and looks up the type in the path. It
// answers 403 or 404 itself and reports false.
func (a AdminAPI) resolve(w http.ResponseWriter, r *http.Request, write bool) (Definition, adminstudio.Collection, bool) {
	if a.Authorize == nil || !a.Authorize(r, write) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return Definition{}, adminstudio.Collection{}, false
	}
	definition, ok := definitionByID(a.types(), r.PathValue("type"))
	if !ok {
		api.WriteError(w, r, http.StatusNotFound, "not found", nil)
		return Definition{}, adminstudio.Collection{}, false
	}
	return definition, AdminCollections([]Definition{definition}, a.Locales.Enabled())[0], true
}

// listLocale resolves ?locale= for lists and reads: empty means the default
// locale; anything else must be configured. The second result says whether
// translations apply to this type in that locale.
func (a AdminAPI) listLocale(values url.Values, definition Definition) (string, bool, bool) {
	locale := values.Get("locale")
	if locale == "" {
		locale = a.Locales.Default
	}
	if locale != a.Locales.Default && !a.Locales.Has(locale) {
		return "", false, false
	}
	return locale, a.translatable(definition.ID, locale), true
}

// expansion loads related records for ?expand=.
func (a AdminAPI) expansion() expansion {
	return expansion{
		service: a.Service,
		target:  func(id string) (Definition, bool) { return definitionByID(a.types(), id) },
		present: func(definition Definition, record collections.Record) map[string]any {
			return a.present(definition.ID, Flatten(record))
		},
	}
}

func (a AdminAPI) list(w http.ResponseWriter, r *http.Request) {
	definition, collection, ok := a.resolve(w, r, false)
	if !ok {
		return
	}
	values := r.URL.Query()
	query, err := parsePublicQuery(values, definition)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	expand, err := a.expansion().parse(values, definition)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	// Records written before drafts were enabled have no status and count as
	// drafts when read, but cannot be matched by an equality filter.
	switch status := values.Get("status"); {
	case status == "":
	case !definition.Drafts || (status != StatusDraft && status != StatusPublished):
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query", nil)
		return
	default:
		query.Where = append(query.Where, collections.Condition{Field: StatusField, Op: collections.Equals, Value: status})
	}
	locale, translate, ok := a.listLocale(values, definition)
	if !ok {
		api.WriteError(w, r, http.StatusBadRequest, "unsupported locale", nil)
		return
	}
	records, hasMore, err := a.page(r, definition.ID, query)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	var translations map[string]map[string]any
	if translate && len(records) > 0 {
		ids := make([]string, 0, len(records))
		for _, record := range records {
			ids = append(ids, record.ID)
		}
		if translations, err = a.ListLocalized(r, collection, ids, locale); err != nil {
			writeServiceError(w, r, err, collections.Schema{})
			return
		}
	}
	expanded, err := a.expansion().load(r.Context(), expand, records)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	items := make([]map[string]any, 0, len(records))
	for _, record := range records {
		item := a.present(definition.ID, Flatten(record))
		if translation, ok := translations[record.ID]; ok {
			item = withFallback(item, translation)
		}
		if related, ok := expanded[record.ID]; ok {
			item["expanded"] = related
		}
		items = append(items, item)
	}
	api.WriteList(w, items, api.PaginationMeta{Limit: query.Limit, Offset: query.Offset, HasMore: hasMore})
}

// page fetches one record past the page to learn whether more follow. At
// the query limit's maximum the extra record cannot be requested, so a full
// last page reports no more.
func (a AdminAPI) page(r *http.Request, typeID string, query collections.Query) ([]collections.Record, bool, error) {
	limit := query.Limit
	if limit < 100 {
		query.Limit = limit + 1
	}
	records, err := a.Service.QueryRecords(r.Context(), typeID, query)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	return records, hasMore, nil
}

func (a AdminAPI) get(w http.ResponseWriter, r *http.Request) {
	definition, collection, ok := a.resolve(w, r, false)
	if !ok {
		return
	}
	locale, translate, ok := a.listLocale(r.URL.Query(), definition)
	if !ok {
		api.WriteError(w, r, http.StatusBadRequest, "unsupported locale", nil)
		return
	}
	id := r.PathValue("id")
	expand, err := a.expansion().parse(r.URL.Query(), definition)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	record, err := a.Service.GetRecord(r.Context(), definition.ID, id)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	item := a.present(definition.ID, Flatten(record))
	expanded, err := a.expansion().load(r.Context(), expand, []collections.Record{record})
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	if related, ok := expanded[record.ID]; ok {
		item["expanded"] = related
	}
	if translate {
		translations, err := a.ListLocalized(r, collection, []string{id}, locale)
		if err != nil {
			writeServiceError(w, r, err, collections.Schema{})
			return
		}
		if translation, ok := translations[id]; ok {
			item = withFallback(item, translation)
		}
	}
	api.WriteData(w, http.StatusOK, item, nil)
}

func (a AdminAPI) create(w http.ResponseWriter, r *http.Request) {
	a.write(w, r, "", http.StatusCreated)
}

func (a AdminAPI) update(w http.ResponseWriter, r *http.Request) {
	a.write(w, r, r.PathValue("id"), http.StatusOK)
}

func (a AdminAPI) write(w http.ResponseWriter, r *http.Request, id string, status int) {
	definition, _, ok := a.resolve(w, r, true)
	if !ok {
		return
	}
	var payload map[string]any
	if !api.DecodeJSON(w, r, &payload, maxBodyBytes) {
		return
	}
	if payload == nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}
	schema, err := a.Service.GetSchema(r.Context(), definition.ID)
	if err != nil {
		writeServiceError(w, r, err, schema)
		return
	}
	record, err := a.save(r, schema, id, payload)
	if err != nil {
		writeServiceError(w, r, err, schema)
		return
	}
	api.WriteData(w, status, record, nil)
}

func (a AdminAPI) delete(w http.ResponseWriter, r *http.Request) {
	definition, _, ok := a.resolve(w, r, true)
	if !ok {
		return
	}
	r = a.withActor(r)
	if err := a.Service.DeleteRecord(r.Context(), definition.ID, r.PathValue("id")); err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// current answers the record after a change so clients learn the new version.
func (a AdminAPI) current(w http.ResponseWriter, r *http.Request, typeID, id string) {
	record, err := a.Service.GetRecord(r.Context(), typeID, id)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	api.WriteData(w, http.StatusOK, a.present(typeID, Flatten(record)), nil)
}

func (a AdminAPI) workflow(w http.ResponseWriter, r *http.Request) {
	definition, collection, ok := a.resolve(w, r, true)
	if !ok {
		return
	}
	var input struct {
		Target  string `json:"target"`
		Version int64  `json:"version"`
	}
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	id := r.PathValue("id")
	if err := a.Transition(r, collection, map[string]any{"id": id}, adminstudio.PublicationState(input.Target), input.Version); err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	a.current(w, r, definition.ID, id)
}

func (a AdminAPI) revisions(w http.ResponseWriter, r *http.Request) {
	definition, collection, ok := a.resolve(w, r, false)
	if !ok {
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > collections.MaxRevisionsPerRecord {
			api.WriteError(w, r, http.StatusBadRequest, "invalid list query", nil)
			return
		}
		limit = value
	}
	id := r.PathValue("id")
	if _, err := a.Service.GetRecord(r.Context(), definition.ID, id); err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	summaries, err := a.ListRevisions(r, collection, id, limit)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	api.WriteList(w, summaries, api.PaginationMeta{Limit: limit, HasMore: false})
}

func revisionVersion(w http.ResponseWriter, r *http.Request) (int64, bool) {
	version, err := strconv.ParseInt(r.PathValue("version"), 10, 64)
	if err != nil || version < 1 {
		api.WriteError(w, r, http.StatusNotFound, "not found", nil)
		return 0, false
	}
	return version, true
}

func (a AdminAPI) revision(w http.ResponseWriter, r *http.Request) {
	_, collection, ok := a.resolve(w, r, false)
	if !ok {
		return
	}
	version, ok := revisionVersion(w, r)
	if !ok {
		return
	}
	snapshot, err := a.GetRevision(r, collection, r.PathValue("id"), version)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	api.WriteData(w, http.StatusOK, snapshot, nil)
}

func (a AdminAPI) restore(w http.ResponseWriter, r *http.Request) {
	definition, collection, ok := a.resolve(w, r, true)
	if !ok {
		return
	}
	version, ok := revisionVersion(w, r)
	if !ok {
		return
	}
	var input struct {
		Version int64 `json:"version"`
	}
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	id := r.PathValue("id")
	if err := a.RestoreRevision(r, collection, id, version, input.Version); err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	a.current(w, r, definition.ID, id)
}

func (a AdminAPI) locales(w http.ResponseWriter, r *http.Request) {
	_, collection, ok := a.resolve(w, r, false)
	if !ok {
		return
	}
	options, err := a.LocaleOptions(r, collection)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	api.WriteData(w, http.StatusOK, options, nil)
}

// localized is the body of a locale read: the localized fields in that
// locale and the version to send back when saving them.
type localized struct {
	Fields  map[string]any `json:"fields"`
	Version int64          `json:"version"`
}

func (a AdminAPI) writeLocalized(w http.ResponseWriter, r *http.Request, collection adminstudio.Collection, id, locale string) {
	fields, version, err := a.GetLocalized(r, collection, id, locale)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	api.WriteData(w, http.StatusOK, localized{Fields: fields, Version: version}, nil)
}

func (a AdminAPI) locale(w http.ResponseWriter, r *http.Request) {
	_, collection, ok := a.resolve(w, r, false)
	if !ok {
		return
	}
	a.writeLocalized(w, r, collection, r.PathValue("id"), r.PathValue("locale"))
}

func (a AdminAPI) saveLocale(w http.ResponseWriter, r *http.Request) {
	_, collection, ok := a.resolve(w, r, true)
	if !ok {
		return
	}
	var input localized
	if !api.DecodeJSON(w, r, &input, maxBodyBytes) {
		return
	}
	if input.Fields == nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}
	id, locale := r.PathValue("id"), r.PathValue("locale")
	if err := a.SaveLocalized(r, collection, id, locale, input.Fields, input.Version); err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	a.writeLocalized(w, r, collection, id, locale)
}

func (a AdminAPI) copyLocale(w http.ResponseWriter, r *http.Request) {
	_, collection, ok := a.resolve(w, r, true)
	if !ok {
		return
	}
	var input struct {
		From    string `json:"from"`
		Version int64  `json:"version"`
	}
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	id, locale := r.PathValue("id"), r.PathValue("locale")
	if err := a.CopyLocale(r, collection, id, input.From, locale, input.Version); err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	a.writeLocalized(w, r, collection, id, locale)
}

func (a AdminAPI) presenceHeartbeat(w http.ResponseWriter, r *http.Request) {
	if a.Presence == nil {
		api.WriteError(w, r, http.StatusNotFound, "presence disabled", nil)
		return
	}
	_, _, ok := a.resolve(w, r, false)
	if !ok {
		return
	}
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	id := r.PathValue("id")
	if len(id) > 128 || strings.ContainsAny(id, "\x00\r\n") {
		api.WriteError(w, r, http.StatusBadRequest, "invalid resource", nil)
		return
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil && scope.TenantID != "" {
		tenantID = scope.TenantID
	}
	name := strings.TrimSpace(strings.SplitN(user.Email, "@", 2)[0])
	if name == "" {
		name = "Editor"
	}
	resource := "content:" + r.PathValue("type") + ":" + id
	if err := a.Presence.Heartbeat(realtime.PresenceSession{TenantID: tenantID, Resource: resource, UserID: user.ID, Email: user.Email, Name: name}); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "presence is unavailable", nil)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, a.Presence.Active(tenantID, resource, user.ID), nil)
}

func (a AdminAPI) presenceList(w http.ResponseWriter, r *http.Request) {
	if a.Presence == nil {
		api.WriteError(w, r, http.StatusNotFound, "presence disabled", nil)
		return
	}
	_, _, ok := a.resolve(w, r, false)
	if !ok {
		return
	}
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil && scope.TenantID != "" {
		tenantID = scope.TenantID
	}
	resource := "content:" + r.PathValue("type") + ":" + r.PathValue("id")
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, a.Presence.Active(tenantID, resource, user.ID), nil)
}

func (a AdminAPI) presenceLeave(w http.ResponseWriter, r *http.Request) {
	if a.Presence == nil {
		api.WriteError(w, r, http.StatusNotFound, "presence disabled", nil)
		return
	}
	_, _, ok := a.resolve(w, r, false)
	if !ok {
		return
	}
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil && scope.TenantID != "" {
		tenantID = scope.TenantID
	}
	resource := "content:" + r.PathValue("type") + ":" + r.PathValue("id")
	a.Presence.Leave(tenantID, resource, user.ID)
	w.WriteHeader(http.StatusNoContent)
}
