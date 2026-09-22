package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestProfileExplainCompactReportsBoundedBaseline(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := profileCommand([]string{"explain", "--profile", "compact"}, &stdout, &stderr); err != nil {
		t.Fatalf("profile explain: %v; stderr=%s", err, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{"profile=compact", "db_max_open=1", "workers=1", "queue_entries=10000", "capability=core", "capability=auth"} {
		if !strings.Contains(output, want) {
			t.Fatalf("profile explain missing %q: %s", want, output)
		}
	}
	for _, notWant := range []string{"capability=distributed-cache", "capability=otel-export"} {
		if strings.Contains(output, notWant) {
			t.Fatalf("Compact profile unexpectedly reports %q: %s", notWant, output)
		}
	}
}

func TestProfileCheckJSONIsMachineReadable(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := profileCommand([]string{"check", "--profile", "distributed", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("profile check: %v; stderr=%s", err, stderr.String())
	}
	var result struct {
		Valid        bool   `json:"valid"`
		Profile      string `json:"profile"`
		Capabilities []struct {
			Name string `json:"name"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode profile check JSON: %v; output=%s", err, stdout.String())
	}
	if !result.Valid || result.Profile != "distributed" {
		t.Fatalf("unexpected profile result: %+v", result)
	}
	if !containsProfileCapability(result.Capabilities, "otel-export") || !containsProfileCapability(result.Capabilities, "tenant-admission") {
		t.Fatalf("distributed-only capabilities missing: %+v", result.Capabilities)
	}
}

func TestProfileCommandRejectsUnknownProfileAndSubcommand(t *testing.T) {
	for _, args := range [][]string{{"explain", "--profile", "unknown"}, {"dump"}} {
		var stdout, stderr bytes.Buffer
		if err := profileCommand(args, &stdout, &stderr); err == nil {
			t.Fatalf("profile command accepted %v", args)
		}
	}
}

func containsProfileCapability(capabilities []struct {
	Name string `json:"name"`
}, name string) bool {
	for _, capability := range capabilities {
		if capability.Name == name {
			return true
		}
	}
	return false
}
