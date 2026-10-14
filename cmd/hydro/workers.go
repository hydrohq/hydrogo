package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	_ "net/http/pprof"
	"strings"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	adminstudiosql "github.com/ardaozyildirim/hydrogo/internal/adminstudio/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	mailstore "github.com/ardaozyildirim/hydrogo/internal/mailer/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	securitystore "github.com/ardaozyildirim/hydrogo/internal/security/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/storage"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/webhooks"
	newsservice "github.com/ardaozyildirim/hydrogo/modules/news"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func runOutboxWorker(ctx context.Context, worker webhooks.Worker, stderr io.Writer) {
	if len(worker.Dispatcher.Endpoints) == 0 {
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		processed, err := worker.RunOnce(ctx, time.Now().UTC())
		if err != nil && ctx.Err() == nil && stderr != nil {
			_, _ = fmt.Fprintf(stderr, "hydro outbox worker: %v\n", err)
		}
		if !processed {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}
}

func runAdminWeeklyDigestOnce(ctx context.Context, now time.Time, db database.QueryExecutor, tx database.Transactor, store adminstudiosql.Store, users *authstore.UserStore, spool mailstore.Spool, scopes []tenant.Scope, requireTenant, analyticsEnabled, membersEnabled bool) (int, error) {
	if err := database.RequireExecutor(db); err != nil {
		return 0, err
	}
	if users == nil {
		return 0, errors.New("digest user store is required")
	}
	if requireTenant && len(scopes) == 0 {
		return 0, tenant.ErrMissingScope
	}
	workScopes := scopes
	if !requireTenant {
		workScopes = []tenant.Scope{{}}
	}
	queued := 0
	for _, scope := range workScopes {
		jobCtx := ctx
		if requireTenant {
			jobCtx = tenant.WithScope(jobCtx, scope)
		}
		statsByWindow := map[string]adminstudio.DigestStats{}
		after := ""
		for {
			preferences, err := store.DigestPreferencesAfter(jobCtx, db, after, 500)
			if err != nil {
				return queued, err
			}
			if len(preferences) == 0 {
				break
			}
			after = preferences[len(preferences)-1].UserID
			duePreferences := make([]adminstudio.DigestPreference, 0, len(preferences))
			digestUsers := make([]adminstudio.DigestUser, 0, len(preferences))
			for _, preference := range preferences {
				frequency := adminstudio.DigestFrequency(preference.Frequency)
				if frequency == "" {
					frequency = adminstudio.DigestWeekly
				}
				_, _, key, due, err := adminstudio.DigestWindow(now, preference.Timezone, frequency)
				if err != nil {
					return queued, err
				}
				if !due || preference.LastSentPeriod == key {
					continue
				}
				account, err := users.GetByID(jobCtx, preference.UserID)
				if errors.Is(err, auth.ErrUserNotFound) {
					continue
				}
				if err != nil {
					return queued, err
				}
				if account.Role != auth.RoleAdmin && account.Role != auth.RoleEditor {
					continue
				}
				duePreferences = append(duePreferences, preference)
				digestUsers = append(digestUsers, adminstudio.DigestUser{ID: account.ID, Email: account.Email, Active: account.Active})
			}
			if len(duePreferences) > 0 {
				err = adminstudio.DeliverDueDigests(jobCtx, now, duePreferences, digestUsers, func(statsCtx context.Context, start, end time.Time) (adminstudio.DigestStats, error) {
					key := start.UTC().Format(time.RFC3339Nano) + "/" + end.UTC().Format(time.RFC3339Nano)
					if stats, ok := statsByWindow[key]; ok {
						return stats, nil
					}
					stats, err := store.WeeklyDigestStats(statsCtx, db, start, end, analyticsEnabled, membersEnabled)
					if err == nil {
						statsByWindow[key] = stats
					}
					return stats, err
				}, func(deliveryCtx context.Context, account adminstudio.DigestUser, period, body, htmlBody string) error {
					return tx.Within(deliveryCtx, func(exec database.Executor) error {
						claimed, err := store.ClaimDigestWeek(deliveryCtx, exec, account.ID, period)
						if err != nil || !claimed {
							return err
						}
						subject, eventKey := "Haftalık site özeti", "admin-weekly:"
						if strings.Contains(body, "Aylık yönetici özeti") {
							subject, eventKey = "Aylık yönetici özeti", "admin-monthly:"
						}
						if err := spool.Enqueue(deliveryCtx, exec, site.EmailNotification{EventKey: eventKey + account.ID + ":" + period, To: account.Email, Subject: subject, Body: body, HTMLBody: htmlBody}); err != nil {
							return err
						}
						queued++
						return nil
					})
				})
				if err != nil {
					return queued, err
				}
			}
			if len(preferences) < 500 {
				break
			}
		}
	}
	return queued, nil
}

func runSessionCleanup(ctx context.Context, cleaner expiredSessionCleaner, stderr io.Writer) {
	runSessionCleanupObserved(ctx, cleaner, stderr, nil)
}

func runSessionCleanupObserved(ctx context.Context, cleaner expiredSessionCleaner, stderr io.Writer, observer sessionCleanupObserver) {
	if cleaner == nil {
		return
	}
	if stderr == nil {
		stderr = io.Discard
	}
	const interval = 5 * time.Minute
	run := func() {
		deleted, err := cleaner.DeleteExpired(ctx, time.Now().UTC())
		if err != nil {
			if ctx.Err() == nil {
				if observer != nil {
					observer.RecordSessionCleanupError()
				}
				_, _ = fmt.Fprintf(stderr, "hydro session cleanup: %v\n", err)
			}
			return
		}
		if observer != nil {
			observer.RecordSessionCleanup(deleted)
		}
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func runStaleMediaTempSweeper(ctx context.Context, mediaDir string, stderr io.Writer) {
	for {
		if _, err := storage.SweepStaleFiles(ctx, mediaDir, 24*time.Hour, time.Now()); err != nil && ctx.Err() == nil {
			_, _ = fmt.Fprintf(stderr, "hydro stale media temp cleanup: %v\n", err)
		}
		timer := time.NewTimer(6 * time.Hour)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func refreshIPBlocklist(ctx context.Context, db *sql.DB, store securitystore.BanStore, list *security.IPBlocklist) error {
	entries, err := store.ListActive(ctx, db, time.Now().UTC(), 4096)
	if err != nil {
		return err
	}
	return list.Replace(entries)
}

func runIPBlocklistRefresh(ctx context.Context, db *sql.DB, store securitystore.BanStore, list *security.IPBlocklist) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshCtx, cancel := context.WithTimeout(ctx, time.Second)
			err := refreshIPBlocklist(refreshCtx, db, store, list)
			cancel()
			if err != nil {
				slog.Warn("security blocklist refresh failed", "error", err)
			}
		}
	}
}

func runNewsWorker(ctx context.Context, db database.Executor, worker newsservice.Worker, metrics *observability.Metrics, stderr io.Writer, concurrency int) {
	if concurrency < 1 {
		concurrency = 1
	}
	var workers sync.WaitGroup
	for index := 0; index < concurrency; index++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			runNewsWorkerLoop(ctx, worker, metrics, stderr, newsWorkerID(index))
		}(index)
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	defer workers.Wait()
	for {
		now := time.Now().UTC()
		if err := worker.EnqueueDue(ctx, db, now); err != nil && ctx.Err() == nil {
			metrics.RecordWorkerEnqueueError()
			_, _ = fmt.Fprintf(stderr, "hydro news worker enqueue: %v\n", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func newsWorkerID(index int) string {
	return fmt.Sprintf("news-worker-%d", index)
}

func runNewsWorkerLoop(ctx context.Context, worker newsservice.Worker, metrics *observability.Metrics, stderr io.Writer, workerID string) {
	const idlePoll = 100 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		processed, err := worker.RunOnce(ctx, workerID, time.Now().UTC())
		if err != nil {
			metrics.RecordWorkerFailure()
			if ctx.Err() == nil {
				_, _ = fmt.Fprintf(stderr, "hydro news worker %s run: %v\n", workerID, err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(idlePoll):
			}
			continue
		}
		if processed {
			metrics.RecordWorkerProcessed()
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(idlePoll):
		}
	}
}
