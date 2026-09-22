package capacity

import (
	"sort"
)

// Capability describes the resource and activation boundary of a platform
// capability. It is metadata only; composition roots remain responsible for
// actually binding providers and routes.
type Capability struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Activation string   `json:"activation"`
	Resources  []string `json:"resources,omitempty"`
}

// Manifest returns the stable baseline capability contract for a capacity
// profile. Provider choices (SQLite/PostgreSQL, local/S3, local/Redis) remain
// deployment inputs and are intentionally not inferred from the profile.
func Manifest(profile Profile) ([]Capability, error) {
	if err := profile.Validate(); err != nil {
		return nil, err
	}
	capabilities := []Capability{
		{Name: "core", Kind: "core", Activation: "always", Resources: []string{"http", "config", "health"}},
		{Name: "auth", Kind: "module", Activation: "explicit", Resources: []string{"routes", "sessions", "database"}},
		{Name: "audit", Kind: "module", Activation: "explicit", Resources: []string{"database", "outbox"}},
		{Name: "server-ui", Kind: "presentation", Activation: "explicit", Resources: []string{"templ", "embedded-assets"}},
		{Name: "local-cache", Kind: "provider", Activation: "profile-default", Resources: []string{"memory"}},
	}
	if profile != Compact {
		capabilities = append(capabilities,
			Capability{Name: "bounded-workers", Kind: "runtime", Activation: "profile-default", Resources: []string{"goroutines", "queue"}},
			Capability{Name: "distributed-cache", Kind: "provider", Activation: "opt-in", Resources: []string{"network", "connections"}},
		)
	}
	if profile == Distributed {
		capabilities = append(capabilities,
			Capability{Name: "tenant-admission", Kind: "module", Activation: "opt-in", Resources: []string{"database", "network"}},
			Capability{Name: "otel-export", Kind: "observability", Activation: "opt-in", Resources: []string{"network", "goroutines"}},
		)
	}
	sort.Slice(capabilities, func(i, j int) bool { return capabilities[i].Name < capabilities[j].Name })
	return capabilities, nil
}
