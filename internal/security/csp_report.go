package security

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const maxCSPReportBytes int64 = 16 << 10

// CSPReport is the bounded, non-persistent projection accepted from browsers.
// Reports are intentionally not logged because document and source URLs may
// contain user-controlled or sensitive path data.
type CSPReport struct {
	DocumentURI        string `json:"document-uri,omitempty"`
	Referrer           string `json:"referrer,omitempty"`
	ViolatedDirective  string `json:"violated-directive,omitempty"`
	EffectiveDirective string `json:"effective-directive,omitempty"`
	BlockedURI         string `json:"blocked-uri,omitempty"`
	SourceFile         string `json:"source-file,omitempty"`
	LineNumber         int64  `json:"line-number,omitempty"`
	ColumnNumber       int64  `json:"column-number,omitempty"`
}

var ErrInvalidCSPReport = errors.New("invalid CSP report")

// ParseCSPReport accepts both the legacy csp-report envelope and the modern
// direct report shape. Unknown browser-specific fields are ignored, while the
// body remains strictly bounded by the caller.
func ParseCSPReport(body []byte) (CSPReport, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return CSPReport{}, ErrInvalidCSPReport
	}
	payload := body
	if nested, ok := envelope["csp-report"]; ok {
		payload = nested
	}
	var report CSPReport
	if err := json.Unmarshal(payload, &report); err != nil {
		return CSPReport{}, ErrInvalidCSPReport
	}
	for _, value := range []string{report.DocumentURI, report.Referrer, report.ViolatedDirective, report.EffectiveDirective, report.BlockedURI, report.SourceFile} {
		if len(value) > 2048 || strings.ContainsAny(value, "\r\n") {
			return CSPReport{}, ErrInvalidCSPReport
		}
	}
	if report.LineNumber < 0 || report.ColumnNumber < 0 {
		return CSPReport{}, ErrInvalidCSPReport
	}
	return report, nil
}

// CSPReportHandler consumes bounded browser reports without turning them into
// application state. It is safe to expose publicly and deliberately returns a
// generic response to avoid creating an oracle for report contents.
func CSPReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	limited := http.MaxBytesReader(w, r.Body, maxCSPReportBytes)
	var body []byte
	if data, err := io.ReadAll(limited); err == nil {
		body = data
	}
	if _, err := ParseCSPReport(body); err != nil {
		http.Error(w, "invalid CSP report", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
