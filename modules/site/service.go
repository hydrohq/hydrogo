package site

import (
	"context"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
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
}

type Service struct {
	Transactions database.Transactor
	Store        Store
}

func (s Service) SaveSetting(ctx context.Context, value Setting) error {
	value.Key = strings.TrimSpace(value.Key)
	if err := ValidateSetting(value); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return s.Store.UpsertSetting(ctx, tx, value) })
}

func (s Service) SaveNavigation(ctx context.Context, locale string, items []NavigationItem) error {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if locale == "" {
		return ErrInvalidNavigation
	}
	seen := make(map[int]struct{}, len(items))
	for _, item := range items {
		item.Locale = locale
		if err := ValidateNavigation(item); err != nil {
			return err
		}
		if _, ok := seen[item.Position]; ok {
			return ErrInvalidNavigation
		}
		seen[item.Position] = struct{}{}
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return s.Store.ReplaceNavigation(ctx, tx, locale, items) })
}

func (s Service) SaveRedirect(ctx context.Context, value Redirect) error {
	value.SourcePath = normalizePath(value.SourcePath)
	value.TargetPath = normalizePath(value.TargetPath)
	if err := ValidateRedirect(value); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return s.Store.PutRedirect(ctx, tx, value) })
}

func (s Service) Submit(ctx context.Context, value Submission) error {
	if err := ValidateSubmission(value); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return s.Store.CreateSubmission(ctx, tx, value) })
}
