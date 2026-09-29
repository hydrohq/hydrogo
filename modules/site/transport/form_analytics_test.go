package transport

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type formMetricsRecorder struct {
	views       int
	tenantID    string
	formID      string
	visitorKey  string
	viewDay     time.Time
	submissions int
}

func (m *formMetricsRecorder) RecordFormView(_ context.Context, tenantID, formID, visitor string, day time.Time) (bool, error) {
	m.views++
	m.tenantID, m.formID, m.visitorKey, m.viewDay = tenantID, formID, visitor, day
	return true, nil
}
func (m *formMetricsRecorder) RecordFormSubmission(context.Context, database.Executor, string, string, time.Time) error {
	m.submissions++
	return nil
}
func (m *formMetricsRecorder) ListFormDaily(context.Context, string, string, string, string) ([]analytics.FormDailyMetric, error) {
	return nil, nil
}

func TestPublicFormViewUsesDailyPseudonymAndKnownFormOnly(t *testing.T) {
	hasher, err := analytics.NewVisitorHasher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	metrics := &formMetricsRecorder{}
	h := Handler{FormsEnabled: true, AllowedForms: map[string]struct{}{"contact": {}}, FormMetrics: metrics, VisitorHasher: hasher, RequireTenantScope: true}
	request := httptest.NewRequest("POST", "/forms/contact/view", nil)
	request.RemoteAddr = "192.0.2.123:4567"
	request.Header.Set("User-Agent", "Example Browser")
	request.SetPathValue("form", "contact")
	request = request.WithContext(tenant.WithScope(request.Context(), tenant.Scope{TenantID: "tenant-a", Slug: "a"}))
	response := httptest.NewRecorder()
	h.recordFormView(response, request)
	if response.Code != 204 || metrics.views != 1 || metrics.tenantID != "tenant-a" || metrics.formID != "contact" || metrics.visitorKey == "" || len(metrics.visitorKey) != 64 {
		t.Fatalf("view response=%d metrics=%+v", response.Code, metrics)
	}

	unknown := httptest.NewRequest("POST", "/forms/unknown/view", nil)
	unknown.SetPathValue("form", "unknown")
	unknown = unknown.WithContext(tenant.WithScope(unknown.Context(), tenant.Scope{TenantID: "tenant-a", Slug: "a"}))
	unknownResponse := httptest.NewRecorder()
	h.recordFormView(unknownResponse, unknown)
	if unknownResponse.Code != 204 || metrics.views != 1 {
		t.Fatalf("unknown form changed metrics: status=%d metrics=%+v", unknownResponse.Code, metrics)
	}
}
