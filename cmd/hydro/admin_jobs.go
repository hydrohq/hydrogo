package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	jobstore "github.com/ardaozyildirim/hydrogo/internal/jobs/sqlstore"
)

// jobsStore builds the durable job store for the admin handlers below, using
// the same dialect and queue capacity as the server's own workers.
func (a *serveApp) jobsStore() jobstore.Store {
	return jobstore.Store{Dialect: a.cfg.Database.Dialect, MaxEntries: a.budget.QueueEntries}
}

func jobsListLimit(r *http.Request) int {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}
	return limit
}

// adminJobsHandler lists claimable and leased jobs (never dead-lettered).
func (a *serveApp) adminJobsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !activePlatformAdmin(user) || !auth.Can(user, auth.PermissionOperations) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	jobs, err := a.jobsStore().List(r.Context(), a.db, jobsListLimit(r))
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "list jobs: "+err.Error(), nil)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, map[string]any{"jobs": jobs}, nil)
}

// adminJobDeadLettersHandler lists dead-lettered jobs awaiting operator review.
func (a *serveApp) adminJobDeadLettersHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !activePlatformAdmin(user) || !auth.Can(user, auth.PermissionOperations) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	listed, err := a.jobsStore().ListDeadLetters(r.Context(), a.db, jobsListLimit(r))
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "list dead-letter jobs: "+err.Error(), nil)
		return
	}
	summaries := make([]jobstore.JobSummary, len(listed))
	for i, job := range listed {
		summaries[i] = jobstore.JobSummaryOf(job)
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, map[string]any{"jobs": summaries}, nil)
}

// adminJobDeadLetterRequeueHandler makes one dead-letter job claimable again.
func (a *serveApp) adminJobDeadLetterRequeueHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !activePlatformAdmin(user) || !auth.Can(user, auth.PermissionOperations) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	id := r.PathValue("id")
	if id == "" {
		api.WriteError(w, r, http.StatusBadRequest, "job id is required", nil)
		return
	}
	if err := a.jobsStore().RequeueDeadLetter(r.Context(), a.db, id, time.Now().UTC()); err != nil {
		api.WriteError(w, r, http.StatusNotFound, "requeue job: "+err.Error(), nil)
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{"id": id}, nil)
}

// adminJobDeadLetterDeleteHandler permanently removes one dead-letter job.
func (a *serveApp) adminJobDeadLetterDeleteHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !activePlatformAdmin(user) || !auth.Can(user, auth.PermissionOperations) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	id := r.PathValue("id")
	if id == "" {
		api.WriteError(w, r, http.StatusBadRequest, "job id is required", nil)
		return
	}
	if err := a.jobsStore().DeleteDeadLetter(r.Context(), a.db, id); err != nil {
		api.WriteError(w, r, http.StatusNotFound, "delete job: "+err.Error(), nil)
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{"id": id}, nil)
}
