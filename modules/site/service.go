package site

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
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

type AuditWriter interface {
	Record(context.Context, database.Executor, string, string, string) error
}

type Service struct {
	Transactions database.Transactor
	Store        Store
	Audit        AuditWriter
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

func normalizeNavigation(locale string, items []NavigationItem) (string, []NavigationItem, error) {
	locale = canonical.Locale(locale)
	if locale == "" {
		return "", nil, ErrInvalidNavigation
	}
	seen := make(map[int]struct{}, len(items))
	for i := range items {
		items[i].Locale = locale
		if err := ValidateNavigation(items[i]); err != nil {
			return "", nil, err
		}
		if _, ok := seen[items[i].Position]; ok {
			return "", nil, ErrInvalidNavigation
		}
		seen[items[i].Position] = struct{}{}
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

func (s Service) Submit(ctx context.Context, value Submission) error {
	if err := ValidateSubmission(value); err != nil {
		return err
	}
	if err := s.requireStore(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Store.CreateSubmission(ctx, tx, value); err != nil {
			return err
		}
		return s.audit(ctx, tx, "site.form.submitted", value.ID)
	})
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
