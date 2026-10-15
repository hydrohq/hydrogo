package transport

import (
	"encoding/csv"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type redirectInput struct {
	Pattern    string `json:"pattern"`
	Kind       string `json:"kind"`
	TargetPath string `json:"target_path"`
	Status     int    `json:"status"`
}

func (h Handler) tenantID(r *http.Request) string {
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		return scope.TenantID
	}
	return ""
}

func (h Handler) adminRedirects(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	limit := 500
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 1000 {
			api.WriteError(w, r, http.StatusBadRequest, "invalid limit", nil)
			return
		}
		limit = parsed
	}
	items, err := h.Service.ListRedirectRules(r.Context(), limit)
	if err != nil {
		writeSiteError(w, r, err, "redirects unavailable")
		return
	}
	if h.Redirects != nil {
		hits := make(map[string]uint64)
		for _, rule := range h.Redirects.List(h.tenantID(r)) {
			hits[rule.ID] = rule.Hits
		}
		for i := range items {
			if count, ok := hits[items[i].ID]; ok {
				items[i].Hits = count
			}
		}
	}
	if items == nil {
		items = []site.RedirectRule{}
	}
	api.WriteData(w, http.StatusOK, items, nil)
}

func (h Handler) ruleFromInput(r *http.Request, id string, input redirectInput) site.RedirectRule {
	rule := site.RedirectRule{ID: id, TenantID: h.tenantID(r), Pattern: strings.TrimSpace(input.Pattern), Kind: strings.TrimSpace(input.Kind), TargetPath: strings.TrimSpace(input.TargetPath), Status: input.Status, CreatedAt: time.Now().UTC()}
	if rule.Kind == "" {
		rule.Kind = "exact"
	}
	if rule.Status == 0 {
		rule.Status = http.StatusMovedPermanently
	}
	return normalizeRedirectRule(rule)
}

func normalizeRedirectRule(rule site.RedirectRule) site.RedirectRule {
	if rule.Kind == "exact" {
		if path, err := canonical.Path(rule.Pattern); err == nil {
			rule.Pattern = "/" + path
		}
	}
	if !strings.HasPrefix(rule.TargetPath, "//") && !strings.Contains(rule.TargetPath, "*") && rule.Kind != "regex" {
		if path, err := canonical.Path(rule.TargetPath); err == nil {
			rule.TargetPath = "/" + path
		}
	}
	return rule
}

func (h Handler) saveRule(w http.ResponseWriter, r *http.Request, rule site.RedirectRule, status int) {
	if err := site.ValidateRedirectRule(rule); err != nil {
		api.WriteValidation(w, r, map[string]string{"rule": "invalid"})
		return
	}
	persist := func() error { return h.Service.SaveRedirectRule(r.Context(), rule) }
	var err error
	if h.Redirects != nil {
		err = h.Redirects.UpsertWith(rule, persist)
	} else {
		err = persist()
	}
	if err != nil {
		writeSiteError(w, r, err, "redirect could not be saved")
		return
	}
	api.WriteData(w, status, rule, nil)
}

func (h Handler) adminCreateRedirect(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	var input redirectInput
	if !decode(w, r, &input) {
		return
	}
	id, err := randomID("redirect-")
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "redirects unavailable", nil)
		return
	}
	h.saveRule(w, r, h.ruleFromInput(r, id, input), http.StatusCreated)
}

func (h Handler) adminUpdateRedirect(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	id := r.PathValue("id")
	if !h.ruleExists(w, r, id) {
		return
	}
	var input redirectInput
	if !decode(w, r, &input) {
		return
	}
	h.saveRule(w, r, h.ruleFromInput(r, id, input), http.StatusOK)
}

func (h Handler) ruleExists(w http.ResponseWriter, r *http.Request, id string) bool {
	rules, err := h.Service.ListRedirectRules(r.Context(), 1000)
	if err != nil {
		writeSiteError(w, r, err, "redirects unavailable")
		return false
	}
	for _, rule := range rules {
		if rule.ID == id {
			return true
		}
	}
	api.WriteError(w, r, http.StatusNotFound, "redirect not found", nil)
	return false
}

