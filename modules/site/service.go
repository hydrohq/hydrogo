package site

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/forms"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type Store interface {
	UpsertSetting(context.Context, database.Executor, Setting) error
	ListPublicSettings(context.Context, database.Executor) ([]Setting, error)
	ReplaceNavigation(context.Context, database.Executor, string, []NavigationItem) error
	ListNavigation(context.Context, database.Executor, string) ([]NavigationItem, error)
	PutRedirect(context.Context, database.Executor, Redirect) error
	GetRedirect(context.Context, database.Executor, string) (Redirect, error)
	CreateSubmission(context.Context, database.Executor, Submission) error
	ListSubmissions(context.Context, database.Executor, string, int) ([]Submission, error)
	UpdateSubmissionStatus(context.Context, database.Executor, string, string) error
	PurgeSubmissionsBefore(context.Context, database.Executor, time.Time) (int64, error)
}

type RedirectRuleStore interface {
	UpsertRedirectRule(context.Context, database.Executor, RedirectRule) error
	DeleteRedirectRule(context.Context, database.Executor, string) error
	ListRedirectRules(context.Context, database.QueryExecutor, int) ([]RedirectRule, error)
}

type NotFoundPathStore interface {
	RecordNotFound(context.Context, database.Executor, string, time.Time) error
	ListNotFound(context.Context, database.QueryExecutor, int) ([]NotFoundPath, error)
	DeleteNotFound(context.Context, database.Executor, string) error
}

type AdminNavigationStore interface {
	ListAdminNavigation(context.Context, database.Executor, string) ([]NavigationItem, error)
}

// SettingsReader exposes private settings to the admin API.
type SettingsReader interface {
	ListSettings(context.Context, database.Executor) ([]Setting, error)
	GetSetting(context.Context, database.Executor, string) (Setting, bool, error)
}

// SubmissionQueryStore is the inbox read surface of durable stores.
type SubmissionQueryStore interface {
	QuerySubmissions(context.Context, database.Executor, SubmissionFilter) ([]Submission, error)
	GetSubmission(context.Context, database.Executor, string) (Submission, bool, error)
}

// IdempotentSubmissionStore is an optional extension implemented by durable stores.
// Keeping it separate preserves compatibility for lightweight module fakes.
type IdempotentSubmissionStore interface {
	FindSubmissionByIdempotency(context.Context, database.Executor, string, string) (Submission, bool, error)
}

type LeadStore interface {
	UpdateLead(context.Context, database.Executor, string, string, string) error
	CreateLeadNote(context.Context, database.Executor, LeadNote) error
	ListLeadNotes(context.Context, database.QueryExecutor, string, int) ([]LeadNote, error)
}

type TransactionalMailQueue interface {
	Enqueue(context.Context, database.Executor, EmailNotification) error
}

type AuditWriter interface {
	Record(context.Context, database.Executor, string, string, string) error
}

type FormSubmissionMetrics interface {
	RecordFormSubmission(context.Context, database.Executor, string, string, time.Time) error
}

type FormRegistry interface {
	Upsert(context.Context, forms.Form) error
	UpsertWithExecutor(context.Context, database.Executor, forms.Form) error
	Get(context.Context, string) (forms.Form, error)
	List(context.Context) ([]forms.Form, error)
	Delete(context.Context, string) error
	DeleteWithExecutor(context.Context, database.Executor, string) error
}

type Service struct {
	Transactions       database.Transactor
	Store              Store
	Audit              AuditWriter
	Forms              FormRegistry
	MailQueue          TransactionalMailQueue
	FormMetrics        FormSubmissionMetrics
	RequireTenantScope bool
}

func (s Service) SaveForm(ctx context.Context, form forms.Form) error {
	if err := form.Validate(); err != nil {
		return err
	}
	if err := s.requireStore(); err != nil || guards.IsNil(s.Forms) {
		return ErrStoreRequired
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Forms.UpsertWithExecutor(ctx, tx, form); err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.form.updated", form.ID)
	})
}

