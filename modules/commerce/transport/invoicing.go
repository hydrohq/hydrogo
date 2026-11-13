package transport

import (
	"net/http"

	"github.com/hydrohq/hydrogo/internal/api"
)

func (h Handler) getInvoicingStatus(w http.ResponseWriter, r *http.Request) {
	provider := ""
	if h.InvoiceProvider != nil {
		provider = h.InvoiceProvider.Name()
	}
	api.WriteData(w, http.StatusOK, map[string]any{
		"configured":         h.InvoiceProvider != nil,
		"provider":           provider,
		"auto_issue_on_paid": h.AutoIssueInvoiceOnPaid,
	}, nil)
}

func (h Handler) issueOrderInvoice(w http.ResponseWriter, r *http.Request) {
	if h.InvoiceProvider == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "no invoice provider is configured", nil)
		return
	}
	order, err := h.Service.IssueOrderInvoice(r.Context(), h.InvoiceProvider, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	api.WriteData(w, http.StatusOK, order, nil)
}