func (h Handler) adminDeleteRedirect(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	id := r.PathValue("id")
	if !h.ruleExists(w, r, id) {
		return
	}
	persist := func() error { return h.Service.DeleteRedirectRule(r.Context(), id) }
	var err error
	if h.Redirects != nil {
		err = h.Redirects.RemoveWith(h.tenantID(r), id, persist)
	} else {
		err = persist()
	}
	if err != nil {
		writeSiteError(w, r, err, "redirect could not be removed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// adminImportRedirects accepts either a JSON body {"rules":[...]} or a
// multipart "file" CSV with pattern,kind,target,status columns.
func (h Handler) adminImportRedirects(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	var inputs []redirectInput
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var payload struct {
			Rules []redirectInput `json:"rules"`
		}
		if !decode(w, r, &payload) {
			return
		}
		inputs = payload.Rules
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			api.WriteError(w, r, http.StatusBadRequest, "invalid CSV upload", nil)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			api.WriteError(w, r, http.StatusBadRequest, "invalid CSV upload", nil)
			return
		}
		defer file.Close()
		inputs, err = parseRedirectCSV(file)
		if err != nil {
			api.WriteError(w, r, http.StatusBadRequest, "invalid CSV", nil)
			return
		}
	}
	if len(inputs) == 0 || len(inputs) > 1000 {
		api.WriteError(w, r, http.StatusBadRequest, "rules list is empty or too long", nil)
		return
	}
	rules := make([]site.RedirectRule, 0, len(inputs))
	fields := map[string]string{}
	for index, input := range inputs {
		id, err := randomID("redirect-")
		if err != nil {
			api.WriteError(w, r, http.StatusServiceUnavailable, "redirects unavailable", nil)
			return
		}
		rule := h.ruleFromInput(r, id, input)
		if err := site.ValidateRedirectRule(rule); err != nil {
			fields[strconv.Itoa(index)] = "invalid"
			continue
		}
		rules = append(rules, rule)
	}
	if len(fields) > 0 {
		api.WriteValidation(w, r, fields)
		return
	}
	persist := func() error { return h.Service.SaveRedirectRules(r.Context(), rules) }
	var err error
	if h.Redirects != nil {
		err = h.Redirects.UpsertManyWith(rules, persist)
	} else {
		err = persist()
	}
	if err != nil {
		writeSiteError(w, r, err, "redirect rules could not be imported")
		return
	}
	api.WriteData(w, http.StatusOK, map[string]int{"imported": len(rules)}, nil)
}

func parseRedirectCSV(file io.Reader) ([]redirectInput, error) {
	reader := csv.NewReader(io.LimitReader(file, 1<<20))
	reader.FieldsPerRecord = 4
	inputs := make([]redirectInput, 0, 64)
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || len(inputs) >= 1000 {
			return nil, errors.New("invalid csv")
		}
		if len(inputs) == 0 && strings.EqualFold(strings.TrimSpace(row[0]), "pattern") {
			continue
		}
		status, err := strconv.Atoi(strings.TrimSpace(row[3]))
		if err != nil {
			return nil, errors.New("invalid csv")
		}
		inputs = append(inputs, redirectInput{Pattern: row[0], Kind: strings.TrimSpace(row[1]), TargetPath: row[2], Status: status})
	}
	return inputs, nil
}

func (h Handler) adminNotFound(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 500 {
			api.WriteError(w, r, http.StatusBadRequest, "invalid limit", nil)
			return
		}
		limit = parsed
	}
	items, err := h.Service.ListNotFound(r.Context(), limit)
	if err != nil {
		writeSiteError(w, r, err, "not-found report unavailable")
		return
	}
	if items == nil {
		items = []site.NotFoundPath{}
	}
	api.WriteData(w, http.StatusOK, items, nil)
}

func cleanPath(value string) (string, bool) {
	path, err := canonical.Path(value)
	if err != nil || path == "" || len(path) > 2048 {
		return "", false
	}
	return "/" + path, true
}

func (h Handler) adminConvertNotFound(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	var payload struct {
		Path   string `json:"path"`
		Target string `json:"target"`
	}
	if !decode(w, r, &payload) {
		return
	}
	source, sourceOK := cleanPath(payload.Path)
	target, targetOK := cleanPath(payload.Target)
	fields := map[string]string{}
	if !sourceOK {
		fields["path"] = "invalid"
	}
	if !targetOK {
		fields["target"] = "invalid"
	}
	if len(fields) > 0 {
		api.WriteValidation(w, r, fields)
		return
	}
	id, err := randomID("not-found-")
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "redirects unavailable", nil)
		return
	}
	rule := site.RedirectRule{ID: id, TenantID: h.tenantID(r), Pattern: source, Kind: "exact", TargetPath: target, Status: http.StatusMovedPermanently, CreatedAt: time.Now().UTC()}
	if err := site.ValidateRedirectRule(rule); err != nil {
		api.WriteValidation(w, r, map[string]string{"rule": "invalid"})
		return
	}
	persist := func() error { return h.Service.SaveRedirectRule(r.Context(), rule) }
	if h.Redirects != nil {
		err = h.Redirects.UpsertWith(rule, persist)
	} else {
		err = persist()
	}
	if err != nil {
		writeSiteError(w, r, err, "redirect could not be saved")
		return
	}
	if err := h.Service.DeleteNotFound(r.Context(), source); err != nil {
		writeSiteError(w, r, err, "redirect saved; report cleanup unavailable")
		return
	}
	api.WriteData(w, http.StatusCreated, rule, nil)
}

func (h Handler) adminDeleteNotFound(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	var payload struct {
		Path string `json:"path"`
	}
	if !decode(w, r, &payload) {
		return
	}
	path, ok := cleanPath(payload.Path)
	if !ok {
		api.WriteValidation(w, r, map[string]string{"path": "invalid"})
		return
	}
	if err := h.Service.DeleteNotFound(r.Context(), path); err != nil {
		writeSiteError(w, r, err, "not-found entry could not be removed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