func (s Service) DeleteForm(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrStoreRequired
	}
	if err := s.requireStore(); err != nil || guards.IsNil(s.Forms) {
		return ErrStoreRequired
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Forms.DeleteWithExecutor(ctx, tx, id); err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.form.deleted", id)
	})
}

func (s Service) ListForms(ctx context.Context) ([]forms.Form, error) {
	if err := s.requireStore(); err != nil || guards.IsNil(s.Forms) {
		return nil, ErrStoreRequired
	}
	return s.Forms.List(ctx)
}

var errAuditIDUnavailable = errors.New("audit id generation unavailable")

func (s Service) requireStore() error {
	if guards.IsNil(s.Store) {
		return ErrStoreRequired
	}
	return nil
}

func (s Service) SaveSetting(ctx context.Context, value Setting) error {
	value.Key = strings.TrimSpace(value.Key)
	if err := ValidateSetting(value); err != nil {
		return err
	}
	if err := s.requireStore(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Store.UpsertSetting(ctx, tx, value); err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.setting.updated", value.Key)
	})
}

func (s Service) SaveContactWidget(ctx context.Context, value ContactWidget) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if err := s.requireStore(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		for _, setting := range value.Settings() {
			if err := s.Store.UpsertSetting(ctx, tx, setting); err != nil {
				return err
			}
		}
		return s.audit(ctx, tx, "site.contact_widget.updated", "contact widget")
	})
}

func (s Service) SaveSocialProfiles(ctx context.Context, profiles []SocialProfile) error {
	profiles, err := NormalizeSocialProfiles(profiles)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(profiles)
	if err != nil {
		return ErrInvalidSetting
	}
	if err := s.requireStore(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Store.UpsertSetting(ctx, tx, Setting{Key: "site.social_profiles", Value: string(payload), Public: true}); err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.social_profiles.updated", "social profiles")
	})
}

func (s Service) SaveNavigation(ctx context.Context, locale string, items []NavigationItem) error {
	locale, items, err := normalizeNavigation(locale, items)
	if err != nil {
		return err
	}
	if err := s.requireStore(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Store.ReplaceNavigation(ctx, tx, locale, items); err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.navigation.updated", locale)
	})
}

// ListSettings returns every stored setting, private ones included.
func (s Service) ListSettings(ctx context.Context) ([]Setting, error) {
	if err := s.requireStore(); err != nil {
		return nil, err
	}
	reader, ok := s.Store.(SettingsReader)
	if !ok {
		return nil, ErrStoreRequired
	}
	var items []Setting
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var queryErr error
		items, queryErr = reader.ListSettings(ctx, tx)
		return queryErr
	})
	return items, err
}

// GetSetting reads one setting, private ones included.
func (s Service) GetSetting(ctx context.Context, key string) (Setting, bool, error) {
	if err := s.requireStore(); err != nil {
		return Setting{}, false, err
	}
	reader, ok := s.Store.(SettingsReader)
	if !ok {
		return Setting{}, false, ErrStoreRequired
	}
	var value Setting
	var found bool
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var queryErr error
		value, found, queryErr = reader.GetSetting(ctx, tx, key)
		return queryErr
	})
	return value, found, err
}

// ListInbox pages the form inbox.
func (s Service) ListInbox(ctx context.Context, filter SubmissionFilter) ([]Submission, error) {
	if err := s.requireStore(); err != nil {
		return nil, err
	}
	store, ok := s.Store.(SubmissionQueryStore)
	if !ok {
		return nil, ErrStoreRequired
	}
	if filter.Status != "" && ValidateInboxStatus(filter.Status) != nil {
		return nil, ErrInvalidInboxState
	}
	var items []Submission
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var queryErr error
		items, queryErr = store.QuerySubmissions(ctx, tx, filter)
		return queryErr
	})
	return items, err
}

// GetSubmission reads one inbox entry; found is false for unknown ids.
func (s Service) GetSubmission(ctx context.Context, id string) (Submission, bool, error) {
	if strings.TrimSpace(id) == "" {
		return Submission{}, false, ErrInvalidInboxState
	}
	if err := s.requireStore(); err != nil {
		return Submission{}, false, err
	}
	store, ok := s.Store.(SubmissionQueryStore)
	if !ok {
		return Submission{}, false, ErrStoreRequired
	}
	var value Submission
	var found bool
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var queryErr error
		value, found, queryErr = store.GetSubmission(ctx, tx, id)
		return queryErr
	})
	return value, found, err
}

