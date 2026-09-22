package observability

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// TracingProfile is deployment-selected configuration. Exporter wiring stays
// outside Core so Compact remains dependency-light; adapters can consume this
// validated profile to install an OTLP provider.
type TracingProfile struct {
	Endpoint string
	Sampling float64
	Enabled  bool
}

func LoadTracingProfile() (TracingProfile, error) {
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	enabled := strings.EqualFold(strings.TrimSpace(os.Getenv("HYDRO_OTEL_ENABLED")), "true") || endpoint != ""
	sampling := 0.1
	if raw := strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER_ARG")); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || value < 0 || value > 1 {
			return TracingProfile{}, errors.New("OTEL_TRACES_SAMPLER_ARG must be between 0 and 1")
		}
		sampling = value
	}
	if endpoint != "" {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return TracingProfile{}, errors.New("OTEL_EXPORTER_OTLP_ENDPOINT must be an absolute URL")
		}
	}
	return TracingProfile{Endpoint: endpoint, Sampling: sampling, Enabled: enabled}, nil
}
