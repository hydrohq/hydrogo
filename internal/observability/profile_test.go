package observability

import "testing"

func TestLoadTracingProfileValidatesBoundedDeploymentConfig(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://collector.example.test:4318")
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.25")
	profile, err := LoadTracingProfile()
	if err != nil {
		t.Fatal(err)
	}
	if !profile.Enabled || profile.Sampling != 0.25 {
		t.Fatalf("unexpected profile: %#v", profile)
	}
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "2")
	if _, err := LoadTracingProfile(); err == nil {
		t.Fatal("invalid sampling accepted")
	}
}
