package socialsync

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type Handler struct {
	Service *SyncService
}

func NewHandler(service *SyncService) *Handler {
	return &Handler{Service: service}
}

// HandleGetPosts returns cached social media posts.
func (h *Handler) HandleGetPosts(w http.ResponseWriter, r *http.Request) {
	limit := 12
	if q := r.URL.Query().Get("limit"); q != "" {
		if parsed, err := strconv.Atoi(q); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	posts, err := h.Service.PublicPosts(r.Context(), limit)
	if err != nil {
		http.Error(w, "failed to retrieve posts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": posts,
	})
}

// HandleGetReviews returns cached Google reviews.
func (h *Handler) HandleGetReviews(w http.ResponseWriter, r *http.Request) {
	minRating := 5
	if q := r.URL.Query().Get("min_rating"); q != "" {
		if parsed, err := strconv.Atoi(q); err == nil && parsed >= 1 && parsed <= 5 {
			minRating = parsed
		}
	}

	limit := 10
	if q := r.URL.Query().Get("limit"); q != "" {
		if parsed, err := strconv.Atoi(q); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	reviews, err := h.Service.PublicReviews(r.Context(), minRating, limit)
	if err != nil {
		http.Error(w, "failed to retrieve reviews", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": reviews,
	})
}