func (s Service) ListAdminNavigation(ctx context.Context, locale string) ([]NavigationItem, error) {
	if err := s.requireStore(); err != nil {
		return nil, err
	}
	store, ok := s.Store.(AdminNavigationStore)
	if !ok {
		return nil, ErrStoreRequired
	}
	var items []NavigationItem
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var queryErr error
		items, queryErr = store.ListAdminNavigation(ctx, tx, canonical.Locale(locale))
		return queryErr
	})
	return items, err
}

func normalizeNavigation(locale string, items []NavigationItem) (string, []NavigationItem, error) {
	locale = canonical.Locale(locale)
	if locale == "" {
		return "", nil, ErrInvalidNavigation
	}
	if len(items) > 500 {
		return "", nil, ErrInvalidNavigation
	}
	seen := make(map[int]struct{}, len(items))
	byID := make(map[string]NavigationItem, len(items))
	for i := range items {
		items[i].Locale = locale
		items[i].ParentID = strings.TrimSpace(items[i].ParentID)
		if items[i].Target == "" {
			items[i].Target = "_self"
		}
		if err := ValidateNavigation(items[i]); err != nil {
			return "", nil, err
		}
		if _, exists := byID[items[i].ID]; exists {
			return "", nil, ErrInvalidNavigation
		}
		byID[items[i].ID] = items[i]
		if _, ok := seen[items[i].Position]; ok {
			return "", nil, ErrInvalidNavigation
		}
		seen[items[i].Position] = struct{}{}
	}
	for _, item := range items {
		if item.ParentID == "" {
			continue
		}
		if _, exists := byID[item.ParentID]; !exists {
			return "", nil, ErrInvalidNavigation
		}
		visited := map[string]struct{}{item.ID: {}}
		parentID := item.ParentID
		depth := 1
		for parentID != "" {
			if _, cycle := visited[parentID]; cycle || depth > 8 {
				return "", nil, ErrInvalidNavigation
			}
			visited[parentID] = struct{}{}
			parentID = byID[parentID].ParentID
			depth++
		}
	}
	return locale, items, nil
}

func (s Service) SaveRedirect(ctx context.Context, value Redirect) error {
	value.SourcePath = normalizePath(value.SourcePath)
	value.TargetPath = normalizePath(value.TargetPath)
	if err := ValidateRedirect(value); err != nil {
		return err
	}
	if err := s.requireStore(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Store.PutRedirect(ctx, tx, value); err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.redirect.updated", value.SourcePath)
	})
}

func (s Service) SaveRedirectRule(ctx context.Context, value RedirectRule) error {
	return s.SaveRedirectRules(ctx, []RedirectRule{value})
}

func (s Service) SaveRedirectRules(ctx context.Context, values []RedirectRule) error {
	if len(values) == 0 || len(values) > 1000 {
		return ErrInvalidRedirect
	}
	for i := range values {
		values[i].Pattern = strings.TrimSpace(values[i].Pattern)
		values[i].TargetPath = strings.TrimSpace(values[i].TargetPath)
		if values[i].Kind == "exact" {
			values[i].Pattern = normalizePath(values[i].Pattern)
		}
		if !strings.Contains(values[i].TargetPath, "*") && values[i].Kind != "regex" {
			values[i].TargetPath = normalizePath(values[i].TargetPath)
		}
		if err := ValidateRedirectRule(values[i]); err != nil {
			return err
		}
	}
	store, ok := s.Store.(RedirectRuleStore)
	if !ok {
		return ErrRedirectStoreRequired
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		for _, value := range values {
			if err := store.UpsertRedirectRule(ctx, tx, value); err != nil {
				return err
			}
			if err := s.audit(ctx, tx, "site.redirect_rule.updated", value.Pattern); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s Service) DeleteRedirectRule(ctx context.Context, id string) error {
	store, ok := s.Store.(RedirectRuleStore)
	if !ok {
		return ErrRedirectStoreRequired
	}
	if strings.TrimSpace(id) == "" || len(id) > 128 {
		return ErrInvalidRedirect
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := store.DeleteRedirectRule(ctx, tx, id); err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.redirect_rule.deleted", id)
	})
}

func (s Service) ListRedirectRules(ctx context.Context, limit int) ([]RedirectRule, error) {
	store, ok := s.Store.(RedirectRuleStore)
	if !ok {
		return nil, ErrRedirectStoreRequired
	}
	var result []RedirectRule
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	err := s.Transactions.WithinQuery(ctx, func(query database.QueryExecutor) error {
		var err error
		result, err = store.ListRedirectRules(ctx, query, limit)
		return err
	})
	return result, err
}

func (s Service) RecordNotFound(ctx context.Context, path string, now time.Time) error {
	store, ok := s.Store.(NotFoundPathStore)
	if !ok {
		return ErrRedirectStoreRequired
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return store.RecordNotFound(ctx, tx, path, now) })
}

