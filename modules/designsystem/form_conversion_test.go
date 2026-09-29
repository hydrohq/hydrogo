package designsystem

import (
	"strings"
	"testing"
)

func TestFormConversionMetricsShowRateAndAccessibleTrend(t *testing.T) {
	markup := render(t, FormConversionMetrics([]FormConversionMetric{{Title: "Contact", Views: 240, Submissions: 36, ConversionPercent: 15, Trend: []FormConversionDay{{Day: "2026-09-29", Submissions: 4}}}}))
	for _, want := range []string{"240", "36", "15.0%", `role="img"`, "2026-09-29: 4 submissions", "bg-sky-600"} {
		if !strings.Contains(markup, want) {
			t.Errorf("conversion report missing %q: %s", want, markup)
		}
	}
}
