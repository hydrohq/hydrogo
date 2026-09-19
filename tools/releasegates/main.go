// Command releasegates validates the evidence manifest used to make profile
// readiness claims. It deliberately keeps external gates pending until their
// dated evidence is recorded.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Evidence struct {
	Path    string `json:"path"`
	Command string `json:"command,omitempty"`
}

type Gate struct {
	ID       string     `json:"id"`
	Category string     `json:"category"`
	Status   string     `json:"status"`
	Evidence []Evidence `json:"evidence,omitempty"`
	Note     string     `json:"note,omitempty"`
}

type Profile struct {
	Name          string   `json:"name"`
	Claim         string   `json:"claim"`
	RequiredGates []string `json:"required_gates"`
}

type Manifest struct {
	FormatVersion int       `json:"format_version"`
	GeneratedDate string    `json:"generated_date"`
	Gates         []Gate    `json:"gates"`
	Profiles      []Profile `json:"profiles"`
}

func load(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse gate manifest: %w", err)
	}
	if err := validate(manifest, filepath.Dir(path)); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func validate(manifest Manifest, root string) error {
	if manifest.FormatVersion != 1 || strings.TrimSpace(manifest.GeneratedDate) == "" {
		return errors.New("format_version 1 and generated_date are required")
	}
	gateByID := make(map[string]Gate, len(manifest.Gates))
	for _, gate := range manifest.Gates {
		if strings.TrimSpace(gate.ID) == "" || strings.TrimSpace(gate.Category) == "" {
			return errors.New("every gate requires id and category")
		}
		if _, exists := gateByID[gate.ID]; exists {
			return fmt.Errorf("duplicate gate %q", gate.ID)
		}
		if gate.Status != "verified" && gate.Status != "pending" {
			return fmt.Errorf("gate %q has invalid status %q", gate.ID, gate.Status)
		}
		if gate.Status == "verified" && len(gate.Evidence) == 0 {
			return fmt.Errorf("verified gate %q has no evidence", gate.ID)
		}
		for _, evidence := range gate.Evidence {
			if err := validateEvidence(root, evidence); err != nil {
				return fmt.Errorf("gate %q: %w", gate.ID, err)
			}
		}
		gateByID[gate.ID] = gate
	}
	profiles := make(map[string]struct{}, len(manifest.Profiles))
	for _, profile := range manifest.Profiles {
		if strings.TrimSpace(profile.Name) == "" || (profile.Claim != "candidate" && profile.Claim != "ready") {
			return fmt.Errorf("profile %q has invalid name or claim", profile.Name)
		}
		if _, exists := profiles[profile.Name]; exists {
			return fmt.Errorf("duplicate profile %q", profile.Name)
		}
		profiles[profile.Name] = struct{}{}
		for _, gateID := range profile.RequiredGates {
			gate, exists := gateByID[gateID]
			if !exists {
				return fmt.Errorf("profile %q references unknown gate %q", profile.Name, gateID)
			}
			if profile.Claim == "ready" && gate.Status != "verified" {
				return fmt.Errorf("profile %q claims ready but gate %q is %s", profile.Name, gateID, gate.Status)
			}
		}
	}
	return nil
}

func validateEvidence(root string, evidence Evidence) error {
	path := strings.TrimSpace(evidence.Path)
	if path == "" || filepath.IsAbs(path) {
		return errors.New("evidence path must be relative")
	}
	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("evidence path escapes manifest root: %q", path)
	}
	if _, err := os.Stat(filepath.Join(root, clean)); err != nil {
		return fmt.Errorf("evidence %q is not present: %w", path, err)
	}
	return nil
}

func findProfile(manifest Manifest, name string) (Profile, bool) {
	for _, profile := range manifest.Profiles {
		if profile.Name == name {
			return profile, true
		}
	}
	return Profile{}, false
}

func main() {
	manifestPath := flag.String("manifest", "docs/validation/release-gate-manifest.json", "gate manifest path")
	profileName := flag.String("profile", "", "profile to inspect")
	requireReady := flag.Bool("require-ready", false, "fail unless the selected profile is ready")
	flag.Parse()

	manifest, err := load(*manifestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
	if *profileName == "" {
		fmt.Printf("PASS: validated %d gates and %d profiles\n", len(manifest.Gates), len(manifest.Profiles))
		return
	}
	profile, ok := findProfile(manifest, *profileName)
	if !ok {
		fmt.Fprintf(os.Stderr, "FAIL: unknown profile %q\n", *profileName)
		os.Exit(1)
	}
	if *requireReady && profile.Claim != "ready" {
		fmt.Fprintf(os.Stderr, "FAIL: profile %q is %s, not ready\n", profile.Name, profile.Claim)
		os.Exit(1)
	}
	fmt.Printf("PASS: profile %q is %s; %d required gates validated\n", profile.Name, profile.Claim, len(profile.RequiredGates))
}