func (s Service) ListNotFound(ctx context.Context, limit int) ([]NotFoundPath, error) {
	store, ok := s.Store.(NotFoundPathStore)
	if !ok {
		return nil, ErrRedirectStoreRequired
	}
	var result []NotFoundPath
	if limit < 1 || limit > 500 {
		limit = 100
	}
	err := s.Transactions.WithinQuery(ctx, func(query database.QueryExecutor) error {
		var err error
		result, err = store.ListNotFound(ctx, query, limit)
		return err
	})
	return result, err
}

func (s Service) DeleteNotFound(ctx context.Context, path string) error {
	store, ok := s.Store.(NotFoundPathStore)
	if !ok {
		return ErrRedirectStoreRequired
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return store.DeleteNotFound(ctx, tx, path) })
}

func (s Service) Submit(ctx context.Context, value Submission) error {
	_, err := s.submit(ctx, value, false)
	return err
}

// SubmitIdempotent deduplicates a form submission when IdempotencyKey is set.
// Empty keys retain the legacy behavior for internal callers.
func (s Service) SubmitIdempotent(ctx context.Context, value Submission) (bool, error) {
	return s.submit(ctx, value, true)
}

func (s Service) submit(ctx context.Context, value Submission, idempotent bool) (bool, error) {
	if err := ValidateSubmission(value); err != nil {
		return false, err
	}
	if value.IdempotencyKey != "" {
		if err := ValidateIdempotencyKey(value.IdempotencyKey); err != nil {
			return false, err
		}
		if strings.TrimSpace(value.RequestHash) == "" {
			return false, ErrIdempotencyConflict
		}
	}
	if idempotent && value.IdempotencyKey == "" {
		return false, ErrInvalidIdempotencyKey
	}
	if err := s.requireStore(); err != nil {
		return false, err
	}
	if !idempotent || value.IdempotencyKey == "" {
		return false, s.Transactions.Within(ctx, func(tx database.Executor) error {
			if err := s.Store.CreateSubmission(ctx, tx, value); err != nil {
				return err
			}
			if err := s.recordFormSubmissionMetric(ctx, tx, value.FormKey); err != nil {
				return err
			}
			if err := s.enqueueNotifications(ctx, tx, value.Notifications); err != nil {
				return err
			}
			return s.audit(ctx, tx, "site.form.submitted", value.ID)
		})
	}
	lookup, ok := s.Store.(IdempotentSubmissionStore)
	if !ok {
		return false, ErrIdempotencyStoreRequired
	}
	replayed := false
	duplicate := false
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		existing, found, err := lookup.FindSubmissionByIdempotency(ctx, tx, value.FormKey, value.IdempotencyKey)
		if err != nil {
			return err
		}
		if found {
			if existing.RequestHash != value.RequestHash {
				return ErrIdempotencyConflict
			}
			replayed = true
			return nil
		}
		if err := s.Store.CreateSubmission(ctx, tx, value); err != nil {
			if database.IsUniqueViolation(err) {
				duplicate = true
				return err
			}
			return err
		}
		if err := s.recordFormSubmissionMetric(ctx, tx, value.FormKey); err != nil {
			return err
		}
		if err := s.enqueueNotifications(ctx, tx, value.Notifications); err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.form.submitted", value.ID)
	})
	if duplicate {
		err = s.Transactions.Within(ctx, func(tx database.Executor) error {
			existing, found, lookupErr := lookup.FindSubmissionByIdempotency(ctx, tx, value.FormKey, value.IdempotencyKey)
			if lookupErr != nil {
				return lookupErr
			}
			if !found || existing.RequestHash != value.RequestHash {
				return ErrIdempotencyConflict
			}
			replayed = true
			return nil
		})
	}
	return replayed, err
}

