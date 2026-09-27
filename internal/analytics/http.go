package analytics

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

const (
	ConsentCookie  = "hydro_analytics_consent_id"
	PurposeCookie  = "hydro_analytics_purpose"
	DefaultPurpose = "content-views-v1"
)

// ContentViewMiddleware records only successful public content responses when
// the browser carries a current, opaque consent cookie. It never blocks the
// content response; queue pressure is observable through Dropped().
func ContentViewMiddleware(next http.Handler, queue *ContentViewQueue, contentType string) http.Handler {
	if next == nil || queue == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status < 200 || rec.status >= 300 {
			return
		}
		consent, err := r.Cookie(ConsentCookie)
		if err != nil || strings.TrimSpace(consent.Value) == "" {
			return
		}
		purpose := DefaultPurpose
		if cookie, cookieErr := r.Cookie(PurposeCookie); cookieErr == nil && cookie.Value != "" {
			purpose = cookie.Value
		}
		id := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(id, "/")
		if len(parts) < 4 {
			return
		}
		contentID, locale := parts[len(parts)-1], parts[len(parts)-2]
		if contentID == "html" && len(parts) >= 5 {
			contentID, locale = parts[len(parts)-2], parts[len(parts)-3]
		}
		tenantID := "default"
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			tenantID = scope.TenantID
		}
		_ = queue.Enqueue(ContentViewJob{ConsentID: consent.Value, PurposeVersion: purpose, Event: v1.ContentViewEvent{Day: time.Now().UTC().Format("2006-01-02"), TenantID: tenantID, ContentType: contentType, ContentID: contentID, Locale: locale}})
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
		w.ResponseWriter.WriteHeader(code)
	}
}
func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

type ConsentStore interface {
	SaveConsent(context.Context, string, v1.AnalyticsConsent) error
}

// ConsentHandler is an explicit opt-in endpoint. The opaque cookie is random
// and only its digest is persisted by the SQL store.
func ConsentHandler(store ConsentStore, secure bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		decision := v1.ConsentDecision(strings.TrimSpace(r.FormValue("decision")))
		if decision != v1.ConsentGranted && decision != v1.ConsentRevoked && decision != v1.ConsentRejected {
			http.Error(w, "invalid decision", http.StatusBadRequest)
			return
		}
		id := ""
		if cookie, err := r.Cookie(ConsentCookie); err == nil {
			id = cookie.Value
		}
		if id == "" {
			raw := make([]byte, 24)
			if _, err := rand.Read(raw); err != nil {
				http.Error(w, "consent unavailable", 500)
				return
			}
			id = hex.EncodeToString(raw)
		}
		consent := v1.AnalyticsConsent{Decision: decision, PurposeVersion: DefaultPurpose, DecidedAt: time.Now().UTC()}
		if decision == v1.ConsentRevoked {
			now := time.Now().UTC()
			consent.RevokedAt = &now
		}
		if err := store.SaveConsent(r.Context(), id, consent); err != nil {
			http.Error(w, fmt.Sprintf("consent unavailable: %v", err), 500)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: ConsentCookie, Value: id, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: 31536000})
		http.SetCookie(w, &http.Cookie{Name: PurposeCookie, Value: DefaultPurpose, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: 31536000})
		w.WriteHeader(http.StatusNoContent)
	})
}

type AggregateEventStore interface {
	RecordEvent(context.Context, string, string, v1.AnalyticsEvent) error
}

// ContactClickHandler records only consented, daily aggregate call/WhatsApp
// clicks. It accepts no page URL, account ID, IP, or visitor identifier.
func ContactClickHandler(store AggregateEventStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
			return
		}
		if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "same-site" && site != "none" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || !strings.EqualFold(u.Host, r.Host) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		consent, err := r.Cookie(ConsentCookie)
		if err != nil || consent.Value == "" || store == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		purpose := DefaultPurpose
		if cookie, err := r.Cookie(PurposeCookie); err == nil && cookie.Value != "" {
			purpose = cookie.Value
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		if err := r.ParseForm(); err != nil && err != io.EOF {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		typ := v1.AnalyticsEventType("")
		switch r.FormValue("action") {
		case "call":
			typ = v1.AnalyticsContactCall
		case "whatsapp":
			typ = v1.AnalyticsContactWhatsApp
		default:
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = store.RecordEvent(r.Context(), consent.Value, purpose, v1.AnalyticsEvent{Type: typ, Day: time.Now().UTC().Format("2006-01-02")})
		w.WriteHeader(http.StatusNoContent)
	})
}
