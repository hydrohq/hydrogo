package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/jobs"
	jobstore "github.com/ardaozyildirim/hydrogo/internal/jobs/sqlstore"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func newJobsTestApp(t *testing.T) *serveApp {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "jobs.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	return &serveApp{db: db, cfg: config.Config{Database: config.Database{Dialect: database.SQLite}}, budget: capacity.Budget{QueueEntries: 1000}}
}

func TestAdminJobsHandlersRequireOperationsPermission(t *testing.T) {
	app := newJobsTestApp(t)
	editor := &auth.User{ID: "e1", Role: auth.RoleEditor, Active: true}

	for _, call := range []func(http.ResponseWriter, *http.Request){app.adminJobsHandler, app.adminJobDeadLettersHandler} {
		recorder := httptest.NewRecorder()
		call(recorder, adminAPIRequest(http.MethodGet, "/api/v1/admin/jobs", "", editor))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("editor jobs access: %d", recorder.Code)
		}
	}
}

func TestAdminJobsListsAndManagesDeadLetters(t *testing.T) {
	app := newJobsTestApp(t)
	admin := &auth.User{ID: "a1", Role: auth.RoleAdmin, Active: true}
	store := app.jobsStore()
	now := time.Now().UTC()

	if err := store.Enqueue(context.Background(), app.db, jobs.Job{ID: "job-1", Type: "news.publish", Payload: "{}", AvailableAt: now, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	app.adminJobsHandler(recorder, adminAPIRequest(http.MethodGet, "/api/v1/admin/jobs", "", admin))
	if recorder.Code != http.StatusOK {
		t.Fatalf("list jobs: %d %s", recorder.Code, recorder.Body)
	}
	var listed struct {
		Data struct {
			Jobs []jobstore.JobSummary `json:"jobs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Data.Jobs) != 1 || listed.Data.Jobs[0].ID != "job-1" {
		t.Fatalf("listed jobs = %+v", listed.Data.Jobs)
	}

	_, token, ok, err := store.Claim(context.Background(), app.db, "worker-a", now, time.Minute)
	if err != nil || !ok {
		t.Fatalf("claim: %v/%v", ok, err)
	}
	if err := store.DeadLetter(context.Background(), app.db, "job-1", "worker-a", token, "boom"); err != nil {
		t.Fatal(err)
	}

	deadLetters := httptest.NewRecorder()
	app.adminJobDeadLettersHandler(deadLetters, adminAPIRequest(http.MethodGet, "/api/v1/admin/jobs/dead-letters", "", admin))
	if deadLetters.Code != http.StatusOK || !containsJobID(t, deadLetters, "job-1") {
		t.Fatalf("dead letters: %d %s", deadLetters.Code, deadLetters.Body)
	}

	requeue := httptest.NewRecorder()
	request := adminAPIRequest(http.MethodPost, "/api/v1/admin/jobs/dead-letters/job-1/requeue", "", admin)
	request.SetPathValue("id", "job-1")
	app.adminJobDeadLetterRequeueHandler(requeue, request)
	if requeue.Code != http.StatusOK {
		t.Fatalf("requeue: %d %s", requeue.Code, requeue.Body)
	}

	// Requeued, so it's claimable again and no longer a dead letter. The handler
	// stamped available_at with its own time.Now(), so claim strictly later.
	_, _, ok, err = store.Claim(context.Background(), app.db, "worker-b", time.Now().UTC().Add(time.Second), time.Minute)
	if err != nil || !ok {
		t.Fatalf("reclaim after requeue: %v/%v", ok, err)
	}

	missing := httptest.NewRecorder()
	missingRequest := adminAPIRequest(http.MethodDelete, "/api/v1/admin/jobs/dead-letters/no-such-job", "", admin)
	missingRequest.SetPathValue("id", "no-such-job")
	app.adminJobDeadLetterDeleteHandler(missing, missingRequest)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("delete missing dead letter: %d", missing.Code)
	}
}

func containsJobID(t *testing.T, recorder *httptest.ResponseRecorder, id string) bool {
	t.Helper()
	var body struct {
		Data struct {
			Jobs []jobstore.JobSummary `json:"jobs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, job := range body.Data.Jobs {
		if job.ID == id {
			return true
		}
	}
	return false
}