func (s Service) recordFormSubmissionMetric(ctx context.Context, tx database.Executor, formID string) error {
	if s.FormMetrics == nil {
		return nil
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(ctx); err == nil {
		tenantID = scope.TenantID
	} else if s.RequireTenantScope {
		return err
	}
	return s.FormMetrics.RecordFormSubmission(ctx, tx, tenantID, formID, time.Now().UTC())
}

func (s Service) SetSubmissionStatus(ctx context.Context, id, status string) error {
	if strings.TrimSpace(id) == "" || ValidateInboxStatus(status) != nil {
		return ErrInvalidInboxState
	}
	if err := s.requireStore(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Store.UpdateSubmissionStatus(ctx, tx, id, strings.TrimSpace(status)); err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.inbox.status_updated", id)
	})
}

func (s Service) UpdateLead(ctx context.Context, id, stage, assignedTo string) error {
	if strings.TrimSpace(id) == "" || ValidateLeadStage(stage) != nil || ValidateLeadAssignee(assignedTo) != nil {
		return ErrInvalidInboxState
	}
	store, ok := s.Store.(LeadStore)
	if !ok {
		return ErrLeadStoreRequired
	}
	if err := s.requireStore(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := store.UpdateLead(ctx, tx, id, strings.TrimSpace(stage), strings.TrimSpace(assignedTo)); err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.inbox.lead_updated", id)
	})
}

func (s Service) AddLeadNote(ctx context.Context, note LeadNote) error {
	if err := ValidateLeadNote(note); err != nil {
		return err
	}
	store, ok := s.Store.(LeadStore)
	if !ok {
		return ErrLeadStoreRequired
	}
	if err := s.requireStore(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := store.CreateLeadNote(ctx, tx, note); err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.inbox.note_added", note.SubmissionID)
	})
}

func (s Service) ListLeadNotes(ctx context.Context, id string, limit int) ([]LeadNote, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidInboxState
	}
	store, ok := s.Store.(LeadStore)
	if !ok {
		return nil, ErrLeadStoreRequired
	}
	if err := s.requireStore(); err != nil {
		return nil, err
	}
	var result []LeadNote
	err := s.Transactions.WithinQuery(ctx, func(tx database.QueryExecutor) error {
		var err error
		result, err = store.ListLeadNotes(ctx, tx, id, limit)
		return err
	})
	return result, err
}

func (s Service) enqueueNotifications(ctx context.Context, tx database.Executor, notifications []EmailNotification) error {
	if len(notifications) == 0 {
		return nil
	}
	if guards.IsNil(s.MailQueue) {
		return ErrMailQueueRequired
	}
	for _, notification := range notifications {
		if err := s.MailQueue.Enqueue(ctx, tx, notification); err != nil {
			return err
		}
	}
	return nil
}

func (s Service) PurgeInbox(ctx context.Context, before time.Time) (int64, error) {
	if before.IsZero() {
		return 0, ErrInvalidInboxState
	}
	if err := s.requireStore(); err != nil {
		return 0, err
	}
	var purged int64
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		purged, err = s.Store.PurgeSubmissionsBefore(ctx, tx, before.UTC())
		if err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.inbox.purged", before.UTC().Format(time.RFC3339))
	})
	return purged, err
}

func (s Service) audit(ctx context.Context, tx database.Executor, action, target string) error {
	if s.Audit == nil {
		return nil
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return errAuditIDUnavailable
	}
	return s.Audit.Record(ctx, tx, "site-"+hex.EncodeToString(idBytes), action, target)
}
